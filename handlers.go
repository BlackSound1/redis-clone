package main

import (
	"log"
	"maps"
	"path/filepath"
)

// Create a handler function type
type Handler func(*Client, *Value, *AppState) *Value

var Handlers = map[string]Handler{
	"COMMAND": command,
	"GET":     get,
	"SET":     set,
	"DEL":     del,
	"EXISTS":  exists,
	"KEYS":    keys,
	"SAVE":    save,
	"BGSAVE":  bgsave,
	"FLUSHDB": flushdb,
	"DBSIZE":  dbsize,
	"AUTH":    auth,
}

// These commands don't need auth
var SafeCommands = []string{
	"COMMAND",
	"AUTH",
}

// handle takes a Client and a Value type and calls the handler
// associated with the bulk string of the first message in the Value.
// It then writes the reply from the handler back to the connection.
func handle(client *Client, v *Value, state *AppState) {
	// Get the bulk string of the first message
	cmd := v.array[0].bulk

	w := NewWriter(client.conn)

	// Get the handler
	handler, ok := Handlers[cmd]
	if !ok {
		w.Write(&Value{typ: ERROR, err: "ERR Invalid command"})
		w.Flush()
		return
	}

	// If auth is needed and we're not logged-in and the command isn't safe, NOAUTH error
	if state.conf.requirepass && !client.authenticated && !contains(SafeCommands, cmd) {
		w.Write(&Value{typ: ERROR, err: "NOAUTH Authentication required"})
		w.Flush()
		return
	}

	// Call the handler with the value
	reply := handler(client, v, state)
	w.Write(reply)
	w.Flush() // For network connections, always flush after writing
}

// command is a stub function that just returns a basic OK string message
func command(client *Client, v *Value, state *AppState) *Value {
	return &Value{typ: STRING, str: "OK"}
}

// get handles the case of GET Redis messages
func get(client *Client, v *Value, state *AppState) *Value {
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
func set(client *Client, v *Value, state *AppState) *Value {
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
func del(client *Client, v *Value, state *AppState) *Value {
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
func exists(client *Client, v *Value, state *AppState) *Value {
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
func keys(client *Client, v *Value, state *AppState) *Value {
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

// save handles the case of SAVE Redis messages
//
// save is considered to be blocking because it uses
// the `SaveRDB` function, which has a `RLock` on its critical section
func save(client *Client, v *Value, state *AppState) *Value {
	SaveRDB(state)
	return &Value{typ: STRING, str: "OK"}
}

// bgsave handles the case of BGSAVE Redis messages
func bgsave(client *Client, v *Value, state *AppState) *Value {
	if state.bgSaveRunning {
		return &Value{typ: ERROR, err: "ERR Background saving already happening"}
	}

	// Make a local copy of the DB
	copy := make(map[string]string, len(DB.store))
	DB.mu.RLock()
	maps.Copy(copy, DB.store)
	DB.mu.RUnlock()

	state.bgSaveRunning = true
	state.dbCopy = copy

	// Save to DB in another thread. Whenever the goroutine finishes, reset the BGSAVE state variables
	go func() {
		defer func() {
			state.bgSaveRunning = false
			state.dbCopy = nil
		}()

		SaveRDB(state)
	}()

	return &Value{typ: STRING, str: "OK"}
}

// flushdb handles the case of FLUSHDB Redis messages
func flushdb(client *Client, v *Value, state *AppState) *Value {
	// Instead of linearly going through each key and deleting it,
	// just set the DB to a new, empty map
	DB.mu.Lock()
	DB.store = map[string]string{}
	DB.mu.Unlock()
	return &Value{typ: STRING, str: "OK"}
}

// dbsize handles the case of DBSIZE Redis messages
func dbsize(client *Client, v *Value, state *AppState) *Value {
	DB.mu.RLock()
	size := len(DB.store)
	DB.mu.RUnlock()

	return &Value{typ: INTEGER, num: size}
}

// auth handles the case of AUTH Redis messages
func auth(client *Client, v *Value, state *AppState) *Value {
	args := v.array[1:]
	if len(args) != 1 {
		return &Value{typ: ERROR, err: "ERR Invalid number of arguments for 'AUTH' command"}
	}

	password := args[0].bulk
	if state.conf.password == password {
		client.authenticated = true
		return &Value{typ: STRING, str: "OK"}
	} else {
		client.authenticated = false
		return &Value{typ: ERROR, err: "ERR Invalid password"}
	}
}
