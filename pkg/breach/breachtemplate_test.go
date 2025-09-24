package breach

import (
	"testing"
)

func TestTemplateFunctions(t *testing.T) {
	tests := []struct {
		name     string
		template string
		context  TemplateContext
		expected string
	}{
		// String manipulation functions
		{
			name:     "printf formatting",
			template: `{{ printf "Found %d items in %s" 5 "directory" }}`,
			expected: "Found 5 items in directory",
		},
		{
			name:     "join function",
			template: `{{ .Breach.Values | join ", " }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{Values: []string{"a", "b", "c"}},
			},
			expected: "a, b, c",
		},
		{
			name:     "truncate function",
			template: `{{ truncate 10 "This is a very long string" }}`,
			expected: "This is...",
		},
		{
			name:     "ellipsis function",
			template: `{{ ellipsis 10 "This is a very long string" }}`,
			expected: "This is a …",
		},
		{
			name:     "case functions",
			template: `{{ upper "hello" }} {{ lower "WORLD" }} {{ title "test case" }}`,
			expected: "HELLO world Test Case",
		},

		// Pluralization and humanization
		{
			name:     "pluralize singular",
			template: `{{ pluralize 1 "file" "files" }}`,
			expected: "file",
		},
		{
			name:     "pluralize plural",
			template: `{{ pluralize 5 "file" "files" }}`,
			expected: "files",
		},
		{
			name:     "humanize number",
			template: `{{ humanize 1500 }}`,
			expected: "1.5K",
		},
		{
			name:     "humanize string",
			template: `{{ humanize "test_string_value" }}`,
			expected: "Test String Value",
		},
		{
			name:     "bytes formatting",
			template: `{{ bytes 1024 }}`,
			expected: "1.0 KB",
		},

		// Array/slice functions
		{
			name:     "len function",
			template: `{{ len .Breach.Values }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{Values: []string{"a", "b", "c"}},
			},
			expected: "3",
		},
		{
			name:     "first function",
			template: `{{ first .Breach.Values }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{Values: []string{"first", "second", "third"}},
			},
			expected: "first",
		},
		{
			name:     "last function",
			template: `{{ last .Breach.Values }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{Values: []string{"first", "second", "third"}},
			},
			expected: "third",
		},

		// Comparison functions
		{
			name:     "eq function true",
			template: `{{ if eq "test" "test" }}equal{{ else }}not equal{{ end }}`,
			expected: "equal",
		},
		{
			name:     "eq function false",
			template: `{{ if eq "test" "other" }}equal{{ else }}not equal{{ end }}`,
			expected: "not equal",
		},
		{
			name:     "gt function",
			template: `{{ if gt 5 3 }}greater{{ else }}not greater{{ end }}`,
			expected: "greater",
		},

		// Regular expressions
		{
			name:     "regexMatch function",
			template: `{{ if regexMatch "^test.*" "testing" }}matches{{ else }}no match{{ end }}`,
			expected: "matches",
		},
		{
			name:     "regexReplace function",
			template: `{{ regexReplace "test" "example" "This is a test string" }}`,
			expected: "This is a example string",
		},

		// Conditional functions
		{
			name:     "default with empty value",
			template: `{{ default "fallback" "" }}`,
			expected: "fallback",
		},
		{
			name:     "default with value",
			template: `{{ default "fallback" "actual" }}`,
			expected: "actual",
		},
		{
			name:     "empty function true",
			template: `{{ if empty "" }}is empty{{ else }}not empty{{ end }}`,
			expected: "is empty",
		},
		{
			name:     "empty function false",
			template: `{{ if empty "value" }}is empty{{ else }}not empty{{ end }}`,
			expected: "not empty",
		},

		// Math functions
		{
			name:     "add function",
			template: `{{ add 5 3 }}`,
			expected: "8",
		},
		{
			name:     "sub function",
			template: `{{ sub 10 3 }}`,
			expected: "7",
		},
		{
			name:     "max function",
			template: `{{ max 5 8 }}`,
			expected: "8",
		},
		{
			name:     "min function",
			template: `{{ min 5 8 }}`,
			expected: "5",
		},

		// Context-aware templates
		{
			name:     "severity-based formatting",
			template: `{{ if eq .Severity "high" }}🚨{{ else if eq .Severity "medium" }}⚠️{{ else }}ℹ️{{ end }} {{ .CheckName }}`,
			context: TemplateContext{
				Severity:  "high",
				CheckName: "security-check",
			},
			expected: "🚨 security-check",
		},
		{
			name:     "output format conditional",
			template: `{{ if eq .OutputFormat "json" }}{"message": "test"}{{ else }}Test message{{ end }}`,
			context: TemplateContext{
				OutputFormat: "pretty",
			},
			expected: "Test message",
		},

		// Complex template combining multiple functions
		{
			name: "complex template",
			template: `{{ if gt (len .Breach.Values) 0 }}Found {{ len .Breach.Values | humanize }} {{ pluralize (len .Breach.Values) "issue" "issues" }}:{{ range slice 0 (min 3 (len .Breach.Values)) .Breach.Values }}
• {{ . | humanize | truncate 30 }}{{ end }}{{ if gt (len .Breach.Values) 3 }}
... and {{ sub (len .Breach.Values) 3 }} more{{ end }}{{ else }}No issues found{{ end }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{Values: []string{"first_issue_found", "second_problem_detected", "third_error_occurred", "fourth_warning_raised", "fifth_alert_triggered"}},
			},
			expected: "Found 5 issues:\n• First Issue Found\n• Second Problem Detected\n• Third Error Occurred\n... and 2 more",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock BreachTemplater for testing
			bt := &mockBreachTemplater{breaches: []Breach{}}
			
			result := EvaluateTemplateStringWithContext(bt, tt.template, tt.context)
			
			if result != tt.expected {
				t.Errorf("Template evaluation failed.\nTemplate: %s\nExpected: %s\nActual: %s", tt.template, tt.expected, result)
			}
			
			// Check that no error breaches were added
			if len(bt.breaches) > 0 {
				t.Errorf("Template evaluation should not have added error breaches, but got: %v", bt.breaches)
			}
		})
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test humanizeNumber
	tests := []struct {
		input    int64
		expected string
	}{
		{500, "500"},
		{1500, "1.5K"},
		{1000000, "1M"},
		{2500000000, "2.5B"},
	}

	for _, tt := range tests {
		result := humanizeNumber(tt.input)
		if result != tt.expected {
			t.Errorf("humanizeNumber(%d) = %s; expected %s", tt.input, result, tt.expected)
		}
	}

	// Test humanizeString
	stringTests := []struct {
		input    string
		expected string
	}{
		{"test_string", "Test String"},
		{"kebab-case", "Kebab Case"},
		{"mixed_case-string", "Mixed Case String"},
	}

	for _, tt := range stringTests {
		result := humanizeString(tt.input)
		if result != tt.expected {
			t.Errorf("humanizeString(%s) = %s; expected %s", tt.input, result, tt.expected)
		}
	}

	// Test humanizeBytes
	byteTests := []struct {
		input    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range byteTests {
		result := humanizeBytes(tt.input)
		if result != tt.expected {
			t.Errorf("humanizeBytes(%d) = %s; expected %s", tt.input, result, tt.expected)
		}
	}
}

func TestColorizeText(t *testing.T) {
	tests := []struct {
		color    string
		text     string
		expected string
	}{
		{"red", "error", "\033[31merror\033[0m"},
		{"green", "success", "\033[32msuccess\033[0m"},
		{"unknown", "text", "text"}, // Should return unchanged for unknown colors
	}

	for _, tt := range tests {
		result := colorizeText(tt.color, tt.text)
		if result != tt.expected {
			t.Errorf("colorizeText(%s, %s) = %s; expected %s", tt.color, tt.text, result, tt.expected)
		}
	}
}

func TestEnhancedTemplateEvaluation(t *testing.T) {
	// Test context-aware template evaluation
	bt := &mockBreachTemplater{
		template: BreachTemplate{
			Template: `{{ if eq .Severity "high" }}CRITICAL{{ else }}NORMAL{{ end }}: {{ .Breach.Value }}`,
		},
	}

	breach := &ValueBreach{
		Severity: "high",
		Value:    "Test issue",
	}

	EvaluateTemplateWithContext(bt, breach, nil, "pretty")

	if len(bt.breaches) != 1 {
		t.Fatalf("Expected 1 breach, got %d", len(bt.breaches))
	}

	result := bt.breaches[0].(*ValueBreach)
	if result.Value != "CRITICAL: Test issue" {
		t.Errorf("Expected 'CRITICAL: Test issue', got '%s'", result.Value)
	}
}

func TestFormatSpecificTemplates(t *testing.T) {
	// Test format-specific templates
	bt := &mockBreachTemplater{
		template: BreachTemplate{
			Templates: map[string]string{
				"json":   `{"error": "{{ .Breach.Value }}"}`,
				"pretty": `🚨 {{ .Breach.Value }}`,
			},
		},
	}

	breach := &ValueBreach{
		Value: "Test error",
	}

	// Test JSON format
	EvaluateTemplateWithContext(bt, breach, nil, "json")
	jsonResult := bt.breaches[0].(*ValueBreach)
	if jsonResult.Value != `{"error": "Test error"}` {
		t.Errorf("JSON template failed: got '%s'", jsonResult.Value)
	}

	// Reset and test pretty format
	bt.breaches = []Breach{}
	EvaluateTemplateWithContext(bt, breach, nil, "pretty")
	prettyResult := bt.breaches[0].(*ValueBreach)
	if prettyResult.Value != `🚨 Test error` {
		t.Errorf("Pretty template failed: got '%s'", prettyResult.Value)
	}
}

// Mock BreachTemplater for testing
type mockBreachTemplater struct {
	template BreachTemplate
	breaches []Breach
}

func (m *mockBreachTemplater) AddBreach(b Breach) {
	m.breaches = append(m.breaches, b)
}

func (m *mockBreachTemplater) GetBreachTemplate() BreachTemplate {
	return m.template
}