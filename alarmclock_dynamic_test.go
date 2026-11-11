package alarmclock

import (
	"sync"
	"testing"
	"time"
)

func expectSpinWindow(t *testing.T, clock *AlarmClock, at time.Time, want time.Duration) {
	t.Helper()
	clock.mu.Lock()
	defer clock.mu.Unlock()
	group := clock.alarms[at.Round(0).UTC()]
	if group == nil {
		t.Fatal("alarm group not found")
	}
	if got := time.Duration(group.spinWindow.Load()); got != want {
		t.Fatalf("spin window = %v, want %v", got, want)
	}
}

func TestDynamicWindowsWithOutOfOrderInsertions(t *testing.T) {
	clock := &AlarmClock{spinWindow: 50 * time.Millisecond}
	a := time.Now().Add(150 * time.Millisecond)
	b := a.Add(20 * time.Millisecond)
	c := a.Add(40 * time.Millisecond)
	d := a.Add(60 * time.Millisecond)

	last := clock.SetAlarm(d.UTC())
	expectSpinWindow(t, clock, d, 50*time.Millisecond)
	second := clock.SetAlarm(b.Round(0))
	expectSpinWindow(t, clock, b, 50*time.Millisecond)
	expectSpinWindow(t, clock, d, 40*time.Millisecond)
	third := clock.SetAlarm(c)
	expectSpinWindow(t, clock, c, 20*time.Millisecond)
	expectSpinWindow(t, clock, d, 20*time.Millisecond)
	first := clock.SetAlarm(a)
	expectSpinWindow(t, clock, a, 50*time.Millisecond)
	expectSpinWindow(t, clock, b, 20*time.Millisecond)
	expectSpinWindow(t, clock, c, 20*time.Millisecond)
	expectSpinWindow(t, clock, d, 20*time.Millisecond)

	// Equivalent targets must reuse channels without resetting the shortened window.
	// 같은 시각을 다시 등록해도 줄어든 구간을 유지하며 채널을 재사용해야 합니다.
	for range alarmChannelCount - 1 {
		clock.SetAlarm(d)
	}
	if got := clock.SetAlarm(d); got != last {
		t.Fatal("equivalent target did not reuse its round-robin channel")
	}
	expectSpinWindow(t, clock, d, 20*time.Millisecond)

	awaitAlarm(t, first, a)
	awaitAlarm(t, second, b)
	awaitAlarm(t, third, c)
	awaitAlarm(t, last, d)
	awaitCleanup(t, clock)
}

func TestConcurrentDistinctTargetsUpdateWindows(t *testing.T) {
	clock := &AlarmClock{spinWindow: 40 * time.Millisecond}
	base := time.Now().Add(250 * time.Millisecond)
	var targets [8]time.Time
	for i := range targets {
		targets[i] = base.Add(time.Duration(i) * 10 * time.Millisecond)
	}
	var channels [256]<-chan struct{}
	var registered sync.WaitGroup
	for i := range channels {
		registered.Add(1)
		go func() {
			defer registered.Done()
			at := targets[i%len(targets)]
			if i%2 == 0 {
				at = at.UTC()
			}
			channels[i] = clock.SetAlarm(at)
		}()
	}
	registered.Wait()
	expectSpinWindow(t, clock, targets[0], 40*time.Millisecond)
	for _, at := range targets[1:] {
		expectSpinWindow(t, clock, at, 10*time.Millisecond)
	}
	for i, ch := range channels {
		awaitAlarm(t, ch, targets[i%len(targets)])
	}
	awaitCleanup(t, clock)
}
