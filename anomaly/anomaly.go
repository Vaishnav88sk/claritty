package anomaly

import (
    "math"
    "sync"
    "time"
)

// OnlineStats maintains mean/variance using Welford's algorithm
type OnlineStats struct {
    n    int64
    mean float64
    m2   float64
    mu   sync.Mutex
}

func NewOnlineStats() *OnlineStats { return &OnlineStats{} }

func (s *OnlineStats) Add(x float64) {
    s.mu.Lock()
    defer s.mu.Unlock()

    s.n++
    delta := x - s.mean
    s.mean += delta / float64(s.n)
    delta2 := x - s.mean
    s.m2 += delta * delta2
}

func (s *OnlineStats) Mean() float64 {
    s.mu.Lock()
    defer s.mu.Unlock()
    return s.mean
}

func (s *OnlineStats) Variance() float64 {
    s.mu.Lock()
    defer s.mu.Unlock()
    if s.n < 2 {
        return 0
    }
    return s.m2 / float64(s.n-1)
}

func (s *OnlineStats) StdDev() float64 {
    return math.Sqrt(s.Variance())
}

// EWMA detector
type EWMA struct {
    alpha float64
    mean  float64
    init  bool
    mu    sync.Mutex
}

func NewEWMA(alpha float64) *EWMA { return &EWMA{alpha: alpha} }

func (e *EWMA) Add(x float64) {
    e.mu.Lock()
    defer e.mu.Unlock()
    if !e.init {
        e.mean = x
        e.init = true
        return
    }
    e.mean = e.alpha*x + (1-e.alpha)*e.mean
}

func (e *EWMA) Value() float64 {
    e.mu.Lock()
    defer e.mu.Unlock()
    return e.mean
}

// ZScore = |x - mean| / stdDev
func ZScore(s *OnlineStats, x float64) float64 {
    if s == nil {
        return 0
    }
    mean := s.Mean()
    sd := s.StdDev()
    if sd == 0 {
        return 0
    }
    return math.Abs(x-mean) / sd
}

// Hysteresis avoids flickering
type Hysteresis struct {
    consecutive int
    threshold   float64
    required    int
    state       bool
    mu          sync.Mutex
    lastChange  time.Time
}

func NewHysteresis(th float64, required int) *Hysteresis {
    return &Hysteresis{threshold: th, required: required}
}

func (h *Hysteresis) Update(score float64) bool {
    h.mu.Lock()
    defer h.mu.Unlock()

    if score >= h.threshold {
        h.consecutive++
        if h.consecutive >= h.required {
            h.state = true
            h.lastChange = time.Now()
        }
    } else {
        h.consecutive = 0
        h.state = false
    }
    return h.state
}
