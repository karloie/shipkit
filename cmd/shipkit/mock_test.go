package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// execRunnerFailAt fails on the Nth call (1-based).
type execRunnerFailAt struct {
	callCount int
	failAt    int
}

func (r *execRunnerFailAt) Run(name string, args ...string) error {
	r.callCount++
	if r.callCount >= r.failAt {
		return os.ErrPermission
	}
	return nil
}

func (r *execRunnerFailAt) RunWithStdin(stdin, name string, args ...string) error {
	return r.Run(name, args...)
}

// ── globExists ───────────────────────────────────────────────────────────────

func TestGlobExists(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(dir)

	if globExists("*.go") {
		t.Error("expected no match before file creation")
	}
	os.WriteFile("main.go", []byte("package main"), 0644)
	if !globExists("*.go") {
		t.Error("expected match after file creation")
	}
}

// ── runDockerHubStatus ───────────────────────────────────────────────────────

func TestRunDockerHubStatusNoDocker(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(dir)

	outFile := filepath.Join(dir, "github_output")
	t.Setenv("GITHUB_OUTPUT", outFile)

	if err := runDockerHubStatus([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, _ := os.ReadFile(outFile)
	if !strings.Contains(string(b), "should_build_docker_goreleaser=false") {
		t.Errorf("expected should_build_docker_goreleaser=false, got: %s", string(b))
	}
}

func TestRunDockerHubStatusWithContainerfile(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(dir)

	os.WriteFile("Containerfile", []byte("FROM scratch"), 0644)
	outFile := filepath.Join(dir, "github_output")
	t.Setenv("GITHUB_OUTPUT", outFile)

	if err := runDockerHubStatus([]string{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, _ := os.ReadFile(outFile)
	if !strings.Contains(string(b), "should_build_docker_goreleaser=true") {
		t.Errorf("expected should_build_docker_goreleaser=true, got: %s", string(b))
	}
}

// ── Docker Hub HTTP API ──────────────────────────────────────────────────────

func newDockerHubTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/users/login/"):
			json.NewEncoder(w).Encode(dockerHubLoginResponse{Token: "test-jwt-token"})
		case r.Method == http.MethodPatch && strings.Contains(r.URL.Path, "/repositories/"):
			w.WriteHeader(http.StatusOK)
		default:
			http.NotFound(w, r)
		}
	}))
	origURL := DockerHubAPIURL
	DockerHubAPIURL = srv.URL
	t.Cleanup(func() {
		DockerHubAPIURL = origURL
		srv.Close()
	})
	return srv
}

func TestDockerHubLoginSuccess(t *testing.T) {
	newDockerHubTestServer(t)
	token, err := dockerHubLogin("user", "pass")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "test-jwt-token" {
		t.Errorf("expected test-jwt-token, got %q", token)
	}
}

func TestDockerHubLoginFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	origURL := DockerHubAPIURL
	DockerHubAPIURL = srv.URL
	t.Cleanup(func() { DockerHubAPIURL = origURL; srv.Close() })

	_, err := dockerHubLogin("user", "wrongpass")
	if err == nil {
		t.Fatal("expected error for 401 response")
	}
}

func TestDockerHubUpdateReadmeSuccess(t *testing.T) {
	newDockerHubTestServer(t)
	if err := dockerHubUpdateReadme("test-jwt-token", "owner", "repo", "# README"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDockerHubUpdateReadmeFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	origURL := DockerHubAPIURL
	DockerHubAPIURL = srv.URL
	t.Cleanup(func() { DockerHubAPIURL = origURL; srv.Close() })

	err := dockerHubUpdateReadme("bad-token", "owner", "repo", "# README")
	if err == nil {
		t.Fatal("expected error for 403 response")
	}
}

func TestRunDockerHubReadme(t *testing.T) {
	newDockerHubTestServer(t)

	dir := t.TempDir()
	readmePath := filepath.Join(dir, "README.md")
	os.WriteFile(readmePath, []byte("# My Project\n"), 0644)

	err := runDockerHubReadme([]string{
		"-repo=owner/myrepo",
		"-username=user",
		"-password=pass",
		"-readme=" + readmePath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunDockerHubReadmeMissingRepo(t *testing.T) {
	err := runDockerHubReadme([]string{"-username=user", "-password=pass"})
	if err == nil || !strings.Contains(err.Error(), "repo is required") {
		t.Errorf("expected repo error, got: %v", err)
	}
}

func TestRunDockerHubReadmeMissingPassword(t *testing.T) {
	dir := t.TempDir()
	readmePath := filepath.Join(dir, "README.md")
	os.WriteFile(readmePath, []byte("# README"), 0644)
	t.Setenv("DOCKERHUB_TOKEN", "")
	t.Setenv("DOCKERHUB_PASSWORD", "")

	err := runDockerHubReadme([]string{
		"-repo=owner/repo",
		"-username=user",
		"-readme=" + readmePath,
	})
	if err == nil || !strings.Contains(err.Error(), "password is required") {
		t.Errorf("expected password error, got: %v", err)
	}
}

// ── runGoReleaser ────────────────────────────────────────────────────────────

func TestRunGoReleaserGeneratesConfig(t *testing.T) {
	dir := t.TempDir()
	old, _ := os.Getwd()
	defer os.Chdir(old)
	os.Chdir(dir)

	os.WriteFile("go.mod", []byte("module github.com/acme/testapp\n\ngo 1.22\n"), 0644)
	outPath := filepath.Join(dir, "goreleaser-test.yml")

	err := runGoReleaser([]string{
		"-owner=acme",
		"-output=" + outPath,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	b, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("output file not created: %v", err)
	}
	if !strings.Contains(string(b), "testapp") {
		t.Errorf("expected project name in config, got: %s", string(b))
	}
}

func TestRunGoReleaserMissingOwner(t *testing.T) {
	err := runGoReleaser([]string{})
	if err == nil || !strings.Contains(err.Error(), "owner is required") {
		t.Errorf("expected owner error, got: %v", err)
	}
}

// ── runGitConfig ─────────────────────────────────────────────────────────────

func TestRunGitConfig(t *testing.T) {
	mock := &ExecRunnerMock{}
	old := defaultRunner
	defaultRunner = mock
	defer func() { defaultRunner = old }()

	err := runGitConfig([]string{"-user-name=Bot", "-user-email=bot@test.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.Calls) != 2 {
		t.Fatalf("expected 2 git config calls, got %d", len(mock.Calls))
	}
}

func TestRunGitConfigError(t *testing.T) {
	mock := &ExecRunnerMock{Err: os.ErrPermission}
	old := defaultRunner
	defaultRunner = mock
	defer func() { defaultRunner = old }()

	if err := runGitConfig([]string{}); err == nil {
		t.Fatal("expected error from failing git config")
	}
}

func TestCreateGitTagSuccess(t *testing.T) {
	mock := &ExecRunnerMock{}
	old := defaultRunner
	defaultRunner = mock
	defer func() { defaultRunner = old }()

	if err := createGitTag("v1.2.3"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// git config x2, git fetch --tags, git tag, git push = 5 calls
	if len(mock.Calls) != 5 {
		t.Fatalf("expected 5 calls, got %d: %v", len(mock.Calls), mock.Calls)
	}
}

func TestCreateGitTagFailsOnTag(t *testing.T) {
	callCount := 0
	mock := &ExecRunnerMock{}
	old := defaultRunner
	defaultRunner = mock
	defer func() { defaultRunner = old }()

	// Fail on the 4th call (git tag, after fetch)
	origErr := mock.Err
	_ = origErr
	mock2 := &execRunnerFailAt{failAt: 4}
	defaultRunner = mock2
	defer func() { defaultRunner = old }()
	_ = callCount

	err := createGitTag("v1.2.3")
	if err == nil {
		t.Fatal("expected error when git tag fails")
	}
}

func TestRunGitTagCleanup(t *testing.T) {
	mock := &ExecRunnerMock{}
	old := defaultRunner
	defaultRunner = mock
	defer func() { defaultRunner = old }()

	if err := runGitTagCleanup([]string{"-tag=v1.0.0"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// git push --delete + git tag -d = 2 calls
	if len(mock.Calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(mock.Calls))
	}
}

func TestRunGitTagCleanupFailures(t *testing.T) {
	// Both calls fail — should warn but not return error
	mock := &ExecRunnerMock{Err: os.ErrNotExist}
	old := defaultRunner
	defaultRunner = mock
	defer func() { defaultRunner = old }()

	if err := runGitTagCleanup([]string{"-tag=v1.0.0"}); err != nil {
		t.Fatalf("cleanup should not return error on warn-only failures: %v", err)
	}
}

func TestRunGitTagMissingFlag(t *testing.T) {
	if err := runGitTag([]string{}); err == nil {
		t.Fatal("expected error for missing -tag flag")
	}
}

func TestDockerLoginMocked(t *testing.T) {
	mock := &ExecRunnerMock{}
	old := defaultRunner
	defaultRunner = mock
	defer func() { defaultRunner = old }()

	if err := dockerLogin("user", "token"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(mock.Calls) != 1 || mock.Calls[0][0] != "docker" {
		t.Errorf("expected one docker call, got: %v", mock.Calls)
	}
}

func TestDockerLoginMockedError(t *testing.T) {
	mock := &ExecRunnerMock{Err: os.ErrPermission}
	old := defaultRunner
	defaultRunner = mock
	defer func() { defaultRunner = old }()

	if err := dockerLogin("user", "token"); err == nil {
		t.Fatal("expected error from failed docker login")
	}
}
