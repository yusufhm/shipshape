package analyse_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/salsadigitalauorg/shipshape/pkg/analyse"
	"github.com/salsadigitalauorg/shipshape/pkg/breach"
	"github.com/salsadigitalauorg/shipshape/pkg/data"
	"github.com/salsadigitalauorg/shipshape/pkg/fact/testdata"
	"github.com/salsadigitalauorg/shipshape/pkg/internal"
	"github.com/salsadigitalauorg/shipshape/pkg/plugin"
)

func TestStaticAnalysisBreachesInit(t *testing.T) {
	assert := assert.New(t)

	// Test that the plugin is registered
	plugin := Manager().GetFactories()["static-analysis:breaches"]("TestStaticAnalysisBreaches")
	assert.NotNil(plugin)
	analyser, ok := plugin.(*StaticAnalysisBreaches)
	assert.True(ok)
	assert.Equal("TestStaticAnalysisBreaches", analyser.Id)

	// Test default values
	assert.True(analyser.CheckSuccess)
	assert.True(analyser.CheckIssues)
	assert.Equal(0, analyser.MaxIssues)
}

func TestStaticAnalysisBreachesPluginName(t *testing.T) {
	instance := NewStaticAnalysisBreaches("TestStaticAnalysisBreaches")
	assert.Equal(t, "static-analysis:breaches", instance.GetName())
}

func TestStaticAnalysisBreachesAnalyse(t *testing.T) {
	// Helper to create JSON result data
	createResultJSON := func(result StaticAnalysisResult) []byte {
		data, _ := json.Marshal(result)
		return data
	}

	// Helper to create string JSON result data
	createResultJSONString := func(result StaticAnalysisResult) string {
		data, _ := json.Marshal(result)
		return string(data)
	}

	tt := []internal.AnalyseTest{
		// Success case - no breaches expected
		{
			Name: "success_no_issues",
			Input: testdata.New(
				"testFact",
				data.FormatRaw,
				createResultJSON(StaticAnalysisResult{
					Success:  true,
					ExitCode: 0,
					Tool:     "phpstan",
					Output:   "No errors found",
				}),
			),
			Analyser: &StaticAnalysisBreaches{
				BaseAnalyser: BaseAnalyser{
					BasePlugin: plugin.BasePlugin{Id: "TestStaticAnalysisBreaches"},
					InputName:  "testFact",
				},
				CheckSuccess: true,
				CheckIssues:  true,
				MaxIssues:    0,
			},
			ExpectedBreaches: []breach.Breach{},
		},

		// Failure case - tool failed
		{
			Name: "tool_failed",
			Input: testdata.New(
				"testFact",
				data.FormatRaw,
				createResultJSON(StaticAnalysisResult{
					Success:  false,
					ExitCode: 1,
					Tool:     "phpstan",
					Output:   "Found 2 errors",
				}),
			),
			Analyser: &StaticAnalysisBreaches{
				BaseAnalyser: BaseAnalyser{
					BasePlugin: plugin.BasePlugin{Id: "TestStaticAnalysisBreaches"},
					InputName:  "testFact",
				},
				CheckSuccess: true,
				CheckIssues:  true,
				MaxIssues:    0,
			},
			ExpectedBreaches: []breach.Breach{
				&breach.ValueBreach{
					BreachType: "value",
					CheckName:  "TestStaticAnalysisBreaches",
					Value:      "phpstan failed with exit code 1",
				},
				&breach.ValueBreach{
					BreachType: "value",
					CheckName:  "TestStaticAnalysisBreaches",
					Value:      "phpstan found issues:\nFound 2 errors",
				},
			},
		},

		// Issues found with detailed reporting
		{
			Name: "issues_with_details",
			Input: testdata.New(
				"testFact",
				data.FormatRaw,
				createResultJSON(StaticAnalysisResult{
					Success:  false,
					ExitCode: 1,
					Tool:     "phpstan",
					Issues: []Issue{
						{
							File:     "src/Test.php",
							Line:     10,
							Message:  "Undefined variable $test",
							Rule:     "undefined.var",
							Severity: "error",
						},
						{
							File:     "src/Test.php",
							Line:     15,
							Message:  "Missing return type",
							Rule:     "missing.return.type",
							Severity: "warning",
						},
					},
				}),
			),
			Analyser: &StaticAnalysisBreaches{
				BaseAnalyser: BaseAnalyser{
					BasePlugin: plugin.BasePlugin{Id: "TestStaticAnalysisBreaches"},
					InputName:  "testFact",
				},
				CheckSuccess: false, // Only check issues, not success
				CheckIssues:  true,
				MaxIssues:    0,
			},
			ExpectedBreaches: []breach.Breach{
				&breach.KeyValuesBreach{
					BreachType: "key-values",
					CheckName:  "TestStaticAnalysisBreaches",
					Key:        "phpstan issues in src/Test.php",
					Values: []string{
						"line 10: Undefined variable $test (undefined.var) [error]",
						"line 15: Missing return type (missing.return.type) [warning]",
					},
				},
			},
		},

		// Test severity filtering
		{
			Name: "severity_filtering",
			Input: testdata.New(
				"testFact",
				data.FormatRaw,
				createResultJSON(StaticAnalysisResult{
					Success:  false,
					ExitCode: 1,
					Tool:     "eslint",
					Issues: []Issue{
						{
							File:     "src/test.js",
							Line:     5,
							Message:  "Syntax error",
							Severity: "error",
						},
						{
							File:     "src/test.js",
							Line:     10,
							Message:  "Unused variable",
							Severity: "warning",
						},
					},
				}),
			),
			Analyser: &StaticAnalysisBreaches{
				BaseAnalyser: BaseAnalyser{
					BasePlugin: plugin.BasePlugin{Id: "TestStaticAnalysisBreaches"},
					InputName:  "testFact",
				},
				CheckSuccess: false,
				CheckIssues:  true,
				MaxIssues:    0,
				MinSeverity:  "error", // Only report errors
			},
			ExpectedBreaches: []breach.Breach{
				&breach.KeyValuesBreach{
					BreachType: "key-values",
					CheckName:  "TestStaticAnalysisBreaches",
					Key:        "eslint issues in src/test.js",
					Values: []string{
						"line 5: Syntax error [error]",
					},
				},
			},
		},

		// Test rule ignoring
		{
			Name: "rule_ignoring",
			Input: testdata.New(
				"testFact",
				data.FormatRaw,
				createResultJSON(StaticAnalysisResult{
					Success:  false,
					ExitCode: 1,
					Tool:     "pylint",
					Issues: []Issue{
						{
							File:     "src/test.py",
							Line:     5,
							Message:  "Missing docstring",
							Rule:     "missing-docstring",
							Severity: "warning",
						},
						{
							File:     "src/test.py",
							Line:     10,
							Message:  "Unused variable",
							Rule:     "unused-variable",
							Severity: "warning",
						},
					},
				}),
			),
			Analyser: &StaticAnalysisBreaches{
				BaseAnalyser: BaseAnalyser{
					BasePlugin: plugin.BasePlugin{Id: "TestStaticAnalysisBreaches"},
					InputName:  "testFact",
				},
				CheckSuccess: false,
				CheckIssues:  true,
				MaxIssues:    0,
				IgnoreRules:  []string{"missing-docstring"}, // Ignore this rule
			},
			ExpectedBreaches: []breach.Breach{
				&breach.KeyValuesBreach{
					BreachType: "key-values",
					CheckName:  "TestStaticAnalysisBreaches",
					Key:        "pylint issues in src/test.py",
					Values: []string{
						"line 10: Unused variable (unused-variable) [warning]",
					},
				},
			},
		},

		// Test max issues threshold
		{
			Name: "max_issues_threshold",
			Input: testdata.New(
				"testFact",
				data.FormatRaw,
				createResultJSON(StaticAnalysisResult{
					Success:  false,
					ExitCode: 1,
					Tool:     "phpstan",
					Issues: []Issue{
						{
							File:     "src/test.php",
							Line:     5,
							Message:  "Error 1",
							Severity: "error",
						},
						{
							File:     "src/test.php",
							Line:     10,
							Message:  "Error 2",
							Severity: "error",
						},
					},
				}),
			),
			Analyser: &StaticAnalysisBreaches{
				BaseAnalyser: BaseAnalyser{
					BasePlugin: plugin.BasePlugin{Id: "TestStaticAnalysisBreaches"},
					InputName:  "testFact",
				},
				CheckSuccess: false,
				CheckIssues:  true,
				MaxIssues:    5, // Allow up to 5 issues
			},
			ExpectedBreaches: []breach.Breach{}, // No breaches expected as under threshold
		},

		// Test string input format
		{
			Name: "string_input_format",
			Input: testdata.New(
				"testFact",
				data.FormatString,
				createResultJSONString(StaticAnalysisResult{
					Success:  false,
					ExitCode: 1,
					Tool:     "phpstan",
					Output:   "Found errors",
				}),
			),
			Analyser: &StaticAnalysisBreaches{
				BaseAnalyser: BaseAnalyser{
					BasePlugin: plugin.BasePlugin{Id: "TestStaticAnalysisBreaches"},
					InputName:  "testFact",
				},
				CheckSuccess: true,
				CheckIssues:  true,
				MaxIssues:    0,
			},
			ExpectedBreaches: []breach.Breach{
				&breach.ValueBreach{
					BreachType: "value",
					CheckName:  "TestStaticAnalysisBreaches",
					Value:      "phpstan failed with exit code 1",
				},
				&breach.ValueBreach{
					BreachType: "value",
					CheckName:  "TestStaticAnalysisBreaches",
					Value:      "phpstan found issues:\nFound errors",
				},
			},
		},

		// Test invalid JSON input
		{
			Name: "invalid_json",
			Input: testdata.New(
				"testFact",
				data.FormatRaw,
				[]byte("invalid json"),
			),
			Analyser: &StaticAnalysisBreaches{
				BaseAnalyser: BaseAnalyser{
					BasePlugin: plugin.BasePlugin{Id: "TestStaticAnalysisBreaches"},
					InputName:  "testFact",
				},
				CheckSuccess: true,
				CheckIssues:  true,
				MaxIssues:    0,
			},
			ExpectedBreaches: []breach.Breach{
				&breach.ValueBreach{
					BreachType: "value",
					CheckName:  "TestStaticAnalysisBreaches",
					Value:      "Failed to parse static analysis result: invalid character 'i' looking for beginning of value",
				},
			},
		},

		// Test unsupported input format
		{
			Name: "unsupported_format",
			Input: testdata.New(
				"testFact",
				data.FormatMapString,
				map[string]string{"test": "data"},
			),
			Analyser: &StaticAnalysisBreaches{
				BaseAnalyser: BaseAnalyser{
					BasePlugin: plugin.BasePlugin{Id: "TestStaticAnalysisBreaches"},
					InputName:  "testFact",
				},
			},
			ExpectedBreaches: []breach.Breach{
				&breach.ValueBreach{
					BreachType: "value",
					CheckName:  "TestStaticAnalysisBreaches",
					Value:      "Unsupported input format: map-string",
				},
			},
		},
	}

	for _, test := range tt {
		t.Run(test.Name, func(t *testing.T) {
			internal.TestAnalyse(t, test)
		})
	}
}

func TestStaticAnalysisBreachesSeverityFilter(t *testing.T) {
	plugin := &StaticAnalysisBreaches{}

	tests := []struct {
		name          string
		minSeverity   string
		issueSeverity string
		expected      bool
	}{
		{"error_min_error_issue", "error", "error", true},
		{"error_min_warning_issue", "error", "warning", false},
		{"warning_min_error_issue", "warning", "error", true},
		{"warning_min_warning_issue", "warning", "warning", true},
		{"warning_min_info_issue", "warning", "info", false},
		{"unknown_severity", "error", "unknown", true}, // Include unknown severities
		{"empty_severity", "error", "", true},          // Include empty severities
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := plugin.MeetsSeverity(tt.issueSeverity, tt.minSeverity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStaticAnalysisBreachesRuleIgnoring(t *testing.T) {
	plugin := &StaticAnalysisBreaches{
		IgnoreRules: []string{"rule1", "rule2", "deprecated.method"},
	}

	tests := []struct {
		name     string
		rule     string
		expected bool
	}{
		{"ignored_rule1", "rule1", true},
		{"ignored_rule2", "rule2", true},
		{"ignored_deprecated", "deprecated.method", true},
		{"not_ignored", "other.rule", false},
		{"empty_rule", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := plugin.IsRuleIgnored(tt.rule)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStaticAnalysisBreachesIssueFiltering(t *testing.T) {
	plugin := &StaticAnalysisBreaches{
		MinSeverity: "warning",
		IgnoreRules: []string{"ignore.me"},
	}

	issues := []Issue{
		{File: "test.php", Line: 1, Message: "Error 1", Rule: "rule1", Severity: "error"},
		{File: "test.php", Line: 2, Message: "Warning 1", Rule: "rule2", Severity: "warning"},
		{File: "test.php", Line: 3, Message: "Info 1", Rule: "rule3", Severity: "info"},       // Filtered out by severity
		{File: "test.php", Line: 4, Message: "Ignored", Rule: "ignore.me", Severity: "error"}, // Filtered out by rule
	}

	filtered := plugin.FilterIssues(issues)

	assert.Len(t, filtered, 2)
	assert.Equal(t, "Error 1", filtered[0].Message)
	assert.Equal(t, "Warning 1", filtered[1].Message)
}
