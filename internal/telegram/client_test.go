package telegram

import (
	"strings"
	"testing"
)

func TestSplitMessageShortTextIsOneChunk(t *testing.T) {
	chunks := splitMessage("hello", maxMessageLength)

	if len(chunks) != 1 || chunks[0] != "hello" {
		t.Fatalf("got %#v, want one chunk of %q", chunks, "hello")
	}
}

func TestSplitMessageRespectsLimit(t *testing.T) {
	text := strings.Repeat("a", 10_000)

	chunks := splitMessage(text, maxMessageLength)

	if len(chunks) != 3 {
		t.Fatalf("got %d chunks, want 3", len(chunks))
	}

	for i, chunk := range chunks {
		if len([]rune(chunk)) > maxMessageLength {
			t.Errorf("chunk %d is %d runes, over the limit", i, len([]rune(chunk)))
		}
	}

	if joined := strings.Join(chunks, ""); joined != text {
		t.Error("joined chunks do not reproduce the original text")
	}
}

func TestSplitMessagePrefersNewlineBoundary(t *testing.T) {
	// A newline just inside the limit should be used as the break point.
	line := strings.Repeat("x", 90) + "\n"

	text := strings.Repeat(line, 3)

	chunks := splitMessage(text, 100)

	if !strings.HasSuffix(chunks[0], "\n") {
		t.Fatalf("first chunk does not end on a newline: %q", chunks[0])
	}

	if joined := strings.Join(chunks, ""); joined != text {
		t.Error("joined chunks do not reproduce the original text")
	}
}

func TestSplitMessageCountsRunesNotBytes(t *testing.T) {
	// Multi-byte characters must not be counted as several characters, and must
	// never be cut in half.
	text := strings.Repeat("é", 150)

	chunks := splitMessage(text, 100)

	if len(chunks) != 2 {
		t.Fatalf("got %d chunks, want 2", len(chunks))
	}

	if len([]rune(chunks[0])) != 100 {
		t.Errorf("first chunk is %d runes, want 100", len([]rune(chunks[0])))
	}

	if joined := strings.Join(chunks, ""); joined != text {
		t.Error("joined chunks do not reproduce the original text")
	}
}
