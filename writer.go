package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
)

type Writer struct {
	writer io.Writer
}

// NewWriter creates a new Writer from a given io.Writer.
func NewWriter(w io.Writer) *Writer {
	return &Writer{writer: bufio.NewWriter(w)}
}

func (w *Writer) Deserialize(v *Value) (reply string) {
	switch v.typ {
	case ARRAY:
		// Specify length of array
		reply = fmt.Sprintf("*%d\r\n", len(v.array))

		// For each item in the array, deserialize it
		for _, sub := range v.array {
			reply += w.Deserialize(&sub)
		}
	case STRING:
		reply = fmt.Sprintf("%s%s\r\n", v.typ, v.str)
	case INTEGER:
		reply = fmt.Sprintf("%s%d\r\n", v.typ, v.num)
	case BULK:
		reply = fmt.Sprintf("%s%d\r\n%s\r\n", v.typ, len(v.bulk), v.bulk)
	case ERROR:
		reply = fmt.Sprintf("%s%s\r\n", v.typ, v.err)
	case NULL:
		reply = "$-1\r\n" // Send a bulk string with a length of -1
	default:
		log.Println("Invalid typ received")
		return reply
	}

	return reply
}

// Write automates the process of creating RESP messages from `Value` objects
func (w *Writer) Write(v *Value) {
	reply := w.Deserialize(v)

	// Write to the writer
	w.writer.Write([]byte(reply))
}

// Flush the buffer to force writing
func (w *Writer) Flush() {
	w.writer.(*bufio.Writer).Flush()
}
