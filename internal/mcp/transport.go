package mcp

import (
	"bufio"
	"encoding/json"
	"io"
)

// Transport handles NDJSON (newline-delimited JSON) communication over stdio.
// It reads JSON-RPC 2.0 messages line by line and writes responses with newline suffixes.
type Transport struct {
	reader *bufio.Scanner
	writer io.Writer
}

// NewTransport creates a new Transport with the given reader and writer.
// The reader is wrapped in a bufio.Scanner for line-by-line reading.
func NewTransport(r io.Reader, w io.Writer) *Transport {
	return &Transport{
		reader: bufio.NewScanner(r),
		writer: w,
	}
}

// Read reads a single NDJSON message from the input stream and unmarshals it into v.
// Returns io.EOF when there are no more messages to read.
// Returns an error if the JSON is malformed.
func (t *Transport) Read(v any) error {
	if !t.reader.Scan() {
		if err := t.reader.Err(); err != nil {
			return err
		}
		return io.EOF
	}

	line := t.reader.Bytes()
	if len(line) == 0 {
		return io.EOF
	}

	return json.Unmarshal(line, v)
}

// Write marshals v to JSON and writes it to the output stream followed by a newline.
// Returns an error if marshaling fails or writing fails.
func (t *Transport) Write(v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}

	data = append(data, '\n')
	_, err = t.writer.Write(data)
	return err
}
