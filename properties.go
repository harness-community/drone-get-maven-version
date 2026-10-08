package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// parseMavenProperties parses the maven_properties setting: key=value pairs
// separated by commas or newlines, each optionally written as -Dkey=value.
// A key without '=' is set to "true", as with mvn -Dkey.
func parseMavenProperties(raw string) (map[string]string, error) {
	props := map[string]string{}
	for _, item := range strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' }) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, value, err := splitProperty(strings.TrimPrefix(item, "-D"))
		if err != nil {
			return nil, fmt.Errorf("invalid maven_properties entry %q: %w", item, err)
		}
		props[key] = value
	}
	return props, nil
}

func splitProperty(s string) (key, value string, err error) {
	key, value, found := strings.Cut(s, "=")
	key = strings.TrimSpace(key)
	if !found {
		value = "true"
	}
	if key == "" {
		return "", "", errors.New("the property name is empty")
	}
	if strings.ContainsAny(key, " \t${}") {
		return "", "", fmt.Errorf("the property name %q contains whitespace or placeholder characters", key)
	}
	return key, strings.TrimSpace(value), nil
}

// findMavenConfig returns the .mvn/maven.config that Maven would use for
// pomFile: the one in the nearest directory, from the POM's directory up,
// that contains a .mvn directory. It returns "" when there is none.
func findMavenConfig(pomFile string) (string, error) {
	dir, err := filepath.Abs(filepath.Dir(pomFile))
	if err != nil {
		return "", err
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, ".mvn")); err == nil && info.IsDir() {
			config := filepath.Join(dir, ".mvn", "maven.config")
			if _, err := os.Stat(config); err != nil {
				if errors.Is(err, fs.ErrNotExist) {
					return "", nil
				}
				return "", err
			}
			return config, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil
		}
		dir = parent
	}
}

// parseMavenConfig extracts the -D properties from .mvn/maven.config
// content. Other options (-T 4, --batch-mode, ...) are ignored.
func parseMavenConfig(content string) (map[string]string, error) {
	props := map[string]string{}
	tokens := splitArgs(strings.TrimPrefix(content, "\ufeff"))
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		var def string
		switch {
		case tok == "-D" || tok == "--define":
			if i+1 >= len(tokens) {
				return nil, fmt.Errorf("%s at the end of the file has no property", tok)
			}
			i++
			def = tokens[i]
		case strings.HasPrefix(tok, "--define="):
			def = strings.TrimPrefix(tok, "--define=")
		case strings.HasPrefix(tok, "-D"):
			def = strings.TrimPrefix(tok, "-D")
		default:
			continue
		}
		key, value, err := splitProperty(unquote(def))
		if err != nil {
			return nil, fmt.Errorf("invalid property %q: %w", tok, err)
		}
		props[key] = unquote(value)
	}
	return props, nil
}

// splitArgs splits on whitespace outside single or double quotes. Quotes are
// kept so that unquote can strip them per key and value.
func splitArgs(s string) []string {
	var tokens []string
	var cur strings.Builder
	var quote rune
	inToken := false
	for _, r := range s {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			}
		case r == '"' || r == '\'':
			quote = r
		case r == ' ' || r == '\t' || r == '\r' || r == '\n':
			if inToken {
				tokens = append(tokens, cur.String())
				cur.Reset()
				inToken = false
			}
			continue
		}
		cur.WriteRune(r)
		inToken = true
	}
	if inToken {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

func unquote(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// loadUserProperties returns the -D style properties for pomFile:
// .mvn/maven.config, overridden by the maven_properties setting.
func loadUserProperties(pomFile string, setting map[string]string, logf func(format string, args ...any)) (map[string]string, error) {
	props := map[string]string{}
	config, err := findMavenConfig(pomFile)
	if err != nil {
		return nil, fmt.Errorf("cannot look for .mvn/maven.config: %w", err)
	}
	if config != "" {
		data, err := os.ReadFile(config)
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", config, err)
		}
		fromConfig, err := parseMavenConfig(string(data))
		if err != nil {
			return nil, fmt.Errorf("cannot parse %s: %w", config, err)
		}
		if len(fromConfig) > 0 {
			logf("properties read from %s: %s", config, strings.Join(sortedKeys(fromConfig), ", "))
		}
		for k, v := range fromConfig {
			props[k] = v
		}
	}
	if len(setting) > 0 {
		logf("properties from maven_properties: %s", strings.Join(sortedKeys(setting), ", "))
	}
	for k, v := range setting {
		props[k] = v
	}
	return props, nil
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
