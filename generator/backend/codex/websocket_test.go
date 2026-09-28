package codex

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestWebSocketProcessUpgradesAndRelaysProtocolMessages(t *testing.T) {
	client, server := newPipeProcesses()
	serverDone := make(chan error, 1)

	go func() {
		reader, err := acceptTestWebSocket(server)
		if err != nil {
			serverDone <- err

			return
		}

		opcode, payload, err := readTestClientFrame(reader)
		if err != nil {
			serverDone <- err

			return
		}

		if opcode != webSocketText || string(payload) != `{"id":1,"method":"initialize"}` {
			serverDone <- fmt.Errorf("client frame = opcode %d payload %q", opcode, payload)

			return
		}

		err = writeTestServerFrame(server, webSocketText, []byte(`{"id":1,"result":{}}`))
		serverDone <- err
	}()

	process, err := openWebSocketProcess(context.Background(), client)
	if err != nil {
		t.Fatalf("openWebSocketProcess() error = %v", err)
	}

	_, err = process.Write([]byte(`{"id":1,"method":"initialize"}` + "\n"))
	if err != nil {
		t.Fatalf("Write() error = %v", err)
	}

	response := make([]byte, 128)

	read, err := process.Read(response)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}

	if string(response[:read]) != `{"id":1,"result":{}}`+"\n" {
		t.Fatalf("Read() = %q, want protocol response", response[:read])
	}

	err = <-serverDone
	if err != nil {
		t.Fatalf("server error = %v", err)
	}

	_ = process.Close()
}

func TestWebSocketProcessRejectsInvalidUpgrade(t *testing.T) {
	client, server := newPipeProcesses()

	go func() {
		reader := bufio.NewReader(server)
		_, _ = http.ReadRequest(reader)
		_, _ = io.WriteString(server, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\n\r\n")
	}()

	_, err := openWebSocketProcess(context.Background(), client)
	assertBackendCode(t, err, "backend_incompatible")
}

func serveTestWebSocket(server Process) <-chan error {
	result := make(chan error, 1)

	go func() {
		_, err := acceptTestWebSocket(server)
		result <- err
	}()

	return result
}

func acceptTestWebSocket(server Process) (*bufio.Reader, error) {
	reader := bufio.NewReader(server)

	request, err := http.ReadRequest(reader)
	if err != nil {
		return nil, err
	}

	if request.URL.Path != "/rpc" || !headerContainsToken(request.Header, "Upgrade", "websocket") {
		return nil, errors.New("invalid WebSocket upgrade request")
	}

	key := request.Header.Get("Sec-WebSocket-Key")

	_, err = base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, errors.New("invalid WebSocket key")
	}

	response := fmt.Sprintf(
		"HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n",
		webSocketAccept(key),
	)

	err = writeAll(server, []byte(response))
	if err != nil {
		return nil, err
	}

	return reader, nil
}

func readTestClientFrame(reader io.Reader) (byte, []byte, error) {
	header := make([]byte, 2)

	_, err := io.ReadFull(reader, header)
	if err != nil {
		return 0, nil, err
	}

	if header[1]&webSocketMask == 0 {
		return 0, nil, errors.New("client frame is not masked")
	}

	length, err := readWebSocketLength(reader, header[1]&0x7f)
	if err != nil {
		return 0, nil, err
	}

	mask := make([]byte, 4)

	_, err = io.ReadFull(reader, mask)
	if err != nil {
		return 0, nil, err
	}

	payload := make([]byte, length)

	_, err = io.ReadFull(reader, payload)
	if err != nil {
		return 0, nil, err
	}

	for index := range payload {
		payload[index] ^= mask[index%len(mask)]
	}

	return header[0] & 0x0f, payload, nil
}

func writeTestServerFrame(writer io.Writer, opcode byte, payload []byte) error {
	frame := []byte{webSocketFinalFrame | opcode}
	frame = appendWebSocketLength(frame, len(payload), false)
	frame = append(frame, payload...)

	return writeAll(writer, frame)
}

func TestHeaderContainsTokenIgnoresCaseAndWhitespace(t *testing.T) {
	header := http.Header{"Connection": []string{"keep-alive, Upgrade"}}

	if !headerContainsToken(header, "Connection", "upgrade") {
		t.Fatal("headerContainsToken() did not match token")
	}

	if headerContainsToken(header, "Connection", strings.Repeat("x", 2)) {
		t.Fatal("headerContainsToken() matched absent token")
	}
}
