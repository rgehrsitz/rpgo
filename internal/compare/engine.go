package compare

import (
	"context"
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/transform"
)

// CompareEngine orchestrates scenario comparison
type CompareEngine struct {
	CalcEngine        *calculation.CalculationEngine
	MetricsCalculator *MetricsCalculator
	TemplateRegistry  *transform.TemplateRegistry
}

// NewCompareEngine creates a new comparison engine
func NewCompareEngine(calcEngine *calculation.CalculationEngine) *CompareEngine {
	return &CompareEngine{
		CalcEngine:        calcEngine,
		MetricsCalculator: NewMetricsCalculator(),
	}
}

// CompareOptions configures comparison behavior
type CompareOptions struct {
	BaseScenarioName string   // Name of the base scenario to compare against
	Templates        []string // List of template names to apply
	ParticipantName  string   // Primary participant for templates
}

// Compare runs multiple scenario comparisons
func (ce *CompareEngine) Compare(
	ctx context.Context,
	config *domain.Configuration,
	options CompareOptions,
) (*ComparisonSet, error) {

	// Initialize template registry for the specified participant
	ce.TemplateRegistry = transform.CreateBuiltInTemplates(options.ParticipantName)

	// Find base scenario
	var baseScenario *domain.GenericScenario
	for i := range config.Scenarios {
		if config.Scenarios[i].Name == options.BaseScenarioName {
			baseScenario = &config.Scenarios[i]
			break
		}
	}

	if baseScenario == nil {
		return nil, fmt.Errorf("base scenario %s not found in configuration", options.BaseScenarioName)
	}

	// Calculate base scenario
	baseSummary, err := ce.CalcEngine.RunGenericScenario(ctx, config, baseScenario)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate base scenario: %w", err)
	}

	baseResult := ce.MetricsCalculator.CalculateMetrics(baseSummary)

	// Calculate alternative scenarios using templates
	alternatives := []ComparisonResult{}

	for _, templateName := range options.Templates {
		template, ok := ce.TemplateRegistry.Get(templateName)
		if !ok {
			return nil, fmt.Errorf("template %s not found", templateName)
		}

		// Apply template to create modified scenario
		modifiedScenario, err := transform.ApplyTemplate(baseScenario, template)
		if err != nil {
			return nil, fmt.Errorf("failed to apply template %s: %w", templateName, err)
		}

		// Update scenario name to reflect the template
		modifiedScenario.Name = baseScenario.Name + "_" + templateName

		// Calculate the modified scenario
		altSummary, err := ce.CalcEngine.RunGenericScenario(ctx, config, modifiedScenario)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate scenario %s: %w", templateName, err)
		}

		// Calculate metrics and comparison
		altResult := ce.MetricsCalculator.CalculateMetrics(altSummary)
		altResult.Description = template.Description
		altResult = ce.MetricsCalculator.CalculateComparison(altResult, baseResult)

		alternatives = append(alternatives, altResult)
	}

	// Create comparison set
	compSet := &ComparisonSet{
		BaseScenarioName:   options.BaseScenarioName,
		BaseResult:         &baseResult,
		AlternativeResults: alternatives,
	}

	// Generate recommendations
	compSet.Recommendations = GenerateRecommendations(compSet)

	return compSet, nil
}

// CompareScenarios compares explicit scenarios (not using templates)
func (ce *CompareEngine) CompareScenarios(
	ctx context.Context,
	config *domain.Configuration,
	baseScenarioName string,
	alternativeScenarioNames []string,
) (*ComparisonSet, error) {

	// Find and calculate base scenario
	var baseSummary *domain.ScenarioSummary
	for i := range config.Scenarios {
		if config.Scenarios[i].Name == baseScenarioName {
			summary, err := ce.CalcEngine.RunGenericScenario(ctx, config, &config.Scenarios[i])
			if err != nil {
				return nil, fmt.Errorf("failed to calculate base scenario: %w", err)
			}
			baseSummary = summary
			break
		}
	}

	if baseSummary == nil {
		return nil, fmt.Errorf("base scenario %s not found", baseScenarioName)
	}

	baseResult := ce.MetricsCalculator.CalculateMetrics(baseSummary)

	// Calculate alternative scenarios
	alternatives := []ComparisonResult{}

	for _, altName := range alternativeScenarioNames {
		var altSummary *domain.ScenarioSummary
		for i := range config.Scenarios {
			if config.Scenarios[i].Name == altName {
				summary, err := ce.CalcEngine.RunGenericScenario(ctx, config, &config.Scenarios[i])
				if err != nil {
					return nil, fmt.Errorf("failed to calculate scenario %s: %w", altName, err)
				}
				altSummary = summary
				break
			}
		}

		if altSummary == nil {
			return nil, fmt.Errorf("alternative scenario %s not found", altName)
		}

		altResult := ce.MetricsCalculator.CalculateMetrics(altSummary)
		altResult = ce.MetricsCalculator.CalculateComparison(altResult, baseResult)

		alternatives = append(alternatives, altResult)
	}

	// Create comparison set
	compSet := &ComparisonSet{
		BaseScenarioName:   baseScenarioName,
		BaseResult:         &baseResult,
		AlternativeResults: alternatives,
	}

	// Generate recommendations
	compSet.Recommendations = GenerateRecommendations(compSet)

	return compSet, nil
}
