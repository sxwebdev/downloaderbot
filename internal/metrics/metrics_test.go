package metrics_test

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	appmetrics "github.com/sxwebdev/downloaderbot/internal/metrics"
)

func TestTrackDownloadComplete(t *testing.T) {
	const (
		source  = "test_complete"
		payload = "complete payload"
	)

	bytesBefore := counterValue(t, appmetrics.MediaDownloadBytes.WithLabelValues(source))
	completedBefore := counterValue(t, appmetrics.MediaDownloadCompletedBytes.WithLabelValues(source))
	successBefore := counterValue(t, appmetrics.MediaDownloads.WithLabelValues(
		source, appmetrics.OutcomeSuccess, appmetrics.ReasonNone,
	))
	incompleteBefore := counterValue(t, appmetrics.MediaDownloads.WithLabelValues(
		source, appmetrics.OutcomeFailure, appmetrics.ReasonIncomplete,
	))

	body := appmetrics.TrackDownload(source, io.NopCloser(strings.NewReader(payload)))
	got, err := io.ReadAll(body)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != payload {
		t.Fatalf("downloaded body = %q, want %q", got, payload)
	}
	if err := body.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// A second close must not turn an already completed download into a failure.
	if err := body.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}

	assertCounterDelta(t, appmetrics.MediaDownloadBytes.WithLabelValues(source), bytesBefore, len(payload))
	assertCounterDelta(t, appmetrics.MediaDownloadCompletedBytes.WithLabelValues(source), completedBefore, len(payload))
	assertCounterDelta(t, appmetrics.MediaDownloads.WithLabelValues(
		source, appmetrics.OutcomeSuccess, appmetrics.ReasonNone,
	), successBefore, 1)
	assertCounterDelta(t, appmetrics.MediaDownloads.WithLabelValues(
		source, appmetrics.OutcomeFailure, appmetrics.ReasonIncomplete,
	), incompleteBefore, 0)
}

func TestTrackDownloadClosedBeforeEOF(t *testing.T) {
	const (
		source  = "test_partial"
		payload = "partial payload"
	)

	bytesBefore := counterValue(t, appmetrics.MediaDownloadBytes.WithLabelValues(source))
	completedBefore := counterValue(t, appmetrics.MediaDownloadCompletedBytes.WithLabelValues(source))
	failureBefore := counterValue(t, appmetrics.MediaDownloads.WithLabelValues(
		source, appmetrics.OutcomeFailure, appmetrics.ReasonIncomplete,
	))

	body := appmetrics.TrackDownload(source, io.NopCloser(strings.NewReader(payload)))
	buf := make([]byte, 3)
	n, err := body.Read(buf)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n != len(buf) {
		t.Fatalf("Read bytes = %d, want %d", n, len(buf))
	}
	if err := body.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	assertCounterDelta(t, appmetrics.MediaDownloadBytes.WithLabelValues(source), bytesBefore, len(buf))
	assertCounterDelta(t, appmetrics.MediaDownloadCompletedBytes.WithLabelValues(source), completedBefore, 0)
	assertCounterDelta(t, appmetrics.MediaDownloads.WithLabelValues(
		source, appmetrics.OutcomeFailure, appmetrics.ReasonIncomplete,
	), failureBefore, 1)
}

func TestTrackDownloadReadError(t *testing.T) {
	const source = "test_read_error"
	readErr := errors.New("upstream read failed")

	bytesBefore := counterValue(t, appmetrics.MediaDownloadBytes.WithLabelValues(source))
	failureBefore := counterValue(t, appmetrics.MediaDownloads.WithLabelValues(
		source, appmetrics.OutcomeFailure, appmetrics.ReasonRead,
	))

	body := appmetrics.TrackDownload(source, &errorReadCloser{data: []byte("abc"), err: readErr})
	buf := make([]byte, 8)
	n, err := body.Read(buf)
	if !errors.Is(err, readErr) {
		t.Fatalf("Read error = %v, want %v", err, readErr)
	}
	if n != 3 {
		t.Fatalf("Read bytes = %d, want 3", n)
	}
	if err := body.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	assertCounterDelta(t, appmetrics.MediaDownloadBytes.WithLabelValues(source), bytesBefore, 3)
	assertCounterDelta(t, appmetrics.MediaDownloads.WithLabelValues(
		source, appmetrics.OutcomeFailure, appmetrics.ReasonRead,
	), failureBefore, 1)
}

func TestTrackDownloadCloseError(t *testing.T) {
	const source = "test_close_error"
	closeErr := errors.New("upstream close failed")
	otherBefore := counterValue(t, appmetrics.MediaDownloads.WithLabelValues(
		source, appmetrics.OutcomeFailure, appmetrics.ReasonOther,
	))

	body := appmetrics.TrackDownload(source, &closeErrorReadCloser{closeErr: closeErr})
	if err := body.Close(); !errors.Is(err, closeErr) {
		t.Fatalf("Close error = %v, want %v", err, closeErr)
	}
	assertCounterDelta(t, appmetrics.MediaDownloads.WithLabelValues(
		source, appmetrics.OutcomeFailure, appmetrics.ReasonOther,
	), otherBefore, 1)
}

func TestExtractionAndDeliveryOutcomes(t *testing.T) {
	t.Run("logical extraction timeout uses bounded labels", func(t *testing.T) {
		before := counterValue(t, appmetrics.MediaExtractionRequests.WithLabelValues(
			"unknown", appmetrics.OutcomeFailure, appmetrics.ReasonTimeout,
		))
		appmetrics.ObserveExtractionRequest("", context.DeadlineExceeded)
		assertCounterDelta(t, appmetrics.MediaExtractionRequests.WithLabelValues(
			"unknown", appmetrics.OutcomeFailure, appmetrics.ReasonTimeout,
		), before, 1)
	})

	t.Run("successful attempt and request are separate", func(t *testing.T) {
		const source = "test_extraction_success"
		attemptBefore := counterValue(t, appmetrics.MediaExtractionAttempts.WithLabelValues(
			source, appmetrics.OutcomeSuccess, appmetrics.ReasonNone,
		))
		requestBefore := counterValue(t, appmetrics.MediaExtractionRequests.WithLabelValues(
			source, appmetrics.OutcomeSuccess, appmetrics.ReasonNone,
		))

		appmetrics.ObserveExtractionAttempt(source, time.Now(), nil)
		appmetrics.ObserveExtractionRequest(source, nil)

		assertCounterDelta(t, appmetrics.MediaExtractionAttempts.WithLabelValues(
			source, appmetrics.OutcomeSuccess, appmetrics.ReasonNone,
		), attemptBefore, 1)
		assertCounterDelta(t, appmetrics.MediaExtractionRequests.WithLabelValues(
			source, appmetrics.OutcomeSuccess, appmetrics.ReasonNone,
		), requestBefore, 1)
	})

	t.Run("canceled attempt updates new and compatibility metrics", func(t *testing.T) {
		const source = "test_extraction_canceled"
		attemptBefore := counterValue(t, appmetrics.MediaExtractionAttempts.WithLabelValues(
			source, appmetrics.OutcomeFailure, appmetrics.ReasonCanceled,
		))
		compatBefore := counterValue(t, appmetrics.ExtractErrors.WithLabelValues(
			source, appmetrics.ReasonCanceled,
		))

		appmetrics.ObserveExtractionAttempt(source, time.Now(), context.Canceled)

		assertCounterDelta(t, appmetrics.MediaExtractionAttempts.WithLabelValues(
			source, appmetrics.OutcomeFailure, appmetrics.ReasonCanceled,
		), attemptBefore, 1)
		assertCounterDelta(t, appmetrics.ExtractErrors.WithLabelValues(
			source, appmetrics.ReasonCanceled,
		), compatBefore, 1)
	})

	t.Run("delivery failure keeps compatibility counter", func(t *testing.T) {
		const (
			source = "test_delivery"
			kind   = "video"
		)
		deliveryBefore := counterValue(t, appmetrics.TelegramDeliveries.WithLabelValues(
			source, kind, appmetrics.OutcomeFailure, appmetrics.ReasonTelegram,
		))
		compatBefore := counterValue(t, appmetrics.TelegramSendErrors.WithLabelValues(kind))

		appmetrics.ObserveTelegramDelivery(source, kind, errors.New("telegram unavailable"))

		assertCounterDelta(t, appmetrics.TelegramDeliveries.WithLabelValues(
			source, kind, appmetrics.OutcomeFailure, appmetrics.ReasonTelegram,
		), deliveryBefore, 1)
		assertCounterDelta(t, appmetrics.TelegramSendErrors.WithLabelValues(kind), compatBefore, 1)
	})

	t.Run("successful delivery has no error reason", func(t *testing.T) {
		const (
			source = "test_delivery_success"
			kind   = "photo"
		)
		before := counterValue(t, appmetrics.TelegramDeliveries.WithLabelValues(
			source, kind, appmetrics.OutcomeSuccess, appmetrics.ReasonNone,
		))
		appmetrics.ObserveTelegramDelivery(source, kind, nil)
		assertCounterDelta(t, appmetrics.TelegramDeliveries.WithLabelValues(
			source, kind, appmetrics.OutcomeSuccess, appmetrics.ReasonNone,
		), before, 1)
	})

	t.Run("download failure normalizes source and reason", func(t *testing.T) {
		before := counterValue(t, appmetrics.MediaDownloads.WithLabelValues(
			"unknown", appmetrics.OutcomeFailure, appmetrics.ReasonOther,
		))
		appmetrics.ObserveDownloadFailure("", "raw upstream error text")
		assertCounterDelta(t, appmetrics.MediaDownloads.WithLabelValues(
			"unknown", appmetrics.OutcomeFailure, appmetrics.ReasonOther,
		), before, 1)
	})
}

type errorReadCloser struct {
	data []byte
	err  error
}

func (r *errorReadCloser) Read(p []byte) (int, error) {
	n := copy(p, r.data)
	r.data = r.data[n:]
	return n, r.err
}

func (*errorReadCloser) Close() error { return nil }

type closeErrorReadCloser struct {
	closeErr error
}

func (*closeErrorReadCloser) Read([]byte) (int, error) { return 0, nil }

func (r *closeErrorReadCloser) Close() error { return r.closeErr }

func counterValue(t *testing.T, collector prometheus.Collector) float64 {
	t.Helper()
	return testutil.ToFloat64(collector)
}

func assertCounterDelta(t *testing.T, collector prometheus.Collector, before float64, want int) {
	t.Helper()
	got := counterValue(t, collector) - before
	if got != float64(want) {
		t.Errorf("counter delta = %v, want %d", got, want)
	}
}
