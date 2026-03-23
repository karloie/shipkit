package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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
	repo := fs.String("repo", "", "Docker Hub repo")
	username := fs.String("username", os.Getenv(EnvDockerHubUsername), "Username")
	password := fs.String("password", "", "Password/token")
	readmePath := fs.String("readme", FileReadme, "README path")
	parseFlagsOrExit(fs, args)

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

	fmt.Println("::group::Login")
	token, err := dockerHubLogin(*username, *password)
	if err != nil {
		return fmt.Errorf("failed to login to Docker Hub: %w", err)
	}
	fmt.Println("::endgroup::")

	fmt.Println("::group::Upload README")
	defer fmt.Println("::endgroup::")
	if err := dockerHubUpdateReadme(token, owner, name, string(readmeContent)); err != nil {
		return fmt.Errorf("failed to update README: %w", err)
	}

	fmt.Fprintf(os.Stderr, "✅ Successfully updated Docker Hub README for %s\n", *repo)
	return nil
}

func dockerLogin(username, token string) error {
	if err := defaultRunner.RunWithStdin(token, "docker", "login", "-u", username, "--password-stdin"); err != nil {
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
