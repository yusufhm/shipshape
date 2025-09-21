package breach

import (
	"strings"
	"testing"
)

func TestTemplateEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		template       string
		context        TemplateContext
		expectedResult string
		shouldError    bool
	}{
		// Nil and empty value handling
		{
			name:           "nil breach value",
			template:       `{{ .Breach.Value | default "no value" }}`,
			context:        TemplateContext{Breach: &ValueBreach{}},
			expectedResult: "no value",
		},
		{
			name:           "empty string value",
			template:       `{{ .Breach.Value | default "empty" }}`,
			context:        TemplateContext{Breach: &ValueBreach{Value: ""}},
			expectedResult: "empty",
		},
		{
			name:           "nil context",
			template:       `{{ .Context.nonexistent | default "default" }}`,
			context:        TemplateContext{Context: nil},
			expectedResult: "default",
		},
		{
			name:           "empty context map",
			template:       `{{ .Context.nonexistent | default "default" }}`,
			context:        TemplateContext{Context: map[string]interface{}{}},
			expectedResult: "default",
		},

		// Array/slice edge cases
		{
			name:           "empty array length",
			template:       `{{ len .Breach.Values }}`,
			context:        TemplateContext{Breach: &KeyValuesBreach{Values: []string{}}},
			expectedResult: "0",
		},
		{
			name:           "nil array length",
			template:       `{{ len .Breach.Values }}`,
			context:        TemplateContext{Breach: &KeyValuesBreach{}},
			expectedResult: "0",
		},
		{
			name:           "first of empty array",
			template:       `{{ first .Breach.Values | default "none" }}`,
			context:        TemplateContext{Breach: &KeyValuesBreach{Values: []string{}}},
			expectedResult: "none",
		},
		{
			name:           "slice beyond bounds",
			template:       `{{ slice 5 10 .Breach.Values }}`,
			context:        TemplateContext{Breach: &KeyValuesBreach{Values: []string{"a", "b", "c"}}},
			expectedResult: "",
		},
		{
			name:           "slice negative start",
			template:       `{{ slice -1 2 .Breach.Values | len }}`,
			context:        TemplateContext{Breach: &KeyValuesBreach{Values: []string{"a", "b", "c"}}},
			expectedResult: "2",
		},

		// String manipulation edge cases
		{
			name:           "truncate negative length",
			template:       `{{ truncate -5 "hello world" }}`,
			context:        TemplateContext{},
			expectedResult: "",
			shouldError:    false,
		},
		{
			name:           "truncate zero length",
			template:       `{{ truncate 0 "hello world" }}`,
			context:        TemplateContext{},
			expectedResult: "",
		},
		{
			name:           "truncate very small length",
			template:       `{{ truncate 1 "hello world" }}`,
			context:        TemplateContext{},
			expectedResult: "h",
		},
		{
			name:           "join empty array",
			template:       `{{ slice 0 0 .Breach.Values | join "," }}`,
			context:        TemplateContext{Breach: &KeyValuesBreach{Values: []string{"a", "b"}}},
			expectedResult: "",
		},
		{
			name:           "join single item",
			template:       `{{ slice 0 1 .Breach.Values | join "," }}`,
			context:        TemplateContext{Breach: &KeyValuesBreach{Values: []string{"single"}}},
			expectedResult: "single",
		},

		// Math edge cases
		{
			name:           "division by zero",
			template:       `{{ div 10 0 }}`,
			context:        TemplateContext{},
			expectedResult: "0",
		},
		{
			name:           "modulo by zero",
			template:       `{{ mod 10 0 }}`,
			context:        TemplateContext{},
			expectedResult: "0",
		},
		{
			name:           "negative numbers",
			template:       `{{ add -5 3 }}`,
			context:        TemplateContext{},
			expectedResult: "-2",
		},

		// Comparison edge cases
		{
			name:           "compare different types",
			template:       `{{ if eq "5" 5 }}equal{{ else }}not equal{{ end }}`,
			context:        TemplateContext{},
			expectedResult: "not equal",
		},
		{
			name:           "compare nil values",
			template:       `{{ if eq .Context.nil1 .Context.nil2 }}equal{{ else }}not equal{{ end }}`,
			context:        TemplateContext{Context: map[string]interface{}{}},
			expectedResult: "equal",
		},

		// Regular expression edge cases
		{
			name:           "invalid regex pattern",
			template:       `{{ regexMatch "[invalid" "test" }}`,
			context:        TemplateContext{},
			expectedResult: "false",
		},
		{
			name:           "regex with empty string",
			template:       `{{ regexMatch ".*" "" }}`,
			context:        TemplateContext{},
			expectedResult: "true",
		},
		{
			name:           "regex replace with invalid pattern",
			template:       `{{ regexReplace "[invalid" "replacement" "test string" }}`,
			context:        TemplateContext{},
			expectedResult: "test string",
		},

		// Humanize edge cases
		{
			name:           "humanize zero",
			template:       `{{ humanize 0 }}`,
			context:        TemplateContext{},
			expectedResult: "0",
		},
		{
			name:           "humanize very large number",
			template:       `{{ humanize 1000000000000 }}`,
			context:        TemplateContext{},
			expectedResult: "1T",
		},
		{
			name:           "humanize string with special characters",
			template:       `{{ humanize "test_with-special.chars" }}`,
			context:        TemplateContext{},
			expectedResult: "Test With Special Chars",
		},

		// Bytes formatting edge cases
		{
			name:           "bytes zero",
			template:       `{{ bytes 0 }}`,
			context:        TemplateContext{},
			expectedResult: "0 B",
		},
		{
			name:           "bytes very large",
			template:       `{{ bytes 1152921504606846976 }}`,
			context:        TemplateContext{},
			expectedResult: "1.0 EB",
		},

		// Color formatting (should work even if not in terminal)
		{
			name:           "colorize unknown color",
			template:       `{{ colorize "unknowncolor" "text" }}`,
			context:        TemplateContext{},
			expectedResult: "text",
		},
		{
			name:           "colorize empty text",
			template:       `{{ colorize "red" "" }}`,
			context:        TemplateContext{},
			expectedResult: "",
		},

		// Complex nested operations
		{
			name:     "deeply nested operations",
			template: `{{ slice 0 (min 3 (max 1 (len .Breach.Values))) .Breach.Values | join " | " | truncate 20 | upper }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{Values: []string{"first", "second", "third", "fourth"}},
			},
			expectedResult: "FIRST | SECOND | ...",
		},

		// Date/time edge cases
		{
			name:           "date formatting",
			template:       `{{ now | date "2006-01-02" }}`,
			context:        TemplateContext{},
			expectedResult: "", // Will be current date, just check it doesn't error
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := &mockBreachTemplater{}
			result := EvaluateTemplateStringWithContext(bt, tt.template, tt.context)

			if tt.shouldError {
				// Check that an error breach was added
				if len(bt.breaches) == 0 {
					t.Errorf("Expected error breach to be added, but none were")
				}
			} else {
				// For non-error cases, check the result
				if tt.expectedResult != "" {
					// For date formatting, just check it's not empty and doesn't error
					if strings.Contains(tt.template, "now | date") {
						if result == "" || len(bt.breaches) > 0 {
							t.Errorf("Date formatting failed: result='%s', errors=%d", result, len(bt.breaches))
						}
					} else if result != tt.expectedResult {
						t.Errorf("Expected: %s\nActual: %s", tt.expectedResult, result)
					}
				}

				// Check that no error breaches were added
				if len(bt.breaches) > 0 {
					t.Errorf("Unexpected error breaches: %v", bt.breaches)
				}
			}
		})
	}
}

func TestHelperFunctionEdgeCases(t *testing.T) {
	// Test humanizeNumber edge cases
	testCases := []struct {
		input    int64
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1K"},
		{1500, "1.5K"},
		{999999, "1000.0K"},
		{1000000, "1M"},
		{-1500, "-1.5K"},
	}

	for _, tc := range testCases {
		result := humanizeNumber(tc.input)
		if result != tc.expected {
			t.Errorf("humanizeNumber(%d) = %s, expected %s", tc.input, result, tc.expected)
		}
	}

	// Test humanizeString edge cases
	stringCases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"single", "Single"},
		{"ALREADY_UPPER", "Already Upper"},
		{"mixed-case_string", "Mixed Case String"},
		{"___multiple___underscores___", "Multiple Underscores"},
		{"123numeric456", "123numeric456"},
	}

	for _, tc := range stringCases {
		result := humanizeString(tc.input)
		if result != tc.expected {
			t.Errorf("humanizeString(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}

	// Test humanizeBytes edge cases
	bytesCases := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{-1024, "-1.0 KB"},
	}

	for _, tc := range bytesCases {
		result := humanizeBytes(tc.input)
		if result != tc.expected {
			t.Errorf("humanizeBytes(%d) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

func TestCompareValuesEdgeCases(t *testing.T) {
	testCases := []struct {
		a        interface{}
		b        interface{}
		expected int
	}{
		// Same types
		{5, 3, 1},
		{3, 5, -1},
		{5, 5, 0},
		{"apple", "banana", -1},
		{"banana", "apple", 1},
		{"same", "same", 0},

		// Different numeric types
		{int64(5), int32(3), 1},
		{uint64(5), uint32(3), 1},
		{float64(5.5), float32(3.2), 1},

		// Mixed types (fall back to string comparison)
		{5, "5", 0}, // Both become "5" as strings
		{"hello", 123, 1},
		{true, false, 1}, // "true" vs "false" as strings

		// Nil values
		{nil, nil, 0},
		{nil, "something", -1},
		{"something", nil, 1},
	}

	for _, tc := range testCases {
		result := compareValues(tc.a, tc.b)
		if (result > 0 && tc.expected <= 0) || (result < 0 && tc.expected >= 0) || (result == 0 && tc.expected != 0) {
			t.Errorf("compareValues(%v, %v) = %d, expected sign of %d", tc.a, tc.b, result, tc.expected)
		}
	}
}

func TestColorizeTextEdgeCases(t *testing.T) {
	testCases := []struct {
		color    string
		text     string
		expected string
	}{
		// Valid colors
		{"red", "error", "\033[31merror\033[0m"},
		{"green", "success", "\033[32msuccess\033[0m"},
		{"BLUE", "info", "\033[34minfo\033[0m"}, // Case insensitive, should match

		// Invalid colors
		{"invalidcolor", "text", "text"},
		{"", "text", "text"},

		// Edge cases
		{"red", "", "\033[31m\033[0m"},
		{"green", "multi\nline\ntext", "\033[32mmulti\nline\ntext\033[0m"},
	}

	for _, tc := range testCases {
		result := colorizeText(tc.color, tc.text)
		if result != tc.expected {
			t.Errorf("colorizeText(%s, %s) = %q, expected %q", tc.color, tc.text, result, tc.expected)
		}
	}
}

func TestTemplateFunctionChaining(t *testing.T) {
	// Test complex function chaining scenarios
	tests := []struct {
		name           string
		template       string
		context        TemplateContext
		expectedResult string
	}{
		{
			name:     "long function chain",
			template: `{{ .Breach.Values | join " " | upper | truncate 20 | regexReplace "FIRST" "1ST" }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{Values: []string{"first", "second", "third"}},
			},
			expectedResult: "1ST SECOND THIRD",
		},
		{
			name:     "mathematical chain",
			template: `{{ add (mul 3 4) (div 10 2) }}`,
			context:  TemplateContext{},
			expectedResult: "17",
		},
		{
			name:     "conditional with function chain",
			template: `{{ if gt (len (.Breach.Values | slice 0 2)) 1 }}{{ .Breach.Values | slice 0 2 | join " and " | title }}{{ else }}None{{ end }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{Values: []string{"first", "second", "third"}},
			},
			expectedResult: "First And Second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bt := &mockBreachTemplater{}
			result := EvaluateTemplateStringWithContext(bt, tt.template, tt.context)

			if result != tt.expectedResult {
				t.Errorf("Expected: %s\nActual: %s", tt.expectedResult, result)
			}

			// Ensure no errors occurred
			if len(bt.breaches) > 0 {
				t.Errorf("Unexpected error breaches: %v", bt.breaches)
			}
		})
	}
}