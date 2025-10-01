package transform

import (
	"fmt"
	"strings"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// TemplateRegistry manages built-in scenario templates
type TemplateRegistry struct {
	templates map[string]Template
}

// Template represents a named collection of transforms
type Template struct {
	Name        string
	Description string
	Transforms  []ScenarioTransform
}

// NewTemplateRegistry creates a new template registry with built-in templates
func NewTemplateRegistry() *TemplateRegistry {
	return &TemplateRegistry{
		templates: make(map[string]Template),
	}
}

// Register adds a template to the registry
func (tr *TemplateRegistry) Register(t Template) {
	tr.templates[strings.ToLower(t.Name)] = t
}

// Get retrieves a template by name (case-insensitive)
func (tr *TemplateRegistry) Get(name string) (Template, bool) {
	t, ok := tr.templates[strings.ToLower(name)]
	return t, ok
}

// List returns all registered template names
func (tr *TemplateRegistry) List() []string {
	names := make([]string, 0, len(tr.templates))
	for name := range tr.templates {
		names = append(names, name)
	}
	return names
}

// CreateBuiltInTemplates creates a template registry with common retirement scenarios
func CreateBuiltInTemplates(participantName string) *TemplateRegistry {
	registry := NewTemplateRegistry()

	// Postpone retirement templates
	registry.Register(Template{
		Name:        "postpone_1yr",
		Description: "Postpone retirement by 1 year (12 months)",
		Transforms: []ScenarioTransform{
			&PostponeRetirement{Participant: participantName, Months: 12},
		},
	})

	registry.Register(Template{
		Name:        "postpone_2yr",
		Description: "Postpone retirement by 2 years (24 months)",
		Transforms: []ScenarioTransform{
			&PostponeRetirement{Participant: participantName, Months: 24},
		},
	})

	registry.Register(Template{
		Name:        "postpone_3yr",
		Description: "Postpone retirement by 3 years (36 months)",
		Transforms: []ScenarioTransform{
			&PostponeRetirement{Participant: participantName, Months: 36},
		},
	})

	// Social Security delay templates
	registry.Register(Template{
		Name:        "delay_ss_67",
		Description: "Delay Social Security claiming to age 67 (Full Retirement Age)",
		Transforms: []ScenarioTransform{
			&DelaySSClaim{Participant: participantName, NewAge: 67},
		},
	})

	registry.Register(Template{
		Name:        "delay_ss_70",
		Description: "Delay Social Security claiming to age 70 (Maximum benefit)",
		Transforms: []ScenarioTransform{
			&DelaySSClaim{Participant: participantName, NewAge: 70},
		},
	})

	// TSP strategy templates
	registry.Register(Template{
		Name:        "tsp_need_based",
		Description: "Switch to need-based TSP withdrawals",
		Transforms: []ScenarioTransform{
			&ModifyTSPStrategy{Participant: participantName, NewStrategy: "need_based"},
		},
	})

	registry.Register(Template{
		Name:        "tsp_fixed_2pct",
		Description: "Switch to fixed percentage TSP withdrawals at 2%",
		Transforms: []ScenarioTransform{
			&ModifyTSPStrategy{Participant: participantName, NewStrategy: "variable_percentage"},
			&AdjustTSPRate{Participant: participantName, NewRate: decimal.NewFromFloat(0.02)},
		},
	})

	registry.Register(Template{
		Name:        "tsp_fixed_3pct",
		Description: "Switch to fixed percentage TSP withdrawals at 3%",
		Transforms: []ScenarioTransform{
			&ModifyTSPStrategy{Participant: participantName, NewStrategy: "variable_percentage"},
			&AdjustTSPRate{Participant: participantName, NewRate: decimal.NewFromFloat(0.03)},
		},
	})

	registry.Register(Template{
		Name:        "tsp_fixed_4pct",
		Description: "Switch to fixed percentage TSP withdrawals at 4% (traditional safe withdrawal rate)",
		Transforms: []ScenarioTransform{
			&ModifyTSPStrategy{Participant: participantName, NewStrategy: "4_percent_rule"},
			&AdjustTSPRate{Participant: participantName, NewRate: decimal.NewFromFloat(0.04)},
		},
	})

	// Combination templates - popular strategies
	registry.Register(Template{
		Name:        "postpone_1yr_delay_ss_70",
		Description: "Postpone retirement 1 year + delay SS to 70",
		Transforms: []ScenarioTransform{
			&PostponeRetirement{Participant: participantName, Months: 12},
			&DelaySSClaim{Participant: participantName, NewAge: 70},
		},
	})

	registry.Register(Template{
		Name:        "postpone_2yr_delay_ss_70",
		Description: "Postpone retirement 2 years + delay SS to 70",
		Transforms: []ScenarioTransform{
			&PostponeRetirement{Participant: participantName, Months: 24},
			&DelaySSClaim{Participant: participantName, NewAge: 70},
		},
	})

	registry.Register(Template{
		Name:        "delay_ss_70_tsp_4pct",
		Description: "Delay SS to 70 + 4% TSP withdrawal rate",
		Transforms: []ScenarioTransform{
			&DelaySSClaim{Participant: participantName, NewAge: 70},
			&ModifyTSPStrategy{Participant: participantName, NewStrategy: "4_percent_rule"},
			&AdjustTSPRate{Participant: participantName, NewRate: decimal.NewFromFloat(0.04)},
		},
	})

	registry.Register(Template{
		Name:        "conservative",
		Description: "Conservative strategy: Postpone 2 years, delay SS to 70, 3% TSP",
		Transforms: []ScenarioTransform{
			&PostponeRetirement{Participant: participantName, Months: 24},
			&DelaySSClaim{Participant: participantName, NewAge: 70},
			&ModifyTSPStrategy{Participant: participantName, NewStrategy: "variable_percentage"},
			&AdjustTSPRate{Participant: participantName, NewRate: decimal.NewFromFloat(0.03)},
		},
	})

	registry.Register(Template{
		Name:        "aggressive",
		Description: "Aggressive strategy: Delay SS to 70, 4% TSP withdrawal",
		Transforms: []ScenarioTransform{
			&DelaySSClaim{Participant: participantName, NewAge: 70},
			&ModifyTSPStrategy{Participant: participantName, NewStrategy: "4_percent_rule"},
			&AdjustTSPRate{Participant: participantName, NewRate: decimal.NewFromFloat(0.04)},
		},
	})

	return registry
}

// ApplyTemplate applies a template to a base scenario
func ApplyTemplate(base *domain.GenericScenario, template Template) (*domain.GenericScenario, error) {
	if len(template.Transforms) == 0 {
		return base.DeepCopy(), nil
	}
	return ApplyTransforms(base, template.Transforms)
}

// ParseTemplateList parses a comma-separated list of template names
func ParseTemplateList(templateList string) []string {
	if templateList == "" {
		return nil
	}

	parts := strings.Split(templateList, ",")
	templates := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			templates = append(templates, trimmed)
		}
	}
	return templates
}

// GetTemplateHelp returns formatted help text for all templates
func GetTemplateHelp(registry *TemplateRegistry) string {
	if len(registry.templates) == 0 {
		return "No templates registered"
	}

	var sb strings.Builder
	sb.WriteString("Available Templates:\n\n")

	// Sort templates by category
	categories := map[string][]Template{
		"Retirement Timing":      {},
		"Social Security":        {},
		"TSP Strategies":         {},
		"Combination Strategies": {},
	}

	for _, template := range registry.templates {
		name := template.Name
		if strings.HasPrefix(name, "postpone_") {
			categories["Retirement Timing"] = append(categories["Retirement Timing"], template)
		} else if strings.HasPrefix(name, "delay_ss_") {
			categories["Social Security"] = append(categories["Social Security"], template)
		} else if strings.HasPrefix(name, "tsp_") {
			categories["TSP Strategies"] = append(categories["TSP Strategies"], template)
		} else {
			categories["Combination Strategies"] = append(categories["Combination Strategies"], template)
		}
	}

	// Print each category
	for _, category := range []string{"Retirement Timing", "Social Security", "TSP Strategies", "Combination Strategies"} {
		templates := categories[category]
		if len(templates) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("%s:\n", category))
		for _, t := range templates {
			sb.WriteString(fmt.Sprintf("  %-30s %s\n", t.Name, t.Description))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("Usage:\n")
	sb.WriteString("  ./rpgo compare base.yaml --with postpone_1yr,delay_ss_70\n")
	sb.WriteString("  ./rpgo compare base.yaml --with conservative,aggressive\n")

	return sb.String()
}
