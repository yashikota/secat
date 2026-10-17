package mask

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func streamString(t *testing.T, in string) string {
	t.Helper()
	var out bytes.Buffer
	if err := Stream(strings.NewReader(in), &out, Default); err != nil {
		t.Fatalf("Stream: %v", err)
	}
	return out.String()
}

func TestStream(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"basic", "A=abcdefgh\nB=x\n", "A=ab******\nB=*\n"},
		{"no trailing newline", "A=abcdefgh", "A=ab******"},
		{"crlf", "A=abcdefgh\r\nB=x\r\n", "A=ab******\r\nB=*\r\n"},
		{"empty", "", ""},
		{"only newlines", "\n\n", "\n\n"},
		{"mixed endings", "A=abcd\r\nB=abcd\nC=abcd", "A=a***\r\nB=a***\nC=a***"},
		{"lone carriage return kept in value", "A=ab\rcd\n", "A=a****\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := streamString(t, tt.in); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStreamLongLine(t *testing.T) {
	n := 1 << 20
	in := "K=" + strings.Repeat("a", n) + "\n"
	want := "K=aaaa" + strings.Repeat("*", n-4) + "\n"
	if got := streamString(t, in); got != want {
		t.Errorf("long line mismatch: got %d bytes, want %d bytes", len(got), len(want))
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("write failed") }

func TestStreamWriteError(t *testing.T) {
	if err := Stream(strings.NewReader("A=abcdefgh\n"), failWriter{}, Default); err == nil {
		t.Error("Stream returned nil, want write error")
	}
}

type failReader struct{}

func (failReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }

func TestStreamReadError(t *testing.T) {
	if err := Stream(failReader{}, &bytes.Buffer{}, Default); err == nil {
		t.Error("Stream returned nil, want read error")
	}
}

func TestStreamMultilineAcrossLines(t *testing.T) {
	in := "KEY=\"-----BEGIN RSA PRIVATE KEY-----\nMIIEow\n-----END RSA PRIVATE KEY-----\"\nNEXT=abcdefgh\n"
	got := streamString(t, in)
	if strings.Contains(got, "MIIEow") {
		t.Errorf("output leaks key body:\n%s", got)
	}
	if !strings.HasSuffix(got, "NEXT=ab******\n") {
		t.Errorf("state did not end after the closing quote:\n%s", got)
	}
}
