package lsp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConnRoundTrip(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	c := NewConn(strings.NewReader(""), &buf)
	msg := &Message{
		JSONRPC: "2.0",
		ID:      json.RawMessage("1"),
		Method:  "initialize",
		Params:  json.RawMessage(`{"rootUri":"file:///tmp"}`),
	}
	require.NoError(t, c.WriteMessage(msg))
	raw := buf.String()
	require.Contains(t, raw, "Content-Length:")
	rc := NewConn(strings.NewReader(raw), &buf)
	got, err := rc.ReadMessage()
	require.NoError(t, err)
	require.Equal(t, "initialize", got.Method)
	require.True(t, got.IsRequest(), "expected request")
}

func TestMergeResultsArrays(t *testing.T) {
	t.Parallel()
	out := mergeResults([]json.RawMessage{
		json.RawMessage(`[{"uri":"a"}]`),
		json.RawMessage(`[{"uri":"b"}]`),
		json.RawMessage(`null`),
	})
	var items []map[string]string
	require.NoError(t, json.Unmarshal(out, &items))
	require.Len(t, items, 2, "%s", out)
}

func TestMergeResultsFirstNonNull(t *testing.T) {
	t.Parallel()
	out := mergeResults([]json.RawMessage{
		json.RawMessage(`null`),
		json.RawMessage(`{"contents":"hi"}`),
		json.RawMessage(`{"contents":"other"}`),
	})
	require.Equal(t, `{"contents":"hi"}`, string(out))
}

func TestReadMessageMissingContentLength(t *testing.T) {
	t.Parallel()
	// Header block ends without Content-Length.
	raw := "Content-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n"
	c := NewConn(strings.NewReader(raw), &bytes.Buffer{})
	_, err := c.ReadMessage()
	require.ErrorIs(t, err, ErrMissingContentLength)
}
