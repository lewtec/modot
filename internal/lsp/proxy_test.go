package lsp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"path/filepath"
	"testing"
	"time"

	"github.com/lewtec/modot/internal/logging"
	"github.com/stretchr/testify/require"
)

func writeLSP(t *testing.T, w io.Writer, v any) {
	t.Helper()
	body, err := json.Marshal(v)
	require.NoError(t, err)
	_, err = io.WriteString(w, "Content-Length: ")
	require.NoError(t, err)
	_, err = io.WriteString(w, string(mustJSON(len(body))))
	require.NoError(t, err)
	_, err = io.WriteString(w, "\r\n\r\n")
	require.NoError(t, err)
	_, err = w.Write(body)
	require.NoError(t, err)
}

func mustJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func TestProxyInitializeEmptyConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()

	serverIn, clientToServer := io.Pipe()    // client writes → server reads
	clientFromServer, serverOut := io.Pipe() // server writes → client reads

	ctx, cancel := context.WithCancel(logging.NewWriterContext(io.Discard))
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, serverIn, serverOut)
	}()

	writeLSP(t, clientToServer, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"processId":    nil,
			"rootUri":      pathToURI(root),
			"capabilities": map[string]any{},
		},
	})

	conn := NewConn(clientFromServer, io.Discard)
	msg, err := conn.ReadMessage()
	require.NoError(t, err)
	require.Nil(t, msg.Error)
	require.NotEmpty(t, msg.Result, "empty result")
	var result map[string]any
	require.NoError(t, json.Unmarshal(msg.Result, &result))
	require.NotNil(t, result["capabilities"], "missing capabilities")

	// hover with no backends -> method not found
	writeLSP(t, clientToServer, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "textDocument/hover",
		"params": map[string]any{
			"textDocument": map[string]any{"uri": pathToURI(filepath.Join(root, "x.go"))},
			"position":     map[string]any{"line": 0, "character": 0},
		},
	})
	msg, err = conn.ReadMessage()
	require.NoError(t, err)
	require.NotNil(t, msg.Error, "want MethodNotFound, result=%s", msg.Result)
	require.Equal(t, CodeMethodNotFound, msg.Error.Code, "result=%s", msg.Result)

	if err := clientToServer.Close(); err != nil {
		t.Logf("close clientToServer: %v", err)
	}
	if err := serverOut.Close(); err != nil {
		t.Logf("close serverOut: %v", err)
	}
	select {
	case <-errCh:
	case <-time.After(2 * time.Second):
		cancel()
	}
}

func TestProxyMultiRootRejected(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	serverIn, clientToServer := io.Pipe()
	clientFromServer, serverOut := io.Pipe()

	ctx, cancel := context.WithCancel(logging.NewWriterContext(io.Discard))
	defer cancel()
	go func() {
		if err := Run(ctx, serverIn, serverOut); err != nil {
			// expected during test shutdown
		}
	}()

	writeLSP(t, clientToServer, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"workspaceFolders": []map[string]any{
				{"uri": pathToURI(root), "name": "a"},
				{"uri": pathToURI(filepath.Join(root, "other")), "name": "b"},
			},
			"capabilities": map[string]any{},
		},
	})
	conn := NewConn(clientFromServer, io.Discard)
	msg, err := conn.ReadMessage()
	require.NoError(t, err)
	require.NotNil(t, msg.Error, "want multi-root error")
	require.Contains(t, msg.Error.Message, "single workspace")
	if err := clientToServer.Close(); err != nil {
		t.Logf("close clientToServer: %v", err)
	}
	cancel()
}

func TestWriteReadFraming(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	c := NewConn(nil, &buf)
	require.NoError(t, c.WriteResult(json.RawMessage("1"), map[string]string{"ok": "yes"}))
	rc := NewConn(bytes.NewReader(buf.Bytes()), io.Discard)
	msg, err := rc.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, "1", string(msg.ID))
}
