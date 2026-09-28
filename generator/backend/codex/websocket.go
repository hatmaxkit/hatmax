package codex

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/sha1" //nolint:gosec // WebSocket accept uses SHA-1 by protocol definition.
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"

	"hatmax.adrianpk.com/generator/eval"
)

const (
	webSocketGUID       = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	webSocketText       = byte(0x1)
	webSocketClose      = byte(0x8)
	webSocketPing       = byte(0x9)
	webSocketPong       = byte(0xa)
	webSocketFinalFrame = byte(0x80)
	webSocketMask       = byte(0x80)
)

type webSocketProcess struct {
	process Process
	reader  *bufio.Reader

	readBuffer []byte
	writeMutex sync.Mutex
	closeOnce  sync.Once
	closeError error
}

func openWebSocketProcess(ctx context.Context, process Process) (Process, error) {
	if process == nil {
		return nil, backendError(eval.BackendUnavailable, "proxy_handshake", "Codex App Server proxy is unavailable")
	}

	keyBytes := make([]byte, 16)

	_, err := rand.Read(keyBytes)
	if err != nil {
		return nil, backendError(eval.BackendStartFailed, "proxy_handshake", "WebSocket key generation failed")
	}

	key := base64.StdEncoding.EncodeToString(keyBytes)
	request := fmt.Sprintf(
		"GET /rpc HTTP/1.1\r\nHost: localhost\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\n\r\n",
		key,
	)

	err = writeAll(process, []byte(request))
	if err != nil {
		return nil, backendError(eval.BackendStartFailed, "proxy_handshake", "WebSocket upgrade request failed")
	}

	reader := bufio.NewReader(process)
	responseChannel := make(chan *http.Response, 1)
	errorChannel := make(chan error, 1)

	go func() {
		response, responseErr := http.ReadResponse(reader, &http.Request{Method: http.MethodGet})
		if responseErr != nil {
			errorChannel <- responseErr

			return
		}

		responseChannel <- response
	}()

	var response *http.Response

	select {
	case <-ctx.Done():
		return nil, contextBackendError("proxy_handshake", ctx.Err())
	case err = <-errorChannel:
		return nil, backendError(eval.BackendStartFailed, "proxy_handshake", "WebSocket upgrade response was invalid")
	case response = <-responseChannel:
	}

	if response.Body != nil {
		defer response.Body.Close()
	}

	expectedAccept := webSocketAccept(key)
	if response.StatusCode != http.StatusSwitchingProtocols ||
		!headerContainsToken(response.Header, "Upgrade", "websocket") ||
		!headerContainsToken(response.Header, "Connection", "upgrade") ||
		response.Header.Get("Sec-WebSocket-Accept") != expectedAccept {
		return nil, backendError(eval.BackendIncompatible, "proxy_handshake", "Codex App Server rejected the WebSocket upgrade")
	}

	return &webSocketProcess{process: process, reader: reader}, nil
}

func (process *webSocketProcess) Read(value []byte) (int, error) {
	for len(process.readBuffer) == 0 {
		opcode, payload, err := readWebSocketFrame(process.reader)
		if err != nil {
			return 0, err
		}

		switch opcode {
		case webSocketText:
			process.readBuffer = append(payload, '\n')
		case webSocketPing:
			err = process.writeFrame(webSocketPong, payload)
			if err != nil {
				return 0, err
			}
		case webSocketPong:
			continue
		case webSocketClose:
			return 0, io.EOF
		default:
			return 0, backendError(eval.BackendProtocolViolation, "proxy_read", "Codex App Server sent an unsupported WebSocket frame")
		}
	}

	read := copy(value, process.readBuffer)
	process.readBuffer = process.readBuffer[read:]

	return read, nil
}

func (process *webSocketProcess) Write(value []byte) (int, error) {
	payload := value
	if len(payload) > 0 && payload[len(payload)-1] == '\n' {
		payload = payload[:len(payload)-1]
	}

	err := process.writeFrame(webSocketText, payload)
	if err != nil {
		return 0, err
	}

	return len(value), nil
}

func (process *webSocketProcess) Close() error {
	process.closeOnce.Do(func() {
		process.closeError = process.process.Close()
	})

	return process.closeError
}

func (process *webSocketProcess) Wait() error { return process.process.Wait() }

func (process *webSocketProcess) Kill() error { return process.process.Kill() }

func (process *webSocketProcess) writeFrame(opcode byte, payload []byte) error {
	if len(payload) > maximumMessageBytes {
		return backendError(eval.BackendProtocolViolation, "proxy_write", "Codex App Server message exceeds transport bounds")
	}

	mask := make([]byte, 4)

	_, err := rand.Read(mask)
	if err != nil {
		return backendError(eval.BackendUnavailable, "proxy_write", "WebSocket mask generation failed")
	}

	frame := make([]byte, 0, len(payload)+14)
	frame = append(frame, webSocketFinalFrame|opcode)
	frame = appendWebSocketLength(frame, len(payload), true)
	frame = append(frame, mask...)

	for index, value := range payload {
		frame = append(frame, value^mask[index%len(mask)])
	}

	process.writeMutex.Lock()
	defer process.writeMutex.Unlock()

	err = writeAll(process.process, frame)
	if err != nil {
		return backendError(eval.BackendUnavailable, "proxy_write", "Codex App Server WebSocket write failed")
	}

	return nil
}

func readWebSocketFrame(reader io.Reader) (byte, []byte, error) {
	header := make([]byte, 2)

	_, err := io.ReadFull(reader, header)
	if err != nil {
		return 0, nil, err
	}

	if header[0]&webSocketFinalFrame == 0 || header[0]&0x70 != 0 || header[1]&webSocketMask != 0 {
		return 0, nil, backendError(eval.BackendProtocolViolation, "proxy_read", "Codex App Server sent an invalid WebSocket frame")
	}

	length, err := readWebSocketLength(reader, header[1]&0x7f)
	if err != nil {
		return 0, nil, err
	}

	if length > maximumMessageBytes {
		return 0, nil, backendError(eval.BackendProtocolViolation, "proxy_read", "Codex App Server message exceeds transport bounds")
	}

	payload := make([]byte, length)

	_, err = io.ReadFull(reader, payload)
	if err != nil {
		return 0, nil, err
	}

	return header[0] & 0x0f, payload, nil
}

func appendWebSocketLength(frame []byte, length int, masked bool) []byte {
	mask := byte(0)
	if masked {
		mask = webSocketMask
	}

	switch {
	case length < 126:
		return append(frame, mask|byte(length))
	case length <= 0xffff:
		return append(frame, mask|126, byte(length>>8), byte(length))
	default:
		return append(
			frame,
			mask|127,
			0, 0, 0, 0,
			byte(length>>24), byte(length>>16), byte(length>>8), byte(length),
		)
	}
}

func readWebSocketLength(reader io.Reader, marker byte) (int, error) {
	switch marker {
	case 126:
		value := make([]byte, 2)

		_, err := io.ReadFull(reader, value)
		if err != nil {
			return 0, err
		}

		return int(value[0])<<8 | int(value[1]), nil
	case 127:
		value := make([]byte, 8)

		_, err := io.ReadFull(reader, value)
		if err != nil {
			return 0, err
		}

		if value[0]|value[1]|value[2]|value[3] != 0 {
			return 0, backendError(eval.BackendProtocolViolation, "proxy_read", "Codex App Server message exceeds transport bounds")
		}

		return int(value[4])<<24 | int(value[5])<<16 | int(value[6])<<8 | int(value[7]), nil
	default:
		return int(marker), nil
	}
}

func writeAll(writer io.Writer, value []byte) error {
	for len(value) > 0 {
		written, err := writer.Write(value)
		if err != nil {
			return err
		}

		if written == 0 {
			return io.ErrShortWrite
		}

		value = value[written:]
	}

	return nil
}

func webSocketAccept(key string) string {
	digest := sha1.Sum([]byte(key + webSocketGUID)) //nolint:gosec // WebSocket accept uses SHA-1 by protocol definition.

	return base64.StdEncoding.EncodeToString(digest[:])
}

func headerContainsToken(header http.Header, name, token string) bool {
	for _, value := range header.Values(name) {
		for _, candidate := range strings.Split(value, ",") {
			if strings.EqualFold(strings.TrimSpace(candidate), token) {
				return true
			}
		}
	}

	return false
}
