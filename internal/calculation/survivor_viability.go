package calculation

import (
	"context"
	"fmt"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// SurvivorViabilityAnalyzer provides comprehensive survivor viability analysis
type SurvivorViabilityAnalyzer struct {
	calcEngine *CalculationEngine
}

// NewSurvivorViabilityAnalyzer creates a new survivor viability analyzer
func NewSurvivorViabilityAnalyzer(calcEngine *CalculationEngine) *SurvivorViabilityAnalyzer {
	return &SurvivorViabilityAnalyzer{
		calcEngine: calcEngine,
	}
}

// AnalyzeSurvivorViability performs comprehensive survivor viability analysis
func (sva *SurvivorViabilityAnalyzer) AnalyzeSurvivorViability(
	ctx context.Context,
	config *domain.Configuration,
	scenario *domain.GenericScenario,
	survivorConfig domain.SurvivorScenarioConfig,
) (*domain.SurvivorViabilityAnalysis, error) {

	// Validate inputs
	if len(config.Household.Participants) < 2 {
		return nil, fmt.Errorf("survivor analysis requires at least 2 participants")
	}

	// Find deceased and survivor participants
	deceasedName := ""
	survivorName := ""
	for name, spec := range scenario.Mortality.Participants {
		if spec != nil && (spec.DeathDate != nil || spec.DeathAge != nil) {
			deceasedName = name
			break
		}
	}

	if deceasedName == "" {
		return nil, fmt.Errorf("no death specification found in scenario")
	}

	// Find survivor (first participant who is not deceased)
	for _, p := range config.Household.Participants {
		if p.Name != deceasedName {
			survivorName = p.Name
			break
		}
	}

	if survivorName == "" {
		return nil, fmt.Errorf("no survivor participant found")
	}

	// Calculate death year and ages
	deathYear, deathAge, survivorAge, err := sva.calculateDeathDetails(config.Household, deceasedName, survivorName, scenario.Mortality.Participants[deceasedName])
	if err != nil {
		return nil, fmt.Errorf("failed to calculate death details: %w", err)
	}

	// Run baseline scenario (no death)
	baselineScenario := scenario.DeepCopy()
	baselineScenario.Mortality = nil // Remove mortality for baseline
	baselineProjection, err := sva.calcEngine.RunGenericScenario(ctx, config, baselineScenario)
	if err != nil {
		return nil, fmt.Errorf("failed to run baseline scenario: %w", err)
	}

	// Run survivor scenario (with death)
	survivorProjection, err := sva.calcEngine.RunGenericScenario(ctx, config, scenario)
	if err != nil {
		return nil, fmt.Errorf("failed to run survivor scenario: %w", err)
	}

	// Analyze pre-death year (year before death)
	preDeathYear := deathYear - 1
	preDeathAnalysis, err := sva.analyzeYear(baselineProjection, preDeathYear, config.Household.FilingStatus, survivorName)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze pre-death year: %w", err)
	}

	// Analyze post-death year (year after death)
	postDeathYear := deathYear + 1
	postDeathAnalysis, err := sva.analyzeYear(survivorProjection, postDeathYear, "single", survivorName)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze post-death year: %w", err)
	}

	// Calculate viability assessment
	viabilityAssessment := sva.calculateViabilityAssessment(preDeathAnalysis, postDeathAnalysis, survivorConfig)

	// Calculate life insurance needs
	lifeInsuranceNeeds := sva.calculateLifeInsuranceNeeds(viabilityAssessment, survivorConfig)

	// Generate recommendations
	recommendations := domain.GenerateSurvivorRecommendations(viabilityAssessment)

	return &domain.SurvivorViabilityAnalysis{
		ScenarioName:        scenario.Name,
		DeceasedParticipant: deceasedName,
		SurvivorParticipant: survivorName,
		DeathYear:           deathYear,
		DeathAge:            deathAge,
		SurvivorAge:         survivorAge,
		PreDeathAnalysis:    preDeathAnalysis,
		PostDeathAnalysis:   postDeathAnalysis,
		ViabilityAssessment: viabilityAssessment,
		LifeInsuranceNeeds:  lifeInsuranceNeeds,
		Recommendations:     recommendations,
	}, nil
}

// calculateDeathDetails calculates death year and ages for deceased and survivor
func (sva *SurvivorViabilityAnalyzer) calculateDeathDetails(
	household *domain.Household,
	deceasedName, survivorName string,
	deathSpec *domain.MortalitySpec,
) (deathYear, deathAge, survivorAge int, err error) {

	// Find participants
	var deceased, survivor *domain.Participant
	for i := range household.Participants {
		if household.Participants[i].Name == deceasedName {
			deceased = &household.Participants[i]
		}
		if household.Participants[i].Name == survivorName {
			survivor = &household.Participants[i]
		}
	}

	if deceased == nil || survivor == nil {
		return 0, 0, 0, fmt.Errorf("participants not found")
	}

	// Calculate death year
	if deathSpec.DeathDate != nil {
		deathYear = deathSpec.DeathDate.Year()
		deathAge = deceased.Age(*deathSpec.DeathDate)
	} else if deathSpec.DeathAge != nil {
		deathAge = *deathSpec.DeathAge
		deathYear = deceased.BirthDate.Year() + deathAge
	} else {
		return 0, 0, 0, fmt.Errorf("no death specification provided")
	}

	// Calculate survivor age at death
	deathDate := time.Date(deathYear, 6, 30, 0, 0, 0, 0, time.UTC) // Assume mid-year death
	survivorAge = survivor.Age(deathDate)

	return deathYear, deathAge, survivorAge, nil
}

// analyzeYear analyzes a specific year in the projection
func (sva *SurvivorViabilityAnalyzer) analyzeYear(
	projection *domain.ScenarioSummary,
	year int,
	filingStatus string,
	survivorName string,
) (domain.SurvivorYearAnalysis, error) {

	// Find the year in projection
	var yearData *domain.AnnualCashFlow
	for i, y := range projection.Projection {
		if y.Date.Year() == year {
			yearData = &projection.Projection[i]
			break
		}
	}

	if yearData == nil {
		return domain.SurvivorYearAnalysis{}, fmt.Errorf("year %d not found in projection", year)
	}

	// Calculate income sources
	incomeSources := domain.SurvivorIncomeSources{
		SurvivorPension:    yearData.Pensions[survivorName],
		SurvivorSS:         yearData.SSBenefits[survivorName],
		DeceasedSurvivorSS: decimal.Zero, // Will be calculated separately
		TSPWithdrawals:     yearData.TSPWithdrawals[survivorName],
		OtherIncome:        decimal.Zero,
	}

	// Calculate total income
	incomeSources.TotalIncome = incomeSources.SurvivorPension.
		Add(incomeSources.SurvivorSS).
		Add(incomeSources.DeceasedSurvivorSS).
		Add(incomeSources.TSPWithdrawals).
		Add(incomeSources.OtherIncome)

	// Calculate TSP analysis
	tspAnalysis := domain.SurvivorTSPAnalysis{
		InitialBalance:   yearData.TSPBalances[survivorName],
		FinalBalance:     yearData.TSPBalances[survivorName], // Same as initial for single year
		AnnualWithdrawal: yearData.TSPWithdrawals[survivorName],
	}

	if tspAnalysis.InitialBalance.GreaterThan(decimal.Zero) {
		tspAnalysis.WithdrawalRate = tspAnalysis.AnnualWithdrawal.Div(tspAnalysis.InitialBalance)
	}

	// Calculate tax impact (simplified)
	taxImpact := yearData.FederalTax.Add(yearData.StateTax).Add(yearData.LocalTax)

	// Determine IRMAA risk
	irmaaRisk := "Low"
	if yearData.IRMAARiskStatus == "Breach" {
		irmaaRisk = "High"
	} else if yearData.IRMAARiskStatus == "Warning" {
		irmaaRisk = "Moderate"
	}

	return domain.SurvivorYearAnalysis{
		Year:            year,
		FilingStatus:    filingStatus,
		NetIncome:       yearData.NetIncome,
		MonthlyIncome:   yearData.NetIncome.Div(decimal.NewFromInt(12)),
		HealthcareCosts: yearData.HealthcareCosts.Total,
		TaxImpact:       taxImpact,
		IncomeSources:   incomeSources,
		TSPAnalysis:     tspAnalysis,
		IRMAARisk:       irmaaRisk,
		IRMAACost:       yearData.IRMAASurcharge.Mul(decimal.NewFromInt(12)),
	}, nil
}

// calculateViabilityAssessment calculates the viability assessment
func (sva *SurvivorViabilityAnalyzer) calculateViabilityAssessment(
	preDeath, postDeath domain.SurvivorYearAnalysis,
	config domain.SurvivorScenarioConfig,
) domain.SurvivorViabilityAssessment {

	// Calculate target income (percentage of pre-death income)
	targetIncome := preDeath.NetIncome.Mul(config.TargetIncomeFactor)

	// Calculate actual income and shortfall
	actualIncome := postDeath.NetIncome
	incomeShortfall := targetIncome.Sub(actualIncome)
	shortfallPercentage := decimal.Zero
	if targetIncome.GreaterThan(decimal.Zero) {
		shortfallPercentage = incomeShortfall.Div(targetIncome).Mul(decimal.NewFromInt(100))
	}

	// Calculate viability score
	viabilityScore, viabilityColor := domain.CalculateSurvivorViabilityScore(shortfallPercentage)

	// Calculate changes
	taxImpactChange := postDeath.TaxImpact.Sub(preDeath.TaxImpact)
	healthcareCostChange := postDeath.HealthcareCosts.Sub(preDeath.HealthcareCosts)

	// Determine IRMAA risk change
	irmaaRiskChange := "Same"
	if postDeath.IRMAARisk == "High" && preDeath.IRMAARisk != "High" {
		irmaaRiskChange = "Higher"
	} else if postDeath.IRMAARisk == "Low" && preDeath.IRMAARisk != "Low" {
		irmaaRiskChange = "Lower"
	}

	return domain.SurvivorViabilityAssessment{
		TargetIncome:         targetIncome,
		ActualIncome:         actualIncome,
		IncomeShortfall:      incomeShortfall,
		ShortfallPercentage:  shortfallPercentage,
		ViabilityScore:       viabilityScore,
		ViabilityColor:       viabilityColor,
		TSPLongevityChange:   0, // Will be calculated separately
		TaxImpactChange:      taxImpactChange,
		HealthcareCostChange: healthcareCostChange,
		IRMAARiskChange:      irmaaRiskChange,
	}
}

// calculateLifeInsuranceNeeds calculates life insurance needs
func (sva *SurvivorViabilityAnalyzer) calculateLifeInsuranceNeeds(
	assessment domain.SurvivorViabilityAssessment,
	config domain.SurvivorScenarioConfig,
) domain.LifeInsuranceAnalysis {

	annualShortfall := assessment.IncomeShortfall
	yearsToBridge := config.AnalysisYears
	discountRate := config.DiscountRate

	// Calculate present value of shortfall
	presentValue := decimal.Zero
	for i := 0; i < yearsToBridge; i++ {
		discountFactor := decimal.NewFromFloat(1).Add(discountRate).Pow(decimal.NewFromInt(int64(i)))
		presentValue = presentValue.Add(annualShortfall.Div(discountFactor))
	}

	// Add 20% buffer for recommended coverage
	recommendedCoverage := presentValue.Mul(decimal.NewFromFloat(1.2))

	// Generate alternative strategies
	var strategies []string
	if assessment.ShortfallPercentage.GreaterThan(decimal.NewFromFloat(0.20)) {
		strategies = append(strategies, "Consider increasing survivor benefit elections")
		strategies = append(strategies, "Build up Roth TSP for tax-free withdrawals")
		strategies = append(strategies, "Consider delaying Social Security for higher survivor benefits")
	}

	return domain.LifeInsuranceAnalysis{
		AnnualShortfall:       annualShortfall,
		YearsToBridge:         yearsToBridge,
		DiscountRate:          discountRate,
		PresentValue:          presentValue,
		RecommendedCoverage:   recommendedCoverage,
		AlternativeStrategies: strategies,
	}
}
