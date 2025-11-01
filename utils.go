package main

import "slices"

// contains checks if a given string exists in a given slice of strings
func contains(slice []string, item string) bool {
	return slices.Contains(slice, item)
}
