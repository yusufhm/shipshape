package output

import (
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/salsadigitalauorg/shipshape/pkg/flagsprovider"
)

func init() {
	flagsprovider.Registry["stdout"] = func() flagsprovider.FlagsProvider {
		return s
	}
	flagsprovider.Registry["file"] = func() flagsprovider.FlagsProvider {
		return f
	}
}

func (f *Stdout) ValidateOutputFormat() bool {
	valid := false
	for _, fm := range OutputFormats {
		if f.Format == fm {
			valid = true
			break
		}
	}
	return valid
}

func (f *File) ValidateOutputFormat() bool {
	valid := false
	for _, fm := range OutputFormats {
		if f.Format == fm {
			valid = true
			break
		}
	}
	return valid
}

func (f *Stdout) AddFlags(c *cobra.Command) {
	c.Flags().StringVarP(&f.Format, "output-format",
		"o", "pretty", `Output format [pretty|table|json|junit]
(env: SHIPSHAPE_OUTPUT_FORMAT)`)

	oldPreRun := c.PreRun
	c.PreRun = func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("output-format") {
			f.formatSetByFlag = true
		}
		if oldPreRun != nil {
			oldPreRun(cmd, args)
		}
	}
}

func (f *File) AddFlags(c *cobra.Command) {
	c.Flags().StringVar(&f.Path, "output-file",
		"", `Path to output file
(env: SHIPSHAPE_OUTPUT_FILE)`)
	c.Flags().StringVar(&f.Format, "output-file-format",
		"", `Format for the output file [pretty|table|json|junit]
(env: SHIPSHAPE_OUTPUT_FILE_FORMAT)`)

	oldPreRun := c.PreRun
	c.PreRun = func(cmd *cobra.Command, args []string) {
		if cmd.Flags().Changed("output-file") {
			f.pathSetByFlag = true
		}
		if cmd.Flags().Changed("output-file-format") {
			f.formatSetByFlag = true
		}
		if oldPreRun != nil {
			oldPreRun(cmd, args)
		}
	}
}

func (f *Stdout) EnvironmentOverrides() {
	if outputFormatEnv := os.Getenv("SHIPSHAPE_OUTPUT_FORMAT"); outputFormatEnv != "" {
		if !f.formatSetByFlag {
			f.Format = outputFormatEnv
			f.formatSetByFlag = true
		}
	}

	if !f.ValidateOutputFormat() {
		log.Fatalf("Invalid output format; needs to be one of: %s.",
			strings.Join(OutputFormats, "|"))
	}
}

func (f *File) EnvironmentOverrides() {
	if outputFileEnv := os.Getenv("SHIPSHAPE_OUTPUT_FILE"); outputFileEnv != "" {
		if !f.pathSetByFlag {
			f.Path = outputFileEnv
			f.pathSetByFlag = true
		}
	}

	if outputFileFormatEnv := os.Getenv("SHIPSHAPE_OUTPUT_FILE_FORMAT"); outputFileFormatEnv != "" {
		if !f.formatSetByFlag {
			f.Format = outputFileFormatEnv
			f.formatSetByFlag = true
		}
	}

	if f.Format != "" && !f.ValidateOutputFormat() {
		log.Fatalf("Invalid output file format; needs to be one of: %s.",
			strings.Join(OutputFormats, "|"))
	}
}
