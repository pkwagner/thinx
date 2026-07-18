package onboarding

import (
	"math/rand"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// Typewriter timing for the onboarding background.
const (
	typeInterval = 45 * time.Millisecond // delay between revealed letters
	pauseBetween = 1 * time.Second       // hold after a todo finishes typing
	fullHold     = 5 * time.Second       // hold once the whole screen is filled
)

// bgStyle renders the drifting todos in a calm, unobtrusive grey.
var bgStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

// bgPhase is the current step of the animation loop.
type bgPhase int

const (
	bgTyping  bgPhase = iota // revealing the current todo letter by letter
	bgPausing                // holding after a finished todo
	bgFull                   // whole screen revealed, holding before reset
)

// bgItem is a single todo placed at a fixed spot on the virtual screen.
type bgItem struct {
	row, col int
	text     string
}

// background is a self-animating field of todos that type themselves in. It owns
// no business logic — it just renders fun behind the onboarding modal.
type background struct {
	width, height int
	items         []bgItem // placed todos, in reveal order
	cur           int      // index of the todo currently typing
	typed         int      // runes revealed of items[cur]
	phase         bgPhase
	rng           *rand.Rand
}

// newBackground seeds a fresh background; pass a fixed seed in tests.
func newBackground(seed int64) *background {
	return &background{rng: rand.New(rand.NewSource(seed))}
}

// resize lays out a new virtual screen for the given size.
func (b *background) resize(width, height int) {
	b.width, b.height = width, height
	b.rebuild()
}

// rebuild shuffles a brand new virtual background.
func (b *background) rebuild() {
	b.items = layout(b.width, b.height, b.rng)
	b.cur, b.typed, b.phase = 0, 0, bgTyping
}

// Spacing within the virtual screen.
const (
	todoGap     = 5  // spaces between todos on a line
	rowStep     = 2  // place on every other row, leaving a blank line between
	minLineClip = 4  // smallest amount a row starts off-screen to the left
	maxLineClip = 24 // largest amount a row starts off-screen to the left
	minVisible  = 3  // keep at least this many leading runes of the first todo on screen
)

// layout fills the screen with non-repeating todos. Filled rows are spaced out (a
// blank line between each); every row starts at a negative column so its first
// todo is clipped on the left, and todos pack with todoGap spaces between them
// until one would start past the right edge (that last one runs off-screen too).
// The renderer clips whatever falls outside the screen, giving a ragged, organic
// edge instead of an aligned grid. Placed todos are shuffled into reveal order.
func layout(width, height int, rng *rand.Rand) []bgItem {
	if width <= 0 || height <= 0 {
		return nil
	}
	pool := append([]string(nil), vocab...)
	rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })

	var items []bgItem
	row, col := 0, 0
	firstInRow := true
	for _, text := range pool {
		w := len([]rune(text))
		if col >= width { // no room left to start another todo on this row
			row += rowStep
			if row >= height {
				break
			}
			firstInRow = true
		}
		if firstInRow {
			col = lineOffset(rng, w) // negative: clip the row's first todo on the left
			firstInRow = false
		}
		items = append(items, bgItem{row: row, col: col, text: text})
		col += w + todoGap
	}

	rng.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
	return items
}

// lineOffset returns a negative starting column so a row begins partway through
// its first todo, raggedly filling the left edge — clamped so at least minVisible
// leading runes of that todo stay on screen.
func lineOffset(rng *rand.Rand, w int) int {
	clip := minLineClip + rng.Intn(maxLineClip-minLineClip+1)
	if maxClip := w - minVisible; clip > maxClip {
		clip = maxClip
	}
	if clip < 0 {
		clip = 0
	}
	return -clip
}

// advance moves the animation forward one tick.
func (b *background) advance() {
	switch b.phase {
	case bgTyping:
		if b.cur >= len(b.items) {
			b.phase = bgFull
			return
		}
		if b.typed < len([]rune(b.items[b.cur].text)) {
			b.typed++
		}
		if b.typed >= len([]rune(b.items[b.cur].text)) {
			b.phase = bgPausing
		}
	case bgPausing:
		b.cur++
		b.typed = 0
		if b.cur >= len(b.items) {
			b.phase = bgFull
		} else {
			b.phase = bgTyping
		}
	case bgFull:
		b.rebuild()
	}
}

// next returns the delay before the following tick.
func (b *background) next() time.Duration {
	switch b.phase {
	case bgPausing:
		return pauseBetween
	case bgFull:
		return fullHold
	default:
		return typeInterval
	}
}

// render draws the current state as a full-screen string (blank where nothing has
// been revealed yet).
func (b *background) render() string {
	if b.width <= 0 || b.height <= 0 {
		return ""
	}
	grid := make([][]rune, b.height)
	for i := range grid {
		grid[i] = []rune(strings.Repeat(" ", b.width))
	}
	draw := func(it bgItem, n int) {
		runes := []rune(it.text)
		if n > len(runes) {
			n = len(runes)
		}
		for k := 0; k < n; k++ {
			col := it.col + k
			if it.row >= 0 && it.row < b.height && col >= 0 && col < b.width {
				grid[it.row][col] = runes[k]
			}
		}
	}
	for i := 0; i < b.cur && i < len(b.items); i++ {
		draw(b.items[i], len([]rune(b.items[i].text)))
	}
	if b.cur < len(b.items) {
		draw(b.items[b.cur], b.typed)
	}

	lines := make([]string, b.height)
	for i := range grid {
		lines[i] = strings.TrimRight(string(grid[i]), " ")
	}
	return bgStyle.Render(strings.Join(lines, "\n"))
}
