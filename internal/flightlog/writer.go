package flightlog

import (
	"encoding/json"
	"os"
	"sync"
)

// Writer handles thread-safe JSONL file writing.
// Per ADR-002, entries are appended one JSON object per line.
type Writer struct {
	mu   sync.Mutex
	file *os.File
}

// NewWriter creates a Writer for the given file path.
// Creates the file if it doesn't exist, opens for append if it does.
func NewWriter(path string) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	return &Writer{file: f}, nil
}

// Write serializes the entry to JSON and appends it to the file.
// Thread-safe via sync.Mutex.
func (w *Writer) Write(entry Entry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	// Append newline for JSONL format
	data = append(data, '\n')

	_, err = w.file.Write(data)
	return err
}

// Close closes the underlying file.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Sync flushes the file to disk.
func (w *Writer) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file != nil {
		return w.file.Sync()
	}
	return nil
}
