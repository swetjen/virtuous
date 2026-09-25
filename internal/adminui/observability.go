package adminui

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	defaultTraceSampleRate = 0.1
	// maxTraceSamples caps the shared ring of sampled traces.
	maxTraceSamples = 200
	// maxErrorFingerprints caps the number of distinct error groups retained;
	// when full, the fingerprint with the oldest lastSeen is evicted.
	maxErrorFingerprints = 200
	// latencyWindowSize caps the per-route ring of recent latency samples used
	// for percentile estimates.
	latencyWindowSize = 256
)

// ObservabilityOptions configures the in-memory tracker.
type ObservabilityOptions struct {
	Advanced   bool
	SampleRate float64
}

// RequestEvent captures one RPC invocation for aggregation.
type RequestEvent struct {
	RPCName        string    `json:"rpcName"`
	Path           string    `json:"path"`
	HTTPMethod     string    `json:"httpMethod"`
	StatusCode     int       `json:"statusCode"`
	DurationMS     int64     `json:"durationMs"`
	Timestamp      time.Time `json:"timestamp"`
	GuardOutcome   string    `json:"guardOutcome,omitempty"`
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	StackSignature string    `json:"stackSignature,omitempty"`
}

// GuardDecisionEvent records one guard allow/deny result.
type GuardDecisionEvent struct {
	Timestamp time.Time `json:"timestamp"`
	RPCName   string    `json:"rpcName"`
	GuardName string    `json:"guardName"`
	Allowed   bool      `json:"allowed"`
}

// RouteAggregate summarizes request activity for one RPC.
//
// Counters are cumulative since process start (they previously covered a
// sliding 24-hour window). Latency percentiles are estimated from a bounded
// ring of the most recent latencyWindowSize samples; min/max/average cover
// every request since start.
type RouteAggregate struct {
	RPCName      string    `json:"rpcName"`
	Path         string    `json:"path"`
	HTTPMethod   string    `json:"httpMethod"`
	Requests     int       `json:"requests"`
	ClientErrors int       `json:"clientErrors"`
	ServerErrors int       `json:"serverErrors"`
	AvgLatencyMS float64   `json:"avgLatencyMs"`
	MinLatencyMS int64     `json:"minLatencyMs"`
	MaxLatencyMS int64     `json:"maxLatencyMs"`
	P50LatencyMS float64   `json:"p50LatencyMs"`
	P95LatencyMS float64   `json:"p95LatencyMs"`
	LastRequest  time.Time `json:"lastRequestAt"`
	TraceSamples int       `json:"traceSamples"`
}

// ErrorFingerprint groups repeated server-side failures for one RPC.
// Count is cumulative since process start, not a 24-hour window.
type ErrorFingerprint struct {
	RPCName         string    `json:"rpcName"`
	ErrorHash       string    `json:"errorHash"`
	ErrorMessage    string    `json:"errorMessage"`
	StackSignature  string    `json:"stackSignature,omitempty"`
	Count           int       `json:"count"`
	LastSeen        time.Time `json:"lastSeen"`
	TraceSampleHint bool      `json:"traceSampleHint"`
}

// GuardAggregate summarizes allow/deny activity for one guard on one RPC.
// Counts are cumulative since process start.
type GuardAggregate struct {
	RPCName           string  `json:"rpcName"`
	GuardName         string  `json:"guardName"`
	AllowedCount      int     `json:"allowedCount"`
	DeniedCount       int     `json:"deniedCount"`
	DenialRatePercent float64 `json:"denialRatePercent"`
}

// TraceSample captures a sampled request for future drill-down support.
type TraceSample struct {
	ID             string    `json:"id"`
	RPCName        string    `json:"rpcName"`
	Path           string    `json:"path"`
	HTTPMethod     string    `json:"httpMethod"`
	StatusCode     int       `json:"statusCode"`
	DurationMS     int64     `json:"durationMs"`
	Timestamp      time.Time `json:"timestamp"`
	GuardOutcome   string    `json:"guardOutcome,omitempty"`
	ErrorMessage   string    `json:"errorMessage,omitempty"`
	StackSignature string    `json:"stackSignature,omitempty"`
}

// MetricsTotals provides top-level summary counts for the dashboard.
// Counts are cumulative since process start.
type MetricsTotals struct {
	Requests     int `json:"requests"`
	ClientErrors int `json:"clientErrors"`
	ServerErrors int `json:"serverErrors"`
}

// MetricsSnapshot is the JSON payload for the observability dashboard.
type MetricsSnapshot struct {
	GeneratedAt   time.Time          `json:"generatedAt"`
	TrackingSince time.Time          `json:"trackingSince"`
	Advanced      bool               `json:"advanced"`
	SampleRate    float64            `json:"sampleRate"`
	Totals        MetricsTotals      `json:"totals"`
	Routes        []RouteAggregate   `json:"routes"`
	Errors        []ErrorFingerprint `json:"errors"`
	Guards        []GuardAggregate   `json:"guards"`
	RecentTraces  []TraceSample      `json:"recentTraces"`
	TraceViewerUI bool               `json:"traceViewerUi"`
}

// observabilityRoute holds fixed-size incremental aggregates for one RPC.
// Memory is O(1) per route: counters plus a bounded latency ring.
type observabilityRoute struct {
	path            string
	httpMethod      string
	requests        int
	clientErrors    int
	serverErrors    int
	totalLatencyMS  int64
	minLatencyMS    int64
	maxLatencyMS    int64
	recentLatencies []int64 // ring of at most latencyWindowSize samples
	latencyNext     int
	lastRequest     time.Time
	traceSamples    int
}

type guardCounter struct {
	rpcName   string
	guardName string
	allowed   int
	denied    int
}

type errorCounter struct {
	rpcName         string
	errorHash       string
	errorMessage    string
	stackSignature  string
	count           int
	lastSeen        time.Time
	traceSampleHint bool
}

// ObservabilityTracker keeps bounded in-memory aggregates updated at record
// time. Memory is O(routes + guards + capped fingerprints + capped traces),
// never O(requests).
type ObservabilityTracker struct {
	mu         sync.RWMutex
	advanced   bool
	sampleRate float64
	startedAt  time.Time
	routes     map[string]*observabilityRoute
	guards     map[string]*guardCounter
	errors     map[string]*errorCounter
	traces     []TraceSample // ring of at most maxTraceSamples
	traceNext  int
	random     *rand.Rand
}

// NewObservabilityTracker returns an in-memory tracker for request metrics.
func NewObservabilityTracker(opts ObservabilityOptions) *ObservabilityTracker {
	sampleRate := opts.SampleRate
	if sampleRate <= 0 {
		sampleRate = defaultTraceSampleRate
	}
	if sampleRate > 1 {
		sampleRate = 1
	}
	return &ObservabilityTracker{
		advanced:   opts.Advanced,
		sampleRate: sampleRate,
		startedAt:  time.Now().UTC(),
		routes:     make(map[string]*observabilityRoute),
		guards:     make(map[string]*guardCounter),
		errors:     make(map[string]*errorCounter),
		random:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Advanced reports whether advanced observability is enabled.
func (t *ObservabilityTracker) Advanced() bool {
	if t == nil {
		return false
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.advanced
}

// SampleRate returns the configured trace sampling rate.
func (t *ObservabilityTracker) SampleRate() float64 {
	if t == nil {
		return 0
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.sampleRate
}

// RecordRequest folds one request and any guard outcomes into the aggregates.
func (t *ObservabilityTracker) RecordRequest(event RequestEvent, guards []GuardDecisionEvent) {
	if t == nil {
		return
	}

	now := event.Timestamp.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}
	event.Timestamp = now
	event.RPCName = strings.TrimSpace(event.RPCName)
	event.Path = strings.TrimSpace(event.Path)
	event.HTTPMethod = strings.ToUpper(strings.TrimSpace(event.HTTPMethod))
	if event.HTTPMethod == "" {
		event.HTTPMethod = http.MethodPost
	}
	if event.StatusCode == 0 {
		event.StatusCode = http.StatusOK
	}
	if event.DurationMS < 0 {
		event.DurationMS = 0
	}
	event.ErrorMessage = strings.TrimSpace(event.ErrorMessage)
	event.StackSignature = strings.TrimSpace(event.StackSignature)
	event.GuardOutcome = strings.TrimSpace(strings.ToLower(event.GuardOutcome))

	t.mu.Lock()
	defer t.mu.Unlock()

	route := t.routes[event.RPCName]
	if route == nil {
		route = &observabilityRoute{}
		t.routes[event.RPCName] = route
	}
	if event.Path != "" {
		route.path = event.Path
	}
	if event.HTTPMethod != "" {
		route.httpMethod = event.HTTPMethod
	}

	route.requests++
	switch {
	case event.StatusCode >= 500:
		route.serverErrors++
	case event.StatusCode >= 400:
		route.clientErrors++
	}
	route.totalLatencyMS += event.DurationMS
	if route.requests == 1 || event.DurationMS < route.minLatencyMS {
		route.minLatencyMS = event.DurationMS
	}
	if event.DurationMS > route.maxLatencyMS {
		route.maxLatencyMS = event.DurationMS
	}
	if len(route.recentLatencies) < latencyWindowSize {
		route.recentLatencies = append(route.recentLatencies, event.DurationMS)
	} else {
		route.recentLatencies[route.latencyNext] = event.DurationMS
	}
	route.latencyNext = (route.latencyNext + 1) % latencyWindowSize
	if event.Timestamp.After(route.lastRequest) {
		route.lastRequest = event.Timestamp
	}

	if t.advanced {
		for _, decision := range guards {
			guardName := strings.TrimSpace(decision.GuardName)
			if guardName == "" {
				continue
			}
			t.recordGuardLocked(event.RPCName, guardName, decision.Allowed)
		}
		t.recordErrorLocked(event)
		if t.shouldSampleTraceLocked(event) {
			route.traceSamples++
			t.recordTraceLocked(TraceSample{
				ID:             traceSampleID(event),
				RPCName:        event.RPCName,
				Path:           event.Path,
				HTTPMethod:     event.HTTPMethod,
				StatusCode:     event.StatusCode,
				DurationMS:     event.DurationMS,
				Timestamp:      event.Timestamp,
				GuardOutcome:   event.GuardOutcome,
				ErrorMessage:   event.ErrorMessage,
				StackSignature: event.StackSignature,
			})
		}
	}
}

func (t *ObservabilityTracker) recordGuardLocked(rpcName, guardName string, allowed bool) {
	key := rpcName + "\x00" + guardName
	counter := t.guards[key]
	if counter == nil {
		counter = &guardCounter{
			rpcName:   rpcName,
			guardName: guardName,
		}
		t.guards[key] = counter
	}
	if allowed {
		counter.allowed++
	} else {
		counter.denied++
	}
}

func (t *ObservabilityTracker) recordErrorLocked(event RequestEvent) {
	if event.StatusCode < 500 {
		return
	}
	if event.ErrorMessage == "" && event.StackSignature == "" {
		return
	}
	hash := fingerprintHash(event.ErrorMessage, event.StackSignature)
	key := event.RPCName + "\x00" + hash
	counter := t.errors[key]
	if counter == nil {
		if len(t.errors) >= maxErrorFingerprints {
			t.evictOldestErrorLocked()
		}
		counter = &errorCounter{
			rpcName:        event.RPCName,
			errorHash:      hash,
			errorMessage:   event.ErrorMessage,
			stackSignature: event.StackSignature,
		}
		t.errors[key] = counter
	}
	counter.count++
	if event.Timestamp.After(counter.lastSeen) {
		counter.lastSeen = event.Timestamp
	}
	counter.traceSampleHint = counter.traceSampleHint || event.StackSignature != ""
}

func (t *ObservabilityTracker) evictOldestErrorLocked() {
	oldestKey := ""
	var oldestSeen time.Time
	for key, counter := range t.errors {
		if oldestKey == "" || counter.lastSeen.Before(oldestSeen) {
			oldestKey = key
			oldestSeen = counter.lastSeen
		}
	}
	if oldestKey != "" {
		delete(t.errors, oldestKey)
	}
}

func (t *ObservabilityTracker) recordTraceLocked(sample TraceSample) {
	if len(t.traces) < maxTraceSamples {
		t.traces = append(t.traces, sample)
	} else {
		t.traces[t.traceNext] = sample
	}
	t.traceNext = (t.traceNext + 1) % maxTraceSamples
}

// Snapshot returns the current aggregates. It is a cheap read: no
// re-aggregation over raw events happens here, only a copy of the
// O(routes)-sized state under a read lock.
func (t *ObservabilityTracker) Snapshot() MetricsSnapshot {
	if t == nil {
		return MetricsSnapshot{
			GeneratedAt: time.Now().UTC(),
		}
	}

	t.mu.RLock()
	defer t.mu.RUnlock()

	now := time.Now().UTC()
	snapshot := MetricsSnapshot{
		GeneratedAt:   now,
		TrackingSince: t.startedAt,
		Advanced:      t.advanced,
		SampleRate:    t.sampleRate,
		Routes:        []RouteAggregate{},
		Errors:        []ErrorFingerprint{},
		Guards:        []GuardAggregate{},
		RecentTraces:  []TraceSample{},
		TraceViewerUI: false,
	}

	for rpcName, route := range t.routes {
		aggregate := summarizeRoute(rpcName, route)
		snapshot.Routes = append(snapshot.Routes, aggregate)

		snapshot.Totals.Requests += aggregate.Requests
		snapshot.Totals.ClientErrors += aggregate.ClientErrors
		snapshot.Totals.ServerErrors += aggregate.ServerErrors
	}

	sort.Slice(snapshot.Routes, func(i, j int) bool {
		if snapshot.Routes[i].ServerErrors != snapshot.Routes[j].ServerErrors {
			return snapshot.Routes[i].ServerErrors > snapshot.Routes[j].ServerErrors
		}
		if snapshot.Routes[i].Requests != snapshot.Routes[j].Requests {
			return snapshot.Routes[i].Requests > snapshot.Routes[j].Requests
		}
		return snapshot.Routes[i].RPCName < snapshot.Routes[j].RPCName
	})

	if t.advanced {
		for _, counter := range t.errors {
			snapshot.Errors = append(snapshot.Errors, ErrorFingerprint{
				RPCName:         counter.rpcName,
				ErrorHash:       counter.errorHash,
				ErrorMessage:    counter.errorMessage,
				StackSignature:  counter.stackSignature,
				Count:           counter.count,
				LastSeen:        counter.lastSeen,
				TraceSampleHint: counter.traceSampleHint,
			})
		}
		sort.Slice(snapshot.Errors, func(i, j int) bool {
			if snapshot.Errors[i].Count != snapshot.Errors[j].Count {
				return snapshot.Errors[i].Count > snapshot.Errors[j].Count
			}
			return snapshot.Errors[i].RPCName < snapshot.Errors[j].RPCName
		})

		for _, counter := range t.guards {
			item := GuardAggregate{
				RPCName:      counter.rpcName,
				GuardName:    counter.guardName,
				AllowedCount: counter.allowed,
				DeniedCount:  counter.denied,
			}
			if total := counter.allowed + counter.denied; total > 0 {
				item.DenialRatePercent = (float64(counter.denied) / float64(total)) * 100
			}
			snapshot.Guards = append(snapshot.Guards, item)
		}
		sort.Slice(snapshot.Guards, func(i, j int) bool {
			if snapshot.Guards[i].DenialRatePercent != snapshot.Guards[j].DenialRatePercent {
				return snapshot.Guards[i].DenialRatePercent > snapshot.Guards[j].DenialRatePercent
			}
			if snapshot.Guards[i].DeniedCount != snapshot.Guards[j].DeniedCount {
				return snapshot.Guards[i].DeniedCount > snapshot.Guards[j].DeniedCount
			}
			if snapshot.Guards[i].RPCName != snapshot.Guards[j].RPCName {
				return snapshot.Guards[i].RPCName < snapshot.Guards[j].RPCName
			}
			return snapshot.Guards[i].GuardName < snapshot.Guards[j].GuardName
		})

		snapshot.RecentTraces = append(snapshot.RecentTraces, t.traces...)
		sort.Slice(snapshot.RecentTraces, func(i, j int) bool {
			return snapshot.RecentTraces[i].Timestamp.After(snapshot.RecentTraces[j].Timestamp)
		})
	}

	return snapshot
}

// ServeJSON serves the current snapshot as JSON.
func (t *ObservabilityTracker) ServeJSON(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(t.Snapshot())
}

func (t *ObservabilityTracker) shouldSampleTraceLocked(event RequestEvent) bool {
	if !t.advanced {
		return false
	}
	if event.StatusCode >= 500 || event.GuardOutcome == "deny" || event.StackSignature != "" {
		return true
	}
	return t.random.Float64() <= t.sampleRate
}

func summarizeRoute(rpcName string, route *observabilityRoute) RouteAggregate {
	out := RouteAggregate{
		RPCName:      rpcName,
		Path:         route.path,
		HTTPMethod:   route.httpMethod,
		Requests:     route.requests,
		ClientErrors: route.clientErrors,
		ServerErrors: route.serverErrors,
		MinLatencyMS: route.minLatencyMS,
		MaxLatencyMS: route.maxLatencyMS,
		LastRequest:  route.lastRequest,
		TraceSamples: route.traceSamples,
	}
	if route.requests > 0 {
		out.AvgLatencyMS = float64(route.totalLatencyMS) / float64(route.requests)
		out.P50LatencyMS = percentile(route.recentLatencies, 0.50)
		out.P95LatencyMS = percentile(route.recentLatencies, 0.95)
	}
	return out
}

func percentile(values []int64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	copied := append([]int64(nil), values...)
	sort.Slice(copied, func(i, j int) bool { return copied[i] < copied[j] })
	if p <= 0 {
		return float64(copied[0])
	}
	if p >= 1 {
		return float64(copied[len(copied)-1])
	}
	index := int(float64(len(copied)-1) * p)
	return float64(copied[index])
}

func fingerprintHash(message, stack string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(message) + "\n" + strings.TrimSpace(stack)))
	return hex.EncodeToString(sum[:6])
}

func traceSampleID(event RequestEvent) string {
	sum := sha1.Sum([]byte(event.RPCName + "|" + event.Timestamp.Format(time.RFC3339Nano) + "|" + event.ErrorMessage + "|" + event.StackSignature))
	return hex.EncodeToString(sum[:8])
}
