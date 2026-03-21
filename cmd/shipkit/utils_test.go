package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHelpers(t *testing.T) {
	if v, err := parseTagVersion("v1.2.3"); err != nil || v != "1.2.3" {
		t.Fatalf("unexpected parseTagVersion result: %q %v", v, err)
	}
	if _, err := parseTagVersion("1.2.3"); err == nil {
		t.Fatalf("expected invalid tag error")
	}
	if mm, err := parseMajorMinor("1.2.3"); err != nil || mm != "1.2" {
		t.Fatalf("unexpected parseMajorMinor result: %q %v", mm, err)
	}
	if got := parseCSV("A, B ,,C"); len(got) != 3 {
		t.Fatalf("unexpected parseCSV result: %#v", got)
	}
	if s := buildSummary("rerelease", "", "v1.2.3", "karloie/kompass", "1.2.3", "abcdef1"); !strings.Contains(s, "Re-released tag") {
		t.Fatalf("unexpected rerelease summary: %s", s)
	}
}

func TestParseMajorMinor(t *testing.T) {
	tests := []struct {
		version string
		want    string
		wantErr bool
	}{
		{"1.2.3", "1.2", false},
		{"0.0.1", "", false},
		{"0.1.2", "", false},
		{"10.20.30", "10.20", false},
		{"1.2", "", true},
		{"1", "", true},
		{"1.2.3.4", "", true},
		{"", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got, err := parseMajorMinor(tt.version)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseMajorMinor(%q) error = %v, wantErr %v", tt.version, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseMajorMinor(%q) = %v, want %v", tt.version, got, tt.want)
			}
		})
	}
}

func TestParseRepoFormat(t *testing.T) {
	tests := []struct {
		repo      string
		wantOwner string
		wantName  string
		wantErr   bool
	}{
		{"owner/repo", "owner", "repo", false},
		{"my-org/my-app", "my-org", "my-app", false},
		{"invalid", "", "", true},
		{"/repo", "", "", true},
		{"owner/", "", "", true},
		{"", "", "", true},
		{"owner/repo/extra", "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.repo, func(t *testing.T) {
			owner, name, err := parseRepoFormat(tt.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseRepoFormat(%q) error = %v, wantErr %v", tt.repo, err, tt.wantErr)
				return
			}
			if owner != tt.wantOwner || name != tt.wantName {
				t.Errorf("parseRepoFormat(%q) = (%v, %v), want (%v, %v)", tt.repo, owner, name, tt.wantOwner, tt.wantName)
			}
		})
	}
}

func TestShortenSHA(t *testing.T) {
	tests := []struct {
		sha  string
		want string
	}{
		{"abcdef1234567890", "abcdef1"},
		{"abc", "abc"},
		{"", ""},
		{"  1234567890  ", "1234567"},
		{"short", "short"},
	}

	for _, tt := range tests {
		t.Run(tt.sha, func(t *testing.T) {
			got := shortenSHA(tt.sha)
			if got != tt.want {
				t.Errorf("shortenSHA(%q) = %v, want %v", tt.sha, got, tt.want)
			}
		})
	}
}

func TestParseCSV(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b , c", []string{"a", "b", "c"}},
		{"a,,b", []string{"a", "b"}},
		{"", nil},
		{"  ", nil},
		{"single", []string{"single"}},
		{" a , , b,  c  ", []string{"a", "b", "c"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseCSV(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("parseCSV(%q) length = %v, want %v", tt.input, len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("parseCSV(%q)[%d] = %v, want %v", tt.input, i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestParsePRLabels(t *testing.T) {
	tests := []struct {
		labels string
		want   string
	}{
		{"release:major", BumpMajor},
		{"bug\nrelease:minor\nenhancement", BumpMinor},
		{"release:patch", BumpPatch},
		{"bug\nenhancement", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.labels, func(t *testing.T) {
			got := parsePRLabels(tt.labels)
			if got != tt.want {
				t.Errorf("parsePRLabels(%q) = %v, want %v", tt.labels, got, tt.want)
			}
		})
	}
}

func TestDetectProjectName(t *testing.T) {
	// Save current dir
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Create temp dir
	tmpDir := t.TempDir()
	os.Chdir(tmpDir)

	// No go.mod
	if name := detectProjectName(); name != "" {
		t.Errorf("detectProjectName() without go.mod = %q, want empty", name)
	}

	// With go.mod
	goMod := `module github.com/owner/test-project

go 1.21
`
	if err := os.WriteFile("go.mod", []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	name := detectProjectName()
	if name != "test-project" {
		t.Errorf("detectProjectName() = %q, want test-project", name)
	}

	// Single name module
	goMod2 := `module myproject

go 1.21
`
	if err := os.WriteFile("go.mod", []byte(goMod2), 0644); err != nil {
		t.Fatal(err)
	}

	name = detectProjectName()
	if name != "myproject" {
		t.Errorf("detectProjectName() single = %q, want myproject", name)
	}
}

func TestWriteOutput(t *testing.T) {
	tmpDir := t.TempDir()
	outputFile := filepath.Join(tmpDir, "output.txt")

	// Simple value
	writeOutput(outputFile, "key1", "value1")
	content, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "key1=value1") {
		t.Errorf("writeOutput simple: got %q, want key1=value1", content)
	}

	// Multiline value
	writeOutput(outputFile, "key2", "line1\nline2\nline3")
	content, err = os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "key2<<EOF") {
		t.Errorf("writeOutput multiline: missing heredoc marker")
	}
	if !strings.Contains(string(content), "line1\nline2\nline3") {
		t.Errorf("writeOutput multiline: missing content")
	}

	// Empty outputFile (stdout)
	writeOutput("", "key3", "value3")
}

func TestNewFlagSet(t *testing.T) {
	fs := newFlagSet("test-command")
	if fs == nil {
		t.Fatal("newFlagSet returned nil")
	}
	if fs.Name() != "test-command" {
		t.Errorf("newFlagSet().Name() = %q, want test-command", fs.Name())
	}
}
