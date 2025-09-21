package breach

import (
	"testing"
)

func BenchmarkTemplateEvaluation(b *testing.B) {
	benchmarks := []struct {
		name     string
		template string
		context  TemplateContext
	}{
		{
			name:     "Simple template",
			template: `{{ .Breach.Value | upper }}`,
			context: TemplateContext{
				Breach: &ValueBreach{Value: "test message"},
			},
		},
		{
			name:     "Complex template with functions",
			template: `{{ if gt (len .Breach.Values) 0 }}Found {{ len .Breach.Values | humanize }} {{ pluralize (len .Breach.Values) "item" "items" }}: {{ .Breach.Values | join ", " | truncate 50 }}{{ else }}No items{{ end }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{
					Values: []string{"item1", "item2", "item3", "item4", "item5"},
				},
			},
		},
		{
			name: "Template with context and conditionals",
			template: `{{ if eq .Context.env "prod" }}🔴 PRODUCTION{{ else }}🟢 DEV{{ end }} - {{ .Severity | upper }} - {{ .CheckName | humanize }}`,
			context: TemplateContext{
				Severity:  "high",
				CheckName: "security_check",
				Context:   map[string]interface{}{"env": "prod"},
			},
		},
		{
			name: "Template with loops and string manipulation",
			template: `{{ range slice 0 (min 5 (len .Breach.Values)) .Breach.Values }}• {{ . | title | truncate 20 }}
{{ end }}Total: {{ len .Breach.Values }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{
					Values: []string{
						"first_item_with_long_name",
						"second_item_with_long_name", 
						"third_item_with_long_name",
						"fourth_item_with_long_name",
						"fifth_item_with_long_name",
						"sixth_item_with_long_name",
					},
				},
			},
		},
		{
			name: "Template with mathematical operations",
			template: `Progress: {{ div (mul .Context.completed 100) .Context.total }}% ({{ .Context.completed }}/{{ .Context.total }})
Remaining: {{ sub .Context.total .Context.completed }}
Average: {{ div .Context.total .Context.days }}`,
			context: TemplateContext{
				Context: map[string]interface{}{
					"completed": 75,
					"total":     100,
					"days":      10,
				},
			},
		},
		{
			name: "Template with regex operations",
			template: `{{ if regexMatch "error|fail" .Breach.Value }}{{ .Breach.Value | regexReplace "error" "ERROR" | regexReplace "fail" "FAIL" | colorize "red" }}{{ else }}{{ .Breach.Value | colorize "green" }}{{ end }}`,
			context: TemplateContext{
				Breach: &ValueBreach{Value: "error in processing failed operation"},
			},
		},
	}

	for _, bm := range benchmarks {
		b.Run(bm.name, func(b *testing.B) {
			bt := &mockBreachTemplater{}
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bt.breaches = []Breach{} // Reset for each iteration
				EvaluateTemplateStringWithContext(bt, bm.template, bm.context)
			}
		})
	}
}

func BenchmarkTemplateFunctions(b *testing.B) {
	// Benchmark individual template functions
	functionBenchmarks := []struct {
		name     string
		template string
		context  TemplateContext
	}{
		{
			name:     "humanize function",
			template: `{{ humanize 1500000 }}`,
			context:  TemplateContext{},
		},
		{
			name:     "join function",
			template: `{{ .Breach.Values | join ", " }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{Values: []string{"a", "b", "c", "d", "e"}},
			},
		},
		{
			name:     "truncate function",
			template: `{{ truncate 50 "This is a very long string that needs to be truncated for display purposes" }}`,
			context:  TemplateContext{},
		},
		{
			name:     "colorize function",
			template: `{{ colorize "red" "error message" }}`,
			context:  TemplateContext{},
		},
		{
			name:     "regex match function",
			template: `{{ regexMatch "test.*pattern" "test string with pattern" }}`,
			context:  TemplateContext{},
		},
		{
			name:     "pluralize function",
			template: `{{ pluralize 5 "item" "items" }}`,
			context:  TemplateContext{},
		},
		{
			name:     "bytes function",
			template: `{{ bytes 1048576 }}`,
			context:  TemplateContext{},
		},
		{
			name:     "slice function",
			template: `{{ slice 0 3 .Breach.Values }}`,
			context: TemplateContext{
				Breach: &KeyValuesBreach{Values: []string{"a", "b", "c", "d", "e", "f", "g"}},
			},
		},
	}

	for _, bm := range functionBenchmarks {
		b.Run(bm.name, func(b *testing.B) {
			bt := &mockBreachTemplater{}
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bt.breaches = []Breach{}
				EvaluateTemplateStringWithContext(bt, bm.template, bm.context)
			}
		})
	}
}

func BenchmarkTemplateCompilation(b *testing.B) {
	// Benchmark template compilation vs execution
	template := `{{ if gt (len .Breach.Values) 0 }}Found {{ len .Breach.Values | humanize }} {{ pluralize (len .Breach.Values) "item" "items" }}: {{ range slice 0 (min 3 (len .Breach.Values)) .Breach.Values }}{{ . | title | truncate 20 }}, {{ end }}{{ else }}No items{{ end }}`
	context := TemplateContext{
		Breach: &KeyValuesBreach{
			Values: []string{"first_item", "second_item", "third_item", "fourth_item"},
		},
	}

	b.Run("with_compilation", func(b *testing.B) {
		// This benchmarks the full template evaluation including compilation
		bt := &mockBreachTemplater{}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bt.breaches = []Breach{}
			EvaluateTemplateStringWithContext(bt, template, context)
		}
	})
}

func BenchmarkLegacyVsEnhancedTemplates(b *testing.B) {
	// Compare legacy template performance with enhanced templates
	breach := &KeyValueBreach{
		Key:   "config.yml",
		Value: "configuration error",
	}

	b.Run("legacy_template", func(b *testing.B) {
		bt := &mockBreachTemplater{
			template: BreachTemplate{
				Type:       BreachTypeKeyValue,
				KeyLabel:   "File",
				Key:        "{{ .Breach.Key }}",
				ValueLabel: "Error",
				Value:      "{{ .Breach.Value }}",
			},
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bt.breaches = []Breach{}
			EvaluateTemplate(bt, breach, nil)
		}
	})

	b.Run("enhanced_template", func(b *testing.B) {
		bt := &mockBreachTemplater{
			template: BreachTemplate{
				Template: `📁 File: {{ .Breach.Key }}
🚨 Error: {{ .Breach.Value | title }}`,
			},
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			bt.breaches = []Breach{}
			EvaluateTemplate(bt, breach, nil)
		}
	})
}

func BenchmarkHelperFunctions(b *testing.B) {
	// Benchmark helper functions directly
	b.Run("humanizeNumber", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			humanizeNumber(1500000)
		}
	})

	b.Run("humanizeString", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			humanizeString("test_string_with_underscores")
		}
	})

	b.Run("humanizeBytes", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			humanizeBytes(1048576)
		}
	})

	b.Run("colorizeText", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			colorizeText("red", "error message")
		}
	})

	b.Run("compareValues", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			compareValues("apple", "banana")
		}
	})
}

func BenchmarkFormatSpecificTemplates(b *testing.B) {
	// Benchmark format-specific template selection and evaluation
	templates := map[string]string{
		"json":   `{"error": "{{ .Breach.Value }}", "severity": "{{ .Severity }}"}`,
		"pretty": `🚨 {{ .Breach.Value }} ({{ .Severity }})`,
		"table":  `{{ .Breach.Value }}\t{{ .Severity }}`,
	}

	breach := &ValueBreach{
		Value:    "test error",
		Severity: "high",
	}

	formats := []string{"json", "pretty", "table"}

	for _, format := range formats {
		b.Run("format_"+format, func(b *testing.B) {
			bt := &mockBreachTemplater{
				template: BreachTemplate{Templates: templates},
			}
			
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				bt.breaches = []Breach{}
				EvaluateTemplateWithContext(bt, breach, nil, format)
			}
		})
	}
}

// Memory allocation benchmarks
func BenchmarkTemplateMemoryAllocation(b *testing.B) {
	template := `{{ range .Breach.Values }}{{ . | title | truncate 20 }} {{ end }}`
	context := TemplateContext{
		Breach: &KeyValuesBreach{
			Values: make([]string, 100), // Large slice to test memory usage
		},
	}

	// Fill the values
	for i := 0; i < 100; i++ {
		context.Breach.(*KeyValuesBreach).Values[i] = "test_item_" + string(rune(i))
	}

	bt := &mockBreachTemplater{}

	b.ReportAllocs()
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		bt.breaches = []Breach{}
		EvaluateTemplateStringWithContext(bt, template, context)
	}
}