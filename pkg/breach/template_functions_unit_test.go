package breach

import (
	"testing"
)

// TestTemplateFunctionsUnit tests individual template functions in isolation
func TestTemplateFunctionsUnit(t *testing.T) {
	// Test that all expected functions are registered
	expectedFunctions := []string{
		// String functions
		"printf", "join", "split", "replace", "trim", "trimLeft", "trimRight",
		"upper", "lower", "title", "repeat", "contains", "hasPrefix", "hasSuffix",
		"truncate", "ellipsis", "pad", "padLeft",
		
		// Formatting functions
		"pluralize", "humanize", "bytes",
		
		// Array functions
		"len", "first", "last", "slice",
		
		// Comparison functions
		"eq", "ne", "lt", "le", "gt", "ge", "and", "or", "not",
		
		// Regex functions
		"regexMatch", "regexReplace", "regexFind",
		
		// Date functions
		"now", "date", "duration",
		
		// Color functions
		"colorize", "bold", "italic", "underline",
		
		// Utility functions
		"default", "empty",
		
		// Math functions
		"add", "sub", "mul", "div", "mod", "max", "min",
	}

	for _, funcName := range expectedFunctions {
		if _, exists := TemplateFuncs[funcName]; !exists {
			t.Errorf("Expected template function '%s' to be registered", funcName)
		}
	}

	t.Logf("Successfully verified %d template functions are registered", len(expectedFunctions))
}

func TestTemplateFunctionTypes(t *testing.T) {
	// Test that functions have the expected types/signatures
	tests := []struct {
		name         string
		functionName string
		testCall     func() interface{}
		expectError  bool
	}{
		{
			name:         "printf function",
			functionName: "printf",
			testCall: func() interface{} {
				fn := TemplateFuncs["printf"].(func(string, ...interface{}) string)
				return fn("Hello %s", "world")
			},
		},
		{
			name:         "humanize function",
			functionName: "humanize",
			testCall: func() interface{} {
				fn := TemplateFuncs["humanize"].(func(interface{}) string)
				return fn(1500)
			},
		},
		{
			name:         "pluralize function",
			functionName: "pluralize",
			testCall: func() interface{} {
				fn := TemplateFuncs["pluralize"].(func(int, string, string) string)
				return fn(5, "item", "items")
			},
		},
		{
			name:         "colorize function",
			functionName: "colorize",
			testCall: func() interface{} {
				fn := TemplateFuncs["colorize"].(func(string, string) string)
				return fn("red", "error")
			},
		},
		{
			name:         "eq function",
			functionName: "eq",
			testCall: func() interface{} {
				fn := TemplateFuncs["eq"].(func(interface{}, interface{}) bool)
				return fn("test", "test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					if !tt.expectError {
						t.Errorf("Function %s panicked: %v", tt.functionName, r)
					}
				}
			}()

			result := tt.testCall()
			if result == nil && !tt.expectError {
				t.Errorf("Function %s returned nil unexpectedly", tt.functionName)
			}
		})
	}
}

func TestTemplateHelperFunctionsIndividually(t *testing.T) {
	// Test helper functions directly
	tests := []struct {
		name     string
		function func() interface{}
		expected interface{}
	}{
		{
			name:     "humanizeNumber small",
			function: func() interface{} { return humanizeNumber(500) },
			expected: "500",
		},
		{
			name:     "humanizeNumber large",
			function: func() interface{} { return humanizeNumber(1500) },
			expected: "1.5K",
		},
		{
			name:     "humanizeString with underscores",
			function: func() interface{} { return humanizeString("test_string") },
			expected: "Test String",
		},
		{
			name:     "humanizeBytes kilobytes",
			function: func() interface{} { return humanizeBytes(1024) },
			expected: "1.0 KB",
		},
		{
			name:     "colorizeText red",
			function: func() interface{} { return colorizeText("red", "error") },
			expected: "\033[31merror\033[0m",
		},
		{
			name:     "compareValues equal",
			function: func() interface{} { return compareValues(5, 5) },
			expected: 0,
		},
		{
			name:     "compareValues greater",
			function: func() interface{} { return compareValues(10, 5) },
			expected: 1, // Should be positive
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.function()
			
			// For compareValues, just check the sign
			if tt.name == "compareValues greater" {
				if result.(int) <= 0 {
					t.Errorf("Expected positive result, got %v", result)
				}
				return
			}
			
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestTemplateContextStructure(t *testing.T) {
	// Test that TemplateContext has all expected fields
	ctx := TemplateContext{
		Breach:       &ValueBreach{Value: "test"},
		OutputFormat: "pretty",
		Severity:     "high",
		CheckName:    "test-check",
		CheckType:    "test-type",
		Context:      map[string]interface{}{"key": "value"},
	}

	// Verify all fields are accessible
	if ctx.Breach == nil {
		t.Error("Breach field should not be nil")
	}
	
	if ctx.OutputFormat != "pretty" {
		t.Error("OutputFormat field not set correctly")
	}
	
	if ctx.Severity != "high" {
		t.Error("Severity field not set correctly")
	}
	
	if ctx.CheckName != "test-check" {
		t.Error("CheckName field not set correctly")
	}
	
	if ctx.CheckType != "test-type" {
		t.Error("CheckType field not set correctly")
	}
	
	if ctx.Context == nil || ctx.Context["key"] != "value" {
		t.Error("Context field not set correctly")
	}
}