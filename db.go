package main

import (
	"errors"
	"log"
	"sort"
	"sync"
	"time"
)

// A Database type containing a key, value store and
// a mutex lock to allow concurrency
type Database struct {
	store map[string]*Item
	mu    sync.RWMutex
	mem   int64
}

// NewDatabase creates a new Database type
func NewDatabase() *Database {
	return &Database{
		store: map[string]*Item{},
		mu:    sync.RWMutex{},
	}
}

// evictKeys evicts keys from the DB according to eviction policies
func (db *Database) evictKeys(state *AppState, requiredMem int64) error {
	if state.conf.eviction == NoEviction {
		return errors.New("maximum memory reached")
	}

	// Get a sample of the keys in the DB
	samples := sampleKeys(state)

	// Local fn to check if enough memory has been freed
	enoughMemoryFreed := func() bool {
		if db.mem+requiredMem < state.conf.maxmem {
			return true
		} else {
			return false
		}
	}

	// Local fn to keep deleting keys from the sample keys until enough memory has been freed
	evictUntilMemoryFreed := func(samples []sample) int {
		var n int
		for _, s := range samples {
			log.Println("EVICTING: ", s.k)
			db.Delete(s.k)
			n++
			if enoughMemoryFreed() {
				break
			}
		}
		return n
	}

	// Evict based on eviction policy
	switch state.conf.eviction {
	case AllKeysRandom:
		evictedKeys := evictUntilMemoryFreed(samples)
		state.generalStats.evicted_keys += evictedKeys
	case AllKeysLRU:
		// Sort by least recently used
		sort.Slice(samples, func(i, j int) bool {
			return samples[i].v.LastAccess.After(samples[j].v.LastAccess)
		})
		evictedKeys := evictUntilMemoryFreed(samples)
		state.generalStats.evicted_keys += evictedKeys
	case AllKeysLFU:
		// Sort by least frequently used
		sort.Slice(samples, func(i, j int) bool {
			return samples[i].v.Accesses < samples[j].v.Accesses
		})
		evictedKeys := evictUntilMemoryFreed(samples)
		state.generalStats.evicted_keys += evictedKeys
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

	key := &Item{V: v}
	keyMem := key.approxMemUsage(k)

	// Check if we would be out of memory from this
	outOfMemory := state.conf.maxmem > 0 && db.mem+keyMem >= state.conf.maxmem
	if outOfMemory {
		err := db.evictKeys(state, keyMem)
		if err != nil {
			return err
		}
	}

	db.store[k] = &Item{V: v}
	db.mem += keyMem
	log.Println("MEMORY: ", db.mem)

	if db.mem > state.peakMem {
		state.peakMem = db.mem
	}

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

// Get is a "public" method to get a key from the database
func (db *Database) Get(key string, state *AppState) (i *Item, ok bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	item, ok := db.store[key]
	if !ok {
		return item, ok
	}
	expired := db.tryToExpire(key, item, state)
	if expired {
		return &Item{}, false
	}
	item.Accesses++
	item.LastAccess = time.Now()
	return item, ok
}

// tryToExpire checks if the given key has expired and should be deleted from the DB
func (db *Database) tryToExpire(key string, item *Item, state *AppState) bool {
	// If there is an expiry that has passed, delete the key and return NULL
	if item.shouldExpire() {
		DB.mu.Lock()
		DB.Delete(key)
		DB.mu.Unlock()
		state.generalStats.expired_keys++
		return true
	}
	return false
}

var DB = NewDatabase()
