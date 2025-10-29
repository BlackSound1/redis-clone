package main

import (
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
)

// Define a RESP message type
type Value struct {
    typ   ValueType
    bulk  string
    str   string
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

    // Block until connection is made
    conn, err := l.Accept()
    if err != nil {
        fmt.Println(err)
        os.Exit(1)
    }
    defer conn.Close()

    for {
        v := Value{typ: ARRAY}
        v.readArray(conn)

        fmt.Println(v.array)

        conn.Write([]byte("+OK\r\n"))
    }
}
