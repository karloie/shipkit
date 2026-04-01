package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunGoBuildWithoutPlanFile(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	goMod := "module example.com/testbuild\n\ngo 1.25.0\n"
	if err := os.WriteFile("go.mod", []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	mainSrc := `package main

var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)

func main() {}
`
	if err := os.WriteFile("main.go", []byte(mainSrc), 0644); err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(tmpDir, "testbin")
	if err := runGoBuild([]string{"--output", output, "--main", "."}); err != nil {
		t.Fatalf("runGoBuild returned error without plan file: %v", err)
	}

	if _, err := os.Stat(output); err != nil {
		t.Fatalf("expected built binary at %s: %v", output, err)
	}
}
