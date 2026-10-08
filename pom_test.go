package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writePOM(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pom.xml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func project(body string) string {
	return `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0">
  <modelVersion>4.0.0</modelVersion>
` + body + `
</project>
`
}

func discard(string, ...any) {}

func TestReadRawGAV(t *testing.T) {
	tests := []struct {
		name    string
		pom     string
		want    GAV
		wantErr string
		wantLog []string
	}{
		{
			name: "all fields present",
			pom:  project(`<groupId>g</groupId><artifactId>a</artifactId><version>1.0</version>`),
			want: GAV{"g", "a", "1.0"},
		},
		{
			name:    "groupId inherited",
			pom:     project(`<parent><groupId>pg</groupId><artifactId>pa</artifactId><version>9</version></parent><artifactId>a</artifactId><version>1.0</version>`),
			want:    GAV{"pg", "a", "1.0"},
			wantLog: []string{"groupId taken from parent"},
		},
		{
			name:    "version inherited",
			pom:     project(`<parent><groupId>pg</groupId><artifactId>pa</artifactId><version>9</version></parent><groupId>g</groupId><artifactId>a</artifactId>`),
			want:    GAV{"g", "a", "9"},
			wantLog: []string{"version taken from parent"},
		},
		{
			name:    "groupId and version inherited",
			pom:     project(`<parent><groupId>pg</groupId><artifactId>pa</artifactId><version>9</version></parent><artifactId>a</artifactId>`),
			want:    GAV{"pg", "a", "9"},
			wantLog: []string{"groupId taken from parent", "version taken from parent"},
		},
		{
			name:    "artifactId is never inherited",
			pom:     project(`<parent><groupId>pg</groupId><artifactId>pa</artifactId><version>9</version></parent>`),
			wantErr: "artifactId is missing",
		},
		{
			name:    "groupId missing without parent",
			pom:     project(`<artifactId>a</artifactId><version>1</version>`),
			wantErr: "groupId is missing",
		},
		{
			name:    "version missing without parent",
			pom:     project(`<groupId>g</groupId><artifactId>a</artifactId>`),
			wantErr: "version is missing",
		},
		{
			name: "property placeholder resolved",
			pom:  project(`<groupId>g</groupId><artifactId>a</artifactId><version>${app.version}</version><properties><app.version>4.5.6</app.version></properties>`),
			want: GAV{"g", "a", "4.5.6"},
		},
		{
			name: "project and parent aliases",
			pom: project(`<parent><groupId>pg</groupId><artifactId>pa</artifactId><version>7</version></parent>
				<groupId>${parent.groupId}</groupId><artifactId>${project.groupId}-x</artifactId><version>${project.parent.version}.1</version>`),
			want: GAV{"pg", "pg-x", "7.1"},
		},
		{
			name:    "unresolved revision",
			pom:     project(`<groupId>g</groupId><artifactId>a</artifactId><version>${revision}</version>`),
			wantErr: "${revision} is not defined",
		},
		{
			name:    "placeholder loop",
			pom:     project(`<groupId>g</groupId><artifactId>a</artifactId><version>${a}</version><properties><a>${b}</a><b>${a}</b></properties>`),
			wantErr: "property loop: a -> b -> a",
		},
		{
			name:    "self reference through project.version",
			pom:     project(`<groupId>g</groupId><artifactId>a</artifactId><version>${project.version}</version>`),
			wantErr: "property loop",
		},
		{
			name:    "nesting too deep",
			pom:     project(`<groupId>g</groupId><artifactId>a</artifactId><version>${p1}</version><properties><p1>${p2}</p1><p2>${p3}</p2><p3>${p4}</p3><p4>${p5}</p4><p5>${p6}</p5><p6>x</p6></properties>`),
			wantErr: "nested more than",
		},
		{
			name:    "property resolving to empty",
			pom:     project(`<groupId>g</groupId><artifactId>a</artifactId><version>${v}</version><properties><v></v></properties>`),
			wantErr: "version is empty",
		},
		{
			name: "no namespace",
			pom:  `<project><groupId>g</groupId><artifactId>a</artifactId><version>1</version></project>`,
			want: GAV{"g", "a", "1"},
		},
		{
			name: "comments and whitespace padding",
			pom: project(`<!-- <version>0.0.0</version> -->
  <groupId>
     g
  </groupId>
  <artifactId> a <!-- inline --> </artifactId>
  <version>	1.0	</version>`),
			want: GAV{"g", "a", "1.0"},
		},
		{
			name: "CRLF line endings",
			pom:  strings.ReplaceAll(project("<groupId>g</groupId>\n<artifactId>a</artifactId>\n<version>1</version>"), "\n", "\r\n"),
			want: GAV{"g", "a", "1"},
		},
		{
			name: "UTF-8 BOM",
			pom:  "\xef\xbb\xbf" + project(`<groupId>g</groupId><artifactId>a</artifactId><version>1</version>`),
			want: GAV{"g", "a", "1"},
		},
		{
			name: "ISO-8859-1 declaration",
			pom:  "<?xml version=\"1.0\" encoding=\"ISO-8859-1\"?>\n<project><groupId>g</groupId><artifactId>caf\xe9</artifactId><version>1</version></project>",
			want: GAV{"g", "café", "1"},
		},
		{
			name:    "unsupported encoding",
			pom:     "<?xml version=\"1.0\" encoding=\"EBCDIC\"?>\n<project/>",
			wantErr: "unsupported XML encoding",
		},
		{
			name:    "malformed XML",
			pom:     project(`<groupId>g</groupId><artifactId>a</artifactId><version>1</versio>`),
			wantErr: "is not valid XML",
		},
		{
			name:    "root is not project",
			pom:     `<settings><groupId>g</groupId></settings>`,
			wantErr: "is not valid XML",
		},
		{
			name:    "empty file",
			pom:     "",
			wantErr: "is empty",
		},
		{
			name:    "whitespace only",
			pom:     " \r\n\t",
			wantErr: "is empty",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writePOM(t, tt.pom)
			var logs []string
			got, err := readRawGAV(path, nil, func(format string, args ...any) {
				logs = append(logs, format)
			})
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got %+v", tt.wantErr, got)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err, tt.wantErr)
				}
				if !strings.Contains(err.Error(), path) {
					t.Errorf("error %q does not name the file %s", err, path)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
			if strings.Join(logs, "|") != strings.Join(tt.wantLog, "|") {
				t.Errorf("logs %q, want %q", logs, tt.wantLog)
			}
		})
	}
}

func TestReadRawGAVUnresolvedSuggestsEffective(t *testing.T) {
	path := writePOM(t, project(`<groupId>g</groupId><artifactId>a</artifactId><version>${revision}</version>`))
	_, err := readRawGAV(path, nil, discard)
	if err == nil || !strings.Contains(err.Error(), "mode: effective") {
		t.Fatalf("expected a hint to use mode: effective, got %v", err)
	}
}

func TestReadRawGAVFixtures(t *testing.T) {
	tests := []struct {
		file string
		want GAV
	}{
		{"full.xml", GAV{"com.example", "demo-app", "2.4.1"}},
		{"leak.xml", GAV{"com.example.parent", "leak-guard", "3.0.0"}},
		{"properties.xml", GAV{"com.example.services", "billing-api", "1.7.3-SNAPSHOT"}},
		{filepath.Join("inheritance", "child", "pom.xml"), GAV{"com.example.inherit", "inherit-child", "1.2.3-SNAPSHOT"}},
		// readRawGAV does not read .mvn/maven.config; loadUserProperties does.
		{filepath.Join("cifriendly", "pom.xml"), GAV{"com.example.cifriendly", "cifriendly-parent", "0.0.0"}},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			got, err := readRawGAV(filepath.Join("testdata", tt.file), nil, discard)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestReadRawGAVFileErrors(t *testing.T) {
	dir := t.TempDir()

	_, err := readRawGAV(filepath.Join(dir, "missing.xml"), nil, discard)
	if err == nil || !strings.Contains(err.Error(), "missing.xml") {
		t.Errorf("missing file: got %v", err)
	}

	_, err = readRawGAV(dir, nil, discard)
	if err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Errorf("directory: got %v", err)
	}
}

func TestReadRawGAVPathWithSpaces(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "my workspace", "sub dir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "my pom.xml")
	if err := os.WriteFile(path, []byte(project(`<groupId>g</groupId><artifactId>a</artifactId><version>1</version>`)), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readRawGAV(path, nil, discard)
	if err != nil {
		t.Fatal(err)
	}
	if got != (GAV{"g", "a", "1"}) {
		t.Errorf("got %+v", got)
	}
}

func TestApplyParentFallback(t *testing.T) {
	parent := GAV{"pg", "pa", "9"}
	tests := []struct {
		name string
		own  GAV
		want GAV
	}{
		{"nothing inherited", GAV{"g", "a", "1"}, GAV{"g", "a", "1"}},
		{"groupId inherited", GAV{"", "a", "1"}, GAV{"pg", "a", "1"}},
		{"version inherited", GAV{"g", "a", ""}, GAV{"g", "a", "9"}},
		{"artifactId not inherited", GAV{"g", "", "1"}, GAV{"g", "", "1"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := applyParentFallback(tt.own, parent, discard); got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
	if got := applyParentFallback(GAV{"", "a", ""}, GAV{}, discard); got != (GAV{"", "a", ""}) {
		t.Errorf("empty parent: got %+v", got)
	}
}
