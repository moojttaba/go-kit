// Package config provides small, dependency-free helpers for reading
// 12-factor configuration from the environment. Services build their own
// typed Config struct on top of these.
package config

import (
	"os"
	"strconv"
	"time"
)

// Getenv returns the value of key, or def when the variable is unset or empty.
func Getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// GetenvInt reads key as an int, falling back to def when unset or unparseable.
func GetenvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// GetenvDuration reads key as a time.Duration (e.g. "5s"), falling back to def
// when unset or unparseable.
func GetenvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
