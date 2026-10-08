package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestLoadConfigPomSource(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		wantFile string
		wantWarn bool
		wantErr  string
	}{
		{
			name:    "neither set",
			env:     map[string]string{},
			wantErr: "POM Path is empty",
		},
		{
			name:     "pom_path only",
			env:      map[string]string{"PLUGIN_POM_PATH": "."},
			wantFile: "pom.xml",
		},
		{
			name:     "pom_path nested",
			env:      map[string]string{"PLUGIN_POM_PATH": "services/api"},
			wantFile: filepath.Join("services", "api", "pom.xml"),
		},
		{
			name:     "pom_file only",
			env:      map[string]string{"PLUGIN_POM_FILE": "build/parent.xml"},
			wantFile: filepath.Join("build", "parent.xml"),
		},
		{
			name:     "both set, pom_file wins",
			env:      map[string]string{"PLUGIN_POM_FILE": "a/pom.xml", "PLUGIN_POM_PATH": "b"},
			wantFile: filepath.Join("a", "pom.xml"),
			wantWarn: true,
		},
		{
			name:     "backslash separators",
			env:      map[string]string{"PLUGIN_POM_FILE": `my app\sub dir\pom.xml`},
			wantFile: filepath.Join("my app", "sub dir", "pom.xml"),
		},
		{
			name:     "backslash pom_path",
			env:      map[string]string{"PLUGIN_POM_PATH": `my app\sub`},
			wantFile: filepath.Join("my app", "sub", "pom.xml"),
		},
		{
			name:     "whitespace-only pom_file falls back to pom_path",
			env:      map[string]string{"PLUGIN_POM_FILE": "  ", "PLUGIN_POM_PATH": "x"},
			wantFile: filepath.Join("x", "pom.xml"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := loadConfig(envFrom(tt.env))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got err %v, want %q", err, tt.wantErr)
				}
				if !strings.Contains(err.Error(), "pom_file") || !strings.Contains(err.Error(), "pom_path") {
					t.Errorf("error should mention both settings: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.PomFile != tt.wantFile {
				t.Errorf("PomFile = %q, want %q", cfg.PomFile, tt.wantFile)
			}
			if got := len(cfg.Warnings) > 0; got != tt.wantWarn {
				t.Errorf("warnings = %q, wantWarn %v", cfg.Warnings, tt.wantWarn)
			}
			if cfg.Mode != modeEffective {
				t.Errorf("default mode = %q", cfg.Mode)
			}
		})
	}
}

func TestLoadConfigAbsolutePath(t *testing.T) {
	abs := filepath.Join(t.TempDir(), "ws", "pom.xml")
	cfg, err := loadConfig(envFrom(map[string]string{"PLUGIN_POM_FILE": abs}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PomFile != abs {
		t.Errorf("PomFile = %q, want %q", cfg.PomFile, abs)
	}

	cfg, err = loadConfig(envFrom(map[string]string{"PLUGIN_POM_FILE": filepath.ToSlash(abs)}))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PomFile != abs {
		t.Errorf("forward-slash PomFile = %q, want %q", cfg.PomFile, abs)
	}
}

func TestLoadConfigMode(t *testing.T) {
	tests := []struct {
		mode    string
		want    string
		wantErr bool
	}{
		{"", modeEffective, false},
		{"effective", modeEffective, false},
		{"raw_gav", modeRawGAV, false},
		{" RAW_GAV ", modeRawGAV, false},
		{"raw", "", true},
		{"gav", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			cfg, err := loadConfig(envFrom(map[string]string{"PLUGIN_POM_PATH": ".", "PLUGIN_MODE": tt.mode}))
			if tt.wantErr {
				if err == nil || !strings.Contains(err.Error(), "invalid mode") {
					t.Fatalf("got %v, want invalid mode error", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Mode != tt.want {
				t.Errorf("Mode = %q, want %q", cfg.Mode, tt.want)
			}
		})
	}
}

func TestLoadConfigPrefix(t *testing.T) {
	base := map[string]string{"PLUGIN_POM_PATH": ".", "PLUGIN_MODE": "raw_gav"}
	with := func(prefix string) map[string]string {
		m := map[string]string{}
		for k, v := range base {
			m[k] = v
		}
		m["PLUGIN_VARIABLE_PREFIX"] = prefix
		return m
	}

	cfg, err := loadConfig(envFrom(base))
	if err != nil || cfg.Prefix != "MAVEN" {
		t.Fatalf("default prefix: %q, %v", cfg.Prefix, err)
	}
	cfg, err = loadConfig(envFrom(with("my-app")))
	if err != nil || cfg.Prefix != "MY_APP" {
		t.Fatalf("my-app: %q, %v", cfg.Prefix, err)
	}
	if _, err := loadConfig(envFrom(with("---"))); err == nil {
		t.Fatal("expected error for a prefix without letters or digits")
	}

	effective := map[string]string{"PLUGIN_POM_PATH": ".", "PLUGIN_VARIABLE_PREFIX": "---"}
	cfg, err = loadConfig(envFrom(effective))
	if err != nil {
		t.Fatalf("prefix must not fail effective mode: %v", err)
	}
	if len(cfg.Warnings) != 1 || !strings.Contains(cfg.Warnings[0], "ignored") {
		t.Errorf("expected an ignored-prefix warning, got %q", cfg.Warnings)
	}
}

func TestNormalizePrefix(t *testing.T) {
	tests := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"maven", "MAVEN", false},
		{"MAVEN", "MAVEN", false},
		{"my-app", "MY_APP", false},
		{"a..b", "A_B", false},
		{"maven.", "MAVEN", false},
		{"_maven_", "MAVEN", false},
		{"  my  app  ", "MY_APP", false},
		{"app2", "APP2", false},
		{"café", "CAF", false},
		{"über-app", "BER_APP", false},
		{"", "", true},
		{"___", "", true},
		{"日本", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := normalizePrefix(tt.in)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
