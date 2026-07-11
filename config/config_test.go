package config

import (
	"testing"
	"time"
)

func TestGetenv(t *testing.T) {
	if got := Getenv("GK_UNSET_KEY", "def"); got != "def" {
		t.Fatalf("unset: want def, got %q", got)
	}
	t.Setenv("GK_SET_KEY", "value")
	if got := Getenv("GK_SET_KEY", "def"); got != "value" {
		t.Fatalf("set: want value, got %q", got)
	}
}

func TestGetenvInt(t *testing.T) {
	if got := GetenvInt("GK_INT_UNSET", 7); got != 7 {
		t.Fatalf("unset: want 7, got %d", got)
	}
	t.Setenv("GK_INT_BAD", "notint")
	if got := GetenvInt("GK_INT_BAD", 7); got != 7 {
		t.Fatalf("invalid: want fallback 7, got %d", got)
	}
	t.Setenv("GK_INT_OK", "42")
	if got := GetenvInt("GK_INT_OK", 7); got != 42 {
		t.Fatalf("valid: want 42, got %d", got)
	}
}

func TestGetenvDuration(t *testing.T) {
	if got := GetenvDuration("GK_DUR_UNSET", time.Second); got != time.Second {
		t.Fatalf("unset: want 1s, got %v", got)
	}
	t.Setenv("GK_DUR_OK", "250ms")
	if got := GetenvDuration("GK_DUR_OK", time.Second); got != 250*time.Millisecond {
		t.Fatalf("valid: want 250ms, got %v", got)
	}
}
