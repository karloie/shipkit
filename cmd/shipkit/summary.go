package main

import (
	"encoding/json"
	"fmt"
	"os"
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
	Version                 string `json:"version"`
	DockerImage             string `json:"docker_image"`
	HasGo                   bool   `json:"has_go"`
	HasDocker               bool   `json:"has_docker"`
	HasMaven                bool   `json:"has_maven"`
	HasNpm                  bool   `json:"has_npm"`
	GoreleaserDocker        bool   `json:"goreleaser_docker"`
	GoreleaserConfigCurrent bool   `json:"goreleaser_config_current"`

	// Job results
	ResultPlan           string `json:"result_plan"`
	ResultBuildNpm       string `json:"result_build_npm"`
	ResultBuildGo        string `json:"result_build_go"`
	ResultBuildMaven     string `json:"result_build_maven"`
	ResultBuildDocker    string `json:"result_build_docker"`
	ResultTag            string `json:"result_tag"`
	ResultUpdateVersions string `json:"result_update_versions"`
	ResultPublishNpm     string `json:"result_publish_npm"`
	ResultPublishMaven   string `json:"result_publish_maven"`
	ResultPublishDocker  string `json:"result_publish_docker"`
	ResultPublishGo      string `json:"result_publish_go"`
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
	sb.WriteString(fmt.Sprintf("| Version | `%s` |\n", inputs.Version))
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
	if jobRan(inputs.ResultBuildNpm) {
		sb.WriteString(fmt.Sprintf("| 🏗️ npm Build | %s |\n", statusBadge(inputs.ResultBuildNpm)))
	}
	if jobRan(inputs.ResultBuildGo) {
		sb.WriteString(fmt.Sprintf("| 🏗️ Go Build | %s |\n", statusBadge(inputs.ResultBuildGo)))
	}
	if jobRan(inputs.ResultBuildMaven) {
		sb.WriteString(fmt.Sprintf("| 🏗️ Maven Build | %s |\n", statusBadge(inputs.ResultBuildMaven)))
	}
	if jobRan(inputs.ResultBuildDocker) {
		sb.WriteString(fmt.Sprintf("| 🏗️ Docker Build | %s |\n", statusBadge(inputs.ResultBuildDocker)))
	}
	if jobRan(inputs.ResultTag) {
		sb.WriteString(fmt.Sprintf("| 🏷️ Tag | %s |\n", statusBadge(inputs.ResultTag)))
	}
	if jobRan(inputs.ResultUpdateVersions) {
		sb.WriteString(fmt.Sprintf("| 📝 Update Versions | %s |\n", statusBadge(inputs.ResultUpdateVersions)))
	}
	if jobRan(inputs.ResultPublishNpm) {
		sb.WriteString(fmt.Sprintf("| 🚀 npm Publish | %s |\n", statusBadge(inputs.ResultPublishNpm)))
	}
	if jobRan(inputs.ResultPublishMaven) {
		sb.WriteString(fmt.Sprintf("| 🚀 Maven Publish | %s |\n", statusBadge(inputs.ResultPublishMaven)))
	}
	if jobRan(inputs.ResultPublishDocker) {
		sb.WriteString(fmt.Sprintf("| 🚀 Docker Publish | %s |\n", statusBadge(inputs.ResultPublishDocker)))
	}
	if jobRan(inputs.ResultPublishGo) {
		sb.WriteString(fmt.Sprintf("| 🚀 Go Publish | %s |\n", statusBadge(inputs.ResultPublishGo)))
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
		return "✅ Success"
	case "failure":
		return "❌ Failed"
	case "skipped":
		return "⏭️ Skipped"
	case "cancelled":
		return "🚫 Cancelled"
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

	// Check if any publish job succeeded
	anyPublished := inputs.ResultPublishGo == "success" ||
		inputs.ResultPublishDocker == "success" ||
		inputs.ResultPublishNpm == "success" ||
		inputs.ResultPublishMaven == "success"

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
	fs := newFlagSet("summary")

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
	goreleaserDocker := fs.Bool("should-build-docker-goreleaser", false, "goreleaser handles docker")
	goreleaserConfigCurrent := fs.Bool("goreleaser-config-current", false, "has custom goreleaser config")

	// Job results
	resultPlan := fs.String("result-plan", "", "plan job result")
	resultBuildNpm := fs.String("result-build-npm", "", "npm-build job result")
	resultBuildGo := fs.String("result-build-go", "", "go-build job result")
	resultBuildMaven := fs.String("result-build-maven", "", "maven-build job result")
	resultBuildDocker := fs.String("result-build-docker", "", "docker-build job result")
	resultTag := fs.String("result-tag", "", "tag job result")
	resultUpdateVersions := fs.String("result-update-versions", "", "update-versions job result")
	resultPublishNpm := fs.String("result-publish-npm", "", "npm-publish job result")
	resultPublishMaven := fs.String("result-publish-maven", "", "maven-publish job result")
	resultPublishDocker := fs.String("result-publish-docker", "", "docker-publish job result")
	resultPublishGo := fs.String("result-publish-go", "", "go-publish job result")

	if err := fs.Parse(args); err != nil {
		return err
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
			Version:                 strings.TrimSpace(*version),
			DockerImage:             strings.TrimSpace(*dockerImage),
			HasGo:                   *hasGo,
			HasDocker:               *hasDocker,
			HasMaven:                *hasMaven,
			HasNpm:                  *hasNpm,
			GoreleaserDocker:        *goreleaserDocker,
			GoreleaserConfigCurrent: *goreleaserConfigCurrent,
			ResultPlan:              strings.TrimSpace(*resultPlan),
			ResultBuildNpm:          strings.TrimSpace(*resultBuildNpm),
			ResultBuildGo:           strings.TrimSpace(*resultBuildGo),
			ResultBuildMaven:        strings.TrimSpace(*resultBuildMaven),
			ResultBuildDocker:       strings.TrimSpace(*resultBuildDocker),
			ResultTag:               strings.TrimSpace(*resultTag),
			ResultUpdateVersions:    strings.TrimSpace(*resultUpdateVersions),
			ResultPublishNpm:        strings.TrimSpace(*resultPublishNpm),
			ResultPublishMaven:      strings.TrimSpace(*resultPublishMaven),
			ResultPublishDocker:     strings.TrimSpace(*resultPublishDocker),
			ResultPublishGo:         strings.TrimSpace(*resultPublishGo),
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
		if *resultBuildNpm != "" {
			inputs.ResultBuildNpm = strings.TrimSpace(*resultBuildNpm)
		}
		if *resultBuildGo != "" {
			inputs.ResultBuildGo = strings.TrimSpace(*resultBuildGo)
		}
		if *resultBuildMaven != "" {
			inputs.ResultBuildMaven = strings.TrimSpace(*resultBuildMaven)
		}
		if *resultBuildDocker != "" {
			inputs.ResultBuildDocker = strings.TrimSpace(*resultBuildDocker)
		}
		if *resultTag != "" {
			inputs.ResultTag = strings.TrimSpace(*resultTag)
		}
		if *resultUpdateVersions != "" {
			inputs.ResultUpdateVersions = strings.TrimSpace(*resultUpdateVersions)
		}
		if *resultPublishNpm != "" {
			inputs.ResultPublishNpm = strings.TrimSpace(*resultPublishNpm)
		}
		if *resultPublishMaven != "" {
			inputs.ResultPublishMaven = strings.TrimSpace(*resultPublishMaven)
		}
		if *resultPublishDocker != "" {
			inputs.ResultPublishDocker = strings.TrimSpace(*resultPublishDocker)
		}
		if *resultPublishGo != "" {
			inputs.ResultPublishGo = strings.TrimSpace(*resultPublishGo)
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
