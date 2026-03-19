# 🚢 Shipkit

<img src="doc/vibecoded.png" width="200" alt="Vibe Coded Badge" align="right">

Reusable GitHub workflow tooling for my GitHub projects.

This repository ships a single CLI:

- `shipkit` (`cmd/shipkit`)

The tool provides subcommands:

**Primary commands:**
- `plan` - Compute release versions and validate publishing requirements
- `version` - Display version information
- `goreleaser` - Generate GoReleaser configuration from template
- `docker-hub-readme` - Upload README.md to Docker Hub repository
- `assets-delete` - Delete GitHub release assets by tag

**Internal commands (used by workflows):**
- `git-config` - Configure git user
- `git-tag` - Create annotated git tags
- `git-tag-cleanup` - Delete git tags on failures
- `docker-hub-status` - Check Docker login status  

## Use In Another Repository

Run directly from GitHub using a tagged version:

```bash
go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 plan -mode=release -bump=patch -image=owner/repo
go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 goreleaser -project=myapp -owner=myorg -description="My application"
```

Pin to a stable tag (recommended) rather than branch names.

### Plan Command (Recommended)

The `plan` subcommand computes release versions and validates publishing requirements:

```bash
# Automatic bump detection (from commits/PR labels)
shipkit plan -mode=release -image=karloie/kompass -sha=${{ github.sha }}

# Manual bump specification
shipkit plan -mode=release -bump=patch -image=karloie/kompass -sha=${{ github.sha }}
```

The plan command automatically detects your project type by checking for common build files and logs what it finds:
- **Go**: `go.mod`
- **Node**: `package.json`
- **Docker**: `Containerfile`, `Dockerfile`
- **GoReleaser**: `.goreleaser.yml`, `.goreleaser.yaml`

**Modes and Required Secrets:**

- `release` - Docker + binary release (requires: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`)
- `rerelease` - Re-publish existing tag (requires: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `HOMEBREW_TAP_GITHUB_TOKEN`)
- `goreleaser` - Binary-only release (requires: `HOMEBREW_TAP_GITHUB_TOKEN`)
- `docker` - Docker-only release (requires: `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`)

Secrets are auto-detected by mode, or specify with `-required-secrets`.

**In GitHub Actions (release mode):**
```yaml
- name: Plan release
  id: plan
  run: >-
    go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 plan
    -mode=release
    -bump=${{ inputs.bump }}
    -image=karloie/kompass
    -sha=${{ github.sha }}
    -owner=${{ github.repository_owner }}
    -repo=${{ github.event.repository.name }}
    -run-id=${{ github.run_id }}
  env:
    GITHUB_TOKEN: ${{ github.token }}
    DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
    DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
    HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

**For goreleaser mode:**
```yaml
- name: Plan release
  id: plan
  run: >-
    go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 plan
    -mode=goreleaser
    -bump=${{ inputs.bump }}
    -owner=${{ github.repository_owner }}
    -repo=${{ github.event.repository.name }}
    -run-id=${{ github.run_id }}
  env:
    GITHUB_TOKEN: ${{ github.token }}
    HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

**Outputs:** `latest_tag`, `next_tag`, `publish`, `dryrun`, `release_tag`, `version`, `version_major_minor`, `dockerfile`, `summary_message`, `goreleaser_yml_current`, `goreleaser_yml_new`, `goreleaser_docker`

**Using plan with a known tag (docker mode):**

When the version is already known (e.g., from a parent workflow), pass `-next-tag` to skip version computation:

```yaml
- name: Docker publish
  run: >-
    go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 plan
    -mode=docker
    -next-tag=${{ needs.release.outputs.next_tag }}
    -image=karloie/kompass
    -sha=${{ github.sha }}
  env:
    GITHUB_EVENT_NAME: push
    DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
    DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
```

### Other Commands

- **`version`** - Computes version bumps from commits or manual input
- **`assets-delete`** - Removes release assets (useful for re-releases)

### GoReleaser Config Generation

The `goreleaser` subcommand creates a `.goreleaser.yml` file with sensible defaults:

```bash
# Basic usage
shipkit goreleaser \
  -project=myapp \
  -owner=myorg \
  -description="My application"

# Full options
shipkit goreleaser \
  -project=kompass \
  -binary=kompass \
  -owner=karloie \
  -repo=kompass \
  -description="Kubernetes resource relationship visualizer" \
  -output=.goreleaser.yml
```

Features:
- Detects `package.json` and includes npm build steps (runs before Go build for embedding)
- Detects `Containerfile.goreleaser` or `Dockerfile.goreleaser` for Docker publishing (when present, GoReleaser handles Docker builds and `docker.yml` workflow is skipped)
- Detects `CHANGELOG.md` to use manual changelog or auto-generate from GitHub
- Parameterized project name, binary name, repository details, license
- Generates config for multi-platform builds (Linux, macOS, Windows on amd64 and arm64)
- Includes Homebrew tap, deb/rpm/apk packages, Docker images, SBOM generation, and changelog formatting

### Docker Hub README Upload

The `docker-hub-readme` subcommand uploads your README.md to Docker Hub:

```bash
# Basic usage (reads README.md from current directory)
shipkit docker-hub-readme -repo=owner/image

# Custom README path
shipkit docker-hub-readme -repo=owner/image -readme=docs/DOCKER.md

# Environment variables (recommended for CI)
export DOCKERHUB_USERNAME=myuser
export DOCKERHUB_TOKEN=mytoken
shipkit docker-hub-readme -repo=owner/image
```

**In GitHub Actions:**
```yaml
- name: Update Docker Hub README
  run: |
    go run github.com/karloie/shipkit/cmd/shipkit@main docker-hub-readme \
      -repo=${{ github.repository_owner }}/${{ github.event.repository.name }}
  env:
    DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
    DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
```

## Local Development

```bash
make test lint
make plan-all  # validates all workflow modes
go test ./...
```

## Reusable Workflows

This repository also provides parameterized reusable workflows for callers:

- `.github/workflows/ci.yml` - CI validation and build/test
- `.github/workflows/release.yml` - Automated releases with GoReleaser + Docker
- `.github/workflows/re-release.yml` - Re-publish existing releases
- `.github/workflows/docker.yml` - Docker-only publishing (called by release.yml)

### CI Workflow

`ci.yml` supports two modes:
- `validation` - Runs Make targets (default: `validate`)
- `build-test` - Builds and tests with hardcoded `make build` and `go test ./...`

**Inputs:**
- `mode` - validation or build-test (default: validation)
- `make_targets` - Make targets for validation mode (default: validate)
- `go_version` - Go version override (default: auto-detect)
- `node_version` - Node version for build-test mode (default: none)
- `node_cache` - npm/yarn/pnpm cache type (default: npm)

**Example validation mode:**
```yaml
jobs:
  validate:
    uses: karloie/shipkit/.github/workflows/ci.yml@v0.1.0
    with:
      mode: validation
      make_targets: test lint
```

**Example build-test with Node.js:**
```yaml
jobs:
  build:
    uses: karloie/shipkit/.github/workflows/ci.yml@v0.1.0
    with:
      mode: build-test
      go_version: '1.25'
      node_version: '22'
```

**Note:** Build-test mode hardcodes `npm ci`, `npm run build`, `make build`, and `go test ./...`. For custom commands, use validation mode or your own workflow.

### Release Workflow

`release.yml` performs automated releases using GoReleaser and optionally Docker publishing.

**Inputs:**
- `image` (required) - Docker image name
- `event_name` (required) - GitHub event name
- `bump` (optional) - Manual version bump
- `tool_ref` (optional) - Shipkit version (default: main)
- `go_version` (optional) - Go version (default: auto-detect)

**Secrets:**
- `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN` - For Docker publishing
- `HOMEBREW_TAP_GITHUB_TOKEN` - For Homebrew tap

**Example:**

```yaml
name: Release

on:
  push:
    branches: [main]
  workflow_dispatch:
    inputs:
      bump:
        type: choice
        options: [patch, minor, major]

jobs:
  release:
    uses: karloie/shipkit/.github/workflows/release.yml@v0.1.0
    with:
      image: owner/repo  # e.g., karloie/kompass
      event_name: ${{ github.event_name }}
      bump: ${{ inputs.bump }}
      tool_ref: v0.1.0
    secrets:
      DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
      DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
      HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

**Available workflow inputs:**
- `image` (required): Docker image name (e.g., `karloie/kompass`)
- `event_name` (required): GitHub event name (`push` or `workflow_dispatch`)
- `bump` (optional): Manual version bump (`patch`, `minor`, `major`)
- `tool_ref` (optional): Specific shipkit version/tag (default: `main`)
- `go_version` (optional): Go version override (default: auto-detect from go.mod)

### Re-release Workflow

`re-release.yml` re-publishes the latest git tag.

**Inputs:**
- `image` (required) - Docker image name
- `tool_ref`, `go_version` (optional)

**Example:**
```yaml
name: Re-release

on:
  workflow_dispatch:

jobs:
  rerelease:
    uses: karloie/shipkit/.github/workflows/re-release.yml@v0.1.0
    with:
      image: owner/repo
      tool_ref: v0.1.0
    secrets:
      DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
      DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
      HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

### Workflow Notes

- **Docker publishing:** If `Containerfile.goreleaser` or `Dockerfile.goreleaser` exists, GoReleaser handles Docker builds and `docker.yml` is skipped. Otherwise, `release.yml` calls `docker.yml` after GoReleaser completes.
- **Node.js projects:** If `package.json` exists, GoReleaser automatically runs `npm ci` and `npm run build` in its `before.hooks` section.
- **Docker build args:** Use uppercase `BUILD_VERSION`, `BUILD_COMMIT`, `BUILD_DATE`
- **Go ldflags:** Use camelCase `buildVersion`, `buildCommit`, `buildDate`
