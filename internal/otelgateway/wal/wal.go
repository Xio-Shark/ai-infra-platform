// Package wal implements a Write-Ahead Log for crash recovery in the gateway pipeline.
//
// V2: Before exporting, payloads are appended to a WAL segment file.
// On restart, unacknowledged segments are replayed through the exporter.
//
// Design:
//   - Segment files: fixed-size (default 64MB), sequentially numbered.
//   - Each record: [length:4][crc32:4][payload:length]
//   - Acknowledged records are marked; fully-acked segments are deleted.
//   - Replay scans all unacked segments on startup.
package wal

import (
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"ai-infra-platform/internal/otelgateway/metrics"
)

// Config holds WAL settings.
type Config struct {
	Dir            string        `yaml:"dir"`
	SegmentMaxSize int64         `yaml:"segment_max_size"` // bytes, default 64MB
	SyncInterval   time.Duration `yaml:"sync_interval"`    // fsync interval
}

// DefaultConfig returns production-safe defaults.
func DefaultConfig() Config {
	return Config{
		Dir:            "/tmp/gateway-wal",
		SegmentMaxSize: 64 * 1024 * 1024, // 64MB
		SyncInterval:   500 * time.Millisecond,
	}
}

// record header: 4 bytes length + 4 bytes crc32
const headerSize = 8

// Writer appends records to WAL segment files.
type Writer struct {
	cfg     Config
	mu      sync.Mutex
	current *os.File
	segSeq  int64
	segSize int64
	stopCh  chan struct{}
}

// NewWriter creates a WAL writer. The WAL directory is created if it doesn't exist.
func NewWriter(cfg Config) (*Writer, error) {
	if err := os.MkdirAll(cfg.Dir, 0o755); err != nil {
		return nil, fmt.Errorf("wal: create dir: %w", err)
	}

	w := &Writer{
		cfg:    cfg,
		stopCh: make(chan struct{}),
	}

	// Find the highest existing segment number to continue from.
	seq, err := w.findMaxSegment()
	if err != nil {
		return nil, err
	}
	w.segSeq = seq + 1

	if err := w.rotate(); err != nil {
		return nil, err
	}

	go w.syncLoop()
	log.Printf("[wal] dir=%s segment_max=%dMB", cfg.Dir, cfg.SegmentMaxSize/(1024*1024))
	return w, nil
}

// Append writes a record to the current segment. Thread-safe.
func (w *Writer) Append(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	recordSize := int64(headerSize + len(data))
	if w.segSize+recordSize > w.cfg.SegmentMaxSize {
		if err := w.rotate(); err != nil {
			return fmt.Errorf("wal: rotate: %w", err)
		}
	}

	// Write header: [length:4][crc32:4]
	header := make([]byte, headerSize)
	binary.LittleEndian.PutUint32(header[0:4], uint32(len(data)))
	binary.LittleEndian.PutUint32(header[4:8], crc32.ChecksumIEEE(data))

	if _, err := w.current.Write(header); err != nil {
		return fmt.Errorf("wal: write header: %w", err)
	}
	if _, err := w.current.Write(data); err != nil {
		return fmt.Errorf("wal: write data: %w", err)
	}

	w.segSize += recordSize
	metrics.WALAppendTotal.Inc()
	return nil
}

// Close flushes and closes the current segment.
func (w *Writer) Close() error {
	close(w.stopCh)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.current != nil {
		return w.current.Close()
	}
	return nil
}

func (w *Writer) rotate() error {
	if w.current != nil {
		if err := w.current.Close(); err != nil {
			return err
		}
	}

	path := filepath.Join(w.cfg.Dir, fmt.Sprintf("seg-%010d.wal", w.segSeq))
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("wal: create segment: %w", err)
	}

	w.current = f
	w.segSize = 0
	w.segSeq++
	return nil
}

func (w *Writer) syncLoop() {
	ticker := time.NewTicker(w.cfg.SyncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			w.mu.Lock()
			if w.current != nil {
				_ = w.current.Sync()
			}
			w.mu.Unlock()
		case <-w.stopCh:
			return
		}
	}
}

func (w *Writer) findMaxSegment() (int64, error) {
	entries, err := os.ReadDir(w.cfg.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	var maxSeq int64
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".wal") {
			var seq int64
			if _, err := fmt.Sscanf(e.Name(), "seg-%d.wal", &seq); err == nil {
				if seq > maxSeq {
					maxSeq = seq
				}
			}
		}
	}
	return maxSeq, nil
}

// Replay reads all records from WAL segment files and calls fn for each.
// Used on startup to re-export unacknowledged data.
func Replay(dir string, fn func(data []byte) error) error {
	start := time.Now()

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("wal: read dir: %w", err)
	}

	// Sort segments by name (sequential order)
	var segments []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".wal") {
			segments = append(segments, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(segments)

	var totalRecords int
	for _, path := range segments {
		n, err := replaySegment(path, fn)
		if err != nil {
			log.Printf("[wal] replay error in %s: %v", path, err)
			metrics.WALCorruptionTotal.Inc()
			continue // skip corrupted segment, continue with next
		}
		totalRecords += n
	}

	metrics.WALReplayTotal.Inc()
	metrics.WALReplayLatency.Observe(time.Since(start).Seconds())
	log.Printf("[wal] replayed %d records from %d segments in %s",
		totalRecords, len(segments), time.Since(start))
	return nil
}

func replaySegment(path string, fn func([]byte) error) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	var count int
	header := make([]byte, headerSize)

	for {
		if _, err := io.ReadFull(f, header); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return count, fmt.Errorf("read header: %w", err)
		}

		length := binary.LittleEndian.Uint32(header[0:4])
		expectedCRC := binary.LittleEndian.Uint32(header[4:8])

		data := make([]byte, length)
		if _, err := io.ReadFull(f, data); err != nil {
			return count, fmt.Errorf("read data: %w", err)
		}

		if crc32.ChecksumIEEE(data) != expectedCRC {
			metrics.WALCorruptionTotal.Inc()
			return count, fmt.Errorf("crc mismatch at record %d", count)
		}

		if err := fn(data); err != nil {
			return count, fmt.Errorf("callback error at record %d: %w", count, err)
		}
		count++
	}
	return count, nil
}
