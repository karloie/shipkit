package main

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"testing"
)

func TestVerifyVersionNpm(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create package.json with version 1.2.3
	pkg := map[string]interface{}{
		"name":    "test-package",
		"version": "1.2.3",
	}
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("package.json", data, 0644); err != nil {
		t.Fatal(err)
	}

	// Test matching version with -version flag
	err = runVerifyVersion([]string{"-type=npm", "-version=1.2.3"})
	if err != nil {
		t.Errorf("expected no error for matching version, got: %v", err)
	}

	// Test matching version with -tag flag (should strip 'v')
	err = runVerifyVersion([]string{"-type=npm", "-tag=v1.2.3"})
	if err != nil {
		t.Errorf("expected no error for matching version with tag, got: %v", err)
	}

	// Test mismatching version
	err = runVerifyVersion([]string{"-type=npm", "-version=1.2.4"})
	if err == nil {
		t.Error("expected error for mismatching version, got nil")
	}
}

func TestVerifyVersionMaven(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create minimal pom.xml with version 1.2.3
	type Project struct {
		XMLName xml.Name `xml:"project"`
		Version string   `xml:"version"`
	}
	pom := Project{Version: "1.2.3"}
	data, err := xml.MarshalIndent(pom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	pomContent := xml.Header + string(data)
	if err := os.WriteFile("pom.xml", []byte(pomContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Test matching version with -version flag
	err = runVerifyVersion([]string{"-type=maven", "-version=1.2.3"})
	if err != nil {
		t.Errorf("expected no error for matching version, got: %v", err)
	}

	// Test matching version with -tag flag (should strip 'v')
	err = runVerifyVersion([]string{"-type=maven", "-tag=v1.2.3"})
	if err != nil {
		t.Errorf("expected no error for matching version with tag, got: %v", err)
	}

	// Test mismatching version
	err = runVerifyVersion([]string{"-type=maven", "-version=1.2.4"})
	if err == nil {
		t.Error("expected error for mismatching version, got nil")
	}
}

func TestGetNpmVersion(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create package.json
	pkg := map[string]interface{}{
		"name":    "test-package",
		"version": "2.3.4",
	}
	data, err := json.MarshalIndent(pkg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile("package.json", data, 0644); err != nil {
		t.Fatal(err)
	}

	version, err := getNpmVersion()
	if err != nil {
		t.Fatalf("getNpmVersion failed: %v", err)
	}

	if version != "2.3.4" {
		t.Errorf("expected version 2.3.4, got %s", version)
	}
}

func TestGetMavenVersion(t *testing.T) {
	// Create temp dir
	tmpDir := t.TempDir()
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldDir)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Create pom.xml
	type Project struct {
		XMLName xml.Name `xml:"project"`
		Version string   `xml:"version"`
	}
	pom := Project{Version: "3.4.5"}
	data, err := xml.MarshalIndent(pom, "", "  ")
	if err != nil {
		t.Fatal(err)
	}

	pomContent := xml.Header + string(data)
	if err := os.WriteFile("pom.xml", []byte(pomContent), 0644); err != nil {
		t.Fatal(err)
	}

	version, err := getMavenVersion()
	if err != nil {
		t.Fatalf("getMavenVersion failed: %v", err)
	}

	if version != "3.4.5" {
		t.Errorf("expected version 3.4.5, got %s", version)
	}
}
