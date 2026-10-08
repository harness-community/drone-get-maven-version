package main

import (
	"fmt"
	"io"
	"os"
)

// Set through -ldflags "-X main.version=... -X main.build=...".
var (
	version = "dev"
	build   = ""
)

func main() {
	os.Exit(run(os.Getenv, os.Stdout, os.Stderr))
}

// run executes the plugin and returns the process exit code.
func run(getenv func(string) string, stdout, stderr io.Writer) int {
	fmt.Fprintf(stdout, "drone-get-maven-version %s %s\n", version, build)

	cfg, err := loadConfig(getenv)
	if err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	for _, w := range cfg.Warnings {
		fmt.Fprintln(stderr, "Warning:", w)
	}
	fmt.Fprintln(stdout, "POM file:", cfg.PomFile)
	fmt.Fprintln(stdout, "Mode:", cfg.Mode)

	var outputs []Output
	switch cfg.Mode {
	case modeRawGAV:
		logf := func(format string, args ...any) {
			fmt.Fprintf(stdout, format+"\n", args...)
		}
		userProps, err := loadUserProperties(cfg.PomFile, cfg.MavenProperties, logf)
		if err != nil {
			fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		gav, err := readRawGAV(cfg.PomFile, userProps, logf)
		if err != nil {
			fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		fmt.Fprintf(stdout, "groupId: %s\nartifactId: %s\nversion: %s\n", gav.GroupID, gav.ArtifactID, gav.Version)
		outputs = []Output{
			{cfg.Prefix + "_GROUP_ID", gav.GroupID},
			{cfg.Prefix + "_ARTIFACT_ID", gav.ArtifactID},
			{cfg.Prefix + "_VERSION", gav.Version},
			{"POM_VERSION", gav.Version},
		}
	default:
		v, err := effectiveVersion(cfg.PomFile, cfg.MavenProperties)
		if err != nil {
			fmt.Fprintln(stderr, "Error:", err)
			return 1
		}
		fmt.Fprintln(stdout, "POM Version:", v)
		outputs = []Output{{"POM_VERSION", v}}
	}

	if err := writeOutputs(cfg.OutputFile, outputs); err != nil {
		fmt.Fprintln(stderr, "Error:", err)
		return 1
	}
	for _, o := range outputs {
		fmt.Fprintf(stdout, "%s written to DRONE_OUTPUT\n", o.Key)
	}
	return 0
}
