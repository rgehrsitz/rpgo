package domain

import (
	"github.com/shopspring/decimal"
)

// SensitivityParameter represents a parameter to sweep in sensitivity analysis
type SensitivityParameter struct {
	Name        string          `yaml:"name" json:"name"`
	MinValue    decimal.Decimal `yaml:"min_value" json:"minValue"`
	MaxValue    decimal.Decimal `yaml:"max_value" json:"maxValue"`
	Steps       int             `yaml:"steps" json:"steps"`
	BaseValue   decimal.Decimal `yaml:"base_value" json:"baseValue"`
	Unit        string          `yaml:"unit" json:"unit"` // "percent", "dollars", "years", etc.
	Description string          `yaml:"description" json:"description"`
}

// ParameterSensitivityAnalysis represents a complete parameter sensitivity analysis
type ParameterSensitivityAnalysis struct {
	BaseScenarioName string                 `json:"baseScenarioName"`
	Parameters       []SensitivityParameter `json:"parameters"`
	Results          []SensitivityResult    `json:"results"`
	Summary          SensitivitySummary     `json:"summary"`
	AnalysisType     string                 `json:"analysisType"` // "single", "multi", "matrix"
}

// SensitivityResult represents the result of a single parameter sweep
type SensitivityResult struct {
	ParameterValues map[string]decimal.Decimal `json:"parameterValues"`
	ScenarioName    string                     `json:"scenarioName"`
	Projection      []AnnualCashFlow           `json:"projection"`
	Summary         ScenarioSummary            `json:"summary"`
	KeyMetrics      SensitivityMetrics         `json:"keyMetrics"`
}

// SensitivityMetrics represents key metrics for sensitivity analysis
type SensitivityMetrics struct {
	Year5NetIncome      decimal.Decimal `json:"year5NetIncome"`
	Year10NetIncome     decimal.Decimal `json:"year10NetIncome"`
	TSPLongevity        int             `json:"tspLongevity"`
	TotalLifetimeIncome decimal.Decimal `json:"totalLifetimeIncome"`
	IRMAATotalCost      decimal.Decimal `json:"irmaaTotalCost"`
	NetIncomeChange     decimal.Decimal `json:"netIncomeChange"`
	NetIncomeChangePct  decimal.Decimal `json:"netIncomeChangePct"`
}

// SensitivitySummary provides overall analysis summary
type SensitivitySummary struct {
	MostSensitiveParameter string                     `json:"mostSensitiveParameter"`
	SensitivityScores      map[string]decimal.Decimal `json:"sensitivityScores"`
	Recommendations        []string                   `json:"recommendations"`
	RiskLevel              string                     `json:"riskLevel"` // "LOW", "MEDIUM", "HIGH", "CRITICAL"
}

// SensitivityMatrix represents a 2D parameter sweep
type SensitivityMatrix struct {
	Parameter1    SensitivityParameter     `json:"parameter1"`
	Parameter2    SensitivityParameter     `json:"parameter2"`
	MatrixResults [][]SensitivityResult    `json:"matrixResults"`
	Summary       SensitivityMatrixSummary `json:"summary"`
}

// SensitivityMatrixSummary provides matrix analysis summary
type SensitivityMatrixSummary struct {
	MostSensitiveCombination string          `json:"mostSensitiveCombination"`
	InteractionEffect        decimal.Decimal `json:"interactionEffect"`
	Recommendations          []string        `json:"recommendations"`
	RiskLevel                string          `json:"riskLevel"`
}

// SensitivityConfig represents configuration for sensitivity analysis
type SensitivityConfig struct {
	BaseScenarioName string                 `yaml:"base_scenario" json:"baseScenario"`
	Parameters       []SensitivityParameter `yaml:"parameters" json:"parameters"`
	OutputFormat     string                 `yaml:"output_format" json:"outputFormat"`
	AnalysisType     string                 `yaml:"analysis_type" json:"analysisType"`
}

// Common sensitivity parameters
var (
	InflationRateParam = SensitivityParameter{
		Name:        "inflation_rate",
		MinValue:    decimal.NewFromFloat(0.015),
		MaxValue:    decimal.NewFromFloat(0.040),
		Steps:       6,
		BaseValue:   decimal.NewFromFloat(0.025),
		Unit:        "percent",
		Description: "General inflation rate affecting COLA and expenses",
	}

	TSPReturnPreRetirementParam = SensitivityParameter{
		Name:        "tsp_return_pre_retirement",
		MinValue:    decimal.NewFromFloat(0.05),
		MaxValue:    decimal.NewFromFloat(0.10),
		Steps:       6,
		BaseValue:   decimal.NewFromFloat(0.08),
		Unit:        "percent",
		Description: "TSP return rate before retirement",
	}

	TSPReturnPostRetirementParam = SensitivityParameter{
		Name:        "tsp_return_post_retirement",
		MinValue:    decimal.NewFromFloat(0.03),
		MaxValue:    decimal.NewFromFloat(0.07),
		Steps:       5,
		BaseValue:   decimal.NewFromFloat(0.065),
		Unit:        "percent",
		Description: "TSP return rate after retirement",
	}

	COLABateParam = SensitivityParameter{
		Name:        "cola_rate",
		MinValue:    decimal.NewFromFloat(0.015),
		MaxValue:    decimal.NewFromFloat(0.035),
		Steps:       5,
		BaseValue:   decimal.NewFromFloat(0.025),
		Unit:        "percent",
		Description: "COLA rate for FERS pension and Social Security",
	}

	FEHBInflationParam = SensitivityParameter{
		Name:        "fehb_inflation",
		MinValue:    decimal.NewFromFloat(0.04),
		MaxValue:    decimal.NewFromFloat(0.08),
		Steps:       5,
		BaseValue:   decimal.NewFromFloat(0.065),
		Unit:        "percent",
		Description: "FEHB premium inflation rate",
	}
)

// GetCommonParameters returns a list of common sensitivity parameters
func GetCommonParameters() []SensitivityParameter {
	return []SensitivityParameter{
		InflationRateParam,
		TSPReturnPreRetirementParam,
		TSPReturnPostRetirementParam,
		COLABateParam,
		FEHBInflationParam,
	}
}

// CalculateSensitivityScore calculates how sensitive a metric is to parameter changes
func (sm *SensitivityMetrics) CalculateSensitivityScore(baseMetrics SensitivityMetrics, parameterChange decimal.Decimal) decimal.Decimal {
	if parameterChange.IsZero() {
		return decimal.Zero
	}

	// Calculate percentage change in key metrics
	netIncomeChange := sm.NetIncomeChangePct
	longevityChange := decimal.NewFromInt(int64(sm.TSPLongevity - baseMetrics.TSPLongevity))

	// Weighted sensitivity score (net income change is most important)
	sensitivityScore := netIncomeChange.Abs().Mul(decimal.NewFromFloat(0.7)).
		Add(longevityChange.Abs().Mul(decimal.NewFromFloat(0.3)))

	return sensitivityScore
}

// DetermineRiskLevel determines the risk level based on sensitivity scores
func (ss *SensitivitySummary) DetermineRiskLevel() string {
	maxScore := decimal.Zero
	for _, score := range ss.SensitivityScores {
		if score.GreaterThan(maxScore) {
			maxScore = score
		}
	}

	if maxScore.LessThan(decimal.NewFromFloat(5.0)) {
		return "LOW"
	} else if maxScore.LessThan(decimal.NewFromFloat(15.0)) {
		return "MEDIUM"
	} else if maxScore.LessThan(decimal.NewFromFloat(30.0)) {
		return "HIGH"
	} else {
		return "CRITICAL"
	}
}

// GenerateRecommendations generates recommendations based on sensitivity analysis
func (ss *SensitivitySummary) GenerateRecommendations() []string {
	recommendations := []string{}

	riskLevel := ss.DetermineRiskLevel()

	switch riskLevel {
	case "LOW":
		recommendations = append(recommendations, "Plan is robust to parameter changes")
		recommendations = append(recommendations, "Current assumptions appear reasonable")
	case "MEDIUM":
		recommendations = append(recommendations, "Monitor key parameters regularly")
		recommendations = append(recommendations, "Consider conservative assumptions for critical parameters")
	case "HIGH":
		recommendations = append(recommendations, "Plan is sensitive to parameter changes")
		recommendations = append(recommendations, "Consider stress testing with extreme scenarios")
		recommendations = append(recommendations, "Review assumptions annually")
	case "CRITICAL":
		recommendations = append(recommendations, "⚠️ Plan is highly sensitive to parameter changes")
		recommendations = append(recommendations, "Consider more conservative assumptions")
		recommendations = append(recommendations, "Implement risk mitigation strategies")
		recommendations = append(recommendations, "Review plan quarterly")
	}

	// Add specific recommendations based on most sensitive parameter
	if ss.MostSensitiveParameter != "" {
		switch ss.MostSensitiveParameter {
		case "inflation_rate":
			recommendations = append(recommendations, "Consider inflation-protected investments")
		case "tsp_return_pre_retirement", "tsp_return_post_retirement":
			recommendations = append(recommendations, "Consider diversifying TSP allocation")
		case "cola_rate":
			recommendations = append(recommendations, "Monitor COLA adjustments closely")
		case "fehb_inflation":
			recommendations = append(recommendations, "Consider healthcare cost mitigation strategies")
		}
	}

	return recommendations
}
