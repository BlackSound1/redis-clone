package main

import (
	"fmt"
	"log"
	"net"
	"time"
)

type Client struct {
	conn          net.Conn
	authenticated bool
}

// NewClient creates a new Client type with a given net.Conn and authenticated set to false.
// Keeps track of the state of each client connection
func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn}
}

// writeMonitorLog logs the command sent to the server by a client to the log stream
func (client *Client) writeMonitorLog(value *Value) {
	log.Println("Relaying command to MONITOR: ", client.conn.LocalAddr().String())

	msg := fmt.Sprintf("%d [%s]", time.Now().Unix(), client.conn.LocalAddr().String())

	for _, v := range value.array {
		msg += fmt.Sprintf(" \"%s\"", v.bulk)
	}

	writer := NewWriter(client.conn)
	reply := Value{typ: STRING, str: msg}
	writer.Write(&reply)
	writer.Flush()
}
