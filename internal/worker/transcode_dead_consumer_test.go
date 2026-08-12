package worker

import (
	"cakecake/internal/model/video"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeCloseable struct {
	closed bool
}

func (f *fakeCloseable) Close() error {
	f.closed = true
	return nil
}

type fakeDeadSubscriber struct {
	calls int
	errs  []error
	ch    *fakeCloseable
	msgs  chan amqp.Delivery
}

func (f *fakeDeadSubscriber) NewTranscodeDeadConsumer(_ string) (interface{ Close() error }, <-chan amqp.Delivery, error) {
	f.calls++
	if len(f.errs) > 0 {
		err := f.errs[0]
		f.errs = f.errs[1:]
		return nil, nil, err
	}
	return f.ch, f.msgs, nil
}

func TestHandleTranscodeDeadLetter_Acks(t *testing.T) {
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 42, RetryCount: 3})
	handleTranscodeDeadLetter(amqp.Delivery{Body: body, Acknowledger: ack}, zap.NewNop(), nil)
	require.Equal(t, 1, ack.acked)
}

func TestHandleTranscodeDeadLetter_MarksProcessed(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{
		VideoID: 42, RetryCount: 3, Reason: "x", PayloadJSON: "{}",
	}).Error)
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 42, RetryCount: 3})
	handleTranscodeDeadLetter(amqp.Delivery{Body: body, Acknowledger: ack}, zap.NewNop(), db)
	require.Equal(t, 1, ack.acked)
	var rec video.TranscodeDeadLetter
	require.NoError(t, db.First(&rec).Error)
	require.NotNil(t, rec.ProcessedAt)
}

func TestConsumeTranscodeDead_AcksAndCloses(t *testing.T) {
	ack := &fakeAck{}
	body, _ := json.Marshal(TranscodeJob{VideoID: 7, RetryCount: 2})
	msgs := make(chan amqp.Delivery, 1)
	msgs <- amqp.Delivery{Body: body, Acknowledger: ack}
	close(msgs)

	closer := &fakeCloseable{}
	consumeTranscodeDead(context.Background(), closer, msgs, zap.NewNop(), nil)
	require.Equal(t, 1, ack.acked)
	require.True(t, closer.closed)
}

func TestStartTranscodeDeadConsumer_ReconnectsUntilCancel(t *testing.T) {
	old := transcodeDeadRetryDelay
	transcodeDeadRetryDelay = time.Millisecond
	defer func() { transcodeDeadRetryDelay = old }()

	sub := &fakeDeadSubscriber{
		errs: []error{errTestBrokerDown, errTestBrokerDown},
		ch:   &fakeCloseable{},
		msgs: make(chan amqp.Delivery),
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		StartTranscodeDeadConsumer(ctx, nil, nil, sub, zap.NewNop())
	}()

	// Let the loop hit the subscribe errors and reach the stable msgs channel.
	deadline := time.Now().Add(2 * time.Second)
	for sub.calls < 3 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	cancel()
	<-done
	require.GreaterOrEqual(t, sub.calls, 3)
}

func TestCleanupTranscodeDeadLetters_RemovesOnlyResolvedOld(t *testing.T) {
	db := newBehavioralWorkerDB(t)
	old := time.Now().Add(-40 * 24 * time.Hour)
	recent := time.Now().Add(-time.Hour)

	tmp := t.TempDir()
	processedRaw := filepath.Join(tmp, "processed.mp4")
	requeuedRaw := filepath.Join(tmp, "requeued.mp4")
	recentRaw := filepath.Join(tmp, "recent.mp4")
	for _, p := range []string{processedRaw, requeuedRaw, recentRaw} {
		require.NoError(t, os.WriteFile(p, []byte("media"), 0o644))
	}
	payload := func(raw string) string {
		b, _ := json.Marshal(map[string]interface{}{"job": TranscodeJob{VideoID: 1, RawPath: raw}, "reason": "x"})
		return string(b)
	}
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{VideoID: 1, RetryCount: 1, Reason: "old processed", ProcessedAt: &old, PayloadJSON: payload(processedRaw)}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{VideoID: 2, RetryCount: 1, Reason: "old requeued", RequeuedAt: &old, PayloadJSON: payload(requeuedRaw)}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{VideoID: 3, RetryCount: 1, Reason: "recent processed", ProcessedAt: &recent, PayloadJSON: payload(recentRaw)}).Error)
	require.NoError(t, db.Create(&video.TranscodeDeadLetter{VideoID: 4, RetryCount: 1, Reason: "pending"}).Error)

	deleted := cleanupTranscodeDeadLetters(db, time.Now().Add(-30*24*time.Hour), zap.NewNop())
	require.Equal(t, int64(2), deleted)
	var left int64
	require.NoError(t, db.Model(&video.TranscodeDeadLetter{}).Count(&left).Error)
	require.Equal(t, int64(2), left)

	_, err := os.Stat(processedRaw)
	require.ErrorIs(t, err, os.ErrNotExist, "files of an old processed row are archived")
	_, err = os.Stat(requeuedRaw)
	require.NoError(t, err, "files of an old requeued row may still feed a successor job")
	_, err = os.Stat(recentRaw)
	require.NoError(t, err, "files of a recent row are not touched")
}

var errTestBrokerDown = &testBrokerError{}

type testBrokerError struct{}

func (*testBrokerError) Error() string { return "broker down" }
