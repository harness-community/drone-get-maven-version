package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

// maxPlaceholderDepth bounds nested ${...} resolution.
const maxPlaceholderDepth = 5

// GAV is a set of Maven coordinates.
type GAV struct {
	GroupID    string
	ArtifactID string
	Version    string
}

// rawPOM holds the direct children of <project> that raw_gav mode reads.
// encoding/xml only matches direct children against these tags, so values
// nested in <dependencies>, <build>, <profiles> and so on never reach it.
type rawPOM struct {
	XMLName    xml.Name `xml:"project"`
	GroupID    string   `xml:"groupId"`
	ArtifactID string   `xml:"artifactId"`
	Version    string   `xml:"version"`
	Parent     struct {
		GroupID    string `xml:"groupId"`
		ArtifactID string `xml:"artifactId"`
		Version    string `xml:"version"`
	} `xml:"parent"`
	Properties struct {
		Entries []struct {
			XMLName xml.Name
			Value   string `xml:",chardata"`
		} `xml:",any"`
	} `xml:"properties"`
}

// readRawGAV reads groupId, artifactId and version from the POM at path
// without running Maven. logf receives informational messages.
func readRawGAV(path string, logf func(format string, args ...any)) (GAV, error) {
	pom, err := parsePOM(path)
	if err != nil {
		return GAV{}, err
	}

	own := GAV{
		GroupID:    strings.TrimSpace(pom.GroupID),
		ArtifactID: strings.TrimSpace(pom.ArtifactID),
		Version:    strings.TrimSpace(pom.Version),
	}
	parent := GAV{
		GroupID:    strings.TrimSpace(pom.Parent.GroupID),
		ArtifactID: strings.TrimSpace(pom.Parent.ArtifactID),
		Version:    strings.TrimSpace(pom.Parent.Version),
	}
	gav := applyParentFallback(own, parent, logf)

	props := map[string]string{}
	for _, e := range pom.Properties.Entries {
		props[e.XMLName.Local] = strings.TrimSpace(e.Value)
	}
	for k, v := range map[string]string{
		"project.groupId":           gav.GroupID,
		"project.artifactId":        gav.ArtifactID,
		"project.version":           gav.Version,
		"project.parent.groupId":    parent.GroupID,
		"project.parent.artifactId": parent.ArtifactID,
		"project.parent.version":    parent.Version,
		"parent.groupId":            parent.GroupID,
		"parent.artifactId":         parent.ArtifactID,
		"parent.version":            parent.Version,
	} {
		props[k] = v
	}

	r := resolver{props: props}
	fields := []struct {
		name  string
		value *string
	}{
		{"groupId", &gav.GroupID},
		{"artifactId", &gav.ArtifactID},
		{"version", &gav.Version},
	}
	for _, f := range fields {
		if *f.value == "" {
			return GAV{}, fmt.Errorf("%s is missing in %s (raw_gav mode)", f.name, path)
		}
		resolved, err := r.resolve(*f.value, nil)
		if err != nil {
			return GAV{}, fmt.Errorf("cannot resolve %s %q in %s (raw_gav mode): %w; use mode: effective to let Maven resolve it", f.name, *f.value, path, err)
		}
		if resolved == "" {
			return GAV{}, fmt.Errorf("%s is empty in %s after property resolution (raw_gav mode)", f.name, path)
		}
		*f.value = resolved
	}
	return gav, nil
}

// applyParentFallback fills groupId and version from <parent> when the
// project does not declare them. artifactId is never inherited.
func applyParentFallback(own, parent GAV, logf func(format string, args ...any)) GAV {
	if own.GroupID == "" && parent.GroupID != "" {
		own.GroupID = parent.GroupID
		logf("groupId taken from parent")
	}
	if own.Version == "" && parent.Version != "" {
		own.Version = parent.Version
		logf("version taken from parent")
	}
	return own
}

func parsePOM(path string) (*rawPOM, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read POM file: %w", err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("POM file %s is a directory; set pom_file to a file or use pom_path for a directory containing pom.xml", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read POM file: %w", err)
	}
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, fmt.Errorf("POM file %s is empty", path)
	}

	dec := xml.NewDecoder(bytes.NewReader(data))
	dec.CharsetReader = charsetReader
	var pom rawPOM
	if err := dec.Decode(&pom); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("POM file %s has no <project> element", path)
		}
		return nil, fmt.Errorf("POM file %s is not valid XML: %w", path, err)
	}
	return &pom, nil
}

// charsetReader handles the non-UTF-8 encodings POM files declare in
// practice. encoding/xml handles UTF-8 itself.
func charsetReader(label string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(label) {
	case "utf8", "us-ascii", "ascii":
		return input, nil
	case "iso-8859-1", "iso8859-1", "latin1", "latin-1":
		data, err := io.ReadAll(input)
		if err != nil {
			return nil, err
		}
		// Every ISO-8859-1 byte is the Unicode code point of the same value.
		var b strings.Builder
		b.Grow(len(data))
		for _, c := range data {
			b.WriteRune(rune(c))
		}
		return strings.NewReader(b.String()), nil
	}
	return nil, fmt.Errorf("unsupported XML encoding %q (supported: UTF-8, US-ASCII, ISO-8859-1)", label)
}

var placeholderRE = regexp.MustCompile(`\$\{([^}]*)\}`)

type resolver struct {
	props map[string]string
}

// resolve replaces ${name} placeholders in value. stack holds the property
// names being resolved, for loop detection and depth limiting.
func (r resolver) resolve(value string, stack []string) (string, error) {
	var resolveErr error
	out := placeholderRE.ReplaceAllStringFunc(value, func(match string) string {
		if resolveErr != nil {
			return match
		}
		name := strings.TrimSpace(match[2 : len(match)-1])
		for _, s := range stack {
			if s == name {
				resolveErr = fmt.Errorf("property loop: %s -> %s", strings.Join(stack, " -> "), name)
				return match
			}
		}
		if len(stack) >= maxPlaceholderDepth {
			resolveErr = fmt.Errorf("placeholders nested more than %d levels deep at ${%s}", maxPlaceholderDepth, name)
			return match
		}
		v, ok := r.props[name]
		if !ok {
			resolveErr = fmt.Errorf("${%s} is not defined in the POM's <properties>", name)
			return match
		}
		resolved, err := r.resolve(v, append(append([]string(nil), stack...), name))
		if err != nil {
			resolveErr = err
			return match
		}
		return resolved
	})
	if resolveErr != nil {
		return "", resolveErr
	}
	return out, nil
}
