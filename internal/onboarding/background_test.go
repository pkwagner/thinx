package onboarding

import (
	"strings"
	"testing"
)

// TestLayoutPlacesUniqueTodos verifies the virtual screen has no repeats, leaves a
// blank line between rows, and keeps every todo at least partly visible while
// allowing it to bleed off either edge.
func TestLayoutPlacesUniqueTodos(t *testing.T) {
	t.Parallel()
	const width, height = 80, 24
	b := newBackground(1)
	b.resize(width, height)

	if len(b.items) == 0 {
		t.Fatal("no todos placed")
	}
	clippedLeft := false
	seen := map[string]bool{}
	for _, it := range b.items {
		if seen[it.text] {
			t.Fatalf("repeated todo %q", it.text)
		}
		seen[it.text] = true
		if it.row < 0 || it.row >= height || it.row%rowStep != 0 {
			t.Fatalf("row not on a spaced line: %#v", it)
		}
		w := len([]rune(it.text))
		if it.col >= width || it.col+w <= 0 {
			t.Fatalf("todo entirely off-screen: %#v", it)
		}
		if it.col < 0 {
			clippedLeft = true
		}
	}
	if !clippedLeft {
		t.Fatal("expected some todos clipped on the left edge")
	}
}

// TestAdvanceTypesThroughThenLoops verifies the animation reaches a full screen
// and then resets to a fresh layout.
func TestAdvanceTypesThroughThenLoops(t *testing.T) {
	t.Parallel()
	b := newBackground(2)
	b.resize(60, 10)

	for i := 0; i < 1_000_000 && b.phase != bgFull; i++ {
		b.advance()
	}
	if b.phase != bgFull {
		t.Fatal("never reached a full screen")
	}

	b.advance() // from full -> rebuild
	if b.phase != bgTyping || b.cur != 0 || b.typed != 0 {
		t.Fatalf("after reset: phase=%v cur=%d typed=%d", b.phase, b.cur, b.typed)
	}
}

// TestRenderRevealsTypedTodo verifies the visible part of a fully typed todo
// appears in the output and a not-yet-revealed one does not.
func TestRenderRevealsTypedTodo(t *testing.T) {
	t.Parallel()
	const width, height = 60, 8
	b := newBackground(3)
	b.resize(width, height)

	first := b.items[0]
	for range []rune(first.text) {
		b.advance()
	}
	out := b.render()

	if vis := visibleText(first, width); !strings.Contains(out, vis) {
		t.Fatalf("render missing visible part %q of typed todo %q", vis, first.text)
	}

	if len(b.items) > 1 {
		// The last item in reveal order should not be drawn yet.
		last := b.items[len(b.items)-1]
		if last.text != first.text && strings.Contains(out, last.text) {
			t.Fatalf("render shows not-yet-revealed todo %q", last.text)
		}
	}
}

// visibleText returns the on-screen portion of an item placed at it.col.
func visibleText(it bgItem, width int) string {
	runes := []rune(it.text)
	start, end := 0, len(runes)
	if it.col < 0 {
		start = -it.col
	}
	if it.col+len(runes) > width {
		end = width - it.col
	}
	return string(runes[start:end])
}

// TestRenderEmptyBeforeResize verifies a zero-size background renders nothing.
func TestRenderEmptyBeforeResize(t *testing.T) {
	t.Parallel()
	if got := newBackground(4).render(); got != "" {
		t.Fatalf("render before resize = %q, want empty", got)
	}
}
