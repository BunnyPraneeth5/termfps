package app

import (
	"strings"
	"testing"
	"time"

	"github.com/BunnyPraneeth5/termfps/internal/ui"
)

// A waves run's break shows a banner with the countdown and what's next;
// when the wave starts, a toast says so and the banner is gone (U4).
func TestWaveBannerAndToast(t *testing.T) {
	a, s := newMenuApp(t)
	now := t0
	press(a, &now, enter, right, enter) // Play, Waves, Start
	st := a.state
	st.Wave, st.BreakLeft = 3, 11.5 // as if wave 3 was just cleared
	press(a, &now)
	for _, want := range []string{"✓ WAVE 3 CLEARED", "next wave in 12", "wave 4: 10 enemies · tier 2", "reload and pick things up", "╭", "▰"} {
		if !screenHas(s, want) {
			t.Errorf("the break's banner doesn't show %q", want)
		}
	}
	if _, style := s.Get((80-bannerW)/2, bannerY); style.Fg != ui.ColAmber {
		t.Errorf("the banner's border is %+v, want amber", style)
	}
	st.BreakLeft = 0.01
	press(a, &now)
	if st.Wave != 4 || !strings.Contains(s.Row(1), "WAVE 4 · 10 enemies · GO") {
		t.Errorf("wave %d started; row 1 = %q, want the GO toast", st.Wave, strings.TrimSpace(s.Row(1)))
	}
	if screenHas(s, "CLEARED") {
		t.Error("the banner still shows after the wave started")
	}
}

// With animations on a toast slides in: first on the top row, then on the
// second. A long toast keeps clear of the minimap.
func TestToastSlidesAndAvoidsMinimap(t *testing.T) {
	a, s := newTestApp(t)
	a.cfg.Animations = true
	now := t0.Add(time.Second)
	a.last = now
	a.notify(now, "TIER 2")
	press(a, &now) // 33 ms in: still sliding
	if !strings.Contains(s.Row(0), "TIER 2") {
		t.Errorf("a new toast isn't on the top row: %q", s.Row(0))
	}
	for range 4 {
		press(a, &now)
	}
	if !strings.Contains(s.Row(1), "TIER 2") || strings.Contains(s.Row(0), "TIER 2") {
		t.Error("the toast didn't settle on the second row")
	}
	long := strings.Repeat("long notice ", 5)
	a.cfg.Animations = false
	a.notify(now, long)
	press(a, &now)
	i := strings.Index(s.Row(1), "long")
	end := len([]rune(s.Row(1)[:i])) + len(long) + 2
	if !a.state.ShowMinimap || end > 80-12 { // the test map's minimap is 12 columns
		t.Errorf("a %d-column toast ends at column %d, inside the minimap", len(long)+4, end)
	}
	for range 30 * 4 {
		press(a, &now)
	}
	if screenHas(s, "long notice") {
		t.Error("the toast is still showing after four seconds")
	}
}

// The view takes every row above the two HUD rows.
func TestViewAboveTwoHUDRows(t *testing.T) {
	a, s := newTestApp(t)
	now := t0
	press(a, &now)
	if a.frame.H != 2*(24-2) {
		t.Errorf("frame height %d at 80x24, want 44", a.frame.H)
	}
	if str, _ := s.Get(0, 21); str == " " || str == "" {
		t.Errorf("view row 21 is empty")
	}
	if !strings.Contains(s.Row(22), "KILLS") || strings.Contains(s.Row(21), "KILLS") {
		t.Error("the HUD isn't on the last two rows")
	}
}
