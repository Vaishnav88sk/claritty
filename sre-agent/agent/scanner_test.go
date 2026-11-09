package main

import (
	"context"
	"testing"
	"time"
)

func TestScanControllerSerializesManualAndScheduledScans(t *testing.T) {
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	started := make(chan context.Context)
	finished := make(chan struct{})
	releaseManual := make(chan struct{})
	scanner := &scanController{
		interval: time.Hour,
		scanSlot: make(chan struct{}, 1),
		run: func(scanCtx context.Context) {
			select {
			case started <- scanCtx:
			case <-scanCtx.Done():
				return
			}
			select {
			case <-releaseManual:
				return
			case <-scanCtx.Done():
				close(finished)
			}
		},
	}

	if !scanner.trigger(ctx) {
		t.Fatal("manual scan was rejected while idle")
	}
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("manual scan did not start")
	}
	if !scanner.startWatch(ctx) || scanner.startWatch(ctx) {
		t.Fatal("expected exactly one watch to start")
	}
	if scanner.trigger(ctx) {
		t.Fatal("second manual scan was accepted while busy")
	}
	select {
	case releaseManual <- struct{}{}:
	case <-started:
		t.Fatal("scheduled scan overlapped the manual scan")
	case <-ctx.Done():
		t.Fatal("manual scan did not finish")
	}
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("scheduled scan did not start after the manual scan")
	}
	if scanner.trigger(ctx) {
		t.Fatal("manual scan was accepted during scheduled scan")
	}
	scanner.stopWatch()
	select {
	case <-finished:
	case <-ctx.Done():
		t.Fatal("stopping watch did not cancel the active scan")
	}
}
