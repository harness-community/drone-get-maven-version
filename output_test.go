package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestWriteOutputsFormatAndAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "output.env")
	if err := os.WriteFile(path, []byte("EXISTING=1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeOutputs(path, []Output{{"MAVEN_GROUP_ID", "com.example"}, {"POM_VERSION", "1.0"}}); err != nil {
		t.Fatal(err)
	}
	if err := writeOutputs(path, []Output{{"OTHER", "a=b c"}}); err != nil {
		t.Fatal(err)
	}
	want := "EXISTING=1\nMAVEN_GROUP_ID=com.example\nPOM_VERSION=1.0\nOTHER=a=b c\n"
	if got := readFile(t, path); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestWriteOutputsCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "new.env")
	if err := writeOutputs(path, []Output{{"POM_VERSION", "2"}}); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); got != "POM_VERSION=2\n" {
		t.Errorf("got %q", got)
	}
}

func TestWriteOutputsMissingDroneOutput(t *testing.T) {
	err := writeOutputs("", []Output{{"POM_VERSION", "1"}})
	if err == nil || !strings.Contains(err.Error(), "DRONE_OUTPUT is not set") {
		t.Fatalf("got %v", err)
	}
}

func TestWriteOutputsRejectsLineBreaks(t *testing.T) {
	for _, v := range []string{"1.0\nINJECTED=1", "1.0\r", "\n"} {
		path := filepath.Join(t.TempDir(), "output.env")
		err := writeOutputs(path, []Output{{"A", "ok"}, {"POM_VERSION", v}})
		if err == nil || !strings.Contains(err.Error(), "line break") {
			t.Fatalf("value %q: got %v", v, err)
		}
		if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
			t.Errorf("value %q: nothing should be written, stat err %v", v, statErr)
		}
	}
}

func TestWriteOutputsRejectsBadKeys(t *testing.T) {
	for _, k := range []string{"", "A=B", "A\nB"} {
		if err := writeOutputs(filepath.Join(t.TempDir(), "o"), []Output{{k, "v"}}); err == nil {
			t.Errorf("key %q: expected error", k)
		}
	}
}
