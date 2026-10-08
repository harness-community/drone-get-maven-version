package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunEffectiveBackwardCompatible(t *testing.T) {
	out := filepath.Join(t.TempDir(), "output.env")
	args := fakeMaven(t, "1.2.3-SNAPSHOT", "", nil)

	var stdout, stderr bytes.Buffer
	code := run(envFrom(map[string]string{"PLUGIN_POM_PATH": ".", "DRONE_OUTPUT": out}), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr %s", code, stderr.String())
	}
	if got := readFile(t, out); got != "POM_VERSION=1.2.3-SNAPSHOT\n" {
		t.Errorf("DRONE_OUTPUT = %q", got)
	}
	if (*args)[1] != "pom.xml" {
		t.Errorf("mvn -f %q", (*args)[1])
	}
}

func TestRunEffectiveFailure(t *testing.T) {
	out := filepath.Join(t.TempDir(), "output.env")
	fakeMaven(t, "[ERROR] boom", "", errors.New("exit status 1"))

	var stdout, stderr bytes.Buffer
	code := run(envFrom(map[string]string{"PLUGIN_POM_PATH": ".", "DRONE_OUTPUT": out}), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "[ERROR] boom") {
		t.Errorf("stderr missing Maven output: %s", stderr.String())
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("DRONE_OUTPUT must not be written on failure")
	}
}

func TestRunNoPomSource(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(envFrom(map[string]string{}), &stdout, &stderr); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "POM Path is empty") {
		t.Errorf("stderr %q", stderr.String())
	}
}

func TestRunRawGAV(t *testing.T) {
	out := filepath.Join(t.TempDir(), "output.env")
	fakeMaven(t, "", "", errors.New("maven must not run in raw_gav mode"))

	var stdout, stderr bytes.Buffer
	code := run(envFrom(map[string]string{
		"PLUGIN_POM_FILE":        filepath.Join("testdata", "inheritance", "child", "pom.xml"),
		"PLUGIN_VARIABLE_PREFIX": "maven",
		"PLUGIN_MODE":            "raw_gav",
		"DRONE_OUTPUT":           out,
	}), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr %s", code, stderr.String())
	}
	want := "MAVEN_GROUP_ID=com.example.inherit\n" +
		"MAVEN_ARTIFACT_ID=inherit-child\n" +
		"MAVEN_VERSION=1.2.3-SNAPSHOT\n" +
		"POM_VERSION=1.2.3-SNAPSHOT\n"
	if got := readFile(t, out); got != want {
		t.Errorf("DRONE_OUTPUT = %q, want %q", got, want)
	}
	for _, s := range []string{"groupId taken from parent", "version taken from parent"} {
		if !strings.Contains(stdout.String(), s) {
			t.Errorf("stdout missing %q", s)
		}
	}
}

func TestRunRawGAVFailure(t *testing.T) {
	out := filepath.Join(t.TempDir(), "output.env")
	pom := writePOM(t, project(`<groupId>g</groupId><artifactId>a</artifactId><version>${revision}</version>`))

	var stdout, stderr bytes.Buffer
	code := run(envFrom(map[string]string{"PLUGIN_POM_FILE": pom, "PLUGIN_MODE": "raw_gav", "DRONE_OUTPUT": out}), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(stderr.String(), "${revision}") {
		t.Errorf("stderr %q", stderr.String())
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Errorf("DRONE_OUTPUT must not be written on failure")
	}
}
