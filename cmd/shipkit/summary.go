package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SummaryInputs contains all inputs for generating the release summary
type SummaryInputs struct {
	// Plan outputs
	Mode                    string `json:"mode"`
	ToolRef                 string `json:"tool_ref"`
	Skip                    bool   `json:"skip"`
	Tag                     string `json:"tag"`
	TagExists               bool   `json:"tag_exists"`
	VersionClean            string `json:"version_clean"`
	DockerImage             string `json:"dockerimage"`
	HasGo                   bool   `json:"has_go"`
	HasDocker               bool   `json:"has_docker"`
	HasMaven                bool   `json:"has_maven"`
	HasNpm                  bool   `json:"has_npm"`
	GoreleaserDocker        bool   `json:"goreleaser_docker"`
	GoreleaserConfigCurrent bool   `json:"goreleaser_config_current"`
	BuildOrchestrator       string `json:"build_orchestrator"`

	// Job results
	ResultPlan    string `json:"result_plan"`
	ResultBuild   string `json:"result_build"` // Single build result from `shipkit build`
	ResultTag     string `json:"result_tag"`
	ResultPublish string `json:"result_publish"` // Single publish result from `shipkit publish`
}

// GenerateSummary creates a markdown summary of the release
func GenerateSummary(inputs SummaryInputs) string {
	var sb strings.Builder

	// Header
	sb.WriteString("# 🚀 Release Summary\n\n")

	// Plan section
	sb.WriteString("## 📋 Plan\n\n")
	sb.WriteString("| Property | Value |\n")
	sb.WriteString("|----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Mode | `%s` |\n", inputs.Mode))
	sb.WriteString(fmt.Sprintf("| Tool Ref | `%s` |\n", inputs.ToolRef))
	sb.WriteString(fmt.Sprintf("| Skip | %v |\n", inputs.Skip))
	sb.WriteString(fmt.Sprintf("| Tag | `%s` |\n", inputs.Tag))
	sb.WriteString(fmt.Sprintf("| Tag Exists | %v |\n", inputs.TagExists))
	sb.WriteString(fmt.Sprintf("| Version | `%s` |\n", inputs.VersionClean))
	if inputs.DockerImage != "" {
		sb.WriteString(fmt.Sprintf("| Docker Image | `%s` |\n", inputs.DockerImage))
	}
	sb.WriteString("\n")

	// Detection section
	sb.WriteString("## 🔍 Detected Projects\n\n")
	sb.WriteString("| Project Type | Detected | GoReleaser Handles |\n")
	sb.WriteString("|--------------|----------|-------------------|\n")
	sb.WriteString(fmt.Sprintf("| Go | %s | - |\n", checkmark(inputs.HasGo)))
	sb.WriteString(fmt.Sprintf("| Docker | %s | %s |\n", checkmark(inputs.HasDocker), checkmark(inputs.GoreleaserDocker)))
	sb.WriteString(fmt.Sprintf("| Maven | %s | - |\n", checkmark(inputs.HasMaven)))
	sb.WriteString(fmt.Sprintf("| npm | %s | - |\n", checkmark(inputs.HasNpm)))
	sb.WriteString("\n")

	// Build orchestrator section
	if inputs.BuildOrchestrator != "" {
		sb.WriteString("## 🔨 Build Orchestrator\n\n")
		orchestratorName := inputs.BuildOrchestrator
		switch orchestratorName {
		case "make":
			orchestratorName = "**Make** (Makefile)"
		case "just":
			orchestratorName = "**just** (justfile)"
		case "task":
			orchestratorName = "**Task** (Taskfile)"
		case "convention":
			orchestratorName = "Convention-based (no Makefile detected)"
		}
		sb.WriteString(fmt.Sprintf("Using: %s\n\n", orchestratorName))
	}

	// GoReleaser config section
	sb.WriteString("## 🚀 GoReleaser Configuration\n\n")
	if inputs.GoreleaserConfigCurrent {
		sb.WriteString("✅ Using **custom** .goreleaser.yml config\n")
	} else {
		sb.WriteString("🔧 Will **auto-generate** GoReleaser config (no .goreleaser.yml found)\n")
	}
	sb.WriteString("\n")

	// Execution section
	sb.WriteString("## ⚙️ Execution Results\n\n")
	sb.WriteString("| Job | Status |\n")
	sb.WriteString("|-----|--------|\n")

	// Only show jobs that actually ran
	if inputs.ResultPlan != "" && inputs.ResultPlan != "skipped" {
		sb.WriteString(fmt.Sprintf("| 🚢 Plan | %s |\n", statusBadge(inputs.ResultPlan)))
	}
	if jobRan(inputs.ResultBuild) {
		sb.WriteString(fmt.Sprintf("| 🔨 Build | %s |\n", statusBadge(inputs.ResultBuild)))
	}
	if jobRan(inputs.ResultTag) {
		sb.WriteString(fmt.Sprintf("| 🏷️ Tag | %s |\n", statusBadge(inputs.ResultTag)))
	}
	if jobRan(inputs.ResultPublish) {
		sb.WriteString(fmt.Sprintf("| 🚀 Publish | %s |\n", statusBadge(inputs.ResultPublish)))
	}
	sb.WriteString("\n")

	// Overall status
	overallStatus := determineOverallStatus(inputs)
	sb.WriteString(overallStatus)

	return sb.String()
}

// checkmark returns ✅ or ❌ based on boolean value
func checkmark(value bool) string {
	if value {
		return "✅"
	}
	return "❌"
}

// statusBadge returns a formatted status badge
func statusBadge(result string) string {
	result = strings.ToLower(strings.TrimSpace(result))
	switch result {
	case "success":
		return "✅ success"
	case "failure":
		return "❌ failure"
	case "skipped":
		return "⏭️ skipped"
	case "cancelled":
		return "🚫 cancelled"
	default:
		return fmt.Sprintf("⚠️ %s", result)
	}
}

// jobRan returns true if the job actually executed (not skipped or empty)
func jobRan(result string) bool {
	result = strings.ToLower(strings.TrimSpace(result))
	return result == "success" || result == "failure"
}

// determineOverallStatus determines the overall status of the release
func determineOverallStatus(inputs SummaryInputs) string {
	if inputs.Skip {
		return "## ℹ️ Overall Status: **SKIPPED**\n\nNo release markers found. Release was skipped.\n"
	}

	// Check if publish succeeded
	anyPublished := inputs.ResultPublish == "success"

	if anyPublished {
		return fmt.Sprintf("## ✅ Overall Status: **SUCCESS**\n\nRelease `%s` completed successfully!\n", inputs.Tag)
	}

	if inputs.ResultPlan == "failure" {
		return "## ❌ Overall Status: **FAILED**\n\nRelease failed during planning phase.\n"
	}

	if inputs.ResultTag == "failure" {
		return "## ❌ Overall Status: **FAILED**\n\nRelease failed during tag creation.\n"
	}

	return "## ⚠️ Overall Status: **PARTIAL**\n\nSome jobs succeeded, but publish phase may have been skipped or failed.\n"
}

func runSummary(args []string) error {
	// Log raw args BEFORE parsing
	logInputs(map[string]string{
		"raw_args": strings.Join(args, " "),
	})
	fs := newFlagSet("summary")

	// Makefile override support
	makefile := fs.String("makefile", "Makefile", "Path to Makefile")
	useMake := fs.Bool("use-make", true, "Check for ci-summary target in Makefile")

	// File-based input (preferred)
	planFile := fs.String("plan-file", "", "Path to plan.json file from plan job")

	// Direct JSON input (fallback)
	jsonInput := fs.String("json", "", "JSON string with all summary inputs")

	// Plan outputs (fallback individual flags)
	mode := fs.String("mode", "", "release mode")
	toolRef := fs.String("tool-ref", "", "tool reference")
	skip := fs.Bool("skip", false, "release was skipped")
	tag := fs.String("tag", "", "release tag")
	tagExists := fs.Bool("tag-exists", false, "tag already exists")
	version := fs.String("version", "", "version")
	dockerImage := fs.String("docker-image", "", "docker image")
	hasGo := fs.Bool("has-go", false, "has go project")
	hasDocker := fs.Bool("has-docker", false, "has docker")
	hasMaven := fs.Bool("has-maven", false, "has maven")
	hasNpm := fs.Bool("has-npm", false, "has npm")
	goreleaserDocker := fs.Bool("goreleaser-docker", false, "goreleaser handles docker")
	goreleaserConfigCurrent := fs.Bool("goreleaser-config-current", false, "has custom goreleaser config")
	buildOrchestrator := fs.String("build-orchestrator", "", "build orchestrator (make, just, task)")

	// Job results
	resultPlan := fs.String("result-plan", "", "plan job result")
	resultBuild := fs.String("result-build", "", "build job result")
	resultTag := fs.String("result-tag", "", "tag job result")
	resultPublish := fs.String("result-publish", "", "publish job result")
	parseFlagsOrExit(fs, args)

	// Check for Makefile ci-summary target first
	if *useMake {
		if hasCISummaryTarget(*makefile) {
			fmt.Fprintf(os.Stderr, "📊 Using Makefile ci-summary target\n")
			return runMakeSummary(*makefile, args)
		}
	}

	var inputs SummaryInputs

	// Load plan data from file if provided
	if *planFile != "" {
		data, err := os.ReadFile(*planFile)
		if err != nil {
			return fmt.Errorf("failed to read plan file: %w", err)
		}
		if err := json.Unmarshal(data, &inputs); err != nil {
			return fmt.Errorf("failed to parse plan JSON: %w", err)
		}
		// Job results will be set from flags below
	} else if *jsonInput != "" {
		// Fallback: direct JSON input
		if err := json.Unmarshal([]byte(*jsonInput), &inputs); err != nil {
			return fmt.Errorf("failed to parse JSON input: %w", err)
		}
	} else {
		// Fallback: individual flags for plan outputs
		inputs = SummaryInputs{
			Mode:                    strings.TrimSpace(*mode),
			ToolRef:                 strings.TrimSpace(*toolRef),
			Skip:                    *skip,
			Tag:                     strings.TrimSpace(*tag),
			TagExists:               *tagExists,
			VersionClean:            strings.TrimSpace(*version),
			DockerImage:             strings.TrimSpace(*dockerImage),
			HasGo:                   *hasGo,
			HasDocker:               *hasDocker,
			HasMaven:                *hasMaven,
			HasNpm:                  *hasNpm,
			GoreleaserDocker:        *goreleaserDocker,
			GoreleaserConfigCurrent: *goreleaserConfigCurrent,
			BuildOrchestrator:       strings.TrimSpace(*buildOrchestrator),
			ResultPlan:              strings.TrimSpace(*resultPlan),
			ResultBuild:             strings.TrimSpace(*resultBuild),
			ResultTag:               strings.TrimSpace(*resultTag),
			ResultPublish:           strings.TrimSpace(*resultPublish),
		}
	}

	// If plan file was loaded, override job results from flags
	if *planFile != "" || *jsonInput != "" {
		// Parse job result flags (only if plan file was used)
		// Override tool_ref if provided (plan.json has empty string)
		if *toolRef != "" {
			inputs.ToolRef = strings.TrimSpace(*toolRef)
		}
		if *resultPlan != "" {
			inputs.ResultPlan = strings.TrimSpace(*resultPlan)
		}
		if *resultBuild != "" {
			inputs.ResultBuild = strings.TrimSpace(*resultBuild)
		}
		if *resultTag != "" {
			inputs.ResultTag = strings.TrimSpace(*resultTag)
		}
		if *resultPublish != "" {
			inputs.ResultPublish = strings.TrimSpace(*resultPublish)
		}
	}

	summary := GenerateSummary(inputs)

	// Write to GITHUB_STEP_SUMMARY
	summaryFile := os.Getenv("GITHUB_STEP_SUMMARY")
	if summaryFile == "" {
		// If not in GitHub Actions, write to stdout
		fmt.Print(summary)
		return nil
	}

	f, err := os.OpenFile(summaryFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open GITHUB_STEP_SUMMARY: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(summary); err != nil {
		return fmt.Errorf("failed to write summary: %w", err)
	}

	fmt.Fprintln(os.Stderr, "📊 Release summary generated successfully")

	return nil
}

// hasCISummaryTarget checks if Makefile has a ci-summary target
func hasCISummaryTarget(makefilePath string) bool {
	graph, err := ParseMakefile(makefilePath)
	if err != nil {
		return false
	}
	_, exists := graph.Targets["ci-summary"]
	return exists
}

// runMakeSummary delegates summary generation to Makefile ci-summary target
func runMakeSummary(makefilePath string, args []string) error {
	// Re-parse args to get all the values we need to pass as env vars
	fs := newFlagSet("summary-make")
	planFile := fs.String("plan-file", "", "Plan file path")
	toolRef := fs.String("tool-ref", "", "Tool ref")
	resultPlan := fs.String("result-plan", "", "Plan result")
	resultBuild := fs.String("result-build", "", "Build result")
	resultTag := fs.String("result-tag", "", "Tag result")
	resultPublish := fs.String("result-publish", "", "Publish result")
	parseFlagsOrExit(fs, args)

	// Set environment variables for Makefile to use
	env := os.Environ()
	if *planFile != "" {
		env = append(env, fmt.Sprintf("SHIPKIT_PLAN_FILE=%s", *planFile))
	}
	if *toolRef != "" {
		env = append(env, fmt.Sprintf("SHIPKIT_TOOL_REF=%s", *toolRef))
	}
	if *resultPlan != "" {
		env = append(env, fmt.Sprintf("SHIPKIT_RESULT_PLAN=%s", *resultPlan))
	}
	if *resultBuild != "" {
		env = append(env, fmt.Sprintf("SHIPKIT_RESULT_BUILD=%s", *resultBuild))
	}
	if *resultTag != "" {
		env = append(env, fmt.Sprintf("SHIPKIT_RESULT_TAG=%s", *resultTag))
	}
	if *resultPublish != "" {
		env = append(env, fmt.Sprintf("SHIPKIT_RESULT_PUBLISH=%s", *resultPublish))
	}

	// Run make ci-summary
	cmd := exec.Command("make", "-f", makefilePath, "ci-summary")
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("make ci-summary failed: %w", err)
	}

	return nil
}
