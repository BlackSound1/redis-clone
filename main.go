package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
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

func (v *Value) readArray(reader io.Reader) {
	// Read into a buffer. Must read 4 bytes
	// because arrays are define as such: `*#\r\n`
	buffer := make([]byte, 4)
	reader.Read(buffer)

	// Since arrays have a length associated with it,
	// read that length
	arrLen, err := strconv.Atoi(string(buffer[1]))
	if err != nil {
		fmt.Println(err)
		return
	}

	// Once we know how many bulk strings are in the message,
	// read those and add them to the array
	for range arrLen {
		bulk := v.readBulk(reader)
		v.array = append(v.array, bulk)
	}
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

func main() {
	// Create a TCP listener on port 6379, the default Redis port
	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal("Cannot listen on part 6379. Quitting.")
	}
	defer l.Close()
	log.Println("Listening on port 6379")

	// Block until connection is made
	conn, err := l.Accept() // TODO: Add ability to accept multiple connections
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer conn.Close()
	log.Println("Connection accepted!")

	for {
		v := Value{typ: ARRAY}
		v.readArray(conn)

		handle(conn, &v)

		fmt.Println(v.array)
	}
}

// Create a handler function type
type Handler func(*Value) *Value

var Handlers = map[string]Handler{
	"COMMAND": command,
	"GET":     get,
	"SET":     set,
}

var DB = map[string]string{}

func handle(conn net.Conn, v *Value) {
	// Get the bulk string of the first message
	cmd := v.array[0].bulk

	// Get the handler
	handler, ok := Handlers[cmd]
	if !ok {
		fmt.Println("Invalid command: ", cmd)
		return
	}

	// Call the handler with the value
	reply := handler(v)
	w := NewWriter(conn)
	w.Write(reply)
}

func command(v *Value) *Value {
	return &Value{typ: STRING, str: "OK"}
}

// get handles the case of GET Redis messages
func get(v *Value) *Value {
	// GET can only take 1 argument
	args := v.array[1:]
	if len(args) != 1 {
		return &Value{typ: ERROR, err: "ERR invalid number of arguments for the 'GET' command"}
	}

	// Get the bulk string from the DB, making sure to lock and unlock the
	// critical section
	name := args[0].bulk
	DB.mu.RLock() // Only locked for reading
	val, ok := DB.store[name]
	DB.mu.RUnlock()
	if !ok {
		return &Value{typ: NULL}
	}

	// Create and return a new bulk string object based on the value
	return &Value{typ: BULK, bulk: val}
}

// set handles the case of SET Redis messages
func set(v *Value) *Value {
	// SET must take 2 arguments
	args := v.array[1:]
	if len(args) != 2 {
		return &Value{typ: ERROR, err: "ERR invalid number of arguments for the 'SET' command"}
	}

	// Get the key and value and set the local "DB" with those in mind
	key := args[0].bulk
	val := args[1].bulk
	DB.mu.Lock()
	DB.store[key] = val
	DB.mu.Unlock()

	return &Value{typ: STRING, str: "OK"}
}

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
