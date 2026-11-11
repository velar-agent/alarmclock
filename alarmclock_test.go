package alarmclock

import (
	"sync"
	"testing"
	"time"
)

func awaitAlarm(t *testing.T, ch <-chan struct{}, at time.Time) {
	t.Helper()
	select {
	case _, ok := <-ch:
		if ok {
			t.Fatal("alarm sent a value instead of closing")
		}
		if time.Until(at) > 0 {
			t.Fatal("alarm fired before the target time")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("alarm did not fire")
	}
}

func awaitCleanup(t *testing.T, clock *AlarmClock) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for {
		clock.mu.Lock()
		remaining := len(clock.alarms)
		clock.mu.Unlock()
		if remaining == 0 {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("alarm groups were not cleaned up: %d", remaining)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestRoundRobinAndEquivalentTimes(t *testing.T) {
	clock := &AlarmClock{}
	at := time.Now().Add(100 * time.Millisecond)
	targets := []time.Time{at, at.Round(0), at.UTC(), at.In(time.FixedZone("offset", 9*60*60))}
	channels := make([]<-chan struct{}, alarmChannelCount)
	seen := make(map[<-chan struct{}]bool)
	for i := 0; i < alarmChannelCount*4; i++ {
		ch := clock.SetAlarm(targets[i%len(targets)])
		if i < alarmChannelCount {
			if seen[ch] {
				t.Fatal("distinct shards returned the same channel")
			}
			seen[ch] = true
			channels[i] = ch
		} else if ch != channels[i%alarmChannelCount] {
			t.Fatal("equivalent targets did not share round-robin channels")
		}
	}
	for _, ch := range channels {
		awaitAlarm(t, ch, at)
	}
	awaitCleanup(t, clock)
}

func TestConcurrentRegistrationsAndAllWaitersWake(t *testing.T) {
	const count = 1000
	clock := &AlarmClock{}
	at := time.Now().Add(250 * time.Millisecond)
	channels := make([]<-chan struct{}, count)
	observed := make([]time.Time, count)
	var registered, completed sync.WaitGroup
	registered.Add(count)
	completed.Add(count)
	for i := range count {
		go func() {
			channels[i] = clock.SetAlarm(at)
			registered.Done()
			<-channels[i]
			observed[i] = time.Now()
			completed.Done()
		}()
	}
	registered.Wait()
	usage := make(map[<-chan struct{}]int)
	for _, ch := range channels {
		usage[ch]++
	}
	if len(usage) != alarmChannelCount {
		t.Fatalf("channel count = %d, want %d", len(usage), alarmChannelCount)
	}
	for _, users := range usage {
		if users != count/alarmChannelCount {
			t.Fatalf("unbalanced channel allocation: %d", users)
		}
	}
	done := make(chan struct{})
	go func() {
		completed.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("not all waiters woke up")
	}
	for _, received := range observed {
		if received.Before(at) {
			t.Fatal("a waiter resumed before the target time")
		}
	}
	awaitCleanup(t, clock)
}

func TestDifferentTargetsAndInstancesAreIndependent(t *testing.T) {
	firstClock, secondClock := &AlarmClock{}, &AlarmClock{}
	firstAt := time.Now().Add(20 * time.Millisecond)
	secondAt := firstAt.Add(20 * time.Millisecond)
	first := firstClock.SetAlarm(firstAt)
	second := firstClock.SetAlarm(secondAt)
	otherInstance := secondClock.SetAlarm(firstAt)
	if first == second || first == otherInstance || second == otherInstance {
		t.Fatal("independent alarms shared a channel")
	}
	awaitAlarm(t, first, firstAt)
	awaitAlarm(t, second, secondAt)
	awaitAlarm(t, otherInstance, firstAt)
	awaitCleanup(t, firstClock)
	awaitCleanup(t, secondClock)
}

func TestPastTargetsAndConcurrentCleanup(t *testing.T) {
	clock := New()
	at := time.Now().Add(-time.Second)
	const count = 200
	channels := make([]<-chan struct{}, count)
	var registered sync.WaitGroup
	registered.Add(count)
	for i := range count {
		go func() {
			channels[i] = clock.SetAlarm(at)
			registered.Done()
		}()
	}
	registered.Wait()
	for _, ch := range channels {
		awaitAlarm(t, ch, at)
	}
	awaitCleanup(t, clock)
	awaitAlarm(t, clock.SetAlarm(at), at)
	awaitCleanup(t, clock)
}

func TestRealSpinLoopWithBothClockBases(t *testing.T) {
	for _, wallOnly := range []bool{false, true} {
		clock := New()
		if clock.spinWindow != time.Minute {
			t.Fatal("default spin window changed")
		}
		at := time.Now().Add(10 * time.Millisecond)
		if wallOnly {
			at = at.Round(0)
		}
		awaitAlarm(t, clock.SetAlarm(at), at)
		awaitCleanup(t, clock)
	}
}

func TestSleepThenSpin(t *testing.T) {
	clock := &AlarmClock{spinWindow: time.Millisecond}
	at := time.Now().Add(20 * time.Millisecond)
	awaitAlarm(t, clock.SetAlarm(at), at)
	awaitCleanup(t, clock)
}
