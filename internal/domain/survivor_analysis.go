package domain

import (
	"github.com/shopspring/decimal"
)

// SurvivorViabilityAnalysis represents comprehensive analysis of financial impact when a spouse dies
type SurvivorViabilityAnalysis struct {
	ScenarioName        string `json:"scenarioName"`
	DeceasedParticipant string `json:"deceasedParticipant"`
	SurvivorParticipant string `json:"survivorParticipant"`
	DeathYear           int    `json:"deathYear"`
	DeathAge            int    `json:"deathAge"`
	SurvivorAge         int    `json:"survivorAge"`

	// Pre-death analysis (year before death)
	PreDeathAnalysis SurvivorYearAnalysis `json:"preDeathAnalysis"`

	// Post-death analysis (year after death)
	PostDeathAnalysis SurvivorYearAnalysis `json:"postDeathAnalysis"`

	// Viability assessment
	ViabilityAssessment SurvivorViabilityAssessment `json:"viabilityAssessment"`

	// Life insurance recommendations
	LifeInsuranceNeeds LifeInsuranceAnalysis `json:"lifeInsuranceNeeds"`

	// Recommendations
	Recommendations []string `json:"recommendations"`
}

// SurvivorYearAnalysis represents financial analysis for a specific year
type SurvivorYearAnalysis struct {
	Year            int             `json:"year"`
	FilingStatus    string          `json:"filingStatus"`
	NetIncome       decimal.Decimal `json:"netIncome"`
	MonthlyIncome   decimal.Decimal `json:"monthlyIncome"`
	HealthcareCosts decimal.Decimal `json:"healthcareCosts"`
	TaxImpact       decimal.Decimal `json:"taxImpact"`

	// Income sources breakdown
	IncomeSources SurvivorIncomeSources `json:"incomeSources"`

	// TSP analysis
	TSPAnalysis SurvivorTSPAnalysis `json:"tspAnalysis"`

	// IRMAA risk
	IRMAARisk string          `json:"irmaaRisk"`
	IRMAACost decimal.Decimal `json:"irmaaCost"`
}

// SurvivorIncomeSources represents income sources for survivor analysis
type SurvivorIncomeSources struct {
	SurvivorPension    decimal.Decimal `json:"survivorPension"`
	SurvivorSS         decimal.Decimal `json:"survivorSS"`
	DeceasedSurvivorSS decimal.Decimal `json:"deceasedSurvivorSS"`
	TSPWithdrawals     decimal.Decimal `json:"tspWithdrawals"`
	OtherIncome        decimal.Decimal `json:"otherIncome"`
	TotalIncome        decimal.Decimal `json:"totalIncome"`
}

// SurvivorTSPAnalysis represents TSP analysis for survivor scenarios
type SurvivorTSPAnalysis struct {
	InitialBalance   decimal.Decimal `json:"initialBalance"`
	FinalBalance     decimal.Decimal `json:"finalBalance"`
	Longevity        int             `json:"longevity"`
	WithdrawalRate   decimal.Decimal `json:"withdrawalRate"`
	AnnualWithdrawal decimal.Decimal `json:"annualWithdrawal"`
}

// SurvivorViabilityAssessment represents the viability assessment of survivor scenario
type SurvivorViabilityAssessment struct {
	TargetIncome        decimal.Decimal `json:"targetIncome"`
	ActualIncome        decimal.Decimal `json:"actualIncome"`
	IncomeShortfall     decimal.Decimal `json:"incomeShortfall"`
	ShortfallPercentage decimal.Decimal `json:"shortfallPercentage"`

	ViabilityScore string `json:"viabilityScore"` // "EXCELLENT", "GOOD", "CAUTION", "RISK", "CRITICAL"
	ViabilityColor string `json:"viabilityColor"` // "🟢", "🟡", "🟠", "🔴"

	// Key metrics
	TSPLongevityChange   int             `json:"tspLongevityChange"`
	TaxImpactChange      decimal.Decimal `json:"taxImpactChange"`
	HealthcareCostChange decimal.Decimal `json:"healthcareCostChange"`
	IRMAARiskChange      string          `json:"irmaaRiskChange"`
}

// LifeInsuranceAnalysis represents life insurance needs analysis
type LifeInsuranceAnalysis struct {
	AnnualShortfall     decimal.Decimal `json:"annualShortfall"`
	YearsToBridge       int             `json:"yearsToBridge"`
	DiscountRate        decimal.Decimal `json:"discountRate"`
	PresentValue        decimal.Decimal `json:"presentValue"`
	RecommendedCoverage decimal.Decimal `json:"recommendedCoverage"`

	// Alternative strategies
	AlternativeStrategies []string `json:"alternativeStrategies"`
}

// SurvivorScenarioConfig represents configuration for survivor analysis scenarios
type SurvivorScenarioConfig struct {
	// Death specification
	DeathSpec MortalitySpec `yaml:"death_spec" json:"deathSpec"`

	// Survivor assumptions
	SurvivorSpendingFactor decimal.Decimal `yaml:"survivor_spending_factor" json:"survivorSpendingFactor"`
	TSPSpousalTransfer     string          `yaml:"tsp_spousal_transfer" json:"tspSpousalTransfer"`
	FilingStatusSwitch     string          `yaml:"filing_status_switch" json:"filingStatusSwitch"`

	// Analysis parameters
	AnalysisYears      int             `yaml:"analysis_years" json:"analysisYears"`
	DiscountRate       decimal.Decimal `yaml:"discount_rate" json:"discountRate"`
	TargetIncomeFactor decimal.Decimal `yaml:"target_income_factor" json:"targetIncomeFactor"`
}

// DefaultSurvivorScenarioConfig returns default configuration for survivor analysis
func DefaultSurvivorScenarioConfig() SurvivorScenarioConfig {
	return SurvivorScenarioConfig{
		SurvivorSpendingFactor: decimal.NewFromFloat(0.75), // 75% of couple's spending
		TSPSpousalTransfer:     "merge",                    // Merge TSP balances
		FilingStatusSwitch:     "next_year",                // Switch to single filing next year
		AnalysisYears:          20,                         // Analyze 20 years post-death
		DiscountRate:           decimal.NewFromFloat(0.04), // 4% discount rate
		TargetIncomeFactor:     decimal.NewFromFloat(0.75), // Target 75% of pre-death income
	}
}

// SurvivorViabilityScore calculates viability score based on income shortfall
func CalculateSurvivorViabilityScore(shortfallPercentage decimal.Decimal) (string, string) {
	if shortfallPercentage.LessThanOrEqual(decimal.NewFromFloat(0.05)) { // ≤5%
		return "EXCELLENT", "🟢"
	} else if shortfallPercentage.LessThanOrEqual(decimal.NewFromFloat(0.15)) { // ≤15%
		return "GOOD", "🟡"
	} else if shortfallPercentage.LessThanOrEqual(decimal.NewFromFloat(0.25)) { // ≤25%
		return "CAUTION", "🟠"
	} else if shortfallPercentage.LessThanOrEqual(decimal.NewFromFloat(0.40)) { // ≤40%
		return "RISK", "🔴"
	} else {
		return "CRITICAL", "🔴"
	}
}

// SurvivorRecommendation generates recommendations based on viability assessment
func GenerateSurvivorRecommendations(assessment SurvivorViabilityAssessment) []string {
	var recommendations []string

	if assessment.ShortfallPercentage.GreaterThan(decimal.NewFromFloat(0.10)) {
		recommendations = append(recommendations, "⚠️ Survivor income falls short of target by "+assessment.ShortfallPercentage.StringFixed(1)+"%")
	}

	if assessment.TSPLongevityChange < -5 {
		recommendations = append(recommendations, "💡 TSP longevity reduced by "+decimal.NewFromInt(int64(-assessment.TSPLongevityChange)).StringFixed(0)+" years - consider increasing life insurance")
	}

	if assessment.TaxImpactChange.GreaterThan(decimal.NewFromFloat(5000)) {
		recommendations = append(recommendations, "💡 Tax impact increases by $"+assessment.TaxImpactChange.StringFixed(0)+"/year - consider Roth conversions")
	}

	if assessment.IRMAARiskChange == "Higher" {
		recommendations = append(recommendations, "💡 IRMAA risk increases - consider Roth TSP withdrawals")
	}

	if assessment.ViabilityScore == "CRITICAL" || assessment.ViabilityScore == "RISK" {
		recommendations = append(recommendations, "🚨 Consider increasing life insurance coverage")
		recommendations = append(recommendations, "💡 Review survivor benefit elections on pensions")
		recommendations = append(recommendations, "💡 Build up Roth TSP (tax-free withdrawals help single filer)")
	}

	return recommendations
}
