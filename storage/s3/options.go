// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: Apache-2.0
//
// This file is part of Hatmax. See LICENSE for license terms.

// Package s3 implements bounded private object I/O with the AWS SDK for Go v2.
package s3

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	sdk "github.com/aws/aws-sdk-go-v2/service/s3"

	"hatmax.adrianpk.com/storage"
)

// Options owns the private bucket, endpoint, credentials and per-operation limits.
// Credentials may be omitted to use the SDK's default credential chain, including
// renewable credentials. Region, SpoolDirectory and MaxSpoolBytes are required.
// Each write's MaxBytes must fit MaxSpoolBytes. N concurrent writes consume up to
// N times that disk capacity plus fixed-size copy buffers and filesystem overhead.
// Callers own aggregate quotas and admission. The spool directory must exist and
// remain application-controlled; files have mode 0600 and are removed on return.
type Options struct {
	Bucket                string
	Region                string
	Endpoint              string
	Credentials           aws.CredentialsProvider
	SpoolDirectory        string
	MaxSpoolBytes         int64
	ReadAttempts          int
	OperationTimeout      time.Duration
	AllowInsecureLoopback bool
}

// Store implements storage.Store in a private, initially unversioned bucket.
// Delete removes the current key; it does not erase historical bucket versions.
// Close after all operations and returned readers finish. Construction neither
// creates a bucket nor probes remote permissions, and starts no workers.
type Store struct {
	client       *sdk.Client
	bucket       string
	spool        *os.Root
	capacity     int64
	readAttempts int
	timeout      time.Duration
	httpClient   *http.Client
}

// NewStore validates configuration and opens the owned spool directory. HTTPS
// uses normal certificate verification. Plain HTTP requires the explicit fixture
// option and a literal loopback address. Custom endpoints use path addressing.
// ReadAttempts defaults to 1 and admits 1-5. OperationTimeout defaults to 30s and
// admits at most 120s; it also bounds returned readers. Mutation attempts are
// always exactly one, independently of SDK environment retry settings.
func NewStore(ctx context.Context, options Options) (*Store, error) {
	if options.ReadAttempts == 0 {
		options.ReadAttempts = 1
	}

	if options.OperationTimeout == 0 {
		options.OperationTimeout = 30 * time.Second
	}

	err := validateOptions(options)
	if err != nil {
		return nil, err
	}

	err = ctx.Err()
	if err != nil {
		return nil, err
	}

	root, err := os.OpenRoot(options.SpoolDirectory)
	if err != nil {
		return nil, fmt.Errorf("open S3 spool: %w", err)
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: options.OperationTimeout, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          10,
		MaxIdleConnsPerHost:   2,
		IdleConnTimeout:       30 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ResponseHeaderTimeout: options.OperationTimeout,
		ExpectContinueTimeout: time.Second,
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   options.OperationTimeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	load := []func(*config.LoadOptions) error{config.WithRegion(options.Region), config.WithHTTPClient(client)}
	if options.Credentials != nil {
		load = append(load, config.WithCredentialsProvider(options.Credentials))
	}

	cfg, err := config.LoadDefaultConfig(ctx, load...)
	if err != nil {
		_ = root.Close()

		transport.CloseIdleConnections()

		return nil, fmt.Errorf("S3 configuration: %w", err)
	}

	api := sdk.NewFromConfig(cfg, func(o *sdk.Options) {
		o.Retryer = aws.NopRetryer{}
		o.RequestChecksumCalculation = aws.RequestChecksumCalculationWhenRequired
		o.DisableLogOutputChecksumValidationSkipped = true

		if options.Endpoint != "" {
			o.BaseEndpoint = aws.String(strings.TrimSuffix(options.Endpoint, "/"))
			o.UsePathStyle = true
		}
	})

	return &Store{client: api, bucket: options.Bucket, spool: root, capacity: options.MaxSpoolBytes, readAttempts: options.ReadAttempts, timeout: options.OperationTimeout, httpClient: client}, nil
}

func validateOptions(options Options) error {
	err := (storage.ReadOptions{MaxBytes: options.MaxSpoolBytes}).Validate()
	if err != nil || options.SpoolDirectory == "" || strings.TrimSpace(options.Region) == "" || options.ReadAttempts < 1 || options.ReadAttempts > 5 || options.OperationTimeout <= 0 || options.OperationTimeout > 120*time.Second {
		return fmt.Errorf("S3 limits or configuration: %w", storage.ErrInvalid)
	}

	if len(options.Bucket) < 3 || len(options.Bucket) > 63 || strings.Contains(options.Bucket, "..") || net.ParseIP(options.Bucket) != nil {
		return fmt.Errorf("S3 bucket: %w", storage.ErrInvalid)
	}

	for i := range len(options.Bucket) {
		c := options.Bucket[i]

		alphanumeric := c >= 'a' && c <= 'z' || c >= '0' && c <= '9'
		if !alphanumeric && (c != '-' && c != '.' || i == 0 || i == len(options.Bucket)-1) {
			return fmt.Errorf("S3 bucket: %w", storage.ErrInvalid)
		}
	}

	if options.Endpoint == "" {
		return nil
	}

	endpoint, err := url.Parse(options.Endpoint)
	if err != nil || endpoint.Hostname() == "" || endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" || endpoint.Path != "" && endpoint.Path != "/" {
		return fmt.Errorf("S3 endpoint: %w", storage.ErrInvalid)
	}

	if endpoint.Scheme == "https" {
		return nil
	}

	ip := net.ParseIP(endpoint.Hostname())
	if endpoint.Scheme != "http" || !options.AllowInsecureLoopback || ip == nil || !ip.IsLoopback() {
		return fmt.Errorf("S3 endpoint TLS policy: %w", storage.ErrInvalid)
	}

	return nil
}

// Close releases the spool handle and owned idle HTTP connections. It does not
// close caller-owned input or response readers, or remove the configured root.
func (s *Store) Close() error {
	s.httpClient.CloseIdleConnections()

	return s.spool.Close()
}

var _ storage.Store = (*Store)(nil)
