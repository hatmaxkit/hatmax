// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package mailer

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"math/big"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

type smtpTestPolicy struct {
	implicit bool
	startTLS bool
	reject   bool
	stall    string
}

type smtpTestResult struct {
	commands  []string
	encrypted bool
	raw       string
	err       error
}

// Encryption policy must be enforced before authentication or message delivery.
// Optional STARTTLS preserves SendMail's opportunistic upgrade behavior.
func TestSMTPEncryption(t *testing.T) {
	for _, tc := range []struct {
		name     string
		server   smtpTestPolicy
		tls      bool
		required bool
		skip     bool
		wantErr  string
	}{
		{name: "required unavailable", required: true, wantErr: "STARTTLS is required"},
		{name: "required upgrade", server: smtpTestPolicy{startTLS: true}, required: true, skip: true},
		{name: "optional upgrade", server: smtpTestPolicy{startTLS: true}, skip: true},
		{name: "plaintext permitted"},
		{name: "required rejected", server: smtpTestPolicy{startTLS: true, reject: true}, required: true, wantErr: "smtp starttls"},
		{name: "optional rejected", server: smtpTestPolicy{startTLS: true, reject: true}, wantErr: "smtp starttls"},
		{name: "upgrade certificate", server: smtpTestPolicy{startTLS: true}, required: true, wantErr: "certificate"},
		{name: "optional certificate", server: smtpTestPolicy{startTLS: true}, wantErr: "certificate"},
		{name: "implicit TLS", server: smtpTestPolicy{implicit: true}, tls: true, skip: true},
		{name: "both flags", server: smtpTestPolicy{implicit: true}, tls: true, required: true, skip: true},
		{name: "implicit certificate", server: smtpTestPolicy{implicit: true}, tls: true, wantErr: "certificate"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, result, _ := smtpTestServer(t, tc.server)

			cfg.TLS, cfg.StartTLS, cfg.InsecureSkipVerify = tc.tls, tc.required, tc.skip
			if tc.server.implicit || tc.server.startTLS || tc.required {
				cfg.Username, cfg.Password = "test-user", "test-password"
			}

			err := NewSMTPMailer(cfg).Send(context.Background(), senderMessage(Address{}, false, ""))
			got := <-result

			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("Send: got %v; want %q", err, tc.wantErr)
				}

				for _, command := range got.commands {
					if strings.HasPrefix(command, "AUTH") || strings.HasPrefix(command, "MAIL") || command == "DATA" {
						t.Fatalf("failed encryption reached %q", command)
					}
				}

				return
			}

			if err != nil || got.err != nil {
				t.Fatalf("Send: %v; server: %v", err, got.err)
			}

			if got.encrypted != (tc.server.implicit || tc.server.startTLS) {
				t.Fatalf("encrypted delivery: %v", got.encrypted)
			}

			if !strings.Contains(got.raw, "From: sender@example.com") {
				t.Fatalf("missing effective sender: %q", got.raw)
			}
		})
	}
}

// A canceled caller must not open a connection or deliver any message.
func TestSMTPCanceled(t *testing.T) {
	cfg, _, _ := smtpTestServer(t, smtpTestPolicy{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := NewSMTPMailer(cfg).Send(ctx, senderMessage(Address{}, false, ""))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Send: got %v; want context.Canceled", err)
	}
}

// Cancellation must interrupt every blocking transport phase and close the
// server connection. Caller deadlines must also interrupt a stalled DATA reply.
func TestSMTPInterrupt(t *testing.T) {
	for _, tc := range []struct {
		name     string
		server   smtpTestPolicy
		deadline bool
	}{
		{name: "greeting", server: smtpTestPolicy{stall: "greeting"}},
		{name: "hello", server: smtpTestPolicy{stall: "EHLO"}},
		{name: "implicit handshake", server: smtpTestPolicy{implicit: true, stall: "handshake"}},
		{name: "upgrade handshake", server: smtpTestPolicy{startTLS: true, stall: "handshake"}},
		{name: "authentication", server: smtpTestPolicy{startTLS: true, stall: "AUTH"}},
		{name: "data command", server: smtpTestPolicy{stall: "DATA"}},
		{name: "data reply", server: smtpTestPolicy{stall: "body"}},
		{name: "quit", server: smtpTestPolicy{stall: "QUIT"}},
		{name: "deadline", server: smtpTestPolicy{stall: "body"}, deadline: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, result, stalled := smtpTestServer(t, tc.server)

			cfg.TLS, cfg.StartTLS, cfg.InsecureSkipVerify = tc.server.implicit, tc.server.startTLS, true
			if tc.server.stall == "AUTH" {
				cfg.Username, cfg.Password = "test-user", "test-password"
			}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			want := context.Canceled

			if tc.deadline {
				cancel()

				ctx, cancel = context.WithTimeout(context.Background(), 100*time.Millisecond)
				defer cancel()

				want = context.DeadlineExceeded
			}

			done := make(chan error, 1)
			go func() { done <- NewSMTPMailer(cfg).Send(ctx, senderMessage(Address{}, false, "")) }()

			select {
			case <-stalled:
			case err := <-done:
				t.Fatalf("Send stopped before target phase: %v", err)
			case <-time.After(time.Second):
				t.Fatal("server did not reach target phase")
			}

			if !tc.deadline {
				cancel()
			}

			select {
			case err := <-done:
				if !errors.Is(err, want) {
					t.Fatalf("Send: got %v; want %v", err, want)
				}
			case <-time.After(time.Second):
				t.Fatal("Send did not stop")
			}

			select {
			case got := <-result:
				if got.err != nil {
					t.Fatalf("server connection did not close: %v", got.err)
				}
			case <-time.After(time.Second):
				t.Fatal("server connection remained open")
			}
		})
	}
}

func smtpTestServer(t *testing.T, policy smtpTestPolicy) (SMTPConfig, <-chan smtpTestResult, <-chan struct{}) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}

	certificate := smtpTestCertificate(t)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	result := make(chan smtpTestResult, 1)
	stalled := make(chan struct{}, 1)

	t.Cleanup(func() {
		cancel()
		listener.Close()
		<-done
	})

	go func() {
		defer close(done)

		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}
		defer conn.Close()

		stop := context.AfterFunc(ctx, func() { conn.Close() })
		defer stop()

		got := smtpTestResult{}

		got.err = conn.SetDeadline(time.Now().Add(3 * time.Second))
		if got.err == nil {
			got.err = smtpTestServe(conn, policy, certificate, stalled, &got)
		}

		result <- got
	}()

	host, portText, err := net.SplitHostPort(listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}

	port, err := strconv.Atoi(portText)
	if err != nil {
		t.Fatal(err)
	}

	return SMTPConfig{Host: host, Port: port, Config: Config{DefaultFrom: Address{Email: "sender@example.com"}}}, result, stalled
}

func smtpTestServe(conn net.Conn, policy smtpTestPolicy, certificate tls.Certificate, stalled chan<- struct{}, got *smtpTestResult) error {
	stall := func() error {
		stalled <- struct{}{}

		_, err := io.Copy(io.Discard, conn)
		// Closing during a handshake may reset unread TLS bytes instead of EOF.
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, syscall.ECONNRESET) {
			return nil
		}

		return err
	}

	upgrade := func() error {
		if policy.stall == "handshake" {
			return stall()
		}

		tlsConn := tls.Server(conn, &tls.Config{Certificates: []tls.Certificate{certificate}})

		err := tlsConn.Handshake()
		if err != nil {
			return err
		}

		conn = tlsConn
		got.encrypted = true

		return nil
	}
	if policy.implicit {
		err := upgrade()
		if err != nil || policy.stall == "handshake" {
			return err
		}
	}

	if policy.stall == "greeting" {
		return stall()
	}

	protocol := textproto.NewConn(conn)

	err := protocol.PrintfLine("220 local SMTP")
	for err == nil {
		var command string

		command, err = protocol.ReadLine()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}

			return err
		}

		verb := strings.Fields(command)[0]
		got.commands = append(got.commands, command)

		if policy.stall == verb {
			return stall()
		}

		switch verb {
		case "EHLO":
			err = protocol.PrintfLine("250-local SMTP")
			if err == nil && policy.startTLS && !got.encrypted {
				err = protocol.PrintfLine("250-STARTTLS")
			}

			if err == nil {
				err = protocol.PrintfLine("250 AUTH PLAIN")
			}
		case "STARTTLS":
			if policy.reject {
				err = protocol.PrintfLine("454 TLS unavailable")

				continue
			}

			err = protocol.PrintfLine("220 Ready")
			if err == nil {
				err = upgrade()
			}

			if err != nil || policy.stall == "handshake" {
				return err
			}

			protocol = textproto.NewConn(conn)
		case "AUTH":
			err = protocol.PrintfLine("235 Authenticated")
		case "MAIL", "RCPT":
			err = protocol.PrintfLine("250 OK")
		case "DATA":
			err = protocol.PrintfLine("354 Send body")
			if err == nil {
				var raw []byte

				raw, err = protocol.ReadDotBytes()
				got.raw = string(raw)
			}

			if err == nil && policy.stall == "body" {
				return stall()
			}

			if err == nil {
				err = protocol.PrintfLine("250 Accepted")
			}
		case "QUIT":
			return protocol.PrintfLine("221 Goodbye")
		default:
			err = protocol.PrintfLine("500 Unsupported")
		}
	}

	return err
}

func smtpTestCertificate(t *testing.T) tls.Certificate {
	t.Helper()

	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	certificate := &x509.Certificate{
		SerialNumber: big.NewInt(1), NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		KeyUsage:    x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	der, err := x509.CreateCertificate(rand.Reader, certificate, certificate, public, private)
	if err != nil {
		t.Fatal(err)
	}

	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: private}
}
