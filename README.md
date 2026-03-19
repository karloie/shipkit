# 🚢 Shipkit

<img src="doc/vibecoded.png" width="120" alt="Vibe Coded Badge" align="right">

Reusable GitHub workflow tooling for my GitHub projects.

This repository ships a single CLI:

- `shipkit` (`cmd/shipkit`)

The tool provides subcommands:

- `version` - Display version information
- `policy` - Compute release policy for workflows
- `plan` - Compute release versions and validate publishing requirements
- `assets-delete` - Delete GitHub release assets by tag
- `goreleaser` - Generate GoReleaser configuration from template
- `docker-readme` - Upload README.md to Docker Hub repository

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
  env:
    GH_TOKEN: ${{ github.token }}
    DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
    DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
```

**For goreleaser mode:**
```yaml
- name: Plan release
  id: plan
  run: >-
    go run github.com/karloie/shipkit/cmd/shipkit@v0.1.0 plan
    -mode=goreleaser
    -bump=${{ inputs.bump }}
  env:
    GH_TOKEN: ${{ github.token }}
    HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```

**Outputs:** `latest_tag`, `next_tag`, `publish`, `should_publish`, `publish_mode`, `release_tag`, `docker_version`, `docker_major_minor`, `dockerfile`, `summary_message`

### Other Commands

The CLI provides additional commands primarily used internally by workflows:

- **`version`** - Computes version bumps from commits or manual input
- **`policy`** - Evaluates release policies (used internally by workflows)
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

The `docker-readme` subcommand uploads your README.md to Docker Hub:

```bash
# Basic usage (reads README.md from current directory)
shipkit docker-readme -repo=owner/image

# Custom README path
shipkit docker-readme -repo=owner/image -readme=docs/DOCKER.md

# Environment variables (recommended for CI)
export DOCKERHUB_USERNAME=myuser
export DOCKERHUB_TOKEN=mytoken
shipkit docker-readme -repo=owner/image
```

**In GitHub Actions:**
```yaml
- name: Update Docker Hub README
  run: |
    go run github.com/karloie/shipkit/cmd/shipkit@main docker-readme \
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

- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `.github/workflows/re-release.yml`
- `.github/workflows/docker.yml`
- `.github/workflows/goreleaser.yml`

### CI Workflow

`ci.yml` supports `validation` (Make-based) and `build-test` modes.

**Example with Node.js:**
```yaml
jobs:
  build-and-test:
    uses: karloie/shipkit/.github/workflows/ci.yml@v0.1.0
    with:
      mode: build-test
      go_version_file: go.mod
      node_version: '22'
      frontend_install_cmd: npm ci
      frontend_build_cmd: npm run build
      build_cmd: make build
      test_cmd: go test ./...
```

**Example without Node.js:**
```yaml
jobs:
  build-and-test:
    uses: karloie/shipkit/.github/workflows/ci.yml@v0.1.0
    with:
      mode: build-test
      go_version_file: go.mod
      build_cmd: make build
      test_cmd: go test ./...
```

### Workflow Configuration Notes

- **Docker builds:** If `Containerfile.goreleaser` or `Dockerfile.goreleaser` exists, GoReleaser handles Docker publishing and the `docker.yml` workflow is automatically skipped
- **Frontend assets:** When using `docker.yml` or `release.yml` with Node.js projects, pass `node_version` and frontend build commands to ensure npm builds run before Docker builds
- **Docker metadata args:** Use uppercase `BUILD_VERSION`, `BUILD_COMMIT`, `BUILD_DATE`
- **Node.js optional:** Set `setup_node: 'false'` for Go-only projects
- **Go variables:** Use camelCase (e.g., `buildVersion`, `buildCommit`)

### Release Workflow Example

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

**With Node.js frontend:**
```yaml
jobs:
  release:
    uses: karloie/shipkit/.github/workflows/release.yml@v0.1.0
    with:
      image: owner/repo
      event_name: ${{ github.event_name }}
      bump: ${{ inputs.bump }}
      node_version: '22'
      frontend_install_cmd: npm ci
      frontend_build_cmd: npm run build
      tool_ref: v0.1.0
    secrets:
      DOCKERHUB_USERNAME: ${{ secrets.DOCKERHUB_USERNAME }}
      DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
      HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
```
