package flightlog

import (
	"encoding/json"
	"io"
	"os"
)

// Logger defines the interface for Flight Log recording.
type Logger interface {
	// Record writes an entry to the Flight Log.
	Record(entry Entry) error

	// Close closes the underlying resources.
	Close() error
}

// FlightLog implements Logger with file and optional stdout output.
// Per ADR-002: JSONL file by default, stdout mirroring with verbose option.
type FlightLog struct {
	writer  *Writer
	verbose bool
	stdout  io.Writer
}

// Option configures FlightLog behavior.
type Option func(*FlightLog)

// WithVerbose enables stdout mirroring of log entries.
func WithVerbose(verbose bool) Option {
	return func(fl *FlightLog) {
		fl.verbose = verbose
	}
}

// WithStdout sets a custom stdout writer (useful for testing).
func WithStdout(w io.Writer) Option {
	return func(fl *FlightLog) {
		fl.stdout = w
	}
}

// New creates a new FlightLog writing to the given path.
func New(path string, opts ...Option) (*FlightLog, error) {
	w, err := NewWriter(path)
	if err != nil {
		return nil, err
	}

	fl := &FlightLog{
		writer:  w,
		verbose: false,
		stdout:  os.Stdout,
	}

	for _, opt := range opts {
		opt(fl)
	}

	return fl, nil
}

// Record writes an entry to the Flight Log.
// If verbose is enabled, also writes to stdout.
func (fl *FlightLog) Record(entry Entry) error {
	// Write to file
	if err := fl.writer.Write(entry); err != nil {
		return err
	}

	// Mirror to stdout if verbose
	if fl.verbose && fl.stdout != nil {
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		data = append(data, '\n')
		if _, err := fl.stdout.Write(data); err != nil {
			return err
		}
	}

	return nil
}

// Close closes the Flight Log and flushes any pending writes.
func (fl *FlightLog) Close() error {
	if fl.writer != nil {
		if err := fl.writer.Sync(); err != nil {
			return err
		}
		return fl.writer.Close()
	}
	return nil
}

// Path returns the file path of the Flight Log.
func (fl *FlightLog) Path() string {
	if fl.writer != nil && fl.writer.file != nil {
		return fl.writer.file.Name()
	}
	return ""
}
