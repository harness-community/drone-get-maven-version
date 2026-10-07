package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Output is one KEY=VALUE line for DRONE_OUTPUT.
type Output struct {
	Key   string
	Value string
}

// writeOutputs appends outputs to the DRONE_OUTPUT file at path. Every output
// is validated before anything is written, so a bad value never leaves a
// partial set behind.
func writeOutputs(path string, outputs []Output) error {
	if path == "" {
		return errors.New("DRONE_OUTPUT is not set; the plugin must run as a Harness plugin step to publish outputs")
	}
	var b strings.Builder
	for _, o := range outputs {
		if o.Key == "" || strings.ContainsAny(o.Key, "=\r\n") {
			return fmt.Errorf("invalid output name %q", o.Key)
		}
		if strings.ContainsAny(o.Value, "\r\n") {
			return fmt.Errorf("value for %s contains a line break; refusing to write it to DRONE_OUTPUT", o.Key)
		}
		fmt.Fprintf(&b, "%s=%s\n", o.Key, o.Value)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("opening DRONE_OUTPUT file: %w", err)
	}
	if _, err := f.WriteString(b.String()); err != nil {
		f.Close()
		return fmt.Errorf("writing DRONE_OUTPUT file %s: %w", path, err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("closing DRONE_OUTPUT file %s: %w", path, err)
	}
	return nil
}
