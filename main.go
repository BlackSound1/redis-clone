package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"
)

// The number of seconds since Jan 1 1970
var UNIX_TIMESTAMP int64 = time.Time{}.Unix()

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

	for {
		// Block until connection is made
		conn, err := l.Accept()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Wait for 1 goroutine to finish
		go func() {
			handleConn(conn, state)
		}()
	}
}

// handleConn calls the handler associated with the bulk string of the first message in the Value.
// It then writes the reply from the handler back to the connection.
// It will continuously read RESP messages from the connection until it is closed
func handleConn(conn net.Conn, state *AppState) {
	log.Println("Accepted new connection: ", conn.LocalAddr().String())
	client := NewClient(conn)
	reader := bufio.NewReader(conn)
	for {
		v := Value{typ: ARRAY}
		if err := v.readArray(reader); err != nil {
			log.Println(err)
			break
		}

		handle(client, &v, state)

		fmt.Println(v.array)
	}
	log.Println("Connection closed: ", conn.LocalAddr().String())
}
