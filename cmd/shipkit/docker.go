package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type dockerHubLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type dockerHubLoginResponse struct {
	Token string `json:"token"`
}

type dockerHubRepoUpdate struct {
	FullDescription string `json:"full_description"`
}

func runDockerHubReadme(args []string) error {
	fs := newFlagSet("docker-hub-readme")

	repo := fs.String("repo", "", "Docker Hub repository (owner/name) (required)")
	username := fs.String("username", os.Getenv(EnvDockerHubUsername), "Docker Hub username (or set DOCKERHUB_USERNAME env)")
	password := fs.String("password", "", "Docker Hub password/token (or set DOCKERHUB_PASSWORD or DOCKERHUB_TOKEN env)")
	readmePath := fs.String("readme", FileReadme, "Path to README file")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *repo == "" {
		return fmt.Errorf("repo is required")
	}

	if *username == "" {
		return fmt.Errorf("username is required (set via -username or %s env)", EnvDockerHubUsername)
	}

	if *password == "" {
		*password = getSecretWithFallbacks(EnvDockerHubPassword, EnvDockerHubToken)
	}
	if *password == "" {
		return fmt.Errorf("password is required (set via -password, %s, or %s env)", EnvDockerHubPassword, EnvDockerHubToken)
	}

	owner, name, err := parseRepoFormat(*repo)
	if err != nil {
		return err
	}

	readmeContent, err := os.ReadFile(*readmePath)
	if err != nil {
		return fmt.Errorf("failed to read README file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "🐳 Uploading %s to Docker Hub repository %s...\n", *readmePath, *repo)

	token, err := dockerHubLogin(*username, *password)
	if err != nil {
		return fmt.Errorf("failed to login to Docker Hub: %w", err)
	}

	if err := dockerHubUpdateReadme(token, owner, name, string(readmeContent)); err != nil {
		return fmt.Errorf("failed to update README: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Successfully updated Docker Hub README for %s\n", *repo)
	return nil
}

func dockerLogin(username, token string) error {
	cmd := exec.Command("docker", "login", "-u", username, "--password-stdin")
	cmd.Stdin = strings.NewReader(token)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker login failed: %w", err)
	}

	return nil
}

func dockerHubLogin(username, password string) (string, error) {
	loginReq := dockerHubLoginRequest{
		Username: username,
		Password: password,
	}

	body, err := json.Marshal(loginReq)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(DockerHubAPIURL+"/users/login/", "application/json", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var loginResp dockerHubLoginResponse
	if err := json.NewDecoder(resp.Body).Decode(&loginResp); err != nil {
		return "", err
	}

	return loginResp.Token, nil
}

func dockerHubUpdateReadme(token, owner, repo, readme string) error {
	updateReq := dockerHubRepoUpdate{
		FullDescription: readme,
	}

	body, err := json.Marshal(updateReq)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/repositories/%s/%s/", DockerHubAPIURL, owner, repo)
	req, err := http.NewRequest(http.MethodPatch, url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "JWT "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed with status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	return nil
}
