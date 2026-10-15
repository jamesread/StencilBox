package clientapi

import (
	"context"
	"testing"
	"time"
)

func TestShutdownWaitsForBuildsAndRefusesNewOnes(t *testing.T) {
	api := &ClientApi{}

	if !api.beginBuild() {
		t.Fatal("expected build to be accepted before shutdown")
	}

	finished := make(chan struct{})
	go func() {
		api.Shutdown(context.Background())
		close(finished)
	}()

	select {
	case <-finished:
		t.Fatal("shutdown returned while a build was still running")
	case <-time.After(50 * time.Millisecond):
	}

	if api.beginBuild() {
		t.Fatal("expected new builds to be refused during shutdown")
	}

	api.builds.Done()

	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("shutdown did not return after the build finished")
	}
}

func TestShutdownTimesOutAndCancelsBuilds(t *testing.T) {
	api := &ClientApi{}
	api.buildCtx, api.buildCancel = context.WithCancel(context.Background())
	api.beginBuild()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	api.Shutdown(ctx)

	if api.buildCtx.Err() == nil {
		t.Fatal("expected build context to be cancelled after shutdown timeout")
	}
}
