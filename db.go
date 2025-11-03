package main

import (
	"sync"
	"time"
)

// A Database type containing a key, value store and
// a mutex lock to allow concurrency
type Database struct {
	store map[string]*Key
	mu    sync.RWMutex
}

// NewDatabase creates a new Database type
func NewDatabase() *Database {
	return &Database{
		store: map[string]*Key{},
		mu:    sync.RWMutex{},
	}
}

// Set is a "public" method to set values to DB keys, which we prefer to act "private"
func (db *Database) Set(k string, v string) {
	db.store[k] = &Key{V: v}
}

// Delete is a "public" method to remove a key from the database
func (db *Database) Delete(k string) {
	delete(db.store, k)
}

var DB = NewDatabase()

// Creating a key allows us to store expiry time
type Key struct {
	V   string
	Exp time.Time
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
