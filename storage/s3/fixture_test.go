//go:build integration

// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

package s3

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"debug/buildinfo"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"
)

var live *remoteFixture

type remoteFixture struct {
	root       string
	endpoint   string
	bucket     string
	provider   aws.CredentialsProvider
	admin      *sdk.Client
	httpClient *http.Client
	command    *exec.Cmd
	wait       chan error
	waited     bool
	created    bool
	log        *os.File
	receipt    *os.File
	recordErr  error
}

// TestMain owns one identified native server and retains cleanup receipts.
func TestMain(m *testing.M) {
	fixture := &remoteFixture{}

	err := fixture.start()
	if err != nil {
		fmt.Fprintln(os.Stderr, "S3 fixture startup:", err)

		_ = fixture.finish()

		os.Exit(1)
	}

	live = fixture
	result := m.Run()

	err = fixture.finish()
	if err != nil {
		fmt.Fprintln(os.Stderr, "S3 fixture cleanup:", err)

		result = 1
	}

	os.Exit(result)
}

func (f *remoteFixture) start() error {
	binary := os.Getenv("HATMAX_STORAGE_MINIO")

	receipt := os.Getenv("HATMAX_STORAGE_FIXTURE_RECEIPT")
	if binary == "" || receipt == "" {
		return errors.New("HATMAX_STORAGE_MINIO and HATMAX_STORAGE_FIXTURE_RECEIPT are required; run the slice runner")
	}

	info, err := buildinfo.ReadFile(binary)
	if err != nil {
		return err
	}

	if info.GoVersion != "go1.27.1" || info.Main.Path != "github.com/minio/minio" || info.Main.Version != "v0.0.0-20251015172955-9e49d5e7a648" || info.Main.Sum != "h1:6TdolSCLSs2nwm8i0PpWDqf9iX2Ty9WQK8wmr7dCnUM=" {
		return errors.New("fixture executable source/module/toolchain identity does not match the approved pin")
	}

	f.receipt, err = os.OpenFile(receipt, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	digest, err := fileHash(binary)
	if err != nil {
		return err
	}

	f.record("source: 9e49d5e7a648f00e26f2246f4dc28e6b07f8c84a\nmodule: %s\nmodule-sum: %s\ngo: %s\nbinary-sha256: %s\n", info.Main.Version, info.Main.Sum, info.GoVersion, digest)

	f.root, err = os.MkdirTemp("", "hatmax-s3-fixture-")
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}

	address := listener.Addr().String()

	err = listener.Close()
	if err != nil {
		return err
	}

	access := "fixture-" + rand.Text()
	secret := rand.Text() + rand.Text()
	f.provider = credentials.NewStaticCredentialsProvider(access, secret, "")
	f.endpoint = "http://" + address
	f.bucket = "storage-" + strings.ToLower(rand.Text())

	f.log, err = os.OpenFile(filepath.Join(f.root, "server.log"), os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}

	f.command = exec.Command(binary, "--quiet", "--anonymous", "--config-dir", filepath.Join(f.root, "config"), "--certs-dir", filepath.Join(f.root, "certs"), "server", "--address", address, "--console-address", "127.0.0.1:0", filepath.Join(f.root, "data"))

	f.command.Env = append(os.Environ(), "MINIO_ROOT_USER="+access, "MINIO_ROOT_PASSWORD="+secret, "MINIO_BROWSER=off")
	f.command.Stdout = f.log
	f.command.Stderr = f.log
	f.record("server: starting\n")

	startup, cancelStartup := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelStartup()

	f.wait, err = startFixtureProcess(f.command)
	if err != nil {
		return err
	}

	f.httpClient = &http.Client{Timeout: time.Second}

	for {
		select {
		case <-f.wait:
			f.waited = true

			return errors.New("owned MinIO process exited before readiness; private server log retained until teardown")
		default:
		}

		request, requestErr := http.NewRequestWithContext(startup, http.MethodGet, f.endpoint+"/minio/health/ready", nil)
		if requestErr != nil {
			return requestErr
		}

		response, requestErr := f.httpClient.Do(request)
		if requestErr == nil {
			_ = response.Body.Close()

			if response.StatusCode == http.StatusOK {
				break
			}
		}

		select {
		case <-startup.Done():
			return errors.New("owned MinIO readiness exceeded 60 seconds")
		case <-time.After(100 * time.Millisecond):
		}
	}

	f.httpClient.Timeout = 10 * time.Second
	f.admin = sdk.NewFromConfig(aws.Config{Region: "us-east-1", Credentials: f.provider, HTTPClient: f.httpClient}, func(o *sdk.Options) {
		o.BaseEndpoint = aws.String(f.endpoint)
		o.UsePathStyle = true
		o.Retryer = aws.NopRetryer{}
	})

	_, err = f.admin.CreateBucket(startup, &sdk.CreateBucketInput{Bucket: aws.String(f.bucket)})
	if err != nil {
		return fmt.Errorf("create owned private fixture bucket: %w", err)
	}

	f.created = true
	f.record("server: ready\nbucket: created private\n")

	return nil
}

func (f *remoteFixture) record(format string, args ...any) {
	if f.receipt != nil {
		_, err := fmt.Fprintf(f.receipt, format, args...)
		f.recordErr = errors.Join(f.recordErr, err)
	}
}

func (f *remoteFixture) cleanBucket() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	objects, err := f.admin.ListObjectsV2(ctx, &sdk.ListObjectsV2Input{Bucket: aws.String(f.bucket), MaxKeys: aws.Int32(1000)})
	if err != nil {
		return err
	}

	if aws.ToBool(objects.IsTruncated) {
		return errors.New("owned fixture exceeded its finite 1000-object cleanup bound")
	}

	f.record("objects-before-cleanup: %d\n", len(objects.Contents))

	for _, object := range objects.Contents {
		_, err = f.admin.DeleteObject(ctx, &sdk.DeleteObjectInput{Bucket: aws.String(f.bucket), Key: object.Key})
		if err != nil {
			return err
		}
	}

	objects, err = f.admin.ListObjectsV2(ctx, &sdk.ListObjectsV2Input{Bucket: aws.String(f.bucket), MaxKeys: aws.Int32(1)})
	if err != nil || len(objects.Contents) != 0 || aws.ToBool(objects.IsTruncated) {
		return errors.New("owned fixture object cleanup was not confirmed")
	}

	_, err = f.admin.DeleteBucket(ctx, &sdk.DeleteBucketInput{Bucket: aws.String(f.bucket)})
	if err != nil {
		return err
	}

	f.record("objects-after-cleanup: 0\nbucket: removed\n")

	return nil
}

func (f *remoteFixture) stop() error {
	if f.command == nil || f.command.Process == nil || f.waited {
		return nil
	}

	_ = f.command.Process.Signal(syscall.SIGTERM)

	select {
	case <-f.wait:
		f.waited = true
		f.record("server: stopped\n")

		return nil
	case <-time.After(30 * time.Second):
		_ = f.command.Process.Kill()
		<-f.wait
		f.waited = true

		return errors.New("owned server exceeded the 30-second teardown bound")
	}
}

func (f *remoteFixture) finish() error {
	var result error

	if f.created {
		result = f.cleanBucket()
	}

	result = errors.Join(result, f.stop())

	if f.httpClient != nil {
		f.httpClient.CloseIdleConnections()
	}

	if f.log != nil {
		result = errors.Join(result, f.log.Close())
		digest, err := fileHash(filepath.Join(f.root, "server.log"))
		result = errors.Join(result, err)

		f.record("server-log-sha256: %s\n", digest)
	}

	if f.root != "" {
		result = errors.Join(result, os.RemoveAll(f.root))

		_, err := os.Stat(f.root)
		if !errors.Is(err, os.ErrNotExist) {
			result = errors.Join(result, errors.New("owned fixture root removal not confirmed"))
		} else {
			f.record("temporary-root: removed\n")
		}
	}

	if f.receipt != nil {
		result = errors.Join(result, f.recordErr, f.receipt.Close())
	}

	return result
}

func fileHash(name string) (string, error) {
	file, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()

	_, err = io.Copy(hash, file)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func liveStore(t *testing.T) (*Store, Options) {
	t.Helper()
	options := testOptions(t, live.endpoint)
	options.Bucket = live.bucket
	options.Credentials = live.provider

	return newTestStore(t, options), options
}
