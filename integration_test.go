//go:build integration

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Needs mvn on PATH and access to a repository serving maven-help-plugin.

func TestIntegrationEffectiveInheritance(t *testing.T) {
	got, err := effectiveVersion(filepath.Join("testdata", "inheritance", "child", "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.2.3-SNAPSHOT" {
		t.Errorf("got %q, want 1.2.3-SNAPSHOT", got)
	}
}

// With only PLUGIN_POM_PATH set, DRONE_OUTPUT must be exactly what the
// pre-change plugin wrote.
func TestIntegrationPomPathOnly(t *testing.T) {
	out := filepath.Join(t.TempDir(), "output.env")
	var stdout, stderr bytes.Buffer
	code := run(envFrom(map[string]string{
		"PLUGIN_POM_PATH": filepath.Join("testdata", "inheritance", "child"),
		"DRONE_OUTPUT":    out,
	}), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "POM_VERSION=1.2.3-SNAPSHOT\n" {
		t.Errorf("DRONE_OUTPUT = %q", b)
	}
}

func TestIntegrationEffectiveFailureShowsMavenError(t *testing.T) {
	pom := writePOM(t, project(`<artifactId>a</artifactId>`))
	_, err := effectiveVersion(pom)
	if err == nil {
		t.Fatal("expected Maven to fail on a POM without groupId and version")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("Maven output")) {
		t.Errorf("error does not include Maven output: %v", err)
	}
}
