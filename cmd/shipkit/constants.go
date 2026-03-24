package main

// Environment variable names
const (
	EnvGitHubToken       = "GITHUB_TOKEN"
	EnvGitHubOutput      = "GITHUB_OUTPUT"
	EnvGitHubEventName   = "GITHUB_EVENT_NAME"
	EnvDockerHubUsername = "DOCKERHUB_USERNAME"
	EnvDockerHubToken    = "DOCKERHUB_TOKEN"
	EnvDockerHubPassword = "DOCKERHUB_PASSWORD"
	EnvHomebrewTapToken  = "HOMEBREW_TAP_GITHUB_TOKEN"
)

// Docker Hub API constants
var (
	DockerHubAPIURL = "https://hub.docker.com/v2"
)

// Default values
const (
	DefaultImage   = "karloie/kompass"
	DefaultLicense = "MIT"
	DefaultMode    = "release"
)

// Mode types
const (
	ModeRelease    = "release"
	ModeRerelease  = "rerelease"
	ModeDocker     = "docker"
	ModeGoreleaser = "goreleaser"
)

// Bump types
const (
	BumpPatch = "patch"
	BumpMinor = "minor"
	BumpMajor = "major"
)

// Release states
const (
	ReleaseTrue  = "true"
	ReleaseSkip  = "skip"
	ReleaseFalse = "false"
)

// PR label prefixes
const (
	PRLabelReleaseMajor = "release:major"
	PRLabelReleaseMinor = "release:minor"
	PRLabelReleasePatch = "release:patch"
)

// File names
const (
	FileChangelog     = "CHANGELOG.md"
	FileReadme        = "README.md"
	FileContainerfile = "Containerfile"
	FileDockerfile    = "Dockerfile"
	FileGoReleaser    = ".goreleaser.yml"
)

// Output keys (normalized to match Plan struct JSON tags)
const (
	OutputTagLatest         = "tag_latest"
	OutputTagNext           = "tag_next"
	OutputTagExists         = "tag_exists"
	OutputReleaseSkip       = "release_skip"
	OutputVersionMajorMinor = "version_major_minor"
	OutputDockerFile        = "docker_file"
	OutputGoreleaserConfig  = "goreleaser_config"
	OutputHasDocker         = "has_docker"
	OutputDockerImage       = "container_image"
	OutputVersionClean      = "version_clean"
	OutputReleaseDocker     = "release_docker"
)
