package output_test

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	. "github.com/salsadigitalauorg/shipshape/pkg/output"
)

func TestFlagPriority(t *testing.T) {
	// Test cases for stdout outputter
	t.Run("stdout flag priority", func(t *testing.T) {
		assert := assert.New(t)

		// Create a new stdout outputter
		stdout := &Stdout{Format: "pretty"}

		// Create a test command
		cmd := &cobra.Command{Use: "test"}

		// Add flags to the command
		stdout.AddFlags(cmd)

		// Set the flag value
		cmd.Flags().Set("output-format", "table")

		// Simulate PreRun to mark the flag as set
		cmd.PreRun(cmd, []string{})

		// Create a config that would override the format
		config := map[string]interface{}{
			"stdout": map[string]interface{}{
				"format": "json",
			},
		}

		// Now test with the actual ParseConfig function
		// Reset the outputters map
		originalOutputters := Outputters
		defer func() { Outputters = originalOutputters }()

		// Register our test stdout outputter
		Outputters = map[string]Outputter{"stdout": stdout}

		// Parse the config
		ParseConfig(config, nil)

		// Verify that the format is still "table" (set by flag)
		assert.Equal("table", stdout.Format)
	})

	// Test cases for file outputter
	t.Run("file flag priority", func(t *testing.T) {
		assert := assert.New(t)

		// Create a new file outputter
		file := &File{}

		// Create a test command
		cmd := &cobra.Command{Use: "test"}

		// Add flags to the command
		file.AddFlags(cmd)

		// Set the flag values
		cmd.Flags().Set("output-file", "test.xml")
		cmd.Flags().Set("output-file-format", "junit")

		// Simulate PreRun to mark the flags as set
		cmd.PreRun(cmd, []string{})

		// Create a config that would override the values
		config := map[string]interface{}{
			"file": map[string]interface{}{
				"path":   "different.json",
				"format": "json",
			},
		}

		// Now test with the actual ParseConfig function
		// Reset the outputters map
		originalOutputters := Outputters
		defer func() { Outputters = originalOutputters }()

		// Register our test file outputter
		Outputters = map[string]Outputter{"file": file}

		// Parse the config
		ParseConfig(config, nil)

		// Verify that the values are still set by flag
		assert.Equal("test.xml", file.Path)
		assert.Equal("junit", file.Format)
	})
}
