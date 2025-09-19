package staticanalysis

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/salsadigitalauorg/shipshape/pkg/command"
	"github.com/salsadigitalauorg/shipshape/pkg/data"
	"github.com/salsadigitalauorg/shipshape/pkg/env"
	"github.com/salsadigitalauorg/shipshape/pkg/fact"
	"github.com/salsadigitalauorg/shipshape/pkg/plugin"
)

// StaticAnalysis represents a static analysis tool execution
type StaticAnalysis struct {
	fact.BaseFact       `yaml:",inline"`
	env.BaseEnvResolver `yaml:",inline"`

	// Tool identification
	Tool string `yaml:"tool"`

	// Common configuration
	Binary string   `yaml:"binary,omitempty"`
	Config string   `yaml:"config,omitempty"`
	Paths  []string `yaml:"paths,omitempty"`

	// Advanced configuration
	Args        []string          `yaml:"args,omitempty"`
	Presets     map[string]string `yaml:"presets,omitempty"`
	Environment map[string]string `yaml:"environment,omitempty"`

	// Behavior control
	IgnoreError     bool     `yaml:"ignore-error,omitempty"`
	OutputFormat    string   `yaml:"output-format,omitempty"`
	FailurePatterns []string `yaml:"failure-patterns,omitempty"`
}

// StaticAnalysisResult represents the structured output of a static analysis run
type StaticAnalysisResult struct {
	Success     bool    `json:"success"`
	ExitCode    int     `json:"exit_code"`
	Output      string  `json:"output"`
	ErrorOutput string  `json:"error_output"`
	Issues      []Issue `json:"issues,omitempty"`
	Tool        string  `json:"tool"`
	Duration    string  `json:"duration"`
}

// Issue represents a single issue found by static analysis
type Issue struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column,omitempty"`
	Message  string `json:"message"`
	Rule     string `json:"rule,omitempty"`
	Severity string `json:"severity,omitempty"`
}

// ToolConfig defines the configuration for a specific tool
type ToolConfig struct {
	Binary           string
	Args             []string
	ConfigFlag       string
	OutputFormats    map[string][]string
	SuccessExitCodes []int
}

// Built-in tool configurations
var toolDefaults = map[string]ToolConfig{
	"phpstan": {
		Binary:     "vendor/phpstan/phpstan/phpstan",
		Args:       []string{"analyse", "--no-progress"},
		ConfigFlag: "--configuration=",
		OutputFormats: map[string][]string{
			"json":  {"--error-format=json"},
			"table": {"--error-format=table"},
		},
		SuccessExitCodes: []int{0}, // Only 0 is true success, we'll analyze content for exit code 1
	},
	"eslint": {
		Binary:     "npx eslint",
		Args:       []string{},
		ConfigFlag: "--config ",
		OutputFormats: map[string][]string{
			"json":    {"--format", "json"},
			"stylish": {"--format", "stylish"},
		},
		SuccessExitCodes: []int{0}, // Only 0 is true success, we'll analyze content for exit code 1
	},
	"pylint": {
		Binary:     "pylint",
		Args:       []string{},
		ConfigFlag: "--rcfile=",
		OutputFormats: map[string][]string{
			"json": {"--output-format=json"},
			"text": {"--output-format=text"},
		},
		SuccessExitCodes: []int{0, 4, 8, 16}, // Pylint has various non-error exit codes
	},
}

//go:generate go run ../../../cmd/gen.go fact-plugin --package=staticanalysis

func init() {
	fact.Manager().RegisterFactory("static-analysis", func(n string) fact.Facter {
		return New(n)
	})
}

func New(id string) *StaticAnalysis {
	return &StaticAnalysis{
		BaseFact: fact.BaseFact{
			BasePlugin: plugin.BasePlugin{
				Id: id,
			},
			Format: data.FormatRaw,
		},
	}
}

func (p *StaticAnalysis) GetName() string {
	return "static-analysis"
}

func (p *StaticAnalysis) Collect() {
	contextLogger := log.WithFields(log.Fields{
		"fact-plugin": p.GetName(),
		"fact":        p.GetId(),
		"tool":        p.Tool,
	})

	startTime := time.Now()

	// Build command
	cmd, args, err := p.BuildCommand()
	if err != nil {
		contextLogger.WithError(err).Error("failed to build command")
		p.AddErrors(err)
		return
	}

	contextLogger.WithFields(log.Fields{
		"cmd":  cmd,
		"args": args,
	}).Debug("executing static analysis")

	// Log the resolved configuration for debugging
	if p.Config != "" {
		contextLogger.WithField("resolved-config", args).Debug("config resolution details")
	}

	// Execute command
	commander := command.ShellCommander(cmd, args...)

	// Set environment variables if specified
	if len(p.Environment) > 0 {
		envMap, err := p.GetEnvMap()
		if err != nil {
			contextLogger.WithError(err).Warn("failed to get environment map")
		}
		for key, value := range p.Environment {
			resolved, err := env.ResolveValue(envMap, value)
			if err != nil {
				contextLogger.WithError(err).WithField("key", key).Warn("failed to resolve environment variable")
				resolved = value // Use original value as fallback
			}
			os.Setenv(key, resolved)
		}
	}

	output, err := commander.Output()
	contextLogger.WithField("output", string(output)).Debug("static analysis output")
	duration := time.Since(startTime)

	// Build result
	result := StaticAnalysisResult{
		Tool:     p.Tool,
		Duration: duration.String(),
		Output:   strings.TrimSpace(string(output)),
	}

	if err != nil {
		result.ExitCode = command.GetExitCode(err)
		result.ErrorOutput = command.GetMsgFromCommandError(err)
		result.Success = p.IsSuccessExitCode(result.ExitCode)

		// Special handling for exit code 1 - could be "issues found" or "execution error"
		if result.ExitCode == 1 && !result.Success {
			isSuccessWithIssues := p.IsSuccessfulWithIssues(result.Output, result.ErrorOutput)
			contextLogger.WithFields(log.Fields{
				"exit-code":              result.ExitCode,
				"is-success-with-issues": isSuccessWithIssues,
				"output-sample":          fmt.Sprintf("%.200s", result.Output),
				"stderr-sample":          fmt.Sprintf("%.200s", result.ErrorOutput),
			}).Info("analyzing exit code 1 content") // Info level for debugging

			if isSuccessWithIssues {
				result.Success = true
				contextLogger.
					WithField("exit-code", result.ExitCode).
					Info("static analysis completed with issues found") // Changed to Info
			} else {
				// This is an actual execution error
				if !p.IgnoreError {
					contextLogger.
						WithError(err).
						WithField("exit-code", result.ExitCode).
						WithField("error", result.ErrorOutput).
						WithField("output", result.Output).
						Error("static analysis tool execution failed")
					p.AddErrors(err)
				}
			}
		} else if !p.IgnoreError && !result.Success {
			// Other non-zero exit codes are definite failures
			contextLogger.
				WithError(err).
				WithField("exit-code", result.ExitCode).
				WithField("error", result.ErrorOutput).
				Error("static analysis tool execution failed")
			p.AddErrors(err)
		}
	} else {
		result.ExitCode = 0
		result.Success = true
	}

	// Try to parse issues if output format supports it
	if p.OutputFormat == "json" || strings.Contains(p.Tool, "json") {
		result.Issues = p.ParseIssues(result.Output)
	}

	// Apply failure patterns if specified
	if len(p.FailurePatterns) > 0 {
		for _, pattern := range p.FailurePatterns {
			if strings.Contains(result.Output, pattern) || strings.Contains(result.ErrorOutput, pattern) {
				result.Success = false
				break
			}
		}
	}

	// Store result as JSON
	resultJSON, err := json.Marshal(result)
	if err != nil {
		contextLogger.WithError(err).Error("failed to marshal result")
		p.AddErrors(err)
		return
	}

	p.SetData(resultJSON)

	logMessage := "static analysis completed"
	if result.ExitCode == 0 {
		logMessage += " with no issues found"
	} else if result.Success && result.ExitCode == 1 {
		logMessage += " and found issues to analyze"
	} else if result.ExitCode == 1 {
		logMessage += " but marked as failed (execution error)"
	}

	contextLogger.WithFields(log.Fields{
		"success":        result.Success,
		"exit-code":      result.ExitCode,
		"duration":       result.Duration,
		"issues":         len(result.Issues),
		"output-preview": fmt.Sprintf("%.300s", result.Output), // First 300 chars
		"error-preview":  fmt.Sprintf("%.200s", result.ErrorOutput),
	}).Info(logMessage) // Changed to Info level for easier debugging
}

func (p *StaticAnalysis) BuildCommand() (string, []string, error) {
	// Get tool config or use custom
	var config ToolConfig
	var exists bool

	if p.Tool == "custom" {
		// For custom tools, use provided binary and args as-is
		if p.Binary == "" {
			return "", nil, fmt.Errorf("binary must be specified for custom tool")
		}
		return p.Binary, p.Args, nil
	}

	if config, exists = toolDefaults[p.Tool]; !exists {
		return "", nil, fmt.Errorf("unsupported tool: %s", p.Tool)
	}

	// Determine binary
	binary := p.Binary
	if binary == "" {
		binary = config.Binary
	}

	// Start with tool's default args
	args := make([]string, 0, len(config.Args)+len(p.Args)+10)
	args = append(args, config.Args...)

	// Add configuration if specified
	if p.Config != "" {
		envMap, err := p.GetEnvMap()
		if err != nil {
			return "", nil, fmt.Errorf("failed to get environment map: %w", err)
		}

		// Resolve environment variables in config path
		configPath, err := env.ResolveValue(envMap, p.Config)
		if err != nil {
			return "", nil, fmt.Errorf("failed to resolve config path '%s': %w", p.Config, err)
		}

		// If no environment resolution happened, try basic OS env substitution
		if configPath == p.Config && strings.Contains(configPath, "${") {
			// Fallback: try to resolve using OS environment for basic cases
			if strings.Contains(configPath, "${PHPSTAN_CONFIG:-") {
				start := strings.Index(configPath, "${PHPSTAN_CONFIG:-") + len("${PHPSTAN_CONFIG:-")
				end := strings.Index(configPath[start:], "}")
				if end != -1 {
					defaultValue := configPath[start : start+end]
					if phpstanConfig := os.Getenv("PHPSTAN_CONFIG"); phpstanConfig != "" {
						configPath = phpstanConfig
					} else {
						configPath = defaultValue
					}
				}
			}
		}

		if config.ConfigFlag != "" {
			args = append(args, config.ConfigFlag+configPath)
		}
	}

	// Add output format if specified and supported
	if p.OutputFormat != "" {
		if formatArgs, exists := config.OutputFormats[p.OutputFormat]; exists {
			args = append(args, formatArgs...)
		}
	}

	// Add presets (tool-specific settings)
	if len(p.Presets) > 0 {
		envMap, err := p.GetEnvMap()
		if err != nil {
			return "", nil, fmt.Errorf("failed to get environment map for presets: %w", err)
		}
		for key, value := range p.Presets {
			resolvedValue, err := env.ResolveValue(envMap, value)
			if err != nil {
				return "", nil, fmt.Errorf("failed to resolve preset %s: %w", key, err)
			}
			args = append(args, fmt.Sprintf("--%s=%s", key, resolvedValue))
		}
	}

	// Add custom args
	args = append(args, p.Args...)

	// Add paths
	if len(p.Paths) > 0 {
		envMap, err := p.GetEnvMap()
		if err != nil {
			return "", nil, fmt.Errorf("failed to get environment map for paths: %w", err)
		}
		for _, path := range p.Paths {
			resolvedPath, err := env.ResolveValue(envMap, path)
			if err != nil {
				return "", nil, fmt.Errorf("failed to resolve path %s: %w", path, err)
			}
			args = append(args, resolvedPath)
		}
	}

	return binary, args, nil
}

func (p *StaticAnalysis) IsSuccessExitCode(exitCode int) bool {
	if p.Tool == "custom" {
		return exitCode == 0
	}

	config, exists := toolDefaults[p.Tool]
	if !exists {
		return exitCode == 0
	}

	for _, successCode := range config.SuccessExitCodes {
		if exitCode == successCode {
			return true
		}
	}
	return false
}

func (p *StaticAnalysis) ParseIssues(output string) []Issue {
	// Basic JSON parsing for common formats
	// This is a simplified parser - in a real implementation,
	// you'd want tool-specific parsers

	if strings.TrimSpace(output) == "" {
		return nil
	}

	// Try to parse as JSON array of issues (common format)
	var issues []Issue
	if err := json.Unmarshal([]byte(output), &issues); err != nil {
		// If direct parsing fails, try to extract from common structures
		return p.extractIssuesFromOutput(output)
	}

	return issues
}

func (p *StaticAnalysis) extractIssuesFromOutput(output string) []Issue {
	// This would contain tool-specific parsing logic
	// For now, return empty - could be extended per tool
	return []Issue{}
}

// IsSuccessfulWithIssues analyzes output content to determine if exit code 1
// represents "successfully found issues" vs "execution error"
func (p *StaticAnalysis) IsSuccessfulWithIssues(stdout, stderr string) bool {
	switch p.Tool {
	case "phpstan":
		return p.isPhpstanSuccessWithIssues(stdout, stderr)
	case "eslint":
		return p.isEslintSuccessWithIssues(stdout, stderr)
	case "pylint":
		return p.isPylintSuccessWithIssues(stdout, stderr)
	default:
		// For custom tools, use generic heuristics
		return p.isGenericSuccessWithIssues(stdout, stderr)
	}
}

func (p *StaticAnalysis) isPhpstanSuccessWithIssues(stdout, stderr string) bool {
	// PHPStan success indicators (even with issues found):
	successPatterns := []string{
		"Found",         // "Found X error" or "Found X errors"
		"[ERROR] Found", // "[ERROR] Found X error"
		"[OK]",          // Sometimes used for summary
		"no errors",     // When no errors found but exit is still 1 (edge case)
		"Line   ",       // Table format header indicates successful analysis
		"------",        // Table format separator indicates successful analysis
	}

	// PHPStan error indicators:
	errorPatterns := []string{
		"does not exist",      // Config file issues
		"not found",           // File/path issues
		"Configuration error", // Config parsing issues
		"Fatal error",         // PHP fatal errors
		"Parse error",         // PHP parse errors
		"could not be read",   // File access issues
		"invalid",             // Invalid config/options
		"Unable to",           // General failure messages
	}

	// Check for error patterns first (more specific)
	combinedOutput := stdout + " " + stderr
	for _, pattern := range errorPatterns {
		if strings.Contains(combinedOutput, pattern) {
			return false
		}
	}

	// Check for success patterns
	for _, pattern := range successPatterns {
		if strings.Contains(stdout, pattern) {
			return true
		}
	}

	// Default: if no clear error patterns and has substantial output, assume success with issues
	return len(strings.TrimSpace(stdout)) > 10
}

func (p *StaticAnalysis) isEslintSuccessWithIssues(stdout, stderr string) bool {
	// ESLint success indicators:
	successPatterns := []string{
		"problem", // "1 problem" or "5 problems"
		"error",   // "1 error"
		"warning", // "1 warning"
		"✖",       // ESLint's error symbol
	}

	// ESLint error indicators:
	errorPatterns := []string{
		"No such file",
		"ENOENT",
		"Configuration",
		"Failed to load",
		"Cannot find",
		"Parsing error", // Config parsing issues
	}

	combinedOutput := stdout + " " + stderr
	for _, pattern := range errorPatterns {
		if strings.Contains(combinedOutput, pattern) {
			return false
		}
	}

	for _, pattern := range successPatterns {
		if strings.Contains(stdout, pattern) {
			return true
		}
	}

	return len(strings.TrimSpace(stdout)) > 10
}

func (p *StaticAnalysis) isPylintSuccessWithIssues(stdout, stderr string) bool {
	// Pylint is more complex with different exit codes, but for exit code 1:
	successPatterns := []string{
		"Your code has been rated", // Standard pylint summary
		"warning",
		"error",
		"convention",
		"refactor",
	}

	errorPatterns := []string{
		"No such file",
		"can't open file",
		"Fatal error",
		"ImportError",
		"SyntaxError",
		"Configuration file",
	}

	combinedOutput := stdout + " " + stderr
	for _, pattern := range errorPatterns {
		if strings.Contains(combinedOutput, pattern) {
			return false
		}
	}

	for _, pattern := range successPatterns {
		if strings.Contains(stdout, pattern) {
			return true
		}
	}

	return len(strings.TrimSpace(stdout)) > 10
}

func (p *StaticAnalysis) isGenericSuccessWithIssues(stdout, stderr string) bool {
	// Generic heuristics for custom tools
	errorPatterns := []string{
		"not found",
		"does not exist",
		"No such file",
		"Permission denied",
		"Configuration error",
		"Fatal error",
		"command not found",
		"ENOENT",
		"invalid",
		"Unable to",
		"Failed to",
		"Cannot",
	}

	combinedOutput := stdout + " " + stderr
	for _, pattern := range errorPatterns {
		if strings.Contains(combinedOutput, pattern) {
			return false
		}
	}

	// If no obvious error patterns and has output, assume it's reporting issues
	return len(strings.TrimSpace(stdout)) > 0
}
