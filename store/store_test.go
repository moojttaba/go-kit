package store

import (
	"testing"
	"testing/fstest"
)

// sqlFiles must return only *.sql files, sorted, ignoring dirs and other files.
func TestSQLFiles(t *testing.T) {
	fsys := fstest.MapFS{
		"0002_add.sql":   {Data: []byte("")},
		"0001_init.sql":  {Data: []byte("")},
		"README.md":      {Data: []byte("")},
		"sub/nested.sql": {Data: []byte("")},
	}
	got, err := sqlFiles(fsys)
	if err != nil {
		t.Fatalf("sqlFiles: %v", err)
	}
	want := []string{"0001_init.sql", "0002_add.sql"}
	if len(got) != len(want) {
		t.Fatalf("want %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("at %d: want %q, got %q", i, want[i], got[i])
		}
	}
}
