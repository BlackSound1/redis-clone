package main

import (
	"errors"
	"log"
	"sync"
	"time"
)

// A Database type containing a key, value store and
// a mutex lock to allow concurrency
type Database struct {
	store map[string]*Key
	mu    sync.RWMutex
	mem   int64
}

// NewDatabase creates a new Database type
func NewDatabase() *Database {
	return &Database{
		store: map[string]*Key{},
		mu:    sync.RWMutex{},
	}
}

// evictKeys evicts keys from the DB according to eviction policies
func (db *Database) evictKeys(state *AppState, requiredMem int64) error {
	if state.conf.eviction == NoEviction {
		return errors.New("maximum memory reached")
	}
	return nil
}

// Set is a "public" method to set values to DB keys, which we prefer to act "private"
func (db *Database) Set(k string, v string, state *AppState) error {
	// If key already exists, subtract existing memory amount before adding new amount
	if old, ok := db.store[k]; ok {
		oldMemory := old.approxMemUsage(k)
		db.mem -= oldMemory
	}

	key := &Key{V: v}
	keyMem := key.approxMemUsage(k)

	// Check if we would be out of memory from this
	outOfMemory := state.conf.maxmem > 0 && db.mem+keyMem >= state.conf.maxmem
	if outOfMemory {
		err := db.evictKeys(state, keyMem)
		if err != nil {
			return err
		}
	}

	db.store[k] = &Key{V: v}
	db.mem += keyMem
	log.Println("MEMORY: ", db.mem)

	return nil
}

// Delete is a "public" method to remove a key from the database
func (db *Database) Delete(k string) {
	key, ok := db.store[k]
	if !ok {
		return
	}
	keyMemory := key.approxMemUsage(k)
	delete(db.store, k)
	db.mem -= keyMemory
	log.Println("MEMORY: ", db.mem)
}

var DB = NewDatabase()

// Creating a key allows us to store expiry time
type Key struct {
	V   string
	Exp time.Time
}

// approxMemUsage approximates the memory usage of a key, given its name
func (key *Key) approxMemUsage(name string) int64 {
	stringHeaderSize := 16 // Bytes
	expiryHeaderSize := 24
	mapEntrySize := 32 // Structs are basically maps which have their own headers

	return int64(stringHeaderSize + len(name) + stringHeaderSize + len(key.V) + expiryHeaderSize + mapEntrySize)
}

// A Transaction is made of multiple commands
type Transaction struct {
	commands []*TxCommand
}

// NewTransaction creates a new Transaction type to group
// multiple commands together and execute them atomically.
// Used to implement the MULTI and EXEC commands
func NewTransaction() *Transaction {
	return &Transaction{}
}

// TxCommand is a command to be executed in a transaction
type TxCommand struct {
	v       *Value
	handler Handler
}
