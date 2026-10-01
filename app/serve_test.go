// SPDX-FileCopyrightText: 2026 Adrian PK
// SPDX-License-Identifier: GPL-3.0-only
//
// This file is part of Hatmax. See COPYING for license terms.

package app

import (
	"bufio"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"hatmax.adrianpk.com/log"
)

// Serving fills only missing connection bounds and never rewrites explicit
// request-body, response, address, or handler policy on the caller's server.
func TestServeBounds(t *testing.T) {
	for _, test := range []struct {
		name         string
		header, idle time.Duration
		wantHeader   time.Duration
		wantIdle     time.Duration
	}{
		{"defaults", 0, 0, 5 * time.Second, time.Minute},
		{"custom", time.Second, 2 * time.Second, time.Second, 2 * time.Second},
		{"header only", time.Second, 0, time.Second, time.Minute},
		{"idle only", 0, time.Second, 5 * time.Second, time.Second},
	} {
		t.Run(test.name, func(t *testing.T) {
			handler := http.NewServeMux()
			server := &http.Server{Addr: "127.0.0.1:0", Handler: handler,
				ReadHeaderTimeout: test.header, IdleTimeout: test.idle,
				ReadTimeout: 3 * time.Second, WriteTimeout: 4 * time.Second, MaxHeaderBytes: 4096}
			// A pre-shutdown server avoids a listener while exercising the real Serve exit.
			err := server.Shutdown(context.Background())
			if err != nil {
				t.Fatal(err)
			}

			err = Serve(server)
			if err != nil {
				t.Fatalf("normal closed-server exit: %v", err)
			}

			if server.ReadHeaderTimeout != test.wantHeader || server.IdleTimeout != test.wantIdle ||
				server.ReadTimeout != 3*time.Second || server.WriteTimeout != 4*time.Second ||
				server.Handler != handler || server.Addr != "127.0.0.1:0" || server.MaxHeaderBytes != 4096 {
				t.Fatalf("caller configuration lost: %+v", server)
			}
		})
	}
}

// Invalid connection policy fails before listening or partially applying defaults.
func TestServeInvalid(t *testing.T) {
	for _, test := range []struct {
		name   string
		server *http.Server
	}{
		{"nil", nil},
		{"negative header", &http.Server{ReadHeaderTimeout: -time.Second}},
		{"negative idle", &http.Server{IdleTimeout: -time.Second}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var header, idle time.Duration
			if test.server != nil {
				header, idle = test.server.ReadHeaderTimeout, test.server.IdleTimeout
			}

			err := Serve(test.server)
			if err == nil {
				t.Fatal("invalid server was accepted")
			}

			if test.server != nil && (test.server.ReadHeaderTimeout != header || test.server.IdleTimeout != idle) {
				t.Fatal("rejected configuration was partially defaulted")
			}
		})
	}
}

// Listen failures retain their underlying network error rather than being
// confused with normal HTTP shutdown.
func TestServeListenError(t *testing.T) {
	server := &http.Server{Addr: "127.0.0.1:invalid"}
	err := Serve(server)

	var listenErr *net.OpError
	if !errors.As(err, &listenErr) || listenErr.Op != "listen" {
		t.Fatalf("listen failure = %v, want a listen operation error", err)
	}
}

// Incomplete request headers and idle keep-alive connections close on the
// configured server deadlines, not on the client's larger safety deadline.
func TestServeConnections(t *testing.T) {
	for _, kind := range []string{"headers", "idle"} {
		t.Run(kind, func(t *testing.T) {
			var calls atomic.Int32

			server := &http.Server{ReadHeaderTimeout: 30 * time.Millisecond, IdleTimeout: 30 * time.Millisecond,
				Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					calls.Add(1)

					_, _ = io.WriteString(w, "ok")
				})}
			address, _ := serveHTTP(t, server)

			conn, err := net.DialTimeout("tcp", address, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()

			err = conn.SetDeadline(time.Now().Add(2 * time.Second))
			if err != nil {
				t.Fatal(err)
			}

			request := "GET / HTTP/1.1\r\nHost: local\r\nX-Slow: "
			if kind == "idle" {
				request = "GET / HTTP/1.1\r\nHost: local\r\n\r\n"
			}

			_, err = io.WriteString(conn, request)
			if err != nil {
				t.Fatal(err)
			}

			reader := bufio.NewReader(conn)
			if kind == "idle" {
				response, readErr := http.ReadResponse(reader, nil)
				if readErr != nil {
					t.Fatal(readErr)
				}

				body, bodyErr := io.ReadAll(response.Body)
				response.Body.Close()

				if bodyErr != nil || response.StatusCode != http.StatusOK || string(body) != "ok" {
					t.Fatalf("normal request failed: %s / %s / %v", response.Status, body, bodyErr)
				}
			}

			_, err = io.ReadAll(reader)

			var timeout net.Error
			if errors.As(err, &timeout) && timeout.Timeout() {
				t.Fatal("server did not close the bounded connection")
			}

			wantCalls := int32(0)
			if kind == "idle" {
				wantCalls = 1
			}

			if calls.Load() != wantCalls {
				t.Fatalf("handler calls = %d, want %d", calls.Load(), wantCalls)
			}
		})
	}
}

// Header/idle limits do not impose a deadline on an active streaming response.
func TestServeStream(t *testing.T) {
	release := make(chan struct{})

	var once sync.Once

	server := &http.Server{ReadHeaderTimeout: 30 * time.Millisecond, IdleTimeout: 30 * time.Millisecond,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "data: first\n\n")
			w.(http.Flusher).Flush()
			<-release

			_, _ = io.WriteString(w, "data: last\n\n")
		})}
	address, _ := serveHTTP(t, server)
	t.Cleanup(func() { once.Do(func() { close(release) }) })

	client := &http.Client{Timeout: 2 * time.Second}
	defer client.CloseIdleConnections()

	response, err := client.Get("http://" + address)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	reader := bufio.NewReader(response.Body)

	first, err := reader.ReadString('\n')
	if err != nil || first != "data: first\n" || response.StatusCode != http.StatusOK {
		t.Fatalf("initial stream failed: %q / %v", first, err)
	}

	time.Sleep(3 * server.ReadHeaderTimeout)
	once.Do(func() { close(release) })

	tail, err := io.ReadAll(reader)
	if err != nil || !strings.Contains(string(tail), "data: last\n\n") {
		t.Fatalf("active stream was timed out: %q / %v", tail, err)
	}

	if server.ReadTimeout != 0 || server.WriteTimeout != 0 {
		t.Fatal("Serve installed a body or whole-response timeout")
	}
}

// Shutdown closes the exact serving listener, lets an admitted request finish,
// and only then stops dependencies. Serve normalizes the owned server's close.
func TestServeShutdown(t *testing.T) {
	entered := make(chan struct{})
	release := make(chan struct{})
	shutdownStarted := make(chan struct{})
	shutdownDone := make(chan struct{})

	var (
		once    sync.Once
		stopped atomic.Bool
	)

	server := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(entered)
		<-release

		_, _ = io.WriteString(w, "finished")
	})}
	server.RegisterOnShutdown(func() { close(shutdownStarted) })
	address, servingDone := serveHTTP(t, server)
	t.Cleanup(func() { once.Do(func() { close(release) }) })

	client := &http.Client{Timeout: 2 * time.Second}
	defer client.CloseIdleConnections()

	requestDone := make(chan error, 1)

	go func() {
		response, err := client.Get("http://" + address)
		if err == nil {
			body, readErr := io.ReadAll(response.Body)
			response.Body.Close()

			if readErr != nil || string(body) != "finished" {
				err = errors.New("active request did not drain")
			}
		}

		requestDone <- err
	}()

	waitHTTP(t, entered)

	go func() {
		Shutdown(server, log.NewNoopLogger(), []func(context.Context) error{func(context.Context) error {
			stopped.Store(true)

			return nil
		}})
		close(shutdownDone)
	}()

	waitHTTP(t, shutdownStarted)

	if stopped.Load() {
		t.Fatal("dependency stopped while a request was active")
	}

	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err == nil {
		conn.Close()
		t.Fatal("Shutdown left the serving listener open")
	}

	once.Do(func() { close(release) })
	waitHTTP(t, shutdownDone)

	if !stopped.Load() {
		t.Fatal("dependency was not stopped after draining the request")
	}

	select {
	case err = <-requestDone:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("request did not finish")
	}

	select {
	case err = <-servingDone:
		if err != nil {
			t.Fatalf("Serve returned shutdown as an error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Serve did not exit after shutdown")
	}
}

// BaseContext exposes the actual ephemeral listener without reserving and
// releasing a port or polling readiness; its hook also synchronizes configuration.
func serveHTTP(t *testing.T, server *http.Server) (string, <-chan error) {
	t.Helper()

	address := make(chan string, 1)
	done := make(chan error, 1)
	server.Addr = "127.0.0.1:0"

	server.BaseContext = func(listener net.Listener) context.Context {
		address <- listener.Addr().String()

		return context.Background()
	}
	go func() {
		done <- Serve(server)

		close(done)
	}()

	t.Cleanup(func() {
		_ = server.Close()

		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Error("HTTP serving goroutine did not exit")
		}
	})

	select {
	case bound := <-address:
		return bound, done
	case err := <-done:
		t.Fatalf("HTTP server failed before readiness: %v", err)
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP listener was not ready")
	}

	return "", done
}

func waitHTTP(t *testing.T, signal <-chan struct{}) {
	t.Helper()

	select {
	case <-signal:
	case <-time.After(2 * time.Second):
		t.Fatal("HTTP lifecycle signal timed out")
	}
}
