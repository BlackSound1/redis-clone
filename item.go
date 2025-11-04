package main

import "time"

// Creating a key allows us to store expiry time
type Item struct {
	V          string
	Exp        time.Time
	LastAccess time.Time
	Accesses   int
}

// shouldExpire decides whether the current item should be expired
func (item *Item) shouldExpire() bool {
	return item.Exp.Unix() != UNIX_TIMESTAMP && time.Until(item.Exp).Seconds() <= 0
}

// approxMemUsage approximates the memory usage of a key, given its name
func (item *Item) approxMemUsage(name string) int64 {
	stringHeaderSize := 16 // Bytes
	expiryHeaderSize := 24
	mapEntrySize := 32 // Structs are basically maps which have their own headers

	return int64(stringHeaderSize + len(name) + stringHeaderSize + len(item.V) + expiryHeaderSize + mapEntrySize)
}
