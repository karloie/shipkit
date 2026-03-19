package main

// Environment variable names
const (
	EnvGitHubToken       = "GH_TOKEN"
	EnvGitHubOutput      = "GITHUB_OUTPUT"
	EnvGitHubEventName   = "GITHUB_EVENT_NAME"
	EnvDockerHubUsername = "DOCKERHUB_USERNAME"
	EnvDockerHubToken    = "DOCKERHUB_TOKEN"
	EnvDockerHubPassword = "DOCKERHUB_PASSWORD"
	EnvHomebrewTapToken  = "HOMEBREW_TAP_GITHUB_TOKEN"
)

// Docker Hub API constants
const (
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

// Publish states
const (
	PublishTrue  = "true"
	PublishSkip  = "skip"
	PublishFalse = "false"
)

// PR label prefixes
const (
	PRLabelReleaseMajor = "release:major"
	PRLabelReleaseMinor = "release:minor"
	PRLabelReleasePatch = "release:patch"
)

// File names
const (
	FilePackageJSON             = "package.json"
	FileChangelog               = "CHANGELOG.md"
	FileReadme                  = "README.md"
	FileContainerfile           = "Containerfile"
	FileDockerfile              = "Dockerfile"
	FileContainerfileGoreleaser = "Containerfile.goreleaser"
	FileDockerfileGoreleaser    = "Dockerfile.goreleaser"
	FileGoReleaser              = ".goreleaser.yml"
	FileGo                      = "go.mod"
	FileMaven                   = "pom.xml"
	FileGradle                  = "build.gradle"
	FileGradleKts               = "build.gradle.kts"
	FileCargo                   = "Cargo.toml"
	FilePython                  = "pyproject.toml"
	FilePythonPy                = "setup.py"
	FilePythonReq               = "requirements.txt"
	FileRuby                    = "Gemfile"
	FilePHP                     = "composer.json"
	FileApplicationProps        = "application.properties"
	FileApplicationYml          = "application.yml"
	FileApplicationYaml         = "application.yaml"
)

// Output keys
const (
	OutputLatestTag        = "latest_tag"
	OutputNextTag          = "next_tag"
	OutputPublish          = "publish"
	OutputShouldPublish    = "should_publish"
	OutputPublishMode      = "publish_mode"
	OutputDockerVersion    = "docker_version"
	OutputDockerMajorMinor = "docker_major_minor"
	OutputDockerfile       = "dockerfile"
	OutputReleaseTag       = "release_tag"
	OutputSummaryMessage   = "summary_message"
)
