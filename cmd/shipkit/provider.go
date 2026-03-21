package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// --------------------------------------------------------------------------------

type EnvProvider interface {
	Getenv(key string) string
}

type EnvProviderReal struct{}

func (r *EnvProviderReal) Getenv(key string) string {
	return os.Getenv(key)
}

// --------------------------------------------------------------------------------

type EnvProviderMock struct {
	values map[string]string
}

func (m *EnvProviderMock) Getenv(key string) string {
	if m.values == nil {
		return ""
	}
	return m.values[key]
}

// --------------------------------------------------------------------------------

type GitProvider interface {
	GetLatestTag() (string, error)
	GetCommitLog(since string) (string, error)
	TagExists(tag string) (bool, error)
}

type GitProviderReal struct{}

func (g *GitProviderReal) GetLatestTag() (string, error) {
	out, err := exec.Command("git", "describe", "--tags", "--abbrev=0").CombinedOutput()
	if err != nil {
		return "", err
	}
	tag := strings.TrimSpace(string(out))
	if tag == "" {
		return "", fmt.Errorf("no tags found in repository")
	}
	return tag, nil
}

func (g *GitProviderReal) GetCommitLog(since string) (string, error) {
	var out []byte
	var err error

	if since == "v0.0.4" {
		out, err = exec.Command("git", "log", "-n", "10", "--oneline").CombinedOutput()
	} else {
		out, err = exec.Command("git", "log", since+"..HEAD", "--oneline").CombinedOutput()
	}
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func (g *GitProviderReal) TagExists(tag string) (bool, error) {
	err := exec.Command("git", "rev-parse", tag).Run()
	return err == nil, nil
}

type GitProviderMock struct {
	LatestTag  string
	CommitLog  string
	ExistsTags map[string]bool
	Err        error
}

func (m *GitProviderMock) GetLatestTag() (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	if m.LatestTag == "" {
		return "v0.0.4", nil
	}
	return m.LatestTag, nil
}

func (m *GitProviderMock) GetCommitLog(since string) (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	return m.CommitLog, nil
}

func (m *GitProviderMock) TagExists(tag string) (bool, error) {
	if m.ExistsTags == nil {
		return false, nil
	}
	return m.ExistsTags[tag], nil
}

// --------------------------------------------------------------------------------

type PRProvider interface {
	GetMergedPRLabels() (string, error)
}

type PRProviderReal struct {
	token string
}

func (p *PRProviderReal) GetMergedPRLabels() (string, error) {
	if p.token == "" {
		return "", nil
	}

	cmd := exec.Command("gh", "pr", "list", "--state", "merged", "--head", "main", "--limit", "1", "--json", "labels", "--jq", ".[0].labels[].name")
	cmd.Env = append(os.Environ(), "GITHUB_TOKEN="+p.token)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}

type PRProviderMock struct {
	Labels string
	Err    error
}

func (m *PRProviderMock) GetMergedPRLabels() (string, error) {
	if m.Err != nil {
		return "", m.Err
	}
	return m.Labels, nil
}

// ExecRunnerMock records calls and returns a configurable error.
type ExecRunnerMock struct {
	// Calls records every (name, args...) invocation.
	Calls [][]string
	// Err is returned by every call when non-nil.
	Err error
}

func (m *ExecRunnerMock) Run(name string, args ...string) error {
	m.Calls = append(m.Calls, append([]string{name}, args...))
	return m.Err
}

func (m *ExecRunnerMock) RunWithStdin(stdin, name string, args ...string) error {
	m.Calls = append(m.Calls, append([]string{name}, args...))
	return m.Err
}
