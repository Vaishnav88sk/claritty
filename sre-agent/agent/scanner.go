package main

import (
	"context"
	"log"
	"sync"
	"time"
)

type scanController struct {
	interval time.Duration
	scanSlot chan struct{}
	run      func(context.Context)

	mu          sync.Mutex
	watchCancel context.CancelFunc
}

func (s *scanController) trigger(ctx context.Context) bool {
	if ctx.Err() != nil {
		return false
	}
	select {
	case s.scanSlot <- struct{}{}:
		go s.scan(ctx)
		return true
	default:
		return false
	}
}

func (s *scanController) startWatch(ctx context.Context) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.watchCancel != nil || ctx.Err() != nil {
		return false
	}
	ctx, s.watchCancel = context.WithCancel(ctx)
	go s.watch(ctx)
	return true
}

func (s *scanController) stopWatch() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.watchCancel != nil {
		s.watchCancel()
	}
}

func (s *scanController) watching() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.watchCancel != nil
}

func (s *scanController) watch(ctx context.Context) {
	log.Printf("Continuous watch started (interval=%s)", s.interval)
	defer func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.watchCancel()
		s.watchCancel = nil
		log.Println("Continuous watch stopped")
	}()

	for {
		select {
		case s.scanSlot <- struct{}{}:
			s.scan(ctx)
		case <-ctx.Done():
			return
		}
		timer := time.NewTimer(s.interval)
		select {
		case <-timer.C:
		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

// scan runs with the slot held until both analysis and report delivery finish.
func (s *scanController) scan(ctx context.Context) {
	defer func() { <-s.scanSlot }()
	if ctx.Err() != nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	s.run(ctx)
}

func (s *scanController) waitForScan(ctx context.Context) {
	select {
	case s.scanSlot <- struct{}{}:
		<-s.scanSlot
	case <-ctx.Done():
	}
}
