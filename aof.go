package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"path"
)

type AOF struct {
	w    *Writer
	f    *os.File
	conf *Config
}

// NewAOF creates a new AOF type with the given Config settings
func NewAOF(conf *Config) *AOF {
	aof := AOF{conf: conf}

	filepath := path.Join(aof.conf.dir, aof.conf.aofFn)
	f, err := os.OpenFile(filepath, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0644)
	if err != nil {
		fmt.Println("Cannot open: ", filepath)
		return &aof
	}

	aof.w = NewWriter(f)
	aof.f = f

	return &aof
}

// Sync reads all RESP messages from the AOF file and applies all SET commands found in it
func (aof *AOF) Sync(maxmem int64, evictionPolicy Eviction, memSamples int) {
	r := bufio.NewReader(aof.f)
	for {
		v := Value{}
		err := v.readArray(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Unexpected error while reading AOF records: ", err)
			break
		}

		// Want a blank app state without AOF enabled
		blankState := NewAppState(&Config{
			maxmem:     maxmem,
			eviction:   evictionPolicy,
			memSamples: memSamples,
		})
		blankClient := Client{}
		set(&blankClient, &v, blankState)
	}
}

// Rewrite rewrites the AOF file to reflect the current state of the DB
func (aof *AOF) Rewrite(copy map[string]*Item) {
	// Reroute future AOF records to buffer because the file will be busy as we rewrite it
	var buffer bytes.Buffer
	aof.w = NewWriter(&buffer)

	// Clear file
	if err := aof.f.Truncate(0); err != nil {
		log.Println("AOF rewrite - Truncate error: ", err)
		return
	}

	// Go back to beginning of file for rewriting
	if _, err := aof.f.Seek(0, 0); err != nil {
		log.Println("AOF rewrite - Seek error: ", err)
		return
	}

	// Create a new writer for the file
	fileWriter := NewWriter(aof.f)

	// An AOF file is just an ARRAY of SET strings
	for k, v := range copy {
		command := Value{typ: BULK, bulk: "SET"}
		key := Value{typ: BULK, bulk: k}
		value := Value{typ: BULK, bulk: v.V}

		arr := Value{typ: ARRAY, array: []Value{
			command, key, value,
		}}
		fileWriter.Write(&arr)
	}

	fileWriter.Flush()

	// Reroute future AOF records back to file
	aof.w = NewWriter(aof.f)
}
