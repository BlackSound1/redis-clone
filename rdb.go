package main

import (
	"encoding/gob"
	"log"
	"os"
	"path"
	"time"
)

type SnapshotTracker struct {
	keys   int
	ticker time.Ticker
	RDB    RDBSnapshot
}

// NewSnapshotTracker creates a new SnapshotTracker type with the given RDB settings.
// The returned SnapshotTracker will tick every rdb.Secs seconds
func NewSnapshotTracker(rdb *RDBSnapshot) *SnapshotTracker {
	return &SnapshotTracker{
		keys:   0,
		ticker: *time.NewTicker(time.Second * time.Duration(rdb.Secs)),
		RDB:    *rdb,
	}
}

var trackers = []*SnapshotTracker{}

// InitRDBTrackers initializes the SnapshotTracker types based on the RDB settings from the
// given Config. It then starts a goroutine for each tracker to keep track of the
// number of keys changed and save to DB every snapshot.Secs seconds if the number of
// keys changed is at least snapshot.KeysChanged
func InitRDBTrackers(conf *Config) {
	// Go through each RDB snapshot and track it
	for _, rdb := range conf.rdb {
		tracker := NewSnapshotTracker(&rdb)
		trackers = append(trackers, tracker)

		// Run without blocking main thread
		go func() {
			defer tracker.ticker.Stop()

			// Keep reading the tickers channel until it closes.
			// If the number of keys changed is at least the threshold, save to DB
			for range tracker.ticker.C {
				if tracker.keys >= tracker.RDB.KeysChanged {
					SaveRDB(conf)
				}

				tracker.keys = 0
			}
		}()
	}
}

// IncrementRDBTrackers increments the key count for each SnapshotTracker.
// This function should be called whenever a key is changed in the database.
// If the number of keys changed is at least the threshold, the next call to
// SaveDB will save the database to disk
func IncrementRDBTrackers() {
	for _, t := range trackers {
		t.keys++
	}
}

// SaveRDB saves the current state of the database to a file on disk, in bytes
func SaveRDB(conf *Config) {
	filepath := path.Join(conf.dir, conf.rdbFn)
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("Error opening RDB file: ", err)
		return
	}
	defer f.Close()

	// Read the contents of the DB store to the encoder, which encodes it to bytes
	err = gob.NewEncoder(f).Encode(&DB.store)
	if err != nil {
		log.Println("Error saving to RDB file: ", err)
		return
	}
}

// SyncRDB reads the contents of the RDB file and decodes it into the
// current state of the database
func SyncRDB(conf *Config) {
	filepath := path.Join(conf.dir, conf.rdbFn)
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDONLY, 0644)
	if err != nil {
		log.Println("Error opening RDB file: ", err)
		return
	}
	defer f.Close()

	err = gob.NewDecoder(f).Decode(&DB.store)
	if err != nil {
		log.Println("Error decoding RDB file: ", err)
	}
}
