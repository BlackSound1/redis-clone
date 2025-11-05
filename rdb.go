package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"io"
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
func InitRDBTrackers(state *AppState) {
	// Go through each RDB snapshot and track it
	for _, rdb := range state.conf.rdb {
		tracker := NewSnapshotTracker(&rdb)
		trackers = append(trackers, tracker)

		// Run without blocking main thread
		go func() {
			defer tracker.ticker.Stop()

			// Keep reading the tickers channel until it closes.
			// If the number of keys changed is at least the threshold, save to DB
			for range tracker.ticker.C {
				if tracker.keys >= tracker.RDB.KeysChanged {
					SaveRDB(state)
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
func SaveRDB(state *AppState) {
	filepath := path.Join(state.conf.dir, state.conf.rdbFn)

	// Create file if not exists, open for reading or writing, and make sure previous content is overwritten
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
	if err != nil {
		log.Println("Error opening RDB file: ", err)
		return
	}
	defer f.Close()

	// Save to a local buffer. If BGSAVE, save a local copy of the DB.
	// If not, save the actual DB
	var buffer bytes.Buffer
	if state.bgSaveRunning {
		err = gob.NewEncoder(&buffer).Encode(&state.dbCopy)
	} else {
		DB.mu.RLock()
		err = gob.NewEncoder(&buffer).Encode(&DB.store)
		DB.mu.RUnlock()
	}

	if err != nil {
		log.Println("Error encoding DB to buffer: ", err)
		return
	}

	// Read the data of the buffer once, so when it's read again
	// in the Hash function, we're not reading a buffer that's already been read
	data := buffer.Bytes()

	// Hash the buffer and get the SHA256 checksum. This will be compared to the file checksum later
	bufferSum, err := Hash(&buffer)
	if err != nil {
		log.Println("RDB - Can't compute buffer checksum: ", err)
		return
	}

	// Actually save to file
	_, err = f.Write(data)
	if err != nil {
		log.Println("RDB - Can't write to file: ", err)
		return
	}
	if err := f.Sync(); /*Force flushing to disk*/ err != nil {
		log.Println("RDB - Can't flush file to disk: ", err)
		return
	}

	// Compute the checksum of the file we just saved to for comparison
	if _, err := f.Seek(0, io.SeekStart); /* Force hash cursor to front of file*/ err != nil {
		log.Println("RDB - Can't seek file: ", err)
		return
	}
	fileSum, err := Hash(f)
	if err != nil {
		log.Println("RDB - Can't compute file checksum: ", err)
		return
	}

	if bufferSum != fileSum {
		log.Printf("RDB - Buffer and file checksums don't match:\nf=%s\nb=%s\n", fileSum, bufferSum)
		return
	}

	log.Println("Saved RDB file successfully")

	state.rdbStats.rdb_last_save_ts = time.Now().Unix()
	state.rdbStats.rdb_saves++
}

// SyncRDB reads the contents of the RDB file and decodes it into the
// current state of the database
func SyncRDB(conf *Config) {
	filepath := path.Join(conf.dir, conf.rdbFn)
	f, err := os.Open(filepath)
	if err != nil {
		log.Println("Error opening RDB file: ", err)
		f.Close()
		return
	}
	defer f.Close()

	err = gob.NewDecoder(f).Decode(&DB.store)
	if err != nil {
		log.Println("Error decoding RDB file: ", err)
	}
}

// Hash takes an io.Reader and returns a SHA-256 hash of its contents.
// The hash is returned as a hexadecimal encoded string
func Hash(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
