package main

import (
	"bufio"
	"fmt"
	"io"
)

type Writer struct {
	writer io.Writer
}

// NewWriter creates a new Writer from a given io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: bufio.NewWriter(w)}
}

// Write automates the process of creating RESP messages from `Value` objects
func (w *Writer) Write(v *Value) {
	var reply string
	switch v.typ {
	case STRING:
		reply = fmt.Sprintf("%s%s\r\n", v.typ, v.str)
	case BULK:
		reply = fmt.Sprintf("%s%d\r\n%s\r\n", v.typ, len(v.bulk), v.bulk)
	case ERROR:
		reply = fmt.Sprintf("%s%s\r\n", v.typ, v.err)
	case NULL:
		reply = "$-1\r\n" // Send a bulk string with a length of -1
	}

	// Write to the writer
	w.writer.Write([]byte(reply))

	// Flush buffer to force writing
	w.writer.(*bufio.Writer).Flush()
}
