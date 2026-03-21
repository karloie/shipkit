a map of deployment rules (?)

a project has different combinations of frameworks like go, node, docker, springboot, maven, gradle

a project has its build sequence determinated by framework types and git tag situation

a build is executed in 2 phases, planning calculates stuff and plans out a new state, running executes and validates accoring to plan

a plan is determenistic and makes all logical chises for a build upfront

a run never decides choices on its own, but errors if state deviates from plan






name: Release Shipkit

on:
  workflow_dispatch:
    inputs:
      bump:
        description: "Version bump (patch, minor, major)"
        required: false
        default: patch
        type: choice
        options:
          - patch
          - minor
          - major
      tool_ref:
        description: "Shipkit version to use"
        required: false
        default: v0.2.23
        type: choice
        options:
          - v0.2.23
          - main
          - v0.2.22
          - v0.2.21
          - v0.1.1
      mode:
        description: "Release mode"
        required: false
        default: release
        type: choice
        options:
          - release
          - rerelease
          - docker
          - goreleaser
      image:
        description: "Docker image repository name"
        required: false
        default: "{owner}/{project}"
        type: string
      go_version:
        description: "Go version (leave empty to auto-detect from go.mod or skip)"
        required: false
        default: ""
        type: string
      node_version:
        description: "Node version for frontend build (leave empty to auto-detect from package.json or skip)"
        required: false
        default: ""
        type: string
      go_main:
        description: "Main package path override (leave empty to use ./cmd/{project})"
        required: false
        default: ""
        type: string
      dry_run:
        description: "Dry run mode (run as normal but just log actual commands instead of executing them)"
        required: false
        default: false
        type: boolean
      force_skip_docker:
        description: "Force skip docker job even if Dockerfile exists"
        required: false
        default: false
        type: boolean
      force_skip_goreleaser:
        description: "Force skip goreleaser job even if .goreleaser.yml exists"
        required: false
        default: false
        type: boolean

permissions:
  contents: write
  pull-requests: read

jobs:
  release:
    concurrency:
      group: release-shipkit-main
      cancel-in-progress: false
    # Note: @ref must be static - GitHub Actions doesn't allow inputs context here
    # The tool_ref input controls which Go code version is used, not the workflow file version
    uses: karloie/shipkit/.github/workflows/release.yml@main
    with:
      image: ${{ inputs.image || format('{0}/{1}', github.repository_owner, github.event.repository.name) }}
      event_name: ${{ github.event_name }}
      bump: ${{ inputs.bump }}
      tool_ref: ${{ inputs.tool_ref || 'v0.2.23' }}
      mode: ${{ inputs.mode || 'release' }}
      go_version: ${{ inputs.go_version }}
      node_version: ${{ inputs.node_version }}
      go_main: ${{ inputs.go_main }}
      dry_run: ${{ inputs.dry_run }}
      force_skip_docker: ${{ inputs.force_skip_docker }}
      force_skip_goreleaser: ${{ inputs.force_skip_goreleaser }}
    secrets:
      DOCKERHUB_TOKEN: ${{ secrets.DOCKERHUB_TOKEN }}
      DOCKERHUB_USERNAME: karloie
      HOMEBREW_TAP_GITHUB_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}
