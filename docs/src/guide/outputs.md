# Outputs

ShipShape supports multiple output formats and destinations for check results. You can configure outputs both in your configuration file and via command-line flags.

## Output Formats

The following output formats are supported:

- `pretty` - Human-readable format with detailed breach information (default)
- `table` - Tabular format showing test status and results
- `json` - JSON format for machine processing
- `junit` - JUnit XML format for CI/CD integration

## Command Line Flags

You can control the output using these flags:

- `--output-format` or `-o`: Set the output format for stdout (overrides config file)
- `--output-file`: Specify a file to write the output to
- `--output-file-format`: Set the format for file output (defaults to the same as stdout)

Example usage:
```bash
# Output to stdout in table format
shipshape run . -o table

# Output to both stdout and file
shipshape run . -o pretty --output-file results.xml --output-file-format junit
```

## Configuration File

You can also configure outputs in your `shipshape.yml` file:

```yaml
output:
  stdout:
    format: pretty  # Format for stdout output
  file:
    path: results/output.xml  # File to write output to
    format: junit  # Format for file output
```

## Flag Priority

Command-line flags take precedence over configuration file settings. This allows you to:

1. Set default output settings in your configuration file
2. Override them when needed using command-line flags

For example, if your configuration file specifies JSON output, but you run with `-o junit`, the output will be in JUnit format.

## Output Format Details

### Pretty Format
The default format, optimized for human readability. It shows:
- Overall status
- Detailed breach information
- Remediation status and results

### Table Format
A compact tabular view showing:
- Check names
- Status (Pass/Fail)
- Pass messages
- Breach messages

### JSON Format
Machine-readable format containing:
- Complete check results
- Breach details
- Remediation information
- Statistics and metadata

### JUnit Format
XML format compatible with CI/CD systems, including:
- Test suite organization
- Individual test case results
- Error details for failures
- Test counts and statistics



