package breach

import (
	"bytes"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/template"
	"time"

	ss_rem "github.com/salsadigitalauorg/shipshape/pkg/remediation"
)

func EvaluateTemplate(bt BreachTemplater, b Breach, remediation interface{}) {
	EvaluateTemplateWithContext(bt, b, remediation, "pretty")
}

func EvaluateTemplateWithContext(bt BreachTemplater, b Breach, remediation interface{}, outputFormat string) {
	t := bt.GetBreachTemplate()

	// No template set, use raw breach.
	if t.Type == "" && t.Template == "" && len(t.Templates) == 0 {
		r := ss_rem.RemediatorFromInterface(remediation)
		b.SetRemediator(r)
		bt.AddBreach(b)
		return
	}

	// Create template context
	ctx := TemplateContext{
		Breach:       b,
		OutputFormat: outputFormat,
		Severity:     b.GetSeverity(),
		CheckName:    b.GetCheckName(),
		CheckType:    b.GetCheckType(),
		Context:      t.Context,
	}

	// Check if we have a rich template (single or format-specific)
	var templateStr string
	if len(t.Templates) > 0 {
		// Try format-specific template first
		if formatTemplate, ok := t.Templates[outputFormat]; ok {
			templateStr = formatTemplate
		} else if defaultTemplate, ok := t.Templates["default"]; ok {
			templateStr = defaultTemplate
		}
	} else if t.Template != "" {
		templateStr = t.Template
	}

	// If we have a rich template, use it to override the entire breach string
	if templateStr != "" {
		renderedValue := EvaluateTemplateStringWithContext(bt, templateStr, ctx)
		
		// Create a new breach with the rendered template as the value
		switch b.GetType() {
		case BreachTypeValue:
			breach := &ValueBreach{
				BreachType: b.GetType(),
				CheckType:  b.GetCheckType(),
				CheckName:  b.GetCheckName(),
				Severity:   b.GetSeverity(),
				Value:      renderedValue,
			}
			r := ss_rem.RemediatorFromInterface(remediation)
			breach.SetRemediator(r)
			bt.AddBreach(breach)
		case BreachTypeKeyValue:
			originalBreach := b.(*KeyValueBreach)
			breach := &KeyValueBreach{
				BreachType: b.GetType(),
				CheckType:  b.GetCheckType(),
				CheckName:  b.GetCheckName(),
				Severity:   b.GetSeverity(),
				KeyLabel:   originalBreach.KeyLabel,
				Key:        originalBreach.Key,
				ValueLabel: originalBreach.ValueLabel,
				Value:      renderedValue,
			}
			r := ss_rem.RemediatorFromInterface(remediation)
			breach.SetRemediator(r)
			bt.AddBreach(breach)
		case BreachTypeKeyValues:
			originalBreach := b.(*KeyValuesBreach)
			breach := &KeyValuesBreach{
				BreachType: b.GetType(),
				CheckType:  b.GetCheckType(),
				CheckName:  b.GetCheckName(),
				Severity:   b.GetSeverity(),
				KeyLabel:   originalBreach.KeyLabel,
				Key:        originalBreach.Key,
				ValueLabel: originalBreach.ValueLabel,
				Values:     []string{renderedValue}, // Convert to single value
			}
			r := ss_rem.RemediatorFromInterface(remediation)
			breach.SetRemediator(r)
			bt.AddBreach(breach)
		}
		return
	}

	// Fall back to legacy template system
	rendered := BreachTemplate{
		Type:       b.GetType(),
		ValueLabel: BreachGetValueLabel(b),
		Value:      BreachGetValue(b),
		KeyLabel:   BreachGetKeyLabel(b),
		Key:        BreachGetKey(b),
	}

	if t.KeyLabel != "" {
		rendered.KeyLabel = EvaluateTemplateStringWithContext(bt, t.KeyLabel, ctx)
	}
	if t.Key != "" {
		rendered.Key = EvaluateTemplateStringWithContext(bt, t.Key, ctx)
	}
	if t.ValueLabel != "" {
		rendered.ValueLabel = EvaluateTemplateStringWithContext(bt, t.ValueLabel, ctx)
	}
	if t.Value != "" {
		rendered.Value = EvaluateTemplateStringWithContext(bt, t.Value, ctx)
	}

	var breachToAdd Breach
	switch rendered.Type {
	case BreachTypeValue:
		breach := b.(*ValueBreach)
		breach.ValueLabel = rendered.ValueLabel
		breach.Value = rendered.Value
		breachToAdd = breach
	case BreachTypeKeyValue:
		breach := b.(*KeyValueBreach)
		breach.KeyLabel = rendered.KeyLabel
		breach.Key = rendered.Key
		breach.ValueLabel = rendered.ValueLabel
		breach.Value = rendered.Value
		breachToAdd = breach
	case BreachTypeKeyValues:
		breach := b.(*KeyValuesBreach)
		breach.KeyLabel = rendered.KeyLabel
		breach.Key = rendered.Key
		breach.ValueLabel = rendered.ValueLabel
		// Keep original values for KeyValues breach type
		breachToAdd = breach
	}

	r := ss_rem.RemediatorFromInterface(remediation)
	breachToAdd.SetRemediator(r)
	bt.AddBreach(breachToAdd)
}

var TemplateFuncs = template.FuncMap{
	// String manipulation functions
	"printf": fmt.Sprintf,
	"join": func(sep string, elems interface{}) string {
		if elems == nil {
			return ""
		}
		rv := reflect.ValueOf(elems)
		if rv.Kind() != reflect.Slice {
			return fmt.Sprintf("%v", elems)
		}
		strs := make([]string, rv.Len())
		for i := 0; i < rv.Len(); i++ {
			strs[i] = fmt.Sprintf("%v", rv.Index(i).Interface())
		}
		return strings.Join(strs, sep)
	},
	"split":     strings.Split,
	"replace":   strings.ReplaceAll,
	"trim":      strings.TrimSpace,
	"trimLeft":  strings.TrimLeft,
	"trimRight": strings.TrimRight,
	"upper":     strings.ToUpper,
	"lower":     strings.ToLower,
	"title":     strings.Title,
	"repeat":    strings.Repeat,
	"contains":  strings.Contains,
	"hasPrefix": strings.HasPrefix,
	"hasSuffix": strings.HasSuffix,
	
	// String formatting and truncation
	"truncate": func(length int, s string) string {
		if len(s) <= length {
			return s
		}
		if length <= 3 {
			return s[:length]
		}
		return s[:length-3] + "..."
	},
	"ellipsis": func(length int, s string) string {
		if len(s) <= length {
			return s
		}
		return s[:length] + "…"
	},
	"pad": func(width int, s string) string {
		return fmt.Sprintf("%-*s", width, s)
	},
	"padLeft": func(width int, s string) string {
		return fmt.Sprintf("%*s", width, s)
	},
	
	// Pluralization and humanization
	"pluralize": func(count int, singular, plural string) string {
		if count == 1 {
			return singular
		}
		return plural
	},
	"humanize": func(v interface{}) string {
		switch val := v.(type) {
		case int, int32, int64:
			num := reflect.ValueOf(val).Int()
			return humanizeNumber(num)
		case float32, float64:
			num := reflect.ValueOf(val).Float()
			return fmt.Sprintf("%.2f", num)
		case string:
			return humanizeString(val)
		default:
			return fmt.Sprintf("%v", val)
		}
	},
	"bytes": func(size int64) string {
		return humanizeBytes(size)
	},
	
	// Array/slice functions
	"len": func(v interface{}) int {
		if v == nil {
			return 0
		}
		rv := reflect.ValueOf(v)
		switch rv.Kind() {
		case reflect.Array, reflect.Slice, reflect.Map, reflect.String:
			return rv.Len()
		default:
			return 0
		}
	},
	"first": func(v interface{}) interface{} {
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice && rv.Len() > 0 {
			return rv.Index(0).Interface()
		}
		return nil
	},
	"last": func(v interface{}) interface{} {
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice && rv.Len() > 0 {
			return rv.Index(rv.Len() - 1).Interface()
		}
		return nil
	},
	"slice": func(start, end int, v interface{}) interface{} {
		rv := reflect.ValueOf(v)
		if rv.Kind() == reflect.Slice {
			if start < 0 {
				start = 0
			}
			if end > rv.Len() {
				end = rv.Len()
			}
			if start >= end {
				return reflect.MakeSlice(rv.Type(), 0, 0).Interface()
			}
			return rv.Slice(start, end).Interface()
		}
		return v
	},
	
	// Comparison and logic functions
	"eq": func(a, b interface{}) bool { return reflect.DeepEqual(a, b) },
	"ne": func(a, b interface{}) bool { return !reflect.DeepEqual(a, b) },
	"lt": func(a, b interface{}) bool { return compareValues(a, b) < 0 },
	"le": func(a, b interface{}) bool { return compareValues(a, b) <= 0 },
	"gt": func(a, b interface{}) bool { return compareValues(a, b) > 0 },
	"ge": func(a, b interface{}) bool { return compareValues(a, b) >= 0 },
	"and": func(a, b bool) bool { return a && b },
	"or":  func(a, b bool) bool { return a || b },
	"not": func(a bool) bool { return !a },
	
	// Regular expressions
	"regexMatch": func(pattern, s string) bool {
		matched, _ := regexp.MatchString(pattern, s)
		return matched
	},
	"regexReplace": func(pattern, replacement, s string) string {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return s
		}
		return re.ReplaceAllString(s, replacement)
	},
	"regexFind": func(pattern, s string) string {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return ""
		}
		return re.FindString(s)
	},
	
	// Date and time functions
	"now": func() time.Time { return time.Now() },
	"date": func(format string, t time.Time) string {
		return t.Format(format)
	},
	"duration": func(d time.Duration) string {
		return d.String()
	},
	
	// Color and formatting functions (for terminal output)
	"colorize": func(color, text string) string {
		return colorizeText(color, text)
	},
	"bold": func(text string) string {
		return fmt.Sprintf("\033[1m%s\033[0m", text)
	},
	"italic": func(text string) string {
		return fmt.Sprintf("\033[3m%s\033[0m", text)
	},
	"underline": func(text string) string {
		return fmt.Sprintf("\033[4m%s\033[0m", text)
	},
	
	// Conditional functions
	"default": func(def, val interface{}) interface{} {
		if val == nil || (reflect.ValueOf(val).Kind() == reflect.String && val.(string) == "") {
			return def
		}
		return val
	},
	"empty": func(val interface{}) bool {
		if val == nil {
			return true
		}
		v := reflect.ValueOf(val)
		switch v.Kind() {
		case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
			return v.Len() == 0
		case reflect.Bool:
			return !v.Bool()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return v.Int() == 0
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return v.Uint() == 0
		case reflect.Float32, reflect.Float64:
			return v.Float() == 0
		}
		return false
	},
	
	// Math functions
	"add": func(a, b int) int { return a + b },
	"sub": func(a, b int) int { return a - b },
	"mul": func(a, b int) int { return a * b },
	"div": func(a, b int) int { 
		if b == 0 { return 0 }
		return a / b 
	},
	"mod": func(a, b int) int { 
		if b == 0 { return 0 }
		return a % b 
	},
	"max": func(a, b int) int {
		if a > b { return a }
		return b
	},
	"min": func(a, b int) int {
		if a < b { return a }
		return b
	},
}

// Helper functions for template functions
func humanizeNumber(n int64) string {
	if n < 1000 {
		return strconv.FormatInt(n, 10)
	}
	
	suffixes := []string{"", "K", "M", "B", "T"}
	suffixIndex := 0
	value := float64(n)
	
	for value >= 1000 && suffixIndex < len(suffixes)-1 {
		value /= 1000
		suffixIndex++
	}
	
	if value == float64(int64(value)) {
		return fmt.Sprintf("%.0f%s", value, suffixes[suffixIndex])
	}
	return fmt.Sprintf("%.1f%s", value, suffixes[suffixIndex])
}

func humanizeString(s string) string {
	// Convert snake_case and kebab-case to Title Case
	s = strings.ReplaceAll(s, "_", " ")
	s = strings.ReplaceAll(s, "-", " ")
	
	// Capitalize first letter of each word
	words := strings.Fields(s)
	for i, word := range words {
		if len(word) > 0 {
			words[i] = strings.ToUpper(string(word[0])) + strings.ToLower(word[1:])
		}
	}
	return strings.Join(words, " ")
}

func humanizeBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"B", "KB", "MB", "GB", "TB", "PB", "EB"}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), units[exp+1])
}

func compareValues(a, b interface{}) int {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	
	// Handle numeric comparisons
	if va.Kind() >= reflect.Int && va.Kind() <= reflect.Int64 {
		if vb.Kind() >= reflect.Int && vb.Kind() <= reflect.Int64 {
			ai, bi := va.Int(), vb.Int()
			if ai < bi { return -1 }
			if ai > bi { return 1 }
			return 0
		}
	}
	
	if va.Kind() >= reflect.Uint && va.Kind() <= reflect.Uint64 {
		if vb.Kind() >= reflect.Uint && vb.Kind() <= reflect.Uint64 {
			au, bu := va.Uint(), vb.Uint()
			if au < bu { return -1 }
			if au > bu { return 1 }
			return 0
		}
	}
	
	if va.Kind() == reflect.Float32 || va.Kind() == reflect.Float64 {
		if vb.Kind() == reflect.Float32 || vb.Kind() == reflect.Float64 {
			af, bf := va.Float(), vb.Float()
			if af < bf { return -1 }
			if af > bf { return 1 }
			return 0
		}
	}
	
	// Handle string comparisons
	if va.Kind() == reflect.String && vb.Kind() == reflect.String {
		as, bs := va.String(), vb.String()
		return strings.Compare(as, bs)
	}
	
	// Default to string representation comparison
	as, bs := fmt.Sprintf("%v", a), fmt.Sprintf("%v", b)
	return strings.Compare(as, bs)
}

func colorizeText(color, text string) string {
	colors := map[string]string{
		"black":   "\033[30m",
		"red":     "\033[31m",
		"green":   "\033[32m",
		"yellow":  "\033[33m",
		"blue":    "\033[34m",
		"magenta": "\033[35m",
		"cyan":    "\033[36m",
		"white":   "\033[37m",
		"gray":    "\033[90m",
		"grey":    "\033[90m",
		
		// Bright colors
		"bright-red":     "\033[91m",
		"bright-green":   "\033[92m",
		"bright-yellow":  "\033[93m",
		"bright-blue":    "\033[94m",
		"bright-magenta": "\033[95m",
		"bright-cyan":    "\033[96m",
		"bright-white":   "\033[97m",
		
		// Background colors
		"bg-red":     "\033[41m",
		"bg-green":   "\033[42m",
		"bg-yellow":  "\033[43m",
		"bg-blue":    "\033[44m",
		"bg-magenta": "\033[45m",
		"bg-cyan":    "\033[46m",
		"bg-white":   "\033[47m",
	}
	
	if colorCode, ok := colors[strings.ToLower(color)]; ok {
		return fmt.Sprintf("%s%s\033[0m", colorCode, text)
	}
	return text // Return unmodified if color not found
}

func EvaluateTemplateString(bt BreachTemplater, t string, b Breach) string {
	ctx := TemplateContext{
		Breach:       b,
		OutputFormat: "pretty",
		Severity:     b.GetSeverity(),
		CheckName:    b.GetCheckName(),
		CheckType:    b.GetCheckType(),
	}
	return EvaluateTemplateStringWithContext(bt, t, ctx)
}

func EvaluateTemplateStringWithContext(bt BreachTemplater, templateStr string, ctx TemplateContext) string {
	templ, err := template.New("breachTemplateString").
		Funcs(TemplateFuncs).Parse(templateStr)
	if err != nil {
		bt.AddBreach(&ValueBreach{
			ValueLabel: "unable to parse breach template",
			Value:      err.Error(),
		})
		return templateStr
	}

	buf := &bytes.Buffer{}
	err = templ.Execute(buf, ctx)
	if err != nil {
		bt.AddBreach(&ValueBreach{
			ValueLabel: "unable to render breach template",
			Value:      err.Error(),
		})
		return templateStr
	}
	return buf.String()
}
