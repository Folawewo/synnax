package time

import (
	"time"
)

// ScaledTicker is a ticker that scales the duration between ticks.
// It provides an identical interface to a time.Ticker.
type ScaledTicker struct {
	C     <-chan time.Duration
	Scale float64
	dur   time.Duration
	stop  chan struct{}
}

// Stop stops the ticker
func (s *ScaledTicker) Stop() { close(s.stop) }

func (s *ScaledTicker) tick(c chan time.Duration) {
	t := time.NewTicker(s.dur)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.dur = time.Duration(float64(s.dur) * s.Scale)
			c <- s.dur
			t.Reset(s.dur)
		}
	}
}

// NewScaledTicker returns a new ScaledTicker that ticks at the given duration and scale.
func NewScaledTicker(d time.Duration, scale float64) *ScaledTicker {
	c := make(chan time.Duration)
	t := &ScaledTicker{dur: d, Scale: scale, stop: make(chan struct{}), C: c}
	go t.tick(c)
	return t
}
