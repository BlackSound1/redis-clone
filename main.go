package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

func main() {
	// Read config file
	log.Println("Reading config file")
	readConf("./redis.conf")

	// Create a TCP listener on port 6379, the default Redis port
	l, err := net.Listen("tcp", ":6379")
	if err != nil {
		log.Fatal("Cannot listen on part 6379. Quitting.")
	}
	defer l.Close()
	log.Println("Listening on port 6379")

	// Block until connection is made
	conn, err := l.Accept() // TODO: Add ability to accept multiple connections
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer conn.Close()
	log.Println("Connection accepted!")

	for {
		v := Value{typ: ARRAY}
		v.readArray(conn)

		handle(conn, &v)

		fmt.Println(v.array)
	}
}
