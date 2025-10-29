package main

import (
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
func (aof *AOF) Sync() {
	for {
		v := Value{}
		err := v.readArray(aof.f)
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Println("Unexpected error while reading AOF records: ", err)
			break
		}

		// Want a blank app state without AOF enabled
		blankState := NewAppState(&Config{})
		set(&v, blankState)
	}
}
