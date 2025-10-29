package main

import (
	"fmt"
	"io"
	"strconv"
)

// Define the symbols that define a RESP message
type ValueType string

const (
	ARRAY  ValueType = "*"
	BULK   ValueType = "$"
	STRING ValueType = "+"
	ERROR  ValueType = "-"
	NULL   ValueType = ""
)

// Define a RESP message type
type Value struct {
	typ   ValueType
	bulk  string
	str   string
	err   string
	array []Value
}

func (v *Value) readArray(reader io.Reader) error {
	// Read into a buffer. Must read 4 bytes
	// because arrays are define as such: `*#\r\n`
	buffer := make([]byte, 4)
	_, err := reader.Read(buffer)
	if err != nil {
		return err
	}

	// Since arrays have a length associated with it,
	// read that length
	arrLen, err := strconv.Atoi(string(buffer[1]))
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Once we know how many bulk strings are in the message,
	// read those and add them to the array
	for range arrLen {
		bulk := v.readBulk(reader)
		v.array = append(v.array, bulk)
	}

	return nil
}

func (v *Value) readBulk(reader io.Reader) Value {
	// Read the bulk buffer. Also must have 4 bytes
	buffer := make([]byte, 4)
	reader.Read(buffer)

	// Get size of string in BULK buffer
	n, err := strconv.Atoi(string(buffer[1]))
	if err != nil {
		fmt.Println(err)
		return Value{}
	}

	// Create buffer for the bulk string, including \r\n
	bulkBuffer := make([]byte, n+2)
	reader.Read(bulkBuffer)

	// The actual bulk string doesn't include \r\n
	bulk := string(bulkBuffer[:n])

	return Value{typ: BULK, bulk: bulk}
}
