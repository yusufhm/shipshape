# Enhanced Templating Guide

ShipShape now includes a powerful templating system that allows you to create rich, context-aware breach reports with advanced formatting capabilities.

## Overview

The enhanced templating system provides:

- **60+ template functions** for string manipulation, formatting, and logic
- **Context-aware templates** that adapt based on output format and severity
- **Format-specific templates** for different output types (pretty, JSON, table)
- **Conditional logic** and flow control within templates
- **Rich formatting** with colors, emojis, and human-friendly output

## Basic Template Usage

### Legacy Template Format (Still Supported)
```yaml
breach-format:
  type: key-value
  key-label: "Location"
  key: '{{ .Breach.Key }}'
  value-label: "Issue"
  value: '{{ .Breach.Value }}'
```

### Enhanced Template Format
```yaml
breach-format:
  template: |
    🚨 {{ .CheckName | humanize }}
    📁 Found {{ len .Breach.Values }} {{ pluralize (len .Breach.Values) "issue" "issues" }}
    {{ range .Breach.Values }}• {{ . | colorize "red" }}{{ end }}
```

## Template Context

Templates have access to a rich context object:

```yaml
# Available context variables
.Breach          # The breach object with Key, Value, Values, etc.
.OutputFormat    # Current output format (pretty, json, table, junit)
.Severity        # Breach severity (low, normal, high, critical)
.CheckName       # Name of the check that generated the breach
.CheckType       # Type of check (e.g., "regex:match")
.Context         # User-defined context data
```

## Template Functions

### String Manipulation
```yaml
template: |
  {{ "hello world" | upper }}                    # HELLO WORLD
  {{ "HELLO WORLD" | lower }}                    # hello world  
  {{ "test_string_value" | humanize }}           # Test String Value
  {{ "very long text here" | truncate 10 }}     # very lo...
  {{ "text" | repeat 3 }}                       # texttexttext
  {{ printf "Found %d items" 5 }}               # Found 5 items
```

### Array/List Operations
```yaml
template: |
  {{ .Breach.Values | len }}                     # Number of items
  {{ .Breach.Values | first }}                   # First item
  {{ .Breach.Values | last }}                    # Last item
  {{ .Breach.Values | join ", " }}               # Comma-separated list
  {{ slice 0 3 .Breach.Values }}                 # First 3 items
```

### Conditional Logic
```yaml
template: |
  {{ if eq .Severity "high" }}🚨 CRITICAL{{ else }}ℹ️ INFO{{ end }}
  {{ if gt (len .Breach.Values) 5 }}
    Many issues found ({{ len .Breach.Values }})
  {{ else }}
    {{ len .Breach.Values }} issues found
  {{ end }}
```

### Number and Data Formatting
```yaml
template: |
  {{ 1500 | humanize }}                          # 1.5K
  {{ 1048576 | bytes }}                          # 1.0 MB
  {{ 5 | pluralize "file" "files" }}             # files
  {{ 1 | pluralize "error" "errors" }}           # error
```

### Regular Expressions
```yaml
template: |
  {{ if regexMatch "^test.*" .Breach.Value }}
    This is a test value
  {{ end }}
  {{ regexReplace "old" "new" .Breach.Value }}
```

### Color and Formatting
```yaml
template: |
  {{ .Breach.Value | colorize "red" }}
  {{ "Important" | bold }}
  {{ "Emphasized" | italic }}
  {{ "Underlined" | underline }}
```

### Math Operations
```yaml
template: |
  Total: {{ add 5 3 }}                           # 8
  Remaining: {{ sub 10 3 }}                      # 7
  Max: {{ max 5 8 }}                             # 8
  Percentage: {{ div (mul .Found 100) .Total }}%
```

### Fact Lookup
```yaml
template: |
  # Generic lookup (handles any data type)
  Form: {{ lookup "webform-titles" .Breach.Key }}
  
  # Lookup with default value
  Status: {{ lookupDefault "status-data" .Breach.Key "unknown" }}
```

## Format-Specific Templates

Create different templates for different output formats:

```yaml
breach-format:
  templates:
    pretty: |
      🚨 {{ .CheckName | humanize }}
      📊 Found {{ len .Breach.Values | humanize }} issues:
      {{ range .Breach.Values }}• {{ . | colorize "red" }}{{ end }}
      
    json: |
      {"check": "{{ .CheckName }}", "count": {{ len .Breach.Values }}, "issues": [{{ range $i, $v := .Breach.Values }}{{ if $i }}, {{ end }}"{{ $v }}"{{ end }}]}
      
    table: |
      {{ .CheckName }}\t{{ len .Breach.Values }}\t{{ .Breach.Values | join ", " | truncate 50 }}
      
    default: |
      {{ .CheckName }}: {{ len .Breach.Values }} issues found
```

## Context-Aware Templates

Use custom context data for more dynamic templates:

```yaml
breach-format:
  context:
    environment: "production"
    team: "security"
    threshold: 10
  template: |
    {{ if eq .Context.environment "production" }}🔴 PROD ALERT{{ else }}🟡 DEV{{ end }}
    
    {{ if gt (len .Breach.Values) .Context.threshold }}
      ⚠️  High volume: {{ len .Breach.Values }} issues (threshold: {{ .Context.threshold }})
    {{ else }}
      ℹ️  Normal volume: {{ len .Breach.Values }} issues
    {{ end }}
    
    👥 Assigned to: {{ .Context.team | title }} team
```

## Advanced Examples

### Security Issue Report
```yaml
breach-format:
  template: |
    {{ if eq .Severity "high" }}🚨{{ else if eq .Severity "medium" }}⚠️{{ else }}ℹ️{{ end }} Security Issue Detected
    
    📋 Check: {{ .CheckName | humanize }}
    🎯 Location: {{ .Breach.Key | default "Global" }}
    📊 Severity: {{ .Severity | upper | colorize (if eq .Severity "high") "red" (if eq .Severity "medium") "yellow" "green" }}
    
    {{ if .Breach.Values }}
    📝 Issues found ({{ len .Breach.Values }}):
    {{ range slice 0 (min 5 (len .Breach.Values)) .Breach.Values }}• {{ . | truncate 80 }}{{ end }}
    {{ if gt (len .Breach.Values) 5 }}... and {{ sub (len .Breach.Values) 5 }} more{{ end }}
    {{ else }}
    📝 Issue: {{ .Breach.Value }}
    {{ end }}
```

### File Analysis Report
```yaml
breach-format:
  template: |
    📁 File Analysis Results
    
    {{ if eq .OutputFormat "pretty" }}
    {{ if gt (len .Breach.Values) 100 }}{{ colorize "red" "⚠️  HIGH" }}{{ else if gt (len .Breach.Values) 50 }}{{ colorize "yellow" "⚠️  MEDIUM" }}{{ else }}{{ colorize "green" "✅ LOW" }}{{ end }} file count
    {{ end }}
    
    📈 Statistics:
    • Total files: {{ len .Breach.Values | humanize }}
    • File types: {{ .Breach.Values | join ", " | regexReplace "\\.[^,]*" "" | split ", " | len }} unique
    • Sample files:
    {{ range slice 0 (min 3 (len .Breach.Values)) .Breach.Values }}  - {{ . | truncate 60 }}{{ end }}
    
    {{ if gt (len .Breach.Values) 1000 }}
    💡 Recommendation: Consider implementing file cleanup policies
    {{ end }}
```

## Migration from Legacy Templates

### Before (Legacy)
```yaml
breach-format:
  type: key-value
  key-label: webform
  key: ' {{ lookupFactAsStringMap "webform-titles" .Breach.Key }}'
  value-label: 'Handler "{{ .Breach.ValueLabel }}" has token'
  value: '{{ .Breach.Value }}'
```

### After (Enhanced)
```yaml
breach-format:
  # Legacy format still works for backward compatibility
  type: key-value
  key-label: webform
  key: ' {{ lookup "webform-titles" .Breach.Key }}'
  value-label: 'Handler "{{ .Breach.ValueLabel }}" has token'
  value: '{{ .Breach.Value }}'
  
  # Enhanced template overrides legacy format
  template: |
    📧 {{ colorize "yellow" "Email Token Issue" }}
    🎯 Form: {{ lookup "webform-titles" .Breach.Key | default "Unknown Form" | title }}
    📝 Handler: {{ .Breach.ValueLabel | printf "\"%s\"" }}
    🔧 Token: {{ .Breach.Value | colorize "red" }}
    💡 Consider using static email addresses for better security
```

## Complete Function Reference

### String Functions
- `printf` - Format strings with placeholders
- `join` - Join array elements with separator  
- `split` - Split string by separator
- `replace` - Replace all occurrences
- `trim`, `trimLeft`, `trimRight` - Trim whitespace
- `upper`, `lower`, `title` - Case conversion
- `truncate`, `ellipsis` - Shorten strings
- `pad`, `padLeft` - Pad strings to width
- `repeat` - Repeat string N times
- `contains`, `hasPrefix`, `hasSuffix` - String tests

### Array/Slice Functions  
- `len` - Get length
- `first`, `last` - Get first/last element
- `slice` - Get sub-slice
- `join` - Join elements with separator

### Comparison Functions
- `eq`, `ne` - Equal/not equal
- `lt`, `le`, `gt`, `ge` - Comparisons  
- `and`, `or`, `not` - Boolean logic

### Math Functions
- `add`, `sub`, `mul`, `div`, `mod` - Arithmetic
- `max`, `min` - Min/max values

### Formatting Functions
- `humanize` - Human-friendly formatting
- `pluralize` - Singular/plural forms
- `bytes` - Format byte sizes
- `colorize`, `bold`, `italic`, `underline` - Text formatting

### Utility Functions
- `default` - Provide default value
- `empty` - Test if empty
- `regexMatch`, `regexReplace`, `regexFind` - Regular expressions
- `now`, `date`, `duration` - Date/time functions
- `lookup`, `lookupDefault` - Fact lookup functions

This enhanced templating system provides the flexibility to create rich, informative breach reports that adapt to different contexts and output formats while maintaining backward compatibility with existing configurations.