package main

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// maxMavenLog bounds how much Maven output is echoed back on failure.
const maxMavenLog = 8 << 10

// runMaven runs Maven with args and returns its stdout and stderr. Tests
// replace it to avoid needing Maven.
var runMaven = func(args ...string) (stdout, stderr []byte, err error) {
	// exec.LookPath resolves mvn.cmd through PATHEXT on Windows.
	mvn, err := exec.LookPath("mvn")
	if err != nil {
		return nil, nil, fmt.Errorf("mvn not found on PATH: %w", err)
	}
	var out, errOut bytes.Buffer
	cmd := exec.Command(mvn, args...)
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	err = cmd.Run()
	return out.Bytes(), errOut.Bytes(), err
}

// effectiveVersion asks Maven for the effective project.version of pomFile.
// props are passed as -Dkey=value; Maven reads .mvn/maven.config itself.
func effectiveVersion(pomFile string, props map[string]string) (string, error) {
	args := []string{"-f", pomFile, "help:evaluate", "-Dexpression=project.version", "-q", "-DforceStdout"}
	for _, k := range sortedKeys(props) {
		args = append(args, "-D"+k+"="+props[k])
	}
	stdout, stderr, err := runMaven(args...)
	if err != nil {
		msg := fmt.Sprintf("mvn help:evaluate failed for %s (effective mode): %v", pomFile, err)
		// With -q Maven reports build errors on stdout, so include both.
		if log := mavenLog(stdout, stderr); log != "" {
			msg += "\nMaven output:\n" + log
		}
		return "", errors.New(msg)
	}
	version := strings.TrimSpace(string(stdout))
	if version == "" {
		msg := fmt.Sprintf("mvn help:evaluate returned an empty project.version for %s (effective mode)", pomFile)
		if log := mavenLog(nil, stderr); log != "" {
			msg += "\nMaven output:\n" + log
		}
		return "", errors.New(msg)
	}
	return version, nil
}

func mavenLog(stdout, stderr []byte) string {
	out, errOut := strings.TrimSpace(string(stdout)), strings.TrimSpace(string(stderr))
	if out == errOut {
		errOut = ""
	}
	log := strings.TrimSpace(out + "\n" + errOut)
	if len(log) > maxMavenLog {
		log = "...(truncated)\n" + log[len(log)-maxMavenLog:]
	}
	return log
}
