package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
)

// A struct defining the data persistence settings
type Config struct {
	dir         string
	rdb         []RDBSnapshot
	rdbFn       string
	aofEnabled  bool
	aofFn       string
	aofFsync    FSyncMode
	requirepass bool
	password    string
	maxmem      int64
	eviction    Eviction
	memSamples  int
}

// NewConfig creates a new Config type with default values
func NewConfig() *Config {
	return &Config{}
}

// For RDB, in how many seconds must how many
// keys be changed to warrant saving to DB?
type RDBSnapshot struct {
	Secs        int
	KeysChanged int
}

// Define the different fsync modes for AOF
type FSyncMode string

const (
	Always   FSyncMode = "always"   // Always sync the file
	EverySec FSyncMode = "everysec" // Sync the file every second
	No       FSyncMode = "no"       // Let OS handle syncing
)

type Eviction string

const (
	NoEviction     Eviction = "noeviction"
	AllKeysRandom  Eviction = "allkeys-random"
	AllKeysLRU     Eviction = "allkeys-lru"
	AllKeysLFU     Eviction = "allkeys-lfu"
	VolatileRandom Eviction = "volatile-random"
	VolatileLRU    Eviction = "volatile-lru"
	VolatileLFU    Eviction = "volatile-lfu"
	VolatileTTL    Eviction = "volatile-ttl"
)

// readConf reads a configuration file and returns a Config type
// with the settings specified in the file. If the file cannot be
// read, it returns a Config type with default values
func readConf(filename string) *Config {
	conf := NewConfig()

	// Try to open the config file
	f, err := os.Open(filename)
	if err != nil {
		fmt.Printf("Cannot read %s - using default config instead\n", filename)
		return conf
	}
	defer f.Close()

	s := bufio.NewScanner(f)

	// For each line in the file, parse it
	for s.Scan() {
		l := s.Text()
		parseLine(l, conf)
	}

	if err := s.Err(); err != nil {
		fmt.Println("Error reading config file: ", err)
		return conf
	}

	// Ensure directory(ies) specified in the config file exist
	if conf.dir != "" {
		os.MkdirAll(conf.dir, 0755)
	}

	return conf
}

// parseLine takes a line from a config file and updates the Config
// accordingly. It splits the line by spaces and uses the first
// item as the command and the rest as arguments.
func parseLine(line string, conf *Config) {
	// Each line in the config file is split by spaces
	args := strings.Split(line, " ")

	// The command is always the first item
	cmd := args[0]

	switch cmd {
	case "save":
		// The number of seconds is always the 2nd item
		secs, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Println("Invalid number of seconds")
			return
		}

		// The number of keys changed is always the 3rd item
		keysChanged, err := strconv.Atoi(args[2])
		if err != nil {
			fmt.Println("Invalid number of keys changed")
			return
		}

		// Update the RDB settings based on these values
		snapshot := RDBSnapshot{
			Secs:        secs,
			KeysChanged: keysChanged,
		}
		conf.rdb = append(conf.rdb, snapshot)
	case "dbfilename":
		conf.rdbFn = args[1]
	case "appendfilename":
		conf.aofFn = args[1]
	case "appendfsync":
		conf.aofFsync = FSyncMode(args[1])
	case "dir":
		conf.dir = args[1]
	case "appendonly":
		if args[1] == "yes" {
			conf.aofEnabled = true
		} else {
			conf.aofEnabled = false
		}
	case "requirepass":
		conf.requirepass = true
		conf.password = args[1]
	case "maxmemory":
		maxmem, err := parseMem(args[1])
		if err != nil {
			log.Println("Can't parse maxmemory. Defaulting to 0: ", err)
			conf.maxmem = 0
			break
		}
		conf.maxmem = maxmem
	case "maxmemory-policy":
		conf.eviction = Eviction(args[1])
	case "maxmemory-samples":
		memSamples, err := strconv.Atoi(args[1])
		if err != nil {
			log.Println("Can't parse maxmemory-samples. Defaulting to 50: ", err)
			conf.memSamples = 50
			break
		}
		conf.memSamples = memSamples
	}

}

// parseMem takes a string representing a memory size and returns an int64 value
// representing that memory in bytes.
//
// It supports "kb", "mb", "gb", and "b" suffixes.
// For example, "1024kb" would be parsed into 1048576
func parseMem(mem string) (int64, error) {
	mem = strings.TrimSpace(strings.ToLower(mem))

	var multiplier int64 = 1

	switch {
	case strings.HasSuffix(mem, "kb"):
		multiplier = 1024
		mem = strings.TrimSuffix(mem, "kb")
	case strings.HasSuffix(mem, "mb"):
		multiplier = 1024 * 1024
		mem = strings.TrimSuffix(mem, "mb")
	case strings.HasSuffix(mem, "gb"):
		multiplier = 1024 * 1024 * 1024
		mem = strings.TrimSuffix(mem, "gb")
	case strings.HasSuffix(mem, "b"):
		multiplier = 1
		mem = strings.TrimSuffix(mem, "b")
	}

	num, err := strconv.ParseInt(mem, 10, 64)
	if err != nil {
		return 0, err
	}

	return num * multiplier, nil
}
