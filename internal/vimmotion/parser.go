package vimmotion

import "time"

type Kind int

const (
	KindNone Kind = iota
	KindDown
	KindUp
	KindTop
	KindBottom
	KindHalfPageDown
	KindHalfPageUp
	KindPageDown
	KindPageUp
)

// AmbiguityTimeout is the window after a keystroke during which the parser
// may still be waiting for a disambiguating follow-up (e.g. "g" → "gg").
const AmbiguityTimeout = 300 * time.Millisecond

type Motion struct {
	Kind  Kind
	Count int
}

// Key is a single vim-style keystroke. Use RuneKey for printable chars and
// the CtrlXxx constants for control sequences.
type Key struct {
	Rune rune
	Ctrl rune // Set to 'd' for Ctrl-d, etc. Zero if not a control.
}

func RuneKey(r rune) Key        { return Key{Rune: r} }
func CtrlKey(ctrl rune) Key     { return Key{Ctrl: ctrl} }

// Parser is a pure state machine: Feed emits a Motion when an input resolves,
// Timeout flushes a pending single-key ambiguous (e.g. lone "g").
type Parser struct {
	count      int
	pendingG   bool
	lastInputAt time.Time
}

// NewParser returns a ready-to-use parser.
func NewParser() *Parser { return &Parser{} }

// Reset clears buffered state (count, pending disambig).
func (p *Parser) Reset() {
	p.count = 0
	p.pendingG = false
}

// Pending reports whether the parser is waiting on a disambiguation.
func (p *Parser) Pending() bool { return p.pendingG }

// Timeout resolves any pending single-key ambiguity if `now` is after the
// window. Returns (motion, true) when a motion is emitted; (_, false) otherwise.
// Ambiguous "g" has no standalone motion, so Timeout just resets without emit.
func (p *Parser) Timeout(now time.Time) (Motion, bool) {
	if p.pendingG && now.Sub(p.lastInputAt) >= AmbiguityTimeout {
		p.Reset()
	}
	return Motion{}, false
}

// Feed processes a keystroke and returns (motion, emitted).
func (p *Parser) Feed(k Key, now time.Time) (Motion, bool) {
	p.lastInputAt = now
	if k.Ctrl != 0 {
		p.pendingG = false
		m, ok := p.resolveCtrl(k.Ctrl)
		return m, ok
	}
	r := k.Rune

	// Digit prefix (but '0' only counts when count is nonzero — '0' alone is line-start in vim, not supported here).
	if r >= '1' && r <= '9' || (r == '0' && p.count > 0) {
		p.count = p.count*10 + int(r-'0')
		p.pendingG = false
		return Motion{}, false
	}

	if p.pendingG {
		p.pendingG = false
		if r == 'g' {
			m := Motion{Kind: KindTop, Count: p.takeCount()}
			return m, true
		}
		// Non-matching second char — drop the pending g, fall through to treat r as a fresh key.
	}

	switch r {
	case 'j':
		return Motion{Kind: KindDown, Count: p.takeCount()}, true
	case 'k':
		return Motion{Kind: KindUp, Count: p.takeCount()}, true
	case 'G':
		return Motion{Kind: KindBottom, Count: p.takeCount()}, true
	case 'g':
		p.pendingG = true
		return Motion{}, false
	}
	// Unknown key — drop accumulated count so a stale prefix doesn't attach to the next motion.
	p.count = 0
	return Motion{}, false
}

func (p *Parser) resolveCtrl(c rune) (Motion, bool) {
	switch c {
	case 'd':
		return Motion{Kind: KindHalfPageDown, Count: p.takeCount()}, true
	case 'u':
		return Motion{Kind: KindHalfPageUp, Count: p.takeCount()}, true
	case 'f':
		return Motion{Kind: KindPageDown, Count: p.takeCount()}, true
	case 'b':
		return Motion{Kind: KindPageUp, Count: p.takeCount()}, true
	}
	p.count = 0
	return Motion{}, false
}

func (p *Parser) takeCount() int {
	c := p.count
	p.count = 0
	if c == 0 {
		return 1
	}
	return c
}
