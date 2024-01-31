package json_test

import (
	"testing"

	. "github.com/salsadigitalauorg/shipshape/pkg/checks/json"
	"github.com/salsadigitalauorg/shipshape/pkg/checks/yaml"
	"github.com/salsadigitalauorg/shipshape/pkg/config"
	"github.com/salsadigitalauorg/shipshape/pkg/result"
	"github.com/stretchr/testify/assert"
)

var cFalse = false
var cTrue = true

func TestJsonCheckMerge(t *testing.T) {
	assertions := assert.New(t)

	c := JsonCheck{
		YamlCheck: yaml.YamlCheck{
			Path:           "path1",
			File:           "file1.json",
			Pattern:        "pattern1",
			ExcludePattern: "excludePattern1",
			IgnoreMissing:  &cFalse,
		},
	}
	err := c.Merge(&JsonCheck{
		YamlCheck: yaml.YamlCheck{
			Path:  "path2",
			Files: []string{"slcFile1.json", "slcFile2.json"},
		},
	})
	if err != nil {
		assertions.Fail("Failed to merge JsonCheck")
		return
	}
	assertions.EqualValues(JsonCheck{
		YamlCheck: yaml.YamlCheck{
			Path:           "path2",
			File:           "file1.json",
			Files:          []string{"slcFile1.json", "slcFile2.json"},
			Pattern:        "pattern1",
			ExcludePattern: "excludePattern1",
			IgnoreMissing:  &cFalse,
		},
	}, c)

	err = c.Merge(&JsonCheck{
		YamlCheck: yaml.YamlCheck{
			YamlBase: yaml.YamlBase{
				CheckBase: config.CheckBase{Name: "Some name"},
			},
			Path:  "path2",
			Files: []string{"slcFile3.json", "slcFile3.json"},
		},
	})
	assertions.EqualError(err, "can only merge checks with the same name")
}

func TestJsonCheckRunCheck(t *testing.T) {
	assertions := assert.New(t)

	mockCheck := func() JsonCheck {
		return JsonCheck{
			KeyValues: []KeyValue{
				{
					KeyValue: yaml.KeyValue{
						Key:   "$.license",
						Value: "MIT",
					},
					DisallowedValues: nil,
					AllowedValues:    nil,
				},
			},
		}
	}

	c := mockCheck()
	c.FetchData()
	assertions.Equal(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Passes)
	assertions.ElementsMatch(
		[]result.Breach{
			&result.ValueBreach{
				BreachType: result.BreachTypeValue,
				ValueLabel: "- no file",
				Value:      "no file provided",
			},
		},
		c.Result.Breaches,
	)

	// Non-existent file.
	config.ProjectDir = "testdata"
	c = mockCheck()
	c.Init(Json)
	c.File = "non-existent.json"
	c.FetchData()
	assertions.Equal(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Passes)
	assertions.ElementsMatch(
		[]result.Breach{
			&result.ValueBreach{
				CheckType:  "json",
				Severity:   "normal",
				BreachType: result.BreachTypeValue,
				ValueLabel: "error reading file: testdata/non-existent.json",
				Value:      "open testdata/non-existent.json: no such file or directory",
			},
		},
		c.Result.Breaches,
	)

	// Non-existent file with ignore missing.
	c = mockCheck()
	c.File = "non-existent.json"
	c.IgnoreMissing = &cTrue
	c.FetchData()
	assertions.Equal(result.Pass, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	assertions.EqualValues([]string{"File testdata/non-existent.json does not exist"}, c.Result.Passes)

	// Single file.
	c = mockCheck()
	c.File = "composer.map.json"
	c.FetchData()
	// Should not fail yet.
	assertions.NotEqual(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	assertions.True(c.HasData(false))
	c.UnmarshalDataMap()
	c.RunCheck()
	assertions.Equal(result.Pass, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	assertions.EqualValues([]string{"[composer.map.json] '$.license' equals 'MIT'"}, c.Result.Passes)

	// Bad File pattern.
	c = mockCheck()
	c.Pattern = "*.composer.json"
	c.Path = ""
	c.FetchData()
	assertions.Equal(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Passes)
	assertions.ElementsMatch(
		[]result.Breach{
			&result.ValueBreach{
				BreachType: result.BreachTypeValue,
				ValueLabel: "error finding files in path: testdata",
				Value:      "error parsing regexp: missing argument to repetition operator: `*`",
			},
		},
		c.Result.Breaches,
	)

	// File pattern with no matching files.
	c = mockCheck()
	c.Pattern = "composer*.json"
	c.FetchData()
	assertions.Equal(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Passes)
	assertions.ElementsMatch(
		[]result.Breach{
			&result.ValueBreach{
				BreachType: result.BreachTypeValue,
				ValueLabel: "- no file",
				Value:      "no matching yaml files found",
			},
		},
		c.Result.Breaches,
	)

	// File pattern with no matching files, ignoring missing.
	c = mockCheck()
	c.Pattern = "composer*.json"
	c.IgnoreMissing = &cTrue
	c.FetchData()
	assertions.Equal(result.Pass, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	assertions.EqualValues([]string{"no matching config files found"}, c.Result.Passes)

	// Correct single file pattern & value.
	c = mockCheck()
	c.Pattern = "composer.map.json"
	c.Path = "dir/subdir"
	c.FetchData()
	assertions.NotEqual(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	c.UnmarshalDataMap()
	c.RunCheck()
	assertions.EqualValues([]string{"[testdata/dir/subdir/composer.map.json] '$.license' equals 'MIT'"}, c.Result.Passes)
	assertions.Empty(c.Result.Breaches)

	// Recursive file lookup.
	c = mockCheck()
	c.Pattern = ".*.*.json"
	c.FetchData()
	assertions.NotEqual(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	c.UnmarshalDataMap()
	c.RunCheck()
	assertions.Equal(result.Fail, c.Result.Status)
	assertions.ElementsMatch(
		[]string{
			"[testdata/dir/composer.map.json] '$.license' equals 'MIT'",
			"[testdata/dir/subdir/composer.map.json] '$.license' equals 'MIT'",
			"[testdata/composer.map.json] '$.license' equals 'MIT'",
		},
		c.Result.Passes)
	assertions.ElementsMatch(
		[]result.Breach{
			&result.KeyValueBreach{
				BreachType:    result.BreachTypeKeyValue,
				KeyLabel:      "testdata/composer.array.json",
				Key:           "$.license",
				ValueLabel:    "actual",
				Value:         "BSD",
				ExpectedValue: "MIT",
			},
			&result.KeyValueBreach{
				BreachType:    result.BreachTypeKeyValue,
				KeyLabel:      "testdata/dir/composer.array.json",
				Key:           "$.license",
				ValueLabel:    "actual",
				Value:         "BSD",
				ExpectedValue: "MIT",
			},
			&result.KeyValueBreach{
				BreachType:    result.BreachTypeKeyValue,
				KeyLabel:      "testdata/dir/subdir/composer.array.json",
				Key:           "$.license",
				ValueLabel:    "actual",
				Value:         "BSD",
				ExpectedValue: "MIT",
			},
		},
		c.Result.Breaches,
	)

	// Test disallowed values.
	c = JsonCheck{
		KeyValues: []KeyValue{
			{
				KeyValue: yaml.KeyValue{
					Key:   "$.license",
					Value: "MIT",
				},
				DisallowedValues: []any{"MIT", "BSD"},
				AllowedValues:    nil,
			},
		},
	}
	c.File = "composer.map.json"
	c.FetchData()
	assertions.NotEqual(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	assertions.True(c.HasData(false))
	c.UnmarshalDataMap()
	c.RunCheck()
	assertions.Equal(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Passes)
	assertions.ElementsMatch(
		[]result.Breach{
			&result.KeyValuesBreach{
				BreachType: result.BreachTypeKeyValues,
				KeyLabel:   "config",
				Key:        "composer.map.json",
				ValueLabel: "disallowed $.license",
				Values:     []string{"MIT"},
			},
		},
		c.Result.Breaches)

	// Test allowed values not matched.
	c = JsonCheck{
		KeyValues: []KeyValue{
			{
				KeyValue: yaml.KeyValue{
					Key:   "$.license",
					Value: "MIT",
				},
				AllowedValues:    []any{"BSD", "GPL"},
				DisallowedValues: nil,
			},
		},
	}
	c.File = "composer.map.json"
	c.FetchData()
	assertions.NotEqual(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	assertions.True(c.HasData(false))
	c.UnmarshalDataMap()
	c.RunCheck()
	assertions.Equal(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Passes)
	assertions.ElementsMatch(
		[]result.Breach{
			&result.KeyValuesBreach{
				BreachType: result.BreachTypeKeyValues,
				KeyLabel:   "config",
				Key:        "composer.map.json",
				ValueLabel: "disallowed $.license",
				Values:     []string{"MIT"},
			},
		},
		c.Result.Breaches)

	// Test incorrect key value.
	c = JsonCheck{
		KeyValues: []KeyValue{
			{
				KeyValue: yaml.KeyValue{
					Key:   "$.**);",
					Value: "foo",
				},
				AllowedValues:    []any{"BSD", "GPL"},
				DisallowedValues: nil,
			},
		},
	}
	c.File = "composer.map.json"
	c.FetchData()
	assertions.NotEqual(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	assertions.True(c.HasData(false))
	c.UnmarshalDataMap()
	c.RunCheck()
	assertions.Equal(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Passes)
	assertions.ElementsMatch(
		[]result.Breach{
			&result.ValueBreach{
				BreachType: result.BreachTypeValue,
				Value:      "json: invalid path format: found invalid path character * after dot",
			},
		},
		c.Result.Breaches)

	// Test non-existent key value.
	c = JsonCheck{
		KeyValues: []KeyValue{
			{
				KeyValue: yaml.KeyValue{
					Key:   "$.authors",
					Value: "foo",
				},
			},
		},
	}
	c.File = "composer.map.json"
	c.FetchData()
	assertions.NotEqual(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	assertions.True(c.HasData(false))
	c.UnmarshalDataMap()
	c.RunCheck()
	assertions.Equal(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Passes)
	assertions.ElementsMatch(
		[]result.Breach{
			&result.KeyValueBreach{
				BreachType: result.BreachTypeKeyValue,
				KeyLabel:   "config",
				Key:        "composer.map.json",
				ValueLabel: "key not found",
				Value:      "$.authors",
			},
		},
		c.Result.Breaches)

	// Test is-list.
	c = JsonCheck{
		KeyValues: []KeyValue{
			{
				KeyValue: yaml.KeyValue{
					Key:    "repositories.*.type",
					IsList: true,
				},
				AllowedValues: []any{"vcs", "library", "project"},
			},
		},
	}
	c.File = "composer.map.json"
	c.FetchData()
	assertions.NotEqual(result.Fail, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	assertions.True(c.HasData(false))
	c.UnmarshalDataMap()
	c.RunCheck()
	assertions.Equal(result.Pass, c.Result.Status)
	assertions.Empty(c.Result.Breaches)
	assertions.ElementsMatch(
		[]string{
			"[composer.map.json] no disallowed 'repositories.*.type'",
		},
		c.Result.Passes)
}
