package adminui

import (
	"fmt"
	"net/http"
	"testing"
	"time"
)

func TestObservabilityTrackerMemoryBounded(t *testing.T) {
	tracker := NewObservabilityTracker(ObservabilityOptions{
		Advanced:   true,
		SampleRate: 1,
	})

	const total = 5 * maxTraceSamples // well beyond every buffer cap
	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < total; i++ {
		tracker.RecordRequest(RequestEvent{
			RPCName:      "svc.Method",
			Path:         "/rpc/svc.Method",
			HTTPMethod:   http.MethodPost,
			StatusCode:   http.StatusInternalServerError,
			DurationMS:   int64(i % 100),
			Timestamp:    base.Add(time.Duration(i) * time.Millisecond),
			ErrorMessage: fmt.Sprintf("failure %d", i), // distinct fingerprint per event
		}, []GuardDecisionEvent{{GuardName: "BearerAuth", Allowed: i%2 == 0}})
	}

	tracker.mu.RLock()
	route := tracker.routes["svc.Method"]
	if route == nil {
		t.Fatalf("expected route aggregate to exist")
	}
	if got := len(route.recentLatencies); got > latencyWindowSize {
		t.Fatalf("latency ring exceeded cap: %d > %d", got, latencyWindowSize)
	}
	if got := len(tracker.traces); got > maxTraceSamples {
		t.Fatalf("trace buffer exceeded cap: %d > %d", got, maxTraceSamples)
	}
	if got := len(tracker.errors); got > maxErrorFingerprints {
		t.Fatalf("error fingerprints exceeded cap: %d > %d", got, maxErrorFingerprints)
	}
	if got := len(tracker.guards); got != 1 {
		t.Fatalf("expected 1 guard counter, got %d", got)
	}
	tracker.mu.RUnlock()

	snapshot := tracker.Snapshot()
	if snapshot.Totals.Requests != total {
		t.Fatalf("expected totals to count all %d requests, got %d", total, snapshot.Totals.Requests)
	}
	if len(snapshot.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(snapshot.Routes))
	}
	routeAgg := snapshot.Routes[0]
	if routeAgg.Requests != total {
		t.Fatalf("expected route counter %d, got %d", total, routeAgg.Requests)
	}
	if routeAgg.ServerErrors != total {
		t.Fatalf("expected %d server errors, got %d", total, routeAgg.ServerErrors)
	}
	if len(snapshot.RecentTraces) > maxTraceSamples {
		t.Fatalf("snapshot traces exceeded cap: %d > %d", len(snapshot.RecentTraces), maxTraceSamples)
	}
	if len(snapshot.Errors) > maxErrorFingerprints {
		t.Fatalf("snapshot errors exceeded cap: %d > %d", len(snapshot.Errors), maxErrorFingerprints)
	}
	if len(snapshot.Guards) != 1 {
		t.Fatalf("expected 1 guard aggregate, got %d", len(snapshot.Guards))
	}
	if got := snapshot.Guards[0].AllowedCount + snapshot.Guards[0].DeniedCount; got != total {
		t.Fatalf("expected guard counters to cover all %d requests, got %d", total, got)
	}
}

func TestObservabilityTrackerEvictsOldestErrorFingerprint(t *testing.T) {
	tracker := NewObservabilityTracker(ObservabilityOptions{Advanced: true})

	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < maxErrorFingerprints+1; i++ {
		tracker.RecordRequest(RequestEvent{
			RPCName:      "svc.Method",
			StatusCode:   http.StatusInternalServerError,
			Timestamp:    base.Add(time.Duration(i) * time.Second),
			ErrorMessage: fmt.Sprintf("failure %d", i),
		}, nil)
	}

	snapshot := tracker.Snapshot()
	if len(snapshot.Errors) != maxErrorFingerprints {
		t.Fatalf("expected %d retained fingerprints, got %d", maxErrorFingerprints, len(snapshot.Errors))
	}
	for _, item := range snapshot.Errors {
		if item.ErrorMessage == "failure 0" {
			t.Fatalf("expected oldest fingerprint to be evicted")
		}
	}
}

func TestObservabilityTrackerLatencyAggregates(t *testing.T) {
	tracker := NewObservabilityTracker(ObservabilityOptions{})

	durations := []int64{10, 20, 30, 40}
	for _, d := range durations {
		tracker.RecordRequest(RequestEvent{
			RPCName:    "svc.Method",
			DurationMS: d,
		}, nil)
	}

	snapshot := tracker.Snapshot()
	if len(snapshot.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(snapshot.Routes))
	}
	route := snapshot.Routes[0]
	if route.MinLatencyMS != 10 || route.MaxLatencyMS != 40 {
		t.Fatalf("unexpected min/max latency: %d/%d", route.MinLatencyMS, route.MaxLatencyMS)
	}
	if route.AvgLatencyMS != 25 {
		t.Fatalf("expected avg latency 25, got %v", route.AvgLatencyMS)
	}
	if route.P50LatencyMS != 20 {
		t.Fatalf("expected p50 latency 20, got %v", route.P50LatencyMS)
	}
	if route.LastRequest.IsZero() {
		t.Fatalf("expected last-request timestamp to be set")
	}
}
