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
	FileNode                    = "package.json"
	FileChangelog               = "CHANGELOG.md"
	FileReadme                  = "README.md"
	FileContainerfile           = "Containerfile"
	FileDockerfile              = "Dockerfile"
	FileGoreleaserContainerfile = "Containerfile.goreleaser"
	FileGoreleaserDockerfile    = "Dockerfile.goreleaser"
	FileGoReleaser              = ".goreleaser.yml"
	FileGo                      = "go.mod"
	FileMaven                   = "pom.xml"
	FileGradle                  = "build.gradle"
	FileGradleKts               = "build.gradle.kts"
	FileSpringProps             = "application.properties"
	FileSpringYml               = "application.yml"
	FileSpringYaml              = "application.yaml"
)

// Output keys
const (
	OutputTagLatest            = "latest_tag"
	OutputTagNext              = "next_tag"
	OutputTagExists            = "tag_exists"
	OutputPublish              = "publish"
	OutputSkip                 = "skip"
	OutputVersion              = "version"
	OutputVersionMajorMinor    = "version_major_minor"
	OutputDockerfile           = "dockerfile"
	OutputReleaseTag           = "release_tag"
	OutputSummaryMessage       = "summary_message"
	OutputGoreleaserYmlCurrent = "goreleaser_config_current"
	OutputGoreleaserYmlNew     = "goreleaser_config_auto"
	OutputGoreleaserDocker     = "goreleaser_docker"
	OutputHasDocker            = "has_docker"
	OutputHasGo                = "has_go"
	OutputHasMaven             = "has_maven"
	OutputHasNpm               = "has_npm"
	OutputTagLatestFlag        = "tag_latest"
	OutputDockerImage          = "docker_image"
	OutputVersionClean         = "version_clean"
)
