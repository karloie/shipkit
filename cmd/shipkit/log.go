package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// logInputs prints input parameters in a consistent format
func logInputs(inputs map[string]string) {
	if len(inputs) == 0 {
		return
	}
	fmt.Println("\nINPUT:")
	for key, value := range inputs {
		fmt.Printf(" - %s=%s\n", key, value)
	}
	fmt.Println()
}

// logOutputs prints output values in a consistent format
func logOutputs(outputs map[string]string) {
	if len(outputs) == 0 {
		return
	}
	fmt.Println("\nOUTPUT:")
	for key, value := range outputs {
		fmt.Printf(" - %s=%s\n", key, value)
	}
}

// logCommand prints the command being executed with secret redaction
func logCommand(cmd string, args ...string) {
	redacted := make([]string, len(args))
	for i, arg := range args {
		redacted[i] = redactSecrets(arg)
	}
	fmt.Printf("EXEC: %s", cmd)
	if len(redacted) > 0 {
		fmt.Printf(" %s", strings.Join(redacted, " "))
	}
	fmt.Println()
}

// logEnv prints environment variables, redacting secrets
func logEnv(varNames []string) {
	if len(varNames) == 0 {
		return
	}
	fmt.Println("\nENV:")
	for _, name := range varNames {
		value := os.Getenv(name)
		if isSecretVar(name) {
			if value != "" {
				fmt.Printf(" - %s=***redacted*** (present)\n", name)
			} else {
				fmt.Printf(" - %s=(not set)\n", name)
			}
		} else {
			if value != "" {
				fmt.Printf(" - %s=%s\n", name, value)
			} else {
				fmt.Printf(" - %s=(not set)\n", name)
			}
		}
	}
	fmt.Println()
}

// redactSecrets redacts sensitive information in command arguments
func redactSecrets(arg string) string {
	// Redact --token=xxx, --password=xxx, --secret=xxx, etc.
	secretFlags := []string{"token", "password", "secret", "key", "auth"}
	for _, flag := range secretFlags {
		pattern := fmt.Sprintf(`--(%s[^=]*)=.+`, flag)
		re := regexp.MustCompile(pattern)
		if re.MatchString(arg) {
			parts := strings.SplitN(arg, "=", 2)
			return parts[0] + "=***redacted***"
		}
	}

	// Redact environment variable assignments like TOKEN=xxx
	if strings.Contains(arg, "=") {
		parts := strings.SplitN(arg, "=", 2)
		if isSecretVar(parts[0]) {
			return parts[0] + "=***redacted***"
		}
	}

	return arg
}

// isSecretVar detects if an environment variable name contains secrets
func isSecretVar(name string) bool {
	upper := strings.ToUpper(name)
	secretSuffixes := []string{"_TOKEN", "_PASSWORD", "_SECRET", "_KEY", "_AUTH"}
	secretPrefixes := []string{"GITHUB_", "GH_", "DOCKER_", "NPM_", "HOMEBREW_"}

	// Check suffixes
	for _, suffix := range secretSuffixes {
		if strings.HasSuffix(upper, suffix) {
			return true
		}
	}

	// Check exact matches
	exactMatches := []string{"TOKEN", "PASSWORD", "SECRET", "API_KEY"}
	for _, match := range exactMatches {
		if upper == match {
			return true
		}
	}

	// Check if it starts with known secret prefixes and ends with TOKEN/KEY
	for _, prefix := range secretPrefixes {
		if strings.HasPrefix(upper, prefix) {
			for _, suffix := range secretSuffixes {
				if strings.HasSuffix(upper, suffix) {
					return true
				}
			}
		}
	}

	return false
}

// logSuccess prints a success message with a checkmark
func logSuccess(message string) {
	fmt.Printf("✓ %s\n", message)
}

// logError prints an error message with an X
func logError(message string) {
	fmt.Printf("✗ %s\n", message)
}

// logWarning prints a warning message
func logWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}
