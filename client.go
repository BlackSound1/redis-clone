package main

import "net"

type Client struct {
	conn          net.Conn
	authenticated bool
}

// NewClient creates a new Client type with a given net.Conn and authenticated set to false.
// Keeps track of the state of each client connection
func NewClient(conn net.Conn) *Client {
	return &Client{conn: conn}
}
