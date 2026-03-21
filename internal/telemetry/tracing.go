package telemetry

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

type SpanRecord struct {
	Name      string            `json:"name"`
	Timestamp time.Time         `json:"timestamp"`
	Fields    map[string]string `json:"fields"`
}

type Tracer struct {
	mu    sync.Mutex
	spans map[string][]SpanRecord
}

func NewTracer() *Tracer {
	return &Tracer{spans: make(map[string][]SpanRecord)}
}

func (t *Tracer) NewTrace(rootSpan string) string {
	traceID := randomHex(16)
	t.Add(traceID, rootSpan, map[string]string{"event": "trace_created"})
	return traceID
}

func (t *Tracer) Add(traceID, name string, fields map[string]string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans[traceID] = append(t.spans[traceID], SpanRecord{
		Name:      name,
		Timestamp: time.Now().UTC(),
		Fields:    fields,
	})
}

func (t *Tracer) Snapshot(traceID string) []SpanRecord {
	t.mu.Lock()
	defer t.mu.Unlock()
	records := t.spans[traceID]
	out := make([]SpanRecord, len(records))
	copy(out, records)
	return out
}

func randomHex(size int) string {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		now := time.Now().UTC().Format("20060102150405.000000000")
		return hex.EncodeToString([]byte(now))
	}
	return hex.EncodeToString(buf)
}
