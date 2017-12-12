package monad

import (
	"sync"
	"time"
)

var (
	defaultInterval  = time.Duration(500 * time.Millisecond)
	defaultCooldDown = time.Duration(10 * time.Second)
)

type Config struct {
	Min                int64
	Max                int64
	Desired            int64
	ThresholdScaleUp   int64
	ThresholdScaleDown int64
	Interval           time.Duration
	CoolDownPeriod     time.Duration
	LenFn              func() int64
	DesireFn           func(n int64)
	StatusFn           func() bool
}

type Monad struct {
	sync.Mutex
	cfg         *Config
	t           *time.Ticker
	done        chan struct{}
	desired     int64
	lastActivty time.Time
}

func New(cfg *Config) *Monad {
	m := &Monad{
		done: make(chan struct{}, 1),
	}
	m.Reload(cfg)

	go m.monitor()

	return m
}

func (m *Monad) Reload(newCfg *Config) {
	m.Lock()
	defer m.Unlock()

	m.cfg = &Config{}
	*m.cfg = *newCfg

	if m.cfg.Interval.Nanoseconds() == 0 {
		m.cfg.Interval = defaultInterval
	}

	if m.cfg.CoolDownPeriod.Nanoseconds() == 0 {
		m.cfg.CoolDownPeriod = defaultCooldDown
	}

	if m.t == nil {
		m.t = time.NewTicker(m.cfg.Interval)
	} else {
		m.t.Stop()
		m.t = time.NewTicker(m.cfg.Interval)
	}
}

func (m *Monad) Exit() {
	m.done <- struct{}{}
}

func (m *Monad) monitor() {
	for {
		select {
		case <-m.done:
			m.t.Stop()
			m.t = nil
			return
		case <-m.t.C:

			l := m.cfg.LenFn()

			switch {
			case l > m.cfg.ThresholdScaleUp && m.desired <= m.cfg.Max:
				m.desired++
				m.cfg.DesireFn(m.desired)
				m.lastActivty = time.Now()
				break
			case m.desired > 0 && l < m.cfg.ThresholdScaleDown && m.desired > m.cfg.Min:
				if m.lastActivty.Add(m.cfg.CoolDownPeriod).Before(time.Now()) {
					m.desired--
					m.cfg.DesireFn(m.desired)
					m.lastActivty = time.Now()
				}
				break
			}

		}
	}
}
