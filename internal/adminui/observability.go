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
	maxTraceSamples        = 200
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
type RouteAggregate struct {
	RPCName             string  `json:"rpcName"`
	Path                string  `json:"path"`
	HTTPMethod          string  `json:"httpMethod"`
	RequestsLastMinute  int     `json:"requestsLastMinute"`
	RequestsLastHour    int     `json:"requestsLastHour"`
	RequestsLast24H     int     `json:"requestsLast24h"`
	AvgLatencyLastHour  float64 `json:"avgLatencyLastHourMs"`
	P50LatencyLastHour  float64 `json:"p50LatencyLastHourMs"`
	P95LatencyLastHour  float64 `json:"p95LatencyLastHourMs"`
	ClientErrorsLast24H int     `json:"clientErrorsLast24h"`
	ServerErrorsLast24H int     `json:"serverErrorsLast24h"`
	TraceSamplesLast24H int     `json:"traceSamplesLast24h"`
}

// ErrorFingerprint groups repeated server-side failures for one RPC.
type ErrorFingerprint struct {
	RPCName         string    `json:"rpcName"`
	ErrorHash       string    `json:"errorHash"`
	ErrorMessage    string    `json:"errorMessage"`
	StackSignature  string    `json:"stackSignature,omitempty"`
	CountLast24H    int       `json:"countLast24h"`
	Sparkline       []int     `json:"sparkline"`
	LastSeen        time.Time `json:"lastSeen"`
	TraceSampleHint bool      `json:"traceSampleHint"`
}

// GuardAggregate summarizes allow/deny activity for one guard on one RPC.
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
type MetricsTotals struct {
	RequestsLastMinute  int `json:"requestsLastMinute"`
	RequestsLastHour    int `json:"requestsLastHour"`
	RequestsLast24H     int `json:"requestsLast24h"`
	ClientErrorsLast24H int `json:"clientErrorsLast24h"`
	ServerErrorsLast24H int `json:"serverErrorsLast24h"`
}

// MetricsSnapshot is the JSON payload for the observability dashboard.
type MetricsSnapshot struct {
	GeneratedAt   time.Time          `json:"generatedAt"`
	Advanced      bool               `json:"advanced"`
	SampleRate    float64            `json:"sampleRate"`
	Totals        MetricsTotals      `json:"totals"`
	Routes        []RouteAggregate   `json:"routes"`
	Errors        []ErrorFingerprint `json:"errors"`
	Guards        []GuardAggregate   `json:"guards"`
	RecentTraces  []TraceSample      `json:"recentTraces"`
	TraceViewerUI bool               `json:"traceViewerUi"`
}

type observabilityRoute struct {
	path       string
	httpMethod string
	requests   []RequestEvent
	guards     []GuardDecisionEvent
	traces     []TraceSample
}

// ObservabilityTracker keeps recent in-memory request history and aggregates.
type ObservabilityTracker struct {
	mu         sync.RWMutex
	advanced   bool
	sampleRate float64
	routes     map[string]*observabilityRoute
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
		routes:     make(map[string]*observabilityRoute),
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

// RecordRequest stores one request and any guard outcomes.
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
	route.requests = append(route.requests, event)

	if t.advanced {
		for _, decision := range guards {
			decision.Timestamp = decision.Timestamp.UTC()
			if decision.Timestamp.IsZero() {
				decision.Timestamp = now
			}
			decision.RPCName = event.RPCName
			decision.GuardName = strings.TrimSpace(decision.GuardName)
			if decision.GuardName == "" {
				continue
			}
			route.guards = append(route.guards, decision)
		}
		if t.shouldSampleTraceLocked(event) {
			route.traces = append(route.traces, TraceSample{
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

	t.trimRouteLocked(route, now)
}

// Snapshot computes the current dashboard view from in-memory events.
func (t *ObservabilityTracker) Snapshot() MetricsSnapshot {
	if t == nil {
		return MetricsSnapshot{
			GeneratedAt: time.Now().UTC(),
		}
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now().UTC()
	snapshot := MetricsSnapshot{
		GeneratedAt:   now,
		Advanced:      t.advanced,
		SampleRate:    t.sampleRate,
		Routes:        []RouteAggregate{},
		Errors:        []ErrorFingerprint{},
		Guards:        []GuardAggregate{},
		RecentTraces:  []TraceSample{},
		TraceViewerUI: false,
	}

	errorMap := map[string]*ErrorFingerprint{}
	guardMap := map[string]*GuardAggregate{}

	for rpcName, route := range t.routes {
		t.trimRouteLocked(route, now)
		aggregate := summarizeRoute(rpcName, route, now)
		snapshot.Routes = append(snapshot.Routes, aggregate)

		snapshot.Totals.RequestsLastMinute += aggregate.RequestsLastMinute
		snapshot.Totals.RequestsLastHour += aggregate.RequestsLastHour
		snapshot.Totals.RequestsLast24H += aggregate.RequestsLast24H
		snapshot.Totals.ClientErrorsLast24H += aggregate.ClientErrorsLast24H
		snapshot.Totals.ServerErrorsLast24H += aggregate.ServerErrorsLast24H

		if t.advanced {
			accumulateErrors(errorMap, rpcName, route, now)
			accumulateGuards(guardMap, rpcName, route, now)
			snapshot.RecentTraces = append(snapshot.RecentTraces, route.traces...)
		}
	}

	sort.Slice(snapshot.Routes, func(i, j int) bool {
		if snapshot.Routes[i].ServerErrorsLast24H != snapshot.Routes[j].ServerErrorsLast24H {
			return snapshot.Routes[i].ServerErrorsLast24H > snapshot.Routes[j].ServerErrorsLast24H
		}
		if snapshot.Routes[i].RequestsLast24H != snapshot.Routes[j].RequestsLast24H {
			return snapshot.Routes[i].RequestsLast24H > snapshot.Routes[j].RequestsLast24H
		}
		return snapshot.Routes[i].RPCName < snapshot.Routes[j].RPCName
	})

	if t.advanced {
		for _, item := range errorMap {
			snapshot.Errors = append(snapshot.Errors, *item)
		}
		sort.Slice(snapshot.Errors, func(i, j int) bool {
			if snapshot.Errors[i].CountLast24H != snapshot.Errors[j].CountLast24H {
				return snapshot.Errors[i].CountLast24H > snapshot.Errors[j].CountLast24H
			}
			return snapshot.Errors[i].RPCName < snapshot.Errors[j].RPCName
		})

		for _, item := range guardMap {
			total := item.AllowedCount + item.DeniedCount
			if total > 0 {
				item.DenialRatePercent = (float64(item.DeniedCount) / float64(total)) * 100
			}
			snapshot.Guards = append(snapshot.Guards, *item)
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

		sort.Slice(snapshot.RecentTraces, func(i, j int) bool {
			return snapshot.RecentTraces[i].Timestamp.After(snapshot.RecentTraces[j].Timestamp)
		})
		if len(snapshot.RecentTraces) > maxTraceSamples {
			snapshot.RecentTraces = snapshot.RecentTraces[:maxTraceSamples]
		}
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

func (t *ObservabilityTracker) trimRouteLocked(route *observabilityRoute, now time.Time) {
	if route == nil {
		return
	}
	cutoff := now.Add(-24 * time.Hour)
	route.requests = trimRequests(route.requests, cutoff)
	route.guards = trimGuardEvents(route.guards, cutoff)
	route.traces = trimTraceSamples(route.traces, cutoff)
	if len(route.traces) > maxTraceSamples {
		route.traces = route.traces[len(route.traces)-maxTraceSamples:]
	}
}

func trimRequests(items []RequestEvent, cutoff time.Time) []RequestEvent {
	if len(items) == 0 {
		return items
	}
	idx := 0
	for idx < len(items) && items[idx].Timestamp.Before(cutoff) {
		idx++
	}
	if idx == 0 {
		return items
	}
	return append([]RequestEvent(nil), items[idx:]...)
}

func trimGuardEvents(items []GuardDecisionEvent, cutoff time.Time) []GuardDecisionEvent {
	if len(items) == 0 {
		return items
	}
	idx := 0
	for idx < len(items) && items[idx].Timestamp.Before(cutoff) {
		idx++
	}
	if idx == 0 {
		return items
	}
	return append([]GuardDecisionEvent(nil), items[idx:]...)
}

func trimTraceSamples(items []TraceSample, cutoff time.Time) []TraceSample {
	if len(items) == 0 {
		return items
	}
	idx := 0
	for idx < len(items) && items[idx].Timestamp.Before(cutoff) {
		idx++
	}
	if idx == 0 {
		return items
	}
	return append([]TraceSample(nil), items[idx:]...)
}

func summarizeRoute(rpcName string, route *observabilityRoute, now time.Time) RouteAggregate {
	cutoffMinute := now.Add(-1 * time.Minute)
	cutoffHour := now.Add(-1 * time.Hour)
	cutoffDay := now.Add(-24 * time.Hour)
	durations := make([]int64, 0, len(route.requests))

	out := RouteAggregate{
		RPCName:    rpcName,
		Path:       route.path,
		HTTPMethod: route.httpMethod,
	}

	var totalDurationHour int64
	for _, event := range route.requests {
		if event.Timestamp.Before(cutoffDay) {
			continue
		}
		out.RequestsLast24H++
		switch {
		case event.StatusCode >= 500:
			out.ServerErrorsLast24H++
		case event.StatusCode >= 400:
			out.ClientErrorsLast24H++
		}
		if !event.Timestamp.Before(cutoffMinute) {
			out.RequestsLastMinute++
		}
		if !event.Timestamp.Before(cutoffHour) {
			out.RequestsLastHour++
			totalDurationHour += event.DurationMS
			durations = append(durations, event.DurationMS)
		}
	}
	if out.RequestsLastHour > 0 {
		out.AvgLatencyLastHour = float64(totalDurationHour) / float64(out.RequestsLastHour)
		out.P50LatencyLastHour = percentile(durations, 0.50)
		out.P95LatencyLastHour = percentile(durations, 0.95)
	}
	out.TraceSamplesLast24H = len(route.traces)
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

func accumulateErrors(out map[string]*ErrorFingerprint, rpcName string, route *observabilityRoute, now time.Time) {
	cutoff := now.Add(-24 * time.Hour)
	for _, event := range route.requests {
		if event.Timestamp.Before(cutoff) {
			continue
		}
		if event.StatusCode < 500 {
			continue
		}
		if event.ErrorMessage == "" && event.StackSignature == "" {
			continue
		}
		hash := fingerprintHash(event.ErrorMessage, event.StackSignature)
		item := out[hash]
		if item == nil {
			item = &ErrorFingerprint{
				RPCName:        rpcName,
				ErrorHash:      hash,
				ErrorMessage:   event.ErrorMessage,
				StackSignature: event.StackSignature,
				Sparkline:      make([]int, 24),
			}
			out[hash] = item
		}
		item.CountLast24H++
		if event.Timestamp.After(item.LastSeen) {
			item.LastSeen = event.Timestamp
		}
		item.TraceSampleHint = item.TraceSampleHint || event.StackSignature != ""
		if bucket := sparklineBucket(now, event.Timestamp); bucket >= 0 && bucket < len(item.Sparkline) {
			item.Sparkline[bucket]++
		}
	}
}

func accumulateGuards(out map[string]*GuardAggregate, rpcName string, route *observabilityRoute, now time.Time) {
	cutoff := now.Add(-24 * time.Hour)
	for _, event := range route.guards {
		if event.Timestamp.Before(cutoff) {
			continue
		}
		key := rpcName + "\x00" + event.GuardName
		item := out[key]
		if item == nil {
			item = &GuardAggregate{
				RPCName:   rpcName,
				GuardName: event.GuardName,
			}
			out[key] = item
		}
		if event.Allowed {
			item.AllowedCount++
		} else {
			item.DeniedCount++
		}
	}
}

func sparklineBucket(now, ts time.Time) int {
	age := now.Sub(ts)
	if age < 0 || age > 24*time.Hour {
		return -1
	}
	hoursAgo := int(age / time.Hour)
	return 23 - hoursAgo
}

func fingerprintHash(message, stack string) string {
	sum := sha1.Sum([]byte(strings.TrimSpace(message) + "\n" + strings.TrimSpace(stack)))
	return hex.EncodeToString(sum[:6])
}

func traceSampleID(event RequestEvent) string {
	sum := sha1.Sum([]byte(event.RPCName + "|" + event.Timestamp.Format(time.RFC3339Nano) + "|" + event.ErrorMessage + "|" + event.StackSignature))
	return hex.EncodeToString(sum[:8])
}
