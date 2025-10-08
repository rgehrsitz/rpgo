package calculation

import (
	"context"
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// SensitivityAnalyzer performs parameter sweep analysis
type SensitivityAnalyzer struct {
	calculationEngine *CalculationEngine
}

// NewSensitivityAnalyzer creates a new sensitivity analyzer
func NewSensitivityAnalyzer() *SensitivityAnalyzer {
	return &SensitivityAnalyzer{
		calculationEngine: NewCalculationEngine(),
	}
}

// AnalyzeSingleParameter performs a single parameter sensitivity analysis
func (sa *SensitivityAnalyzer) AnalyzeSingleParameter(
	config *domain.Configuration,
	parameter SensitivityParameter,
	baseScenarioName string,
) (*domain.ParameterSensitivityAnalysis, error) {

	// Get base scenario
	baseScenario, err := sa.getBaseScenario(config, baseScenarioName)
	if err != nil {
		return nil, fmt.Errorf("failed to get base scenario: %w", err)
	}

	// Generate parameter values
	parameterValues := sa.generateParameterValues(parameter)

	// Run analysis for each parameter value
	results := make([]domain.SensitivityResult, 0, len(parameterValues))

	for _, value := range parameterValues {
		// Create modified config with new parameter value
		modifiedConfig := sa.modifyConfigParameter(config, parameter.Name, value)

		// Run scenario using the full calculation engine
		summary, err := sa.calculationEngine.RunGenericScenario(context.Background(), modifiedConfig, baseScenario)
		if err != nil {
			return nil, fmt.Errorf("failed to run scenario for %s=%v: %w", parameter.Name, value, err)
		}

		// Calculate sensitivity metrics
		metrics := sa.calculateSensitivityMetrics(summary.Projection, &config.GlobalAssumptions)
		metrics.Year5NetIncome = summary.Year5NetIncome
		metrics.Year10NetIncome = summary.Year10NetIncome
		metrics.TSPLongevity = summary.TSPLongevity
		metrics.TotalLifetimeIncome = summary.TotalLifetimeIncome

		// Create result
		result := domain.SensitivityResult{
			ParameterValues: map[string]decimal.Decimal{parameter.Name: value},
			ScenarioName:    fmt.Sprintf("%s_%s_%.3f", baseScenarioName, parameter.Name, value.InexactFloat64()),
			Projection:      summary.Projection,
			Summary:         *summary,
			KeyMetrics:      metrics,
		}

		results = append(results, result)
	}

	// Calculate sensitivity summary
	sensitivitySummary := sa.calculateSensitivitySummary(results, parameter)

	analysis := &domain.ParameterSensitivityAnalysis{
		BaseScenarioName: baseScenarioName,
		Parameters:       []domain.SensitivityParameter{parameter},
		Results:          results,
		Summary:          sensitivitySummary,
		AnalysisType:     "single",
	}

	return analysis, nil
}

// AnalyzeMultipleParameters performs a multi-parameter sensitivity analysis
func (sa *SensitivityAnalyzer) AnalyzeMultipleParameters(
	config *domain.Configuration,
	parameters []domain.SensitivityParameter,
	baseScenarioName string,
) (*domain.ParameterSensitivityAnalysis, error) {

	// For now, implement as sequential single-parameter analysis
	// TODO: Implement true multi-parameter analysis with interaction effects

	allResults := make([]domain.SensitivityResult, 0)
	allParameters := make([]domain.SensitivityParameter, 0)

	for _, param := range parameters {
		analysis, err := sa.AnalyzeSingleParameter(config, param, baseScenarioName)
		if err != nil {
			return nil, fmt.Errorf("failed to analyze parameter %s: %w", param.Name, err)
		}

		allResults = append(allResults, analysis.Results...)
		allParameters = append(allParameters, param)
	}

	// Calculate combined sensitivity summary
	sensitivitySummary := sa.calculateMultiParameterSensitivitySummary(allResults, allParameters)

	analysis := &domain.ParameterSensitivityAnalysis{
		BaseScenarioName: baseScenarioName,
		Parameters:       allParameters,
		Results:          allResults,
		Summary:          sensitivitySummary,
		AnalysisType:     "multi",
	}

	return analysis, nil
}

// AnalyzeParameterMatrix performs a 2D parameter matrix analysis
func (sa *SensitivityAnalyzer) AnalyzeParameterMatrix(
	config *domain.Configuration,
	param1, param2 domain.SensitivityParameter,
	baseScenarioName string,
) (*domain.SensitivityMatrix, error) {

	// Generate parameter values for both parameters
	values1 := sa.generateParameterValues(param1)
	values2 := sa.generateParameterValues(param2)

	// Create matrix results
	matrixResults := make([][]domain.SensitivityResult, len(values1))

	for i, value1 := range values1 {
		matrixResults[i] = make([]domain.SensitivityResult, len(values2))

		for j, value2 := range values2 {
			// Create modified config with both parameter values
			modifiedConfig := sa.modifyConfigParameter(config, param1.Name, value1)
			modifiedConfig = sa.modifyConfigParameter(modifiedConfig, param2.Name, value2)

			// Get base scenario
			baseScenario, err := sa.getBaseScenario(config, baseScenarioName)
			if err != nil {
				return nil, fmt.Errorf("failed to get base scenario: %w", err)
			}

			// Run scenario using the full calculation engine
			summary, err := sa.calculationEngine.RunGenericScenario(context.Background(), modifiedConfig, baseScenario)
			if err != nil {
				return nil, fmt.Errorf("failed to run scenario for %s=%v, %s=%v: %w",
					param1.Name, value1, param2.Name, value2, err)
			}

			// Calculate sensitivity metrics
			metrics := sa.calculateSensitivityMetrics(summary.Projection, &config.GlobalAssumptions)
			metrics.Year5NetIncome = summary.Year5NetIncome
			metrics.Year10NetIncome = summary.Year10NetIncome
			metrics.TSPLongevity = summary.TSPLongevity
			metrics.TotalLifetimeIncome = summary.TotalLifetimeIncome

			// Create result
			result := domain.SensitivityResult{
				ParameterValues: map[string]decimal.Decimal{
					param1.Name: value1,
					param2.Name: value2,
				},
				ScenarioName: fmt.Sprintf("%s_%s_%.3f_%s_%.3f",
					baseScenarioName, param1.Name, value1.InexactFloat64(),
					param2.Name, value2.InexactFloat64()),
				Projection: summary.Projection,
				Summary:    *summary,
				KeyMetrics: metrics,
			}

			matrixResults[i][j] = result
		}
	}

	// Calculate matrix summary
	matrixSummary := sa.calculateMatrixSummary(matrixResults, param1, param2)

	matrix := &domain.SensitivityMatrix{
		Parameter1:    param1,
		Parameter2:    param2,
		MatrixResults: matrixResults,
		Summary:       matrixSummary,
	}

	return matrix, nil
}

// generateParameterValues generates values for a parameter sweep
func (sa *SensitivityAnalyzer) generateParameterValues(param domain.SensitivityParameter) []decimal.Decimal {
	values := make([]decimal.Decimal, 0, param.Steps)

	if param.Steps <= 1 {
		return []decimal.Decimal{param.BaseValue}
	}

	// Calculate step size
	stepSize := param.MaxValue.Sub(param.MinValue).Div(decimal.NewFromInt(int64(param.Steps - 1)))

	// Generate values
	for i := 0; i < param.Steps; i++ {
		value := param.MinValue.Add(stepSize.Mul(decimal.NewFromInt(int64(i))))
		values = append(values, value)
	}

	return values
}

// modifyConfigParameter modifies a configuration parameter
func (sa *SensitivityAnalyzer) modifyConfigParameter(config *domain.Configuration, paramName string, value decimal.Decimal) *domain.Configuration {
	// Create a deep copy of the config
	modifiedConfig := *config
	modifiedConfig.GlobalAssumptions = config.GlobalAssumptions

	// Modify the specific parameter
	switch paramName {
	case "inflation_rate":
		modifiedConfig.GlobalAssumptions.InflationRate = value
	case "tsp_return_pre_retirement":
		modifiedConfig.GlobalAssumptions.TSPReturnPreRetirement = value
	case "tsp_return_post_retirement":
		modifiedConfig.GlobalAssumptions.TSPReturnPostRetirement = value
	case "cola_rate":
		modifiedConfig.GlobalAssumptions.COLAGeneralRate = value
	case "fehb_inflation":
		modifiedConfig.GlobalAssumptions.FEHBPremiumInflation = value
	default:
		// Unknown parameter, return original config
		return config
	}

	return &modifiedConfig
}

// getBaseScenario gets the base scenario from config
func (sa *SensitivityAnalyzer) getBaseScenario(config *domain.Configuration, scenarioName string) (*domain.GenericScenario, error) {
	for _, scenario := range config.Scenarios {
		if scenario.Name == scenarioName {
			return &scenario, nil
		}
	}
	return nil, fmt.Errorf("scenario '%s' not found", scenarioName)
}

// calculateSensitivityMetrics calculates sensitivity-specific metrics
func (sa *SensitivityAnalyzer) calculateSensitivityMetrics(projection []domain.AnnualCashFlow, assumptions *domain.GlobalAssumptions) domain.SensitivityMetrics {
	// Calculate IRMAA total cost
	irmaaTotalCost := decimal.Zero
	for i := range projection {
		irmaaTotalCost = irmaaTotalCost.Add(projection[i].IRMAASurcharge.Mul(decimal.NewFromInt(12)))
	}

	return domain.SensitivityMetrics{
		Year5NetIncome:      decimal.Zero, // Will be set from summary
		Year10NetIncome:     decimal.Zero, // Will be set from summary
		TSPLongevity:        0,            // Will be set from summary
		TotalLifetimeIncome: decimal.Zero, // Will be set from summary
		IRMAATotalCost:      irmaaTotalCost,
		NetIncomeChange:     decimal.Zero,
		NetIncomeChangePct:  decimal.Zero,
	}
}

// calculateSensitivitySummary calculates overall sensitivity summary
func (sa *SensitivityAnalyzer) calculateSensitivitySummary(results []domain.SensitivityResult, parameter domain.SensitivityParameter) domain.SensitivitySummary {
	if len(results) == 0 {
		return domain.SensitivitySummary{}
	}

	// Find base case (closest to base value)
	baseResult := results[0]
	minDiff := results[0].ParameterValues[parameter.Name].Sub(parameter.BaseValue).Abs()

	for _, result := range results[1:] {
		diff := result.ParameterValues[parameter.Name].Sub(parameter.BaseValue).Abs()
		if diff.LessThan(minDiff) {
			minDiff = diff
			baseResult = result
		}
	}

	// Calculate sensitivity scores
	sensitivityScores := make(map[string]decimal.Decimal)
	maxScore := decimal.Zero
	mostSensitiveParam := parameter.Name

	for _, result := range results {
		if result.ParameterValues[parameter.Name].Equal(baseResult.ParameterValues[parameter.Name]) {
			continue // Skip base case
		}

		// Calculate parameter change percentage
		paramChange := result.ParameterValues[parameter.Name].Sub(parameter.BaseValue).Div(parameter.BaseValue).Mul(decimal.NewFromInt(100))

		// Calculate metric changes
		netIncomeChange := result.KeyMetrics.Year5NetIncome.Sub(baseResult.KeyMetrics.Year5NetIncome)
		netIncomeChangePct := netIncomeChange.Div(baseResult.KeyMetrics.Year5NetIncome).Mul(decimal.NewFromInt(100))

		// Calculate sensitivity score
		sensitivityScore := netIncomeChangePct.Abs().Div(paramChange.Abs())
		sensitivityScores[result.ScenarioName] = sensitivityScore

		if sensitivityScore.GreaterThan(maxScore) {
			maxScore = sensitivityScore
		}
	}

	// Generate recommendations
	recommendations := []string{}
	if maxScore.GreaterThan(decimal.NewFromFloat(10.0)) {
		recommendations = append(recommendations, fmt.Sprintf("⚠️ High sensitivity to %s changes", parameter.Name))
		recommendations = append(recommendations, "Consider conservative assumptions")
	} else if maxScore.GreaterThan(decimal.NewFromFloat(5.0)) {
		recommendations = append(recommendations, fmt.Sprintf("Moderate sensitivity to %s changes", parameter.Name))
		recommendations = append(recommendations, "Monitor parameter regularly")
	} else {
		recommendations = append(recommendations, fmt.Sprintf("Low sensitivity to %s changes", parameter.Name))
		recommendations = append(recommendations, "Plan appears robust")
	}

	summary := domain.SensitivitySummary{
		MostSensitiveParameter: mostSensitiveParam,
		SensitivityScores:      sensitivityScores,
		Recommendations:        recommendations,
		RiskLevel:              "MEDIUM", // Will be calculated by DetermineRiskLevel()
	}

	summary.RiskLevel = summary.DetermineRiskLevel()
	summary.Recommendations = summary.GenerateRecommendations()

	return summary
}

// calculateMultiParameterSensitivitySummary calculates summary for multi-parameter analysis
func (sa *SensitivityAnalyzer) calculateMultiParameterSensitivitySummary(results []domain.SensitivityResult, parameters []domain.SensitivityParameter) domain.SensitivitySummary {
	// Group results by parameter
	paramResults := make(map[string][]domain.SensitivityResult)
	for _, result := range results {
		for paramName := range result.ParameterValues {
			paramResults[paramName] = append(paramResults[paramName], result)
		}
	}

	// Calculate sensitivity scores for each parameter
	sensitivityScores := make(map[string]decimal.Decimal)
	maxScore := decimal.Zero
	mostSensitiveParam := ""

	for paramName, paramResults := range paramResults {
		// Find parameter definition
		var param domain.SensitivityParameter
		for _, p := range parameters {
			if p.Name == paramName {
				param = p
				break
			}
		}

		// Calculate sensitivity score for this parameter
		paramSummary := sa.calculateSensitivitySummary(paramResults, param)
		paramScore := decimal.Zero
		for _, score := range paramSummary.SensitivityScores {
			if score.GreaterThan(paramScore) {
				paramScore = score
			}
		}

		sensitivityScores[paramName] = paramScore
		if paramScore.GreaterThan(maxScore) {
			maxScore = paramScore
			mostSensitiveParam = paramName
		}
	}

	// Generate recommendations
	recommendations := []string{}
	if maxScore.GreaterThan(decimal.NewFromFloat(15.0)) {
		recommendations = append(recommendations, "⚠️ High overall sensitivity detected")
		recommendations = append(recommendations, "Consider stress testing with extreme scenarios")
	} else if maxScore.GreaterThan(decimal.NewFromFloat(8.0)) {
		recommendations = append(recommendations, "Moderate overall sensitivity")
		recommendations = append(recommendations, "Monitor key parameters regularly")
	} else {
		recommendations = append(recommendations, "Low overall sensitivity")
		recommendations = append(recommendations, "Plan appears robust to parameter changes")
	}

	summary := domain.SensitivitySummary{
		MostSensitiveParameter: mostSensitiveParam,
		SensitivityScores:      sensitivityScores,
		Recommendations:        recommendations,
		RiskLevel:              "MEDIUM",
	}

	summary.RiskLevel = summary.DetermineRiskLevel()
	summary.Recommendations = summary.GenerateRecommendations()

	return summary
}

// calculateMatrixSummary calculates summary for matrix analysis
func (sa *SensitivityAnalyzer) calculateMatrixSummary(matrixResults [][]domain.SensitivityResult, param1, param2 domain.SensitivityParameter) domain.SensitivityMatrixSummary {
	// Find base case (closest to base values)
	baseResult := matrixResults[0][0]
	minDiff1 := matrixResults[0][0].ParameterValues[param1.Name].Sub(param1.BaseValue).Abs()
	minDiff2 := matrixResults[0][0].ParameterValues[param2.Name].Sub(param2.BaseValue).Abs()
	minTotalDiff := minDiff1.Add(minDiff2)

	for i := range matrixResults {
		for j := range matrixResults[i] {
			diff1 := matrixResults[i][j].ParameterValues[param1.Name].Sub(param1.BaseValue).Abs()
			diff2 := matrixResults[i][j].ParameterValues[param2.Name].Sub(param2.BaseValue).Abs()
			totalDiff := diff1.Add(diff2)

			if totalDiff.LessThan(minTotalDiff) {
				minTotalDiff = totalDiff
				baseResult = matrixResults[i][j]
			}
		}
	}

	// Calculate interaction effects
	interactionEffect := decimal.Zero
	maxSensitivity := decimal.Zero
	mostSensitiveCombination := ""

	for i := range matrixResults {
		for j := range matrixResults[i] {
			result := matrixResults[i][j]
			if result.ParameterValues[param1.Name].Equal(baseResult.ParameterValues[param1.Name]) &&
				result.ParameterValues[param2.Name].Equal(baseResult.ParameterValues[param2.Name]) {
				continue // Skip base case
			}

			// Calculate sensitivity score
			param1Change := result.ParameterValues[param1.Name].Sub(param1.BaseValue).Div(param1.BaseValue)
			param2Change := result.ParameterValues[param2.Name].Sub(param2.BaseValue).Div(param2.BaseValue)

			netIncomeChange := result.KeyMetrics.Year5NetIncome.Sub(baseResult.KeyMetrics.Year5NetIncome)
			netIncomeChangePct := netIncomeChange.Div(baseResult.KeyMetrics.Year5NetIncome)

			// Calculate interaction effect
			expectedChange := param1Change.Add(param2Change)
			actualChange := netIncomeChangePct
			interaction := actualChange.Sub(expectedChange)

			if interaction.Abs().GreaterThan(interactionEffect.Abs()) {
				interactionEffect = interaction
			}

			// Calculate sensitivity score
			sensitivityScore := netIncomeChangePct.Abs().Div(param1Change.Abs().Add(param2Change.Abs()))
			if sensitivityScore.GreaterThan(maxSensitivity) {
				maxSensitivity = sensitivityScore
				mostSensitiveCombination = fmt.Sprintf("%s=%.3f, %s=%.3f",
					param1.Name, result.ParameterValues[param1.Name].InexactFloat64(),
					param2.Name, result.ParameterValues[param2.Name].InexactFloat64())
			}
		}
	}

	// Generate recommendations
	recommendations := []string{}
	if maxSensitivity.GreaterThan(decimal.NewFromFloat(10.0)) {
		recommendations = append(recommendations, "⚠️ High sensitivity to parameter combinations")
		recommendations = append(recommendations, "Consider conservative assumptions for both parameters")
	} else if maxSensitivity.GreaterThan(decimal.NewFromFloat(5.0)) {
		recommendations = append(recommendations, "Moderate sensitivity to parameter combinations")
		recommendations = append(recommendations, "Monitor both parameters regularly")
	} else {
		recommendations = append(recommendations, "Low sensitivity to parameter combinations")
		recommendations = append(recommendations, "Plan appears robust to parameter interactions")
	}

	if interactionEffect.Abs().GreaterThan(decimal.NewFromFloat(0.1)) {
		recommendations = append(recommendations, "⚠️ Significant interaction effects detected")
		recommendations = append(recommendations, "Parameters are not independent")
	}

	riskLevel := "MEDIUM"
	if maxSensitivity.GreaterThan(decimal.NewFromFloat(15.0)) {
		riskLevel = "HIGH"
	} else if maxSensitivity.LessThan(decimal.NewFromFloat(5.0)) {
		riskLevel = "LOW"
	}

	return domain.SensitivityMatrixSummary{
		MostSensitiveCombination: mostSensitiveCombination,
		InteractionEffect:        interactionEffect,
		Recommendations:          recommendations,
		RiskLevel:                riskLevel,
	}
}

// SensitivityParameter is a local type for the analyzer
type SensitivityParameter = domain.SensitivityParameter
