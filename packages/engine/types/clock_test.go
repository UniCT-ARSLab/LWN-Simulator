package types

import (
	"context"
	"testing"
	"time"
)

func TestManualClockDoesNotMoveBackwards(t *testing.T) {
	clock := NewManualClock(10 * time.Millisecond)

	if err := clock.Advance(5 * time.Millisecond); err != nil {
		t.Fatalf("advance clock: %v", err)
	}
	if got := clock.Now(); got != 15*time.Millisecond {
		t.Fatalf("expected 15ms, got %s", got)
	}

	if err := clock.Set(14 * time.Millisecond); err == nil {
		t.Fatal("expected backwards clock movement to fail")
	}
	if err := clock.Advance(-time.Millisecond); err == nil {
		t.Fatal("expected negative clock delta to fail")
	}
	if err := clock.Set(20 * time.Millisecond); err != nil || clock.Now() != 20*time.Millisecond {
		t.Fatalf("expected clock to move forward with Set, got %s / %v", clock.Now(), err)
	}
}

func TestManualClockWaitsUntilTarget(t *testing.T) {
	clock := NewManualClock(0)
	done := make(chan error, 1)

	go func() {
		done <- clock.WaitUntil(context.Background(), 100*time.Millisecond, 1, make(chan struct{}))
	}()

	select {
	case err := <-done:
		t.Fatalf("clock returned before target: %v", err)
	case <-time.After(10 * time.Millisecond):
	}

	if err := clock.Advance(100 * time.Millisecond); err != nil {
		t.Fatalf("advance clock: %v", err)
	}

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("wait until target: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("clock did not reach target")
	}
}

func TestRealClockPauseFreezesSimulationTime(t *testing.T) {
	clock := NewRealClock(20)
	clock.Start()
	time.Sleep(5 * time.Millisecond)
	beforePause := clock.Now()
	clock.Pause()
	time.Sleep(15 * time.Millisecond)
	if got := clock.Now(); got-beforePause > 2*time.Millisecond {
		t.Fatalf("paused clock advanced from %s to %s", beforePause, got)
	}
	clock.Resume()
	time.Sleep(5 * time.Millisecond)
	if got := clock.Now(); got <= beforePause {
		t.Fatalf("resumed clock did not advance: before=%s after=%s", beforePause, got)
	}
}

func TestRealClockWaitUntilWakesOnPause(t *testing.T) {
	clock := NewRealClock(1)
	clock.Start()
	wake := make(chan struct{}, 1)
	done := make(chan error, 1)
	go func() {
		done <- clock.WaitUntil(context.Background(), time.Second, 1, wake)
	}()
	time.Sleep(5 * time.Millisecond)
	clock.Pause()
	wake <- struct{}{}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("wait returned an error after wake: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("clock wait did not wake after pause")
	}
}

func TestRealClockChangesSpeedWithoutResettingElapsedTime(t *testing.T) {
	clock := NewRealClock(1)
	clock.Start()
	time.Sleep(5 * time.Millisecond)
	clock.Pause()
	pausedElapsed := clock.Now()

	if err := clock.SetSpeed(4); err != nil {
		t.Fatalf("set clock speed: %v", err)
	}
	if got := clock.Now(); got != pausedElapsed {
		t.Fatalf("changing speed while paused changed elapsed time: before=%s after=%s", pausedElapsed, got)
	}

	clock.Resume()
	time.Sleep(5 * time.Millisecond)
	if got := clock.Now(); got <= pausedElapsed {
		t.Fatalf("clock did not advance after changing speed: before=%s after=%s", pausedElapsed, got)
	}
}

func TestRealClockRestoresElapsedTimeAndRejectsInvalidSpeed(t *testing.T) {
	clock := NewRealClock(1)
	clock.SetElapsed(42 * time.Second)
	if got := clock.Now(); got != 42*time.Second {
		t.Fatalf("restored elapsed time is %s", got)
	}
	clock.Start()
	clock.SetElapsed(3 * time.Second)
	if got := clock.Now(); got < 3*time.Second || got > 3100*time.Millisecond {
		t.Fatalf("running clock did not reset around restored elapsed time: %s", got)
	}
	if err := clock.SetSpeed(0); err == nil {
		t.Fatal("zero speed was accepted")
	}
	if err := clock.SetSpeed(-1); err == nil {
		t.Fatal("negative speed was accepted")
	}
}

func TestClockWaitCancellationAndImmediateTargets(t *testing.T) {
	manual := NewManualClock(time.Second)
	if err := manual.WaitUntil(context.Background(), time.Second, 1, nil); err != nil {
		t.Fatalf("manual clock should immediately satisfy past target: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := manual.WaitUntil(ctx, 2*time.Second, 1, nil); err == nil {
		t.Fatal("manual clock ignored cancellation")
	}

	real := NewRealClock(1)
	real.Start()
	if err := real.WaitUntil(context.Background(), 0, 1, nil); err != nil {
		t.Fatalf("real clock should immediately satisfy past target: %v", err)
	}
	ctx, cancel = context.WithCancel(context.Background())
	cancel()
	if err := real.WaitUntil(ctx, time.Hour, 1, nil); err == nil {
		t.Fatal("real clock ignored cancellation")
	}
}
