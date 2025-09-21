package breach

import "github.com/salsadigitalauorg/shipshape/pkg/remediation"

// Breach provides a representation for different breach types.
type Breach interface {
	GetCheckName() string
	GetCheckType() string
	GetRemediator() remediation.Remediator
	GetRemediationResult() *remediation.RemediationResult
	GetSeverity() string
	GetType() BreachType
	SetCommonValues(checkType string, checkName string, severity string)
	SetRemediator(remediation.Remediator)
	PerformRemediation()
	SetRemediation(status remediation.RemediationStatus, msg string)
	String() string
}

type BreachType string

type BreachTemplate struct {
	Type       BreachType `yaml:"type"`
	KeyLabel   string     `yaml:"key-label,omitempty"`
	Key        string     `yaml:"key,omitempty"`
	ValueLabel string     `yaml:"value-label,omitempty"`
	Value      string     `yaml:"value,omitempty"`
	
	// Enhanced templating support
	Template  string                       `yaml:"template,omitempty"`   // Single template for all formats
	Templates map[string]string           `yaml:"templates,omitempty"`  // Format-specific templates
	Context   map[string]interface{}      `yaml:"context,omitempty"`    // Additional context data
}

// TemplateContext provides the context for template evaluation
type TemplateContext struct {
	Breach       Breach
	OutputFormat string
	Severity     string
	CheckName    string
	CheckType    string
	Context      map[string]interface{} // User-defined context
}

type BreachTemplater interface {
	AddBreach(b Breach)
	GetBreachTemplate() BreachTemplate
}
