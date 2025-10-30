package main

import (
	"fmt"
	"log"
	"net"
	"path/filepath"
)

// Create a handler function type
type Handler func(*Value, *AppState) *Value

var Handlers = map[string]Handler{
	"COMMAND": command,
	"GET":     get,
	"SET":     set,
	"DEL":     del,
	"EXISTS":  exists,
	"KEYS":    keys,
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

	// Get the key and value and set the DB with those in mind
	key := args[0].bulk
	val := args[1].bulk
	DB.mu.Lock()
	DB.store[key] = val

	// If AOF is enabled, write to its buffer
	if state.conf.aofEnabled {
		log.Println("Saving AOF record")
		state.aof.w.Write(v)

		if state.conf.aofFsync == Always {
			state.aof.w.Flush()
		}
	}

	// If there are RDB snapshots, increment the keys
	if len(state.conf.rdb) >= 0 {
		IncrementRDBTrackers()
	}

	DB.mu.Unlock()

	return &Value{typ: STRING, str: "OK"}
}

// del handles the case of DEL Redis messages
func del(v *Value, state *AppState) *Value {
	args := v.array[1:]

	var numDeleted int

	// Lock for reading/ writing because deleting is somewhat like a write
	DB.mu.Lock()
	// Go through all keys to delete (may be multiple)
	for _, arg := range args {
		_, ok := DB.store[arg.bulk]
		delete(DB.store, arg.bulk)
		if ok {
			numDeleted++
		}
	}
	DB.mu.Unlock()

	return &Value{typ: INTEGER, num: numDeleted}
}

// exists handles the case of EXISTS Redis messages
func exists(v *Value, state *AppState) *Value {
	args := v.array[1:]

	var numExists int

	// Only lock for reading
	DB.mu.RLock()
	// Go through all the space-separated keys, and if they
	// exist in the DB, increment counter
	for _, arg := range args {
		_, ok := DB.store[arg.bulk]
		if ok {
			numExists++
		}
	}

	DB.mu.RUnlock()

	return &Value{typ: INTEGER, num: numExists}
}

// keys handles the case of KEYS Redis messages
//
// In prod, may be better to use SCAN
func keys(v *Value, state *AppState) *Value {
	args := v.array[1:]

	// KEYS can only take 1 argument
	if len(args) > 1 {
		return &Value{typ: ERROR, err: "ERR Invalid number of arguments for 'KEYS' command"}
	}

	pattern := args[0].bulk

	DB.mu.RLock()

	var matches []string
	// Loop over all keys
	for key := range DB.store {
		matched, err := filepath.Match(pattern, key) // Can use this to offload some of the pattern-matching difficulty
		if err != nil {
			log.Printf("Error matching keys: (pattern: %s), (key: %s) - %v", pattern, key, err)
			continue
		}

		// If we matched, add to the matches
		if matched {
			matches = append(matches, key)
		}
	}

	DB.mu.RUnlock()

	reply := Value{typ: ARRAY}

	// For each match, add to the reply's array a new bulk string
	for _, m := range matches {
		reply.array = append(reply.array, Value{typ: BULK, bulk: m})
	}

	return &reply
}
