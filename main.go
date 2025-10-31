package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

func main() {
	// Read config file
	log.Println("Reading config file")
	conf := readConf("./redis.conf")

	state := NewAppState(conf)

	if conf.aofEnabled {
		log.Println("Syncing AOF records")
		state.aof.Sync()
	}

	// If there are any RDB snapshots, save to memory any RDB values saved to the file
	if len(conf.rdb) > 0 {
		SyncRDB(conf)
		InitRDBTrackers(state)
	}

	// Create a TCP listener on port 6379, the default Redis port
	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal("Cannot listen on part 6379. Quitting.")
	}
	defer l.Close()
	log.Println("Listening on port 6379")

	var wg sync.WaitGroup

	for {
		// Block until connection is made
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Wait for 1 goroutine to finish
		wg.Add(1)
		go func() {
			handleConn(conn, state)
			wg.Done() // Decrement wg counter by 1
		}()
	}
	wg.Wait() // Block until wg counter is 0
}

// handleConn calls the handler associated with the bulk string of the first message in the Value.
// It then writes the reply from the handler back to the connection.
// It will continuously read RESP messages from the connection until it is closed
func handleConn(conn net.Conn, state *AppState) {
	log.Println("Accepted new connection: ", conn.LocalAddr().String())
	for {
		v := Value{typ: ARRAY}
		if err := v.readArray(conn); err != nil {
			log.Println(err)
			break
		}

		handle(conn, &v, state)

		fmt.Println(v.array)
	}
	log.Println("Connection closed: ", conn.LocalAddr().String())
}

type AppState struct {
	conf          *Config
	aof           *AOF
	bgSaveRunning bool
	dbCopy        map[string]string
}

// NewAppState creates a new AppState type with the given Config settings
// If the Config type specifies that AOF should be enabled, it will create a new AOF type
// and, if necessary, a new goroutine to flush the writer every second
func NewAppState(conf *Config) *AppState {
	state := AppState{
		conf: conf,
	}

	if conf.aofEnabled {
		state.aof = NewAOF(conf)

		// If aofSync mode is everysec, set up a new goroutine
		// that, every second, flushes the writers buffer
		if conf.aofFsync == EverySec {
			go func() {
				t := time.NewTicker(time.Second)
				defer t.Stop()

				for range t.C {
					state.aof.w.Flush()
				}
			}()
		}
	}

	return &state
}
