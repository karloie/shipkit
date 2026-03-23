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
	PlanResult           string `json:"plan_result"`
	NpmBuildResult       string `json:"npm_build_result"`
	GoBuildResult        string `json:"go_build_result"`
	MavenBuildResult     string `json:"maven_build_result"`
	DockerBuildResult    string `json:"docker_build_result"`
	TagResult            string `json:"tag_result"`
	UpdateVersionsResult string `json:"update_versions_result"`
	NpmPublishResult     string `json:"npm_publish_result"`
	MavenPublishResult   string `json:"maven_publish_result"`
	DockerPublishResult  string `json:"docker_publish_result"`
	GoPublishResult      string `json:"go_publish_result"`
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
	if inputs.PlanResult != "" && inputs.PlanResult != "skipped" {
		sb.WriteString(fmt.Sprintf("| 🚢 Plan | %s |\n", statusBadge(inputs.PlanResult)))
	}
	if jobRan(inputs.NpmBuildResult) {
		sb.WriteString(fmt.Sprintf("| 🏗️ npm Build | %s |\n", statusBadge(inputs.NpmBuildResult)))
	}
	if jobRan(inputs.GoBuildResult) {
		sb.WriteString(fmt.Sprintf("| 🏗️ Go Build | %s |\n", statusBadge(inputs.GoBuildResult)))
	}
	if jobRan(inputs.MavenBuildResult) {
		sb.WriteString(fmt.Sprintf("| 🏗️ Maven Build | %s |\n", statusBadge(inputs.MavenBuildResult)))
	}
	if jobRan(inputs.DockerBuildResult) {
		sb.WriteString(fmt.Sprintf("| 🏗️ Docker Build | %s |\n", statusBadge(inputs.DockerBuildResult)))
	}
	if jobRan(inputs.TagResult) {
		sb.WriteString(fmt.Sprintf("| 🏷️ Tag | %s |\n", statusBadge(inputs.TagResult)))
	}
	if jobRan(inputs.UpdateVersionsResult) {
		sb.WriteString(fmt.Sprintf("| 📝 Update Versions | %s |\n", statusBadge(inputs.UpdateVersionsResult)))
	}
	if jobRan(inputs.NpmPublishResult) {
		sb.WriteString(fmt.Sprintf("| 🚀 npm Publish | %s |\n", statusBadge(inputs.NpmPublishResult)))
	}
	if jobRan(inputs.MavenPublishResult) {
		sb.WriteString(fmt.Sprintf("| 🚀 Maven Publish | %s |\n", statusBadge(inputs.MavenPublishResult)))
	}
	if jobRan(inputs.DockerPublishResult) {
		sb.WriteString(fmt.Sprintf("| 🚀 Docker Publish | %s |\n", statusBadge(inputs.DockerPublishResult)))
	}
	if jobRan(inputs.GoPublishResult) {
		sb.WriteString(fmt.Sprintf("| 🚀 Go Publish | %s |\n", statusBadge(inputs.GoPublishResult)))
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
	anyPublished := inputs.GoPublishResult == "success" ||
		inputs.DockerPublishResult == "success" ||
		inputs.NpmPublishResult == "success" ||
		inputs.MavenPublishResult == "success"

	if anyPublished {
		return fmt.Sprintf("## ✅ Overall Status: **SUCCESS**\n\nRelease `%s` completed successfully!\n", inputs.Tag)
	}

	if inputs.PlanResult == "failure" {
		return "## ❌ Overall Status: **FAILED**\n\nRelease failed during planning phase.\n"
	}

	if inputs.TagResult == "failure" {
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
	goreleaserDocker := fs.Bool("goreleaser-docker", false, "goreleaser handles docker")
	goreleaserConfigCurrent := fs.Bool("goreleaser-config-current", false, "has custom goreleaser config")

	// Job results
	planResult := fs.String("plan-result", "", "plan job result")
	npmBuildResult := fs.String("build-result-npm", "", "npm-build job result")
	goBuildResult := fs.String("build-result-go", "", "go-build job result")
	mavenBuildResult := fs.String("build-result-maven", "", "maven-build job result")
	dockerBuildResult := fs.String("build-result-docker", "", "docker-build job result")
	tagResult := fs.String("tag-result", "", "tag job result")
	updateVersionsResult := fs.String("update-versions-result", "", "update-versions job result")
	npmPublishResult := fs.String("publish-result-npm", "", "npm-publish job result")
	mavenPublishResult := fs.String("publish-result-maven", "", "maven-publish job result")
	dockerPublishResult := fs.String("publish-result-docker", "", "docker-publish job result")
	goPublishResult := fs.String("publish-result-go", "", "go-publish job result")

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
			PlanResult:              strings.TrimSpace(*planResult),
			NpmBuildResult:          strings.TrimSpace(*npmBuildResult),
			GoBuildResult:           strings.TrimSpace(*goBuildResult),
			MavenBuildResult:        strings.TrimSpace(*mavenBuildResult),
			DockerBuildResult:       strings.TrimSpace(*dockerBuildResult),
			TagResult:               strings.TrimSpace(*tagResult),
			UpdateVersionsResult:    strings.TrimSpace(*updateVersionsResult),
			NpmPublishResult:        strings.TrimSpace(*npmPublishResult),
			MavenPublishResult:      strings.TrimSpace(*mavenPublishResult),
			DockerPublishResult:     strings.TrimSpace(*dockerPublishResult),
			GoPublishResult:         strings.TrimSpace(*goPublishResult),
		}
	}

	// If plan file was loaded, override job results from flags
	if *planFile != "" || *jsonInput != "" {
		// Parse job result flags (only if plan file was used)
		// Override tool_ref if provided (plan.json has empty string)
		if *toolRef != "" {
			inputs.ToolRef = strings.TrimSpace(*toolRef)
		}
		if *planResult != "" {
			inputs.PlanResult = strings.TrimSpace(*planResult)
		}
		if *npmBuildResult != "" {
			inputs.NpmBuildResult = strings.TrimSpace(*npmBuildResult)
		}
		if *goBuildResult != "" {
			inputs.GoBuildResult = strings.TrimSpace(*goBuildResult)
		}
		if *mavenBuildResult != "" {
			inputs.MavenBuildResult = strings.TrimSpace(*mavenBuildResult)
		}
		if *dockerBuildResult != "" {
			inputs.DockerBuildResult = strings.TrimSpace(*dockerBuildResult)
		}
		if *tagResult != "" {
			inputs.TagResult = strings.TrimSpace(*tagResult)
		}
		if *updateVersionsResult != "" {
			inputs.UpdateVersionsResult = strings.TrimSpace(*updateVersionsResult)
		}
		if *npmPublishResult != "" {
			inputs.NpmPublishResult = strings.TrimSpace(*npmPublishResult)
		}
		if *mavenPublishResult != "" {
			inputs.MavenPublishResult = strings.TrimSpace(*mavenPublishResult)
		}
		if *dockerPublishResult != "" {
			inputs.DockerPublishResult = strings.TrimSpace(*dockerPublishResult)
		}
		if *goPublishResult != "" {
			inputs.GoPublishResult = strings.TrimSpace(*goPublishResult)
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
