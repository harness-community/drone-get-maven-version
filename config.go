package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	modeEffective = "effective"
	modeRawGAV    = "raw_gav"

	defaultPrefix = "MAVEN"
	defaultPom    = "pom.xml"
)

var errNoPomSource = errors.New("POM Path is empty: set pom_file (PLUGIN_POM_FILE) or pom_path (PLUGIN_POM_PATH), exiting...")

// Config is the validated plugin configuration.
type Config struct {
	// PomFile is the POM to read, relative to the working directory unless
	// absolute.
	PomFile string
	// Mode is modeEffective or modeRawGAV.
	Mode string
	// Prefix is the normalized output prefix. Only used in raw_gav mode.
	Prefix string
	// OutputFile is the DRONE_OUTPUT file outputs are appended to.
	OutputFile string
	// Warnings are non-fatal configuration notes for the user.
	Warnings []string
}

// loadConfig reads the PLUGIN_* settings through getenv and validates them.
func loadConfig(getenv func(string) string) (Config, error) {
	cfg := Config{OutputFile: getenv("DRONE_OUTPUT")}

	pomFile := strings.TrimSpace(getenv("PLUGIN_POM_FILE"))
	pomPath := strings.TrimSpace(getenv("PLUGIN_POM_PATH"))
	switch {
	case pomFile != "":
		cfg.PomFile = nativePath(pomFile)
		if pomPath != "" {
			cfg.Warnings = append(cfg.Warnings, fmt.Sprintf("both pom_file (%s) and pom_path (%s) are set; using pom_file", pomFile, pomPath))
		}
	case pomPath != "":
		cfg.PomFile = filepath.Join(nativePath(pomPath), defaultPom)
	default:
		return cfg, errNoPomSource
	}

	mode := strings.ToLower(strings.TrimSpace(getenv("PLUGIN_MODE")))
	switch mode {
	case "":
		cfg.Mode = modeEffective
	case modeEffective, modeRawGAV:
		cfg.Mode = mode
	default:
		return cfg, fmt.Errorf("invalid mode %q: use %q or %q", getenv("PLUGIN_MODE"), modeEffective, modeRawGAV)
	}

	rawPrefix := strings.TrimSpace(getenv("PLUGIN_VARIABLE_PREFIX"))
	if cfg.Mode != modeRawGAV {
		if rawPrefix != "" {
			cfg.Warnings = append(cfg.Warnings, fmt.Sprintf("variable_prefix is ignored in %s mode", cfg.Mode))
		}
		return cfg, nil
	}
	if rawPrefix == "" {
		cfg.Prefix = defaultPrefix
		return cfg, nil
	}
	prefix, err := normalizePrefix(rawPrefix)
	if err != nil {
		return cfg, err
	}
	cfg.Prefix = prefix
	return cfg, nil
}

// normalizePrefix uppercases the prefix, turns every run of characters
// outside [A-Za-z0-9] into a single '_' and trims leading/trailing '_'.
func normalizePrefix(raw string) (string, error) {
	var b strings.Builder
	pendingSep := false
	for _, r := range raw {
		isAlnum := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if !isAlnum {
			pendingSep = true
			continue
		}
		if pendingSep && b.Len() > 0 {
			b.WriteByte('_')
		}
		pendingSep = false
		if r >= 'a' && r <= 'z' {
			r -= 'a' - 'A'
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("invalid variable_prefix %q: it must contain at least one letter or digit", raw)
	}
	return b.String(), nil
}

// nativePath accepts both '/' and '\' as separators and returns the path with
// the separators of the running OS.
func nativePath(p string) string {
	if runtime.GOOS == "windows" {
		return filepath.FromSlash(p)
	}
	return strings.ReplaceAll(p, `\`, "/")
}
