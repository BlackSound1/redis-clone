package main

import (
	"fmt"
	"log"
	"net"
)

// Create a handler function type
type Handler func(*Value, *AppState) *Value

var Handlers = map[string]Handler{
	"COMMAND": command,
	"GET":     get,
	"SET":     set,
}

// handle takes a net.Conn and a Value type and calls the handler
// associated with the bulk string of the first message in the Value.
// It then writes the reply from the handler back to the connection.
func handle(conn net.Conn, v *Value, state *AppState) {
	// Get the bulk string of the first message
	cmd := v.array[0].bulk

	// Get the handler
	handler, ok := Handlers[cmd]
	if !ok {
		fmt.Println("Invalid command: ", cmd)
		return
	}

	// Call the handler with the value
	reply := handler(v, state)
	w := NewWriter(conn)
	w.Write(reply)
	w.Flush() // For network connections, always flush after writing
}

// command is a stub function that just returns a basic OK string message
func command(v *Value, state *AppState) *Value {
	return &Value{typ: STRING, str: "OK"}
}

// get handles the case of GET Redis messages
func get(v *Value, state *AppState) *Value {
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
func set(v *Value, state *AppState) *Value {
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

	if state.conf.aofEnabled {
		log.Println("Saving AOF record")
		state.aof.w.Write(v)

		if state.conf.aofFsync == Always {
			state.aof.w.Flush()
		}
	}
	DB.mu.Unlock()

	return &Value{typ: STRING, str: "OK"}
}
