package analyse

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/salsadigitalauorg/shipshape/pkg/breach"
	"github.com/salsadigitalauorg/shipshape/pkg/data"
	"github.com/salsadigitalauorg/shipshape/pkg/plugin"
)

// StaticAnalysisBreaches analyzes static analysis results for issues
type StaticAnalysisBreaches struct {
	BaseAnalyser `yaml:",inline"`

	// What to check for
	CheckSuccess bool     `yaml:"check-success,omitempty"` // Default: true
	CheckIssues  bool     `yaml:"check-issues,omitempty"`  // Default: true
	MinSeverity  string   `yaml:"min-severity,omitempty"`  // error, warning, info
	IgnoreRules  []string `yaml:"ignore-rules,omitempty"`
	MaxIssues    int      `yaml:"max-issues,omitempty"` // Fail if > N issues (0 = fail on any)
}

// StaticAnalysisResult mirrors the struct from the fact plugin
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

func init() {
	Manager().RegisterFactory("static-analysis:breaches", func(id string) Analyser {
		return NewStaticAnalysisBreaches(id)
	})
}

func NewStaticAnalysisBreaches(id string) *StaticAnalysisBreaches {
	return &StaticAnalysisBreaches{
		BaseAnalyser: BaseAnalyser{
			BasePlugin: plugin.BasePlugin{
				Id: id,
			},
		},
		CheckSuccess: true,
		CheckIssues:  true,
		MaxIssues:    0, // Fail on any issues by default
	}
}

func (p *StaticAnalysisBreaches) GetName() string {
	return "static-analysis:breaches"
}

func (p *StaticAnalysisBreaches) Analyse() {
	input := p.GetInput()
	if input == nil {
		return
	}

	contextLogger := log.WithFields(log.Fields{
		"analyser": p.GetName(),
		"input":    p.InputName,
	})

	contextLogger.WithField("input-format", input.GetFormat()).Debug("analysing static analysis results")

	var result StaticAnalysisResult

	// Parse the input data
	switch input.GetFormat() {
	case data.FormatRaw:
		// Parse JSON result from fact plugin
		rawData := input.GetData().([]byte)
		if err := json.Unmarshal(rawData, &result); err != nil {
			contextLogger.WithError(err).Error("failed to parse static analysis result")
			breach.EvaluateTemplate(p, &breach.ValueBreach{
				Value: fmt.Sprintf("Failed to parse static analysis result: %v", err),
			}, p.Remediation)
			return
		}
	case data.FormatString:
		// Try to parse as JSON string
		stringData := data.AsString(input.GetData())
		if err := json.Unmarshal([]byte(stringData), &result); err != nil {
			contextLogger.WithError(err).Error("failed to parse static analysis result from string")
			breach.EvaluateTemplate(p, &breach.ValueBreach{
				Value: fmt.Sprintf("Failed to parse static analysis result: %v", err),
			}, p.Remediation)
			return
		}
	default:
		contextLogger.WithField("format", input.GetFormat()).Error("unsupported input format")
		breach.EvaluateTemplate(p, &breach.ValueBreach{
			Value: fmt.Sprintf("Unsupported input format: %s", input.GetFormat()),
		}, p.Remediation)
		return
	}

	contextLogger.WithFields(log.Fields{
		"tool":           result.Tool,
		"success":        result.Success,
		"exit-code":      result.ExitCode,
		"issues":         len(result.Issues),
		"duration":       result.Duration,
		"output-preview": fmt.Sprintf("%.200s", result.Output), // First 200 chars of output
	}).Info("parsed static analysis result") // Changed to Info so it shows by default

	// Check success status if enabled
	if p.CheckSuccess && !result.Success {
		breach.EvaluateTemplate(p, &breach.ValueBreach{
			Value: fmt.Sprintf("%s failed with exit code %d", result.Tool, result.ExitCode),
		}, p.Remediation)
	}

	// Check issues if enabled
	if p.CheckIssues {
		filteredIssues := p.FilterIssues(result.Issues)

		contextLogger.WithFields(log.Fields{
			"filtered-issues": len(filteredIssues),
			"max-issues":      p.MaxIssues,
			"has-output":      len(result.Output) > 0,
			"output-length":   len(result.Output),
		}).Info("checking for issues")

		// Determine if we should report issues based on max-issues threshold
		shouldReport := false
		if p.MaxIssues == 0 {
			// If max-issues is 0, report any issues found (structured or unstructured)
			shouldReport = len(filteredIssues) > 0 || (result.Output != "" && !result.Success)
		} else {
			// If max-issues > 0, only report if we exceed the threshold
			shouldReport = len(filteredIssues) > p.MaxIssues
		}

		contextLogger.WithFields(log.Fields{
			"should-report":          shouldReport,
			"max-issues-is-zero":     p.MaxIssues == 0,
			"has-structured-issues":  len(filteredIssues) > 0,
			"has-output-and-success": result.Output != "" && result.Success,
		}).Info("determining whether to report issues")

		if shouldReport {
			if len(result.Issues) > 0 {
				// Report structured issues
				p.reportDetailedIssues(filteredIssues, result.Tool)
			} else if result.Output != "" {
				// Fall back to raw output when no structured parsing succeeded
				breach.EvaluateTemplate(p, &breach.ValueBreach{
					Value: fmt.Sprintf("%s found issues:\n%s", result.Tool, result.Output),
				}, p.Remediation)
			} else {
				// Generic failure message
				breach.EvaluateTemplate(p, &breach.ValueBreach{
					Value: fmt.Sprintf("%s detected %d issue(s)", result.Tool, len(filteredIssues)),
				}, p.Remediation)
			}
		}
	}
}

func (p *StaticAnalysisBreaches) FilterIssues(issues []Issue) []Issue {
	if len(issues) == 0 {
		return issues
	}

	filtered := make([]Issue, 0, len(issues))

	for _, issue := range issues {
		// Filter by severity if specified
		if p.MinSeverity != "" && !p.MeetsSeverity(issue.Severity, p.MinSeverity) {
			continue
		}

		// Filter by ignored rules
		if len(p.IgnoreRules) > 0 && p.IsRuleIgnored(issue.Rule) {
			continue
		}

		filtered = append(filtered, issue)
	}

	return filtered
}

func (p *StaticAnalysisBreaches) MeetsSeverity(issueSeverity, minSeverity string) bool {
	// Define severity levels (highest to lowest)
	severityLevels := map[string]int{
		"error":   3,
		"warning": 2,
		"info":    1,
	}

	issueLevel, issueExists := severityLevels[strings.ToLower(issueSeverity)]
	minLevel, minExists := severityLevels[strings.ToLower(minSeverity)]

	if !issueExists || !minExists {
		// If severity is unknown, include the issue
		return true
	}

	return issueLevel >= minLevel
}

func (p *StaticAnalysisBreaches) IsRuleIgnored(rule string) bool {
	for _, ignoredRule := range p.IgnoreRules {
		if rule == ignoredRule {
			return true
		}
	}
	return false
}

func (p *StaticAnalysisBreaches) reportDetailedIssues(issues []Issue, tool string) {
	// Group issues by file for cleaner reporting
	fileIssues := make(map[string][]Issue)
	for _, issue := range issues {
		fileIssues[issue.File] = append(fileIssues[issue.File], issue)
	}

	for file, issues := range fileIssues {
		issueStrings := make([]string, len(issues))
		for i, issue := range issues {
			location := fmt.Sprintf("line %d", issue.Line)
			if issue.Column > 0 {
				location += fmt.Sprintf(", column %d", issue.Column)
			}

			issueStr := fmt.Sprintf("%s: %s", location, issue.Message)
			if issue.Rule != "" {
				issueStr += fmt.Sprintf(" (%s)", issue.Rule)
			}
			if issue.Severity != "" {
				issueStr += fmt.Sprintf(" [%s]", issue.Severity)
			}

			issueStrings[i] = issueStr
		}

		breach.EvaluateTemplate(p, &breach.KeyValuesBreach{
			Key:    fmt.Sprintf("%s issues in %s", tool, file),
			Values: issueStrings,
		}, p.Remediation)
	}
}
