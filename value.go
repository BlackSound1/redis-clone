package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

// Define the symbols that define a RESP message
type ValueType string

const (
	ARRAY   ValueType = "*"
	BULK    ValueType = "$"
	STRING  ValueType = "+"
	ERROR   ValueType = "-"
	NULL    ValueType = ""
	INTEGER ValueType = ":"
)

// Define a RESP message type
type Value struct {
	typ   ValueType
	bulk  string
	str   string
	err   string
	num   int
	array []Value
}

// readLine reads a line from the user and trims the \r\n characters
func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSuffix(line, "\r\n"), nil
}

// readArray reads an array RESP message from the given io.Reader into the Value type.
// It first reads the length of the array, then reads that many bulk strings from the reader.
// The bulk strings are added to the Value's array field
func (v *Value) readArray(reader *bufio.Reader) error {

	// Get the line from the user
	line, err := readLine(reader)
	if err != nil {
		return err
	}

	// If the line doesn't begin with *, then it's not an ARRAY type
	if line[0] != '*' {
		return errors.New("expected array")
	}

	// Since arrays have a length associated with it, read that length
	arrLen, err := strconv.Atoi(line[1:])
	if err != nil {
		fmt.Println(err)
		return err
	}

	// Once we know how many bulk strings are in the message,
	// read those and add them to the array
	for range arrLen {
		bulk, err := v.readBulk(reader)
		if err != nil {
			log.Println(err)
			break
		}
		v.array = append(v.array, bulk)
	}

	return nil
}

// readBulk reads a bulk RESP message from the given io.Reader into the Value type.
// It first reads the size of the bulk string, then reads that many bytes from the reader.
// The bulk string is returned as a Value with the BULK type
func (v *Value) readBulk(reader *bufio.Reader) (Value, error) {
	line, err := readLine(reader)
	if err != nil {
		return Value{}, err
	}

	// Get size of string in BULK buffer
	n, err := strconv.Atoi(line[1:])
	if err != nil {
		return Value{}, err
	}

	// Create buffer for the bulk string, including \r\n
	bulkBuffer := make([]byte, n+2)

	// Read all of the reader into the bulkBuffer
	if _, err := io.ReadFull(reader, bulkBuffer); err != nil {
		return Value{}, err
	}

	// The actual bulk string doesn't include \r\n
	bulk := string(bulkBuffer[:n])

	return Value{typ: BULK, bulk: bulk}, nil
}
