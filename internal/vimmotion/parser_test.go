package vimmotion

import (
	"testing"
	"time"
)

func feed(p *Parser, keys ...Key) []Motion {
	var out []Motion
	now := time.Now()
	for _, k := range keys {
		if m, ok := p.Feed(k, now); ok {
			out = append(out, m)
		}
		now = now.Add(10 * time.Millisecond)
	}
	return out
}

func TestSimpleMotions(t *testing.T) {
	cases := []struct {
		name string
		keys []Key
		want []Motion
	}{
		{"j", []Key{RuneKey('j')}, []Motion{{KindDown, 1}}},
		{"k", []Key{RuneKey('k')}, []Motion{{KindUp, 1}}},
		{"G", []Key{RuneKey('G')}, []Motion{{KindBottom, 1}}},
		{"gg", []Key{RuneKey('g'), RuneKey('g')}, []Motion{{KindTop, 1}}},
		{"Ctrl-d", []Key{CtrlKey('d')}, []Motion{{KindHalfPageDown, 1}}},
		{"Ctrl-u", []Key{CtrlKey('u')}, []Motion{{KindHalfPageUp, 1}}},
		{"Ctrl-f", []Key{CtrlKey('f')}, []Motion{{KindPageDown, 1}}},
		{"Ctrl-b", []Key{CtrlKey('b')}, []Motion{{KindPageUp, 1}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewParser()
			got := feed(p, tc.keys...)
			if len(got) != len(tc.want) {
				t.Fatalf("emitted %v, want %v", got, tc.want)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Fatalf("motion[%d] = %+v, want %+v", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestCountPrefix(t *testing.T) {
	cases := []struct {
		name string
		keys []Key
		want Motion
	}{
		{"5j", []Key{RuneKey('5'), RuneKey('j')}, Motion{KindDown, 5}},
		{"10k", []Key{RuneKey('1'), RuneKey('0'), RuneKey('k')}, Motion{KindUp, 10}},
		{"3gg", []Key{RuneKey('3'), RuneKey('g'), RuneKey('g')}, Motion{KindTop, 3}},
		{"2G", []Key{RuneKey('2'), RuneKey('G')}, Motion{KindBottom, 2}},
		{"4C-d", []Key{RuneKey('4'), CtrlKey('d')}, Motion{KindHalfPageDown, 4}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := NewParser()
			got := feed(p, tc.keys...)
			if len(got) != 1 || got[0] != tc.want {
				t.Fatalf("got %+v, want [%+v]", got, tc.want)
			}
		})
	}
}

func TestPendingGTimesOut(t *testing.T) {
	p := NewParser()
	now := time.Now()
	if _, ok := p.Feed(RuneKey('g'), now); ok {
		t.Fatal("single g should not emit")
	}
	if !p.Pending() {
		t.Fatal("expected pending after single g")
	}
	_, _ = p.Timeout(now.Add(AmbiguityTimeout + time.Millisecond))
	if p.Pending() {
		t.Fatal("expected cleared pending after timeout")
	}
}

func TestStaleCountClearsOnUnknown(t *testing.T) {
	p := NewParser()
	_ = feed(p, RuneKey('5'), RuneKey('x')) // unknown key drops count
	got := feed(p, RuneKey('j'))
	if len(got) != 1 || got[0] != (Motion{KindDown, 1}) {
		t.Fatalf("got %+v, want [{Down 1}]", got)
	}
}

func TestPendingGBreaksOnMismatch(t *testing.T) {
	p := NewParser()
	// g then j — g drops, j emits as Down
	got := feed(p, RuneKey('g'), RuneKey('j'))
	if len(got) != 1 || got[0] != (Motion{KindDown, 1}) {
		t.Fatalf("got %+v, want [{Down 1}]", got)
	}
}

func TestZeroDigitOnlyAfterCount(t *testing.T) {
	p := NewParser()
	// "0j" alone should not count as 0j (no 0-as-prefix support). Lone 0 treated as unknown.
	got := feed(p, RuneKey('0'), RuneKey('j'))
	if len(got) != 1 || got[0] != (Motion{KindDown, 1}) {
		t.Fatalf("got %+v, want [{Down 1}]", got)
	}

	// "10j" should work
	p.Reset()
	got = feed(p, RuneKey('1'), RuneKey('0'), RuneKey('j'))
	if len(got) != 1 || got[0] != (Motion{KindDown, 10}) {
		t.Fatalf("got %+v, want [{Down 10}]", got)
	}
}
