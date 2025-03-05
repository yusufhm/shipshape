package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/salsadigitalauorg/shipshape/pkg/result"
)

type File struct {
	// Plugin-specific fields.
	Path   string `yaml:"path"`
	Format string `yaml:"format"`
	// Track if values were set by flags
	pathSetByFlag   bool
	formatSetByFlag bool
}

var f = &File{}

func init() {
	// Register the file outputter
	Outputters["file"] = f
}

// WasSetByFlag implements the FlagAware interface
func (p *File) WasSetByFlag() bool {
	// Consider values set by flag if either path or format was set
	return p.pathSetByFlag || p.formatSetByFlag
}

func (p *File) Output(rl *result.ResultList) ([]byte, error) {
	// If no path is provided, skip file output
	if p.Path == "" {
		return nil, nil
	}

	var buf bytes.Buffer
	// Use the same format as stdout
	format := s.Format
	if p.Format != "" {
		format = p.Format
	}

	switch format {
	case "pretty":
		s := &Stdout{}
		s.Pretty(rl, &buf)
	case "table":
		s := &Stdout{}
		s.Table(rl, &buf)
	case "json":
		data, err := json.Marshal(rl)
		if err != nil {
			return nil, fmt.Errorf("unable to convert result to json: %+v", err)
		}
		fmt.Fprintln(&buf, string(data))
	case "junit":
		s := &Stdout{}
		s.JUnit(rl, &buf)
	default:
		return nil, fmt.Errorf("unsupported output format: %s", format)
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(p.Path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create output directory: %v", err)
		}
	}

	// Write to file
	if err := os.WriteFile(p.Path, buf.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("failed to write output file: %v", err)
	}

	return buf.Bytes(), nil
}
