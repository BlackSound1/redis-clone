package main

import "sync"

// A Database type containing a key, value store and
// a mutex lock to allow concurrency
type Database struct {
	store map[string]string
	mu    sync.RWMutex
}

// NewDatabase creates a new Database type
func NewDatabase() *Database {
	return &Database{
		store: map[string]string{},
		mu:    sync.RWMutex{},
	}
}

var DB = NewDatabase()
