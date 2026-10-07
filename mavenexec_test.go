package main

import (
	"errors"
	"strings"
	"testing"
)

// fakeMaven replaces runMaven for the duration of the test.
func fakeMaven(t *testing.T, stdout, stderr string, err error) *[]string {
	t.Helper()
	var gotArgs []string
	orig := runMaven
	runMaven = func(args ...string) ([]byte, []byte, error) {
		gotArgs = args
		return []byte(stdout), []byte(stderr), err
	}
	t.Cleanup(func() { runMaven = orig })
	return &gotArgs
}

func TestEffectiveVersionSuccess(t *testing.T) {
	args := fakeMaven(t, "  1.2.3-SNAPSHOT\r\n", "", nil)
	got, err := effectiveVersion("dir/pom.xml")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.2.3-SNAPSHOT" {
		t.Errorf("got %q", got)
	}
	want := "-f dir/pom.xml help:evaluate -Dexpression=project.version -q -DforceStdout"
	if strings.Join(*args, " ") != want {
		t.Errorf("args %q, want %q", *args, want)
	}
}

func TestEffectiveVersionFailureIncludesMavenOutput(t *testing.T) {
	fakeMaven(t, "[ERROR] Non-resolvable parent POM", "picked up JAVA_TOOL_OPTIONS", errors.New("exit status 1"))
	_, err := effectiveVersion("pom.xml")
	if err == nil {
		t.Fatal("expected error")
	}
	for _, s := range []string{"exit status 1", "pom.xml", "effective mode", "Non-resolvable parent POM", "JAVA_TOOL_OPTIONS"} {
		if !strings.Contains(err.Error(), s) {
			t.Errorf("error %q does not contain %q", err, s)
		}
	}
}

func TestEffectiveVersionEmptyOutput(t *testing.T) {
	fakeMaven(t, " \n", "", nil)
	_, err := effectiveVersion("pom.xml")
	if err == nil || !strings.Contains(err.Error(), "empty project.version") {
		t.Fatalf("got %v", err)
	}
}

func TestMavenLogTruncates(t *testing.T) {
	log := mavenLog([]byte(strings.Repeat("x", maxMavenLog*2)), nil)
	if len(log) > maxMavenLog+64 || !strings.HasPrefix(log, "...(truncated)") {
		t.Errorf("log not truncated: %d bytes", len(log))
	}
}
