package vdom

import (
	"fmt"
	"math/rand"
	"time"
)

var globalRand = rand.New(rand.NewSource(time.Now().UnixNano()))

// GenerateUUID generates a basic UUID-like string using the current time and random numbers
func GenerateUUID() string {
	timestamp := time.Now().UnixNano()
	randPart1 := globalRand.Int63()
	randPart2 := globalRand.Int63()
	return fmt.Sprintf("%x-%x-%x", timestamp, randPart1, randPart2)
}

// GenerateID generates a unique ID for a node
func GenerateID(prefix string) string {
	return fmt.Sprintf("%s-%s", prefix, GenerateUUID())
}

// Helper function for mapping slices
func Map[T, U any](ts []T, f func(T) U) []U {
	us := make([]U, len(ts))
	for i, t := range ts {
		us[i] = f(t)
	}
	return us
}
