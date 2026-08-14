package worker

import (
	"context"
	"testing"
)

func TestQueueDepthCollectorDisabledWithoutMgmtURL(t *testing.T) {
	// StartQueueDepthCollector must return immediately when unconfigured;
	// it should not block the caller.
	done := make(chan struct{})
	go func() {
		StartQueueDepthCollector(context.Background(), nil, nil)
		close(done)
	}()
	<-done
}
