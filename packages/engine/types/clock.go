package types

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Clock represents simulation time. Implementations must be safe to use from
// the engine loop and from control operations that interrupt a wait.
type Clock interface {
	Now() time.Duration
	WaitUntil(ctx context.Context, target time.Duration, speed float64, wake <-chan struct{}) error
}

// LifecycleClock lets the engine freeze and resume wall-clock based time.
// ManualClock deliberately does not implement it: tests control that clock by
// advancing it explicitly.
type LifecycleClock interface {
	Clock
	Start()
	Pause()
	Resume()
}

// RealClock maps simulation time to wall-clock time using the configured
// simulation speed.
type RealClock struct {
	mu        sync.Mutex
	startedAt time.Time
	elapsed   time.Duration
	speed     float64
	started   bool
	paused    bool
}

func NewRealClock(speed float64) *RealClock {
	return &RealClock{speed: speed}
}

func (c *RealClock) Now() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.nowLocked(time.Now())
}

func (c *RealClock) Start() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.startedAt = time.Now()
	c.started = true
	c.paused = false
}

func (c *RealClock) SetElapsed(elapsed time.Duration) {
	if elapsed < 0 {
		elapsed = 0
	}
	c.mu.Lock()
	c.elapsed = elapsed
	c.mu.Unlock()
}

// SetSpeed changes the wall-clock multiplier while preserving the simulation
// time already accumulated. The current elapsed value is captured before the
// new multiplier is applied so changing speed never freezes or jumps the
// simulation clock.
func (c *RealClock) SetSpeed(speed float64) error {
	if speed <= 0 {
		return fmt.Errorf("clock speed must be greater than zero")
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.started && !c.paused {
		now := time.Now()
		c.elapsed = c.nowLocked(now)
		c.startedAt = now
	}
	c.speed = speed
	return nil
}

func (c *RealClock) Pause() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started || c.paused {
		return
	}
	c.elapsed = c.nowLocked(time.Now())
	c.paused = true
}

func (c *RealClock) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.started || !c.paused {
		return
	}
	c.startedAt = time.Now()
	c.paused = false
}

func (c *RealClock) nowLocked(now time.Time) time.Duration {
	if !c.started || c.paused {
		return c.elapsed
	}
	return c.elapsed + time.Duration(float64(now.Sub(c.startedAt))*c.speed)
}

func (c *RealClock) WaitUntil(
	ctx context.Context,
	target time.Duration,
	speed float64,
	wake <-chan struct{},
) error {
	if speed <= 0 {
		return fmt.Errorf("clock speed must be greater than zero")
	}

	delta := target - c.Now()
	if delta <= 0 {
		return nil
	}

	wallDuration := time.Duration(float64(delta) / speed)
	timer := time.NewTimer(wallDuration)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	case <-wake:
		return nil
	}
}

// ManualClock is a deterministic clock intended for tests and step-by-step
// simulation control. Advance never moves time backwards.
type ManualClock struct {
	mu      sync.Mutex
	now     time.Duration
	changed chan struct{}
}

func NewManualClock(start time.Duration) *ManualClock {
	return &ManualClock{
		now:     start,
		changed: make(chan struct{}),
	}
}

func (c *ManualClock) Now() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *ManualClock) Advance(delta time.Duration) error {
	if delta < 0 {
		return fmt.Errorf("clock delta cannot be negative")
	}

	c.mu.Lock()
	c.now += delta
	close(c.changed)
	c.changed = make(chan struct{})
	c.mu.Unlock()

	return nil
}

func (c *ManualClock) Set(target time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if target < c.now {
		return fmt.Errorf("clock cannot move backwards")
	}

	c.now = target
	close(c.changed)
	c.changed = make(chan struct{})
	return nil
}

func (c *ManualClock) WaitUntil(
	ctx context.Context,
	target time.Duration,
	_ float64,
	wake <-chan struct{},
) error {
	for {
		c.mu.Lock()
		if c.now >= target {
			c.mu.Unlock()
			return nil
		}
		changed := c.changed
		c.mu.Unlock()

		select {
		case <-changed:
		case <-wake:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
