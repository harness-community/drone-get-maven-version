package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// writeTree writes files (slash-separated paths relative to a temp dir) and
// returns the temp dir.
func writeTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

const ciFriendlyParent = `<groupId>com.example</groupId><artifactId>parent</artifactId><version>${revision}</version>
<packaging>pom</packaging>
<properties><revision>1.4.0</revision></properties>`

const ciFriendlyChild = `<parent><groupId>com.example</groupId><artifactId>parent</artifactId><version>${revision}</version></parent>
<artifactId>child</artifactId>`

func TestParseMavenProperties(t *testing.T) {
	tests := []struct {
		raw     string
		want    map[string]string
		wantErr string
	}{
		{"", map[string]string{}, ""},
		{"revision=1.2.3", map[string]string{"revision": "1.2.3"}, ""},
		{"-Drevision=1.2.3, changelist=", map[string]string{"revision": "1.2.3", "changelist": ""}, ""},
		{"revision=1.2.3\r\n-Dsha1=-abc\n\n", map[string]string{"revision": "1.2.3", "sha1": "-abc"}, ""},
		{"skipTests", map[string]string{"skipTests": "true"}, ""},
		{"a=b=c", map[string]string{"a": "b=c"}, ""},
		{"=1", nil, "property name is empty"},
		{"my prop=1", nil, "whitespace"},
	}
	for _, tt := range tests {
		got, err := parseMavenProperties(tt.raw)
		if tt.wantErr != "" {
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) || !strings.Contains(err.Error(), "maven_properties") {
				t.Errorf("%q: got %v, want error containing %q", tt.raw, err, tt.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("%q: %v", tt.raw, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q: got %v, want %v", tt.raw, got, tt.want)
		}
	}
}

func TestParseMavenConfig(t *testing.T) {
	content := "\ufeff-T 4 --batch-mode\r\n-Drevision=2.0.0 -D changelist=-SNAPSHOT\n" +
		"--define=sha1=abc --define \"quoted='x y'\"\n-Dflag\n-Dq=\"1.0\"\n"
	got, err := parseMavenConfig(content)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{"revision": "2.0.0", "changelist": "-SNAPSHOT", "sha1": "abc", "quoted": "x y", "flag": "true", "q": "1.0"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if _, err := parseMavenConfig("-Drevision=1 -D"); err == nil {
		t.Error("expected an error for a trailing -D")
	}
}

func TestFindMavenConfig(t *testing.T) {
	root := writeTree(t, map[string]string{
		".mvn/maven.config":      "-Drevision=1",
		"module/sub/pom.xml":     "",
		"other/.mvn/jvm.config":  "",
		"other/module/pom.xml":   "",
		"module/sub/.keep/x.txt": "",
	})
	got, err := findMavenConfig(filepath.Join(root, "module", "sub", "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(root, ".mvn", "maven.config"); got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// The nearest .mvn directory wins even without a maven.config, like Maven.
	got, err = findMavenConfig(filepath.Join(root, "other", "module", "pom.xml"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Errorf("got %q, want none", got)
	}
}

func TestLoadUserPropertiesPrecedence(t *testing.T) {
	root := writeTree(t, map[string]string{
		".mvn/maven.config": "-Drevision=2.0.0 -Dchangelist=-SNAPSHOT",
		"pom.xml":           "",
	})
	got, err := loadUserProperties(filepath.Join(root, "pom.xml"), map[string]string{"changelist": ""}, discard)
	if err != nil {
		t.Fatal(err)
	}
	if want := map[string]string{"revision": "2.0.0", "changelist": ""}; !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestReadRawGAVParentFromDisk(t *testing.T) {
	tests := []struct {
		name      string
		files     map[string]string
		pom       string
		userProps map[string]string
		want      GAV
		wantErr   string
	}{
		{
			name:  "revision defined in parent on disk",
			files: map[string]string{"pom.xml": project(ciFriendlyParent), "child/pom.xml": project(ciFriendlyChild)},
			pom:   "child/pom.xml",
			want:  GAV{"com.example", "child", "1.4.0"},
		},
		{
			name: "child properties win over parent",
			files: map[string]string{
				"pom.xml":       project(ciFriendlyParent),
				"child/pom.xml": project(ciFriendlyChild + `<properties><revision>1.5.0</revision></properties>`),
			},
			pom:  "child/pom.xml",
			want: GAV{"com.example", "child", "1.5.0"},
		},
		{
			name: "-D wins over every POM",
			files: map[string]string{
				"pom.xml":       project(ciFriendlyParent),
				"child/pom.xml": project(ciFriendlyChild + `<properties><revision>1.5.0</revision></properties>`),
			},
			pom:       "child/pom.xml",
			userProps: map[string]string{"revision": "9.9.9"},
			want:      GAV{"com.example", "child", "9.9.9"},
		},
		{
			name: "grandparent properties",
			files: map[string]string{
				"pom.xml": project(`<groupId>com.example</groupId><artifactId>root</artifactId><version>1</version>
<properties><revision>3.0.0</revision><changelist>-SNAPSHOT</changelist></properties>`),
				"parent/pom.xml": project(`<parent><groupId>com.example</groupId><artifactId>root</artifactId><version>1</version></parent>
<artifactId>parent</artifactId><version>${revision}${changelist}</version>`),
				"parent/child/pom.xml": project(`<parent><groupId>com.example</groupId><artifactId>parent</artifactId><version>${revision}${changelist}</version></parent>
<artifactId>child</artifactId>`),
			},
			pom:  "parent/child/pom.xml",
			want: GAV{"com.example", "child", "3.0.0-SNAPSHOT"},
		},
		{
			name: "custom relativePath to a directory",
			files: map[string]string{
				"build/parent/pom.xml": project(ciFriendlyParent),
				"app/pom.xml": project(`<parent><groupId>com.example</groupId><artifactId>parent</artifactId><version>${revision}</version>
<relativePath>../build/parent</relativePath></parent><artifactId>child</artifactId>`),
			},
			pom:  "app/pom.xml",
			want: GAV{"com.example", "child", "1.4.0"},
		},
		{
			name: "custom relativePath with backslashes",
			files: map[string]string{
				"build/parent-pom.xml": project(ciFriendlyParent),
				"app/pom.xml": project(`<parent><groupId>com.example</groupId><artifactId>parent</artifactId><version>${revision}</version>
<relativePath>..\build\parent-pom.xml</relativePath></parent><artifactId>child</artifactId>`),
			},
			pom:  "app/pom.xml",
			want: GAV{"com.example", "child", "1.4.0"},
		},
		{
			name: "empty relativePath skips the disk",
			files: map[string]string{
				"pom.xml": project(ciFriendlyParent),
				"child/pom.xml": project(`<parent><groupId>com.example</groupId><artifactId>parent</artifactId><version>${revision}</version>
<relativePath/></parent><artifactId>child</artifactId>`),
			},
			pom:     "child/pom.xml",
			wantErr: "empty <relativePath>",
		},
		{
			name: "different artifactId on disk is not used",
			files: map[string]string{
				"pom.xml":       project(strings.Replace(ciFriendlyParent, "<artifactId>parent</artifactId>", "<artifactId>aggregator</artifactId>", 1)),
				"child/pom.xml": project(ciFriendlyChild),
			},
			pom:     "child/pom.xml",
			wantErr: `artifactId "aggregator", expected "parent"`,
		},
		{
			name: "different literal version on disk is not used",
			files: map[string]string{
				"pom.xml": project(`<groupId>com.example</groupId><artifactId>parent</artifactId><version>1.0</version>
<properties><svc>1</svc></properties>`),
				"child/pom.xml": project(`<parent><groupId>com.example</groupId><artifactId>parent</artifactId><version>2.0</version></parent>
<artifactId>child</artifactId><version>${svc}</version>`),
			},
			pom:     "child/pom.xml",
			wantErr: `version "1.0", expected "2.0"`,
		},
		{
			name:    "parent not on disk",
			files:   map[string]string{"child/pom.xml": project(ciFriendlyChild)},
			pom:     "child/pom.xml",
			wantErr: "not found on disk",
		},
		{
			name: "parent loop",
			files: map[string]string{
				"a/pom.xml": project(`<parent><groupId>g</groupId><artifactId>b</artifactId><version>1</version><relativePath>../b/pom.xml</relativePath></parent>
<groupId>g</groupId><artifactId>a</artifactId><version>1</version>`),
				"b/pom.xml": project(`<parent><groupId>g</groupId><artifactId>a</artifactId><version>1</version><relativePath>../a/pom.xml</relativePath></parent>
<groupId>g</groupId><artifactId>b</artifactId><version>1</version>`),
				"a/child/pom.xml": project(`<parent><groupId>g</groupId><artifactId>a</artifactId><version>1</version></parent>
<artifactId>c</artifactId><version>${missing}</version>`),
			},
			pom:     "a/child/pom.xml",
			wantErr: "parent loop",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := writeTree(t, tt.files)
			got, err := readRawGAV(filepath.Join(root, filepath.FromSlash(tt.pom)), tt.userProps, discard)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got %+v, %v; want error containing %q", got, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestReadRawGAVParentMalformedIsReported(t *testing.T) {
	root := writeTree(t, map[string]string{"pom.xml": "<project>", "child/pom.xml": project(ciFriendlyChild)})
	_, err := readRawGAV(filepath.Join(root, "child", "pom.xml"), nil, discard)
	if err == nil || !strings.Contains(err.Error(), "parent POM not read") {
		t.Fatalf("got %v", err)
	}
}

func TestRunRawGAVCIFriendly(t *testing.T) {
	root := writeTree(t, map[string]string{
		".mvn/maven.config": "-Drevision=5.6.7 -Dchangelist=-SNAPSHOT",
		"pom.xml": project(`<groupId>com.example</groupId><artifactId>parent</artifactId><version>${revision}${changelist}</version>
<properties><revision>0.0.0</revision><changelist></changelist></properties>`),
		"service/pom.xml": project(`<parent><groupId>com.example</groupId><artifactId>parent</artifactId><version>${revision}${changelist}</version></parent>
<artifactId>service</artifactId>`),
	})
	out := filepath.Join(t.TempDir(), "output.env")
	fakeMaven(t, "", "", errors.New("maven must not run in raw_gav mode"))

	var stdout, stderr bytes.Buffer
	code := run(envFrom(map[string]string{
		"PLUGIN_POM_FILE":         filepath.Join(root, "service", "pom.xml"),
		"PLUGIN_MODE":             "raw_gav",
		"PLUGIN_MAVEN_PROPERTIES": "changelist=-RC1",
		"DRONE_OUTPUT":            out,
	}), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr %s", code, stderr.String())
	}
	want := "MAVEN_GROUP_ID=com.example\nMAVEN_ARTIFACT_ID=service\nMAVEN_VERSION=5.6.7-RC1\nPOM_VERSION=5.6.7-RC1\n"
	if got := readFile(t, out); got != want {
		t.Errorf("DRONE_OUTPUT = %q, want %q", got, want)
	}
	for _, s := range []string{"maven.config: changelist, revision", "maven_properties: changelist", "properties read from parent POM"} {
		if !strings.Contains(stdout.String(), s) {
			t.Errorf("stdout missing %q:\n%s", s, stdout.String())
		}
	}
}

func TestRunInvalidMavenProperties(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run(envFrom(map[string]string{"PLUGIN_POM_PATH": ".", "PLUGIN_MAVEN_PROPERTIES": "=x"}), &stdout, &stderr)
	if code != 1 || !strings.Contains(stderr.String(), "maven_properties") {
		t.Fatalf("exit %d, stderr %q", code, stderr.String())
	}
}

func TestRunEffectivePassesMavenProperties(t *testing.T) {
	out := filepath.Join(t.TempDir(), "output.env")
	args := fakeMaven(t, "5.6.7", "", nil)

	var stdout, stderr bytes.Buffer
	code := run(envFrom(map[string]string{
		"PLUGIN_POM_PATH":         ".",
		"PLUGIN_MAVEN_PROPERTIES": "-Drevision=5.6.7,changelist=",
		"DRONE_OUTPUT":            out,
	}), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d, stderr %s", code, stderr.String())
	}
	want := "-f pom.xml help:evaluate -Dexpression=project.version -q -DforceStdout -Dchangelist= -Drevision=5.6.7"
	if got := strings.Join(*args, " "); got != want {
		t.Errorf("args %q, want %q", got, want)
	}
}
