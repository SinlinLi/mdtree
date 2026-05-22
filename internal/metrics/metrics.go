// Package metrics collects lightweight in-process runtime metrics for mdtree,
// exposed through the /api/stats endpoint for observability.
package metrics

import (
	"net/http"
	"sync"
	"time"
)

// Metrics holds counters and timers for observability. All methods are safe
// for concurrent use.
type Metrics struct {
	mu                sync.Mutex
	startTime         time.Time
	requests          int64
	requestErrors     int64
	totalLatency      time.Duration
	fileReads         int64
	fileWrites        int64
	searches          int64
	indexedFiles      int
	lastIndexAt       time.Time
	lastIndexDuration time.Duration
}

// New returns a Metrics with the start time set to now.
func New() *Metrics {
	return &Metrics{startTime: time.Now()}
}

// Snapshot is an immutable, serializable view of the metrics.
type Snapshot struct {
	UptimeSeconds    float64 `json:"uptimeSeconds"`
	Requests         int64   `json:"requests"`
	RequestErrors    int64   `json:"requestErrors"`
	AvgLatencyMs     float64 `json:"avgLatencyMs"`
	FileReads        int64   `json:"fileReads"`
	FileWrites       int64   `json:"fileWrites"`
	Searches         int64   `json:"searches"`
	IndexedFiles     int     `json:"indexedFiles"`
	LastIndexAt      string  `json:"lastIndexAt,omitempty"`
	LastIndexBuildMs float64 `json:"lastIndexBuildMs"`
}

// ObserveRequest records a completed HTTP request.
func (m *Metrics) ObserveRequest(latency time.Duration, isError bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requests++
	m.totalLatency += latency
	if isError {
		m.requestErrors++
	}
}

// IncFileRead records a markdown file read.
func (m *Metrics) IncFileRead() {
	m.mu.Lock()
	m.fileReads++
	m.mu.Unlock()
}

// IncFileWrite records a markdown file write (save, create or rename).
func (m *Metrics) IncFileWrite() {
	m.mu.Lock()
	m.fileWrites++
	m.mu.Unlock()
}

// IncSearch records a filename search.
func (m *Metrics) IncSearch() {
	m.mu.Lock()
	m.searches++
	m.mu.Unlock()
}

// SetIndex records the result of a search index build.
func (m *Metrics) SetIndex(count int, dur time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.indexedFiles = count
	m.lastIndexAt = time.Now()
	m.lastIndexDuration = dur
}

// Snapshot returns a point-in-time copy of the metrics.
func (m *Metrics) Snapshot() Snapshot {
	m.mu.Lock()
	defer m.mu.Unlock()
	avg := 0.0
	if m.requests > 0 {
		avg = float64(m.totalLatency.Microseconds()) / float64(m.requests) / 1000.0
	}
	s := Snapshot{
		UptimeSeconds:    time.Since(m.startTime).Seconds(),
		Requests:         m.requests,
		RequestErrors:    m.requestErrors,
		AvgLatencyMs:     avg,
		FileReads:        m.fileReads,
		FileWrites:       m.fileWrites,
		Searches:         m.searches,
		IndexedFiles:     m.indexedFiles,
		LastIndexBuildMs: float64(m.lastIndexDuration.Microseconds()) / 1000.0,
	}
	if !m.lastIndexAt.IsZero() {
		s.LastIndexAt = m.lastIndexAt.Format(time.RFC3339)
	}
	return s
}

// Middleware records request count, latency and server-error rate.
func (m *Metrics) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)
		m.ObserveRequest(time.Since(start), rec.status >= 500)
	})
}

// statusRecorder captures the response status code.
type statusRecorder struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (r *statusRecorder) WriteHeader(code int) {
	if !r.wroteHeader {
		r.status = code
		r.wroteHeader = true
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	r.wroteHeader = true
	return r.ResponseWriter.Write(b)
}
