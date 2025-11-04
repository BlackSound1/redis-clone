package main

// A Transaction is made of multiple commands
type Transaction struct {
	commands []*TxCommand
}

// NewTransaction creates a new Transaction type to group
// multiple commands together and execute them atomically.
// Used to implement the MULTI and EXEC commands
func NewTransaction() *Transaction {
	return &Transaction{}
}

// TxCommand is a command to be executed in a transaction
type TxCommand struct {
	v       *Value
	handler Handler
}
