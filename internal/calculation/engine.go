package calculation

import (
	"context"
	"fmt"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

type NetIncomeCalculator struct {
	TaxCalc *ComprehensiveTaxCalculator
	Logger  Logger
}

func NewNetIncomeCalculator(taxCalc *ComprehensiveTaxCalculator, logger Logger) *NetIncomeCalculator {
	return &NetIncomeCalculator{
		TaxCalc: taxCalc,
		Logger:  logger,
	}
}

// CalculationEngine orchestrates all retirement calculations
type CalculationEngine struct {
	TaxCalc               *ComprehensiveTaxCalculator
	MedicareCalc          *MedicareCalculator
	LifecycleFundLoader   *LifecycleFundLoader
	NetIncomeCalc         *NetIncomeCalculator
	HistoricalData        *HistoricalDataManager
	MonteCarloFundReturns map[string]decimal.Decimal // Monte Carlo generated fund returns for TSP allocation calculations
	Debug                 bool                       // Enable debug output for detailed calculations
	Logger                Logger
}

// NewCalculationEngine creates a new calculation engine
func NewCalculationEngine() *CalculationEngine {
	taxCalc := NewComprehensiveTaxCalculator()
	logger := NopLogger{}
	return &CalculationEngine{
		TaxCalc:       taxCalc,
		MedicareCalc:  NewMedicareCalculator(),
		NetIncomeCalc: NewNetIncomeCalculator(taxCalc, logger),
		Logger:        logger,
	}
}

// NewCalculationEngineWithConfig creates a new calculation engine with configurable tax settings
func NewCalculationEngineWithConfig(federalRules domain.FederalRules) *CalculationEngine {
	taxCalc := NewComprehensiveTaxCalculatorWithConfig(federalRules)
	logger := NopLogger{}
	engine := &CalculationEngine{
		TaxCalc:             taxCalc,
		MedicareCalc:        NewMedicareCalculatorWithConfig(federalRules.MedicareConfig),
		LifecycleFundLoader: NewLifecycleFundLoader("data"),
		NetIncomeCalc:       NewNetIncomeCalculator(taxCalc, logger),
		Logger:              logger,
	}

	// Load lifecycle fund data
	if err := engine.LifecycleFundLoader.LoadAllLifecycleFunds(); err != nil {
		// Log error but don't fail - fall back to default allocations
		engine.Logger.Warnf("Failed to load lifecycle fund data: %v", err)
	}

	return engine
}

// SetLogger sets the logger for the calculation engine. If nil is provided, a no-op logger is used.
func (ce *CalculationEngine) SetLogger(l Logger) {
	if l == nil {
		ce.Logger = NopLogger{}
		return
	}
	ce.Logger = l
}

// RunScenarioAuto automatically detects the configuration format and runs the appropriate scenario
func (ce *CalculationEngine) RunScenarioAuto(ctx context.Context, config *domain.Configuration, scenarioIndex int) (*domain.ScenarioSummary, error) {
	if scenarioIndex >= len(config.Scenarios) {
		return nil, fmt.Errorf("scenario index %d out of range (have %d scenarios)", scenarioIndex, len(config.Scenarios))
	}
	return ce.RunGenericScenario(ctx, config, &config.Scenarios[scenarioIndex])
}

// RunGenericScenario calculates a complete retirement scenario using the generic household format
func (ce *CalculationEngine) RunGenericScenario(ctx context.Context, config *domain.Configuration, scenario *domain.GenericScenario) (*domain.ScenarioSummary, error) {
	// Use the new generic projection method directly
	projection := ce.GenerateAnnualProjectionGeneric(config.Household, scenario, &config.GlobalAssumptions, config.GlobalAssumptions.FederalRules)

	// Create scenario summary (guard Year5/Year10 for short projections)
	first := decimal.Zero
	if len(projection) > 0 {
		first = projection[0].NetIncome
	}
	year5 := decimal.Zero
	if len(projection) > 4 {
		year5 = projection[4].NetIncome
	}
	year10 := decimal.Zero
	if len(projection) > 9 {
		year10 = projection[9].NetIncome
	}

	// Calculate absolute calendar year comparisons for apples-to-apples analysis
	netIncome2030 := ce.getNetIncomeForYear(projection, 2030)
	netIncome2035 := ce.getNetIncomeForYear(projection, 2035)
	netIncome2040 := ce.getNetIncomeForYear(projection, 2040)

	// Calculate pre-retirement baseline projections with COLA growth
	currentNetIncome := ce.calculateCurrentNetIncomeGeneric(config.Household)
	preRetirement2030 := ce.projectPreRetirementNetIncome(currentNetIncome, 2030, config.GlobalAssumptions.COLAGeneralRate)
	preRetirement2035 := ce.projectPreRetirementNetIncome(currentNetIncome, 2035, config.GlobalAssumptions.COLAGeneralRate)
	preRetirement2040 := ce.projectPreRetirementNetIncome(currentNetIncome, 2040, config.GlobalAssumptions.COLAGeneralRate)

	summary := &domain.ScenarioSummary{
		Name:                 scenario.Name,
		FirstYearNetIncome:   first,
		Year5NetIncome:       year5,
		Year10NetIncome:      year10,
		Projection:           projection,
		NetIncome2030:        netIncome2030,
		NetIncome2035:        netIncome2035,
		NetIncome2040:        netIncome2040,
		PreRetirementNet2030: preRetirement2030,
		PreRetirementNet2035: preRetirement2035,
		PreRetirementNet2040: preRetirement2040,
	}

	// Calculate total lifetime income (present value)
	var totalPV decimal.Decimal
	discountRate := decimal.NewFromFloat(0.03) // 3% discount rate
	for i, year := range projection {
		discountFactor := decimal.NewFromFloat(1).Add(discountRate).Pow(decimal.NewFromInt(int64(i)))
		totalPV = totalPV.Add(year.NetIncome.Div(discountFactor))
	}
	summary.TotalLifetimeIncome = totalPV

	// Determine TSP longevity
	for i, year := range projection {
		if year.IsTSPDepleted() {
			summary.TSPLongevity = i + 1
			break
		}
	}
	if summary.TSPLongevity == 0 {
		summary.TSPLongevity = len(projection) // Lasted full projection
	}

	// Set initial and final TSP balances using the new generic methods
	if len(projection) > 0 {
		summary.InitialTSPBalance = projection[0].GetTotalTSPBalance()
		summary.FinalTSPBalance = projection[len(projection)-1].GetTotalTSPBalance()

		// Calculate success rate for deterministic scenarios based on TSP sustainability
		summary.SuccessRate = ce.calculateDeterministicSuccessRate(projection, summary.TSPLongevity)
	}

	// Perform IRMAA risk analysis
	isMarried := config.Household.FilingStatus == "married_filing_jointly"
	mc := NewMedicareCalculator()
	summary.IRMAAAnalysis = AnalyzeIRMAARisk(projection, isMarried, mc)

	return summary, nil
}

// RunScenario calculates a complete retirement scenario (legacy format)
// (Legacy RunScenario removed)

// getNetIncomeForYear finds the net income for a specific calendar year in the projection
func (ce *CalculationEngine) getNetIncomeForYear(projection []domain.AnnualCashFlow, targetYear int) decimal.Decimal {
	for _, year := range projection {
		if year.Date.Year() == targetYear {
			return year.NetIncome
		}
	}
	return decimal.Zero // Year not found in projection
}

// projectPreRetirementNetIncome projects current net income to future year with COLA growth
func (ce *CalculationEngine) projectPreRetirementNetIncome(currentNet decimal.Decimal, targetYear int, colaRate decimal.Decimal) decimal.Decimal {
	currentYear := 2025 // Base year
	yearsToProject := targetYear - currentYear

	if yearsToProject <= 0 {
		return currentNet
	}

	// Apply COLA growth for the number of years
	growthFactor := decimal.NewFromFloat(1).Add(colaRate).Pow(decimal.NewFromInt(int64(yearsToProject)))

	return currentNet.Mul(growthFactor)
}

// calculateDeterministicSuccessRate calculates success rate based on TSP sustainability and growth
func (ce *CalculationEngine) calculateDeterministicSuccessRate(projection []domain.AnnualCashFlow, tspLongevity int) decimal.Decimal {
	if len(projection) == 0 {
		return decimal.Zero
	}

	projectionLength := len(projection)

	// If TSP lasts the full projection period, success rate is 100%
	if tspLongevity >= projectionLength {
		// Additional check: TSP should be growing or stable, not just lasting
		firstTSP := projection[0].GetTotalTSPBalance()
		lastTSP := projection[projectionLength-1].GetTotalTSPBalance()

		if lastTSP.GreaterThanOrEqual(firstTSP) {
			return decimal.NewFromFloat(100.0) // 100% success - TSP lasted and grew
		} else {
			return decimal.NewFromFloat(95.0) // 95% success - TSP lasted but declined
		}
	}

	// If TSP depletes before end of projection, calculate percentage based on longevity
	successRate := decimal.NewFromInt(int64(tspLongevity)).Div(decimal.NewFromInt(int64(projectionLength))).Mul(decimal.NewFromFloat(100.0))

	// Minimum 10% success rate for any scenario that makes it past year 1
	if successRate.LessThan(decimal.NewFromFloat(10.0)) && tspLongevity > 1 {
		return decimal.NewFromFloat(10.0)
	}

	return successRate
}

// calculateCurrentNetIncomeGeneric calculates current net income for a household using generic participants
func (ce *CalculationEngine) calculateCurrentNetIncomeGeneric(household *domain.Household) decimal.Decimal {
	// Calculate gross income from all participants
	grossIncome := decimal.Zero
	for _, participant := range household.Participants {
		if participant.IsFederal && participant.CurrentSalary != nil {
			grossIncome = grossIncome.Add(*participant.CurrentSalary)
		}
	}

	// Calculate FEHB premiums (from primary holder)
	fehbPremium := decimal.Zero
	for _, participant := range household.Participants {
		if participant.IsPrimaryFEHBHolder && participant.FEHBPremiumPerPayPeriod != nil {
			fehbPremium = participant.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26)) // 26 pay periods per year
			break                                                                         // Only one primary holder
		}
	}

	// Calculate TSP contributions (pre-tax) from all federal participants
	tspContributions := decimal.Zero
	for _, participant := range household.Participants {
		if participant.IsFederal && participant.CurrentSalary != nil && participant.TSPContributionPercent != nil {
			contribution := participant.CurrentSalary.Mul(*participant.TSPContributionPercent)
			agencyMatch := participant.AgencyMatch()
			tspContributions = tspContributions.Add(contribution).Add(agencyMatch)
		}
	}

	// Calculate taxes - use projection start date for age calculation
	projectionStartYear := ProjectionBaseYear
	projectionStartDate := time.Date(projectionStartYear, 1, 1, 0, 0, 0, 0, time.UTC)

	ages := make([]int, 0)
	salaries := make([]decimal.Decimal, 0)
	for _, participant := range household.Participants {
		ages = append(ages, participant.Age(projectionStartDate))
		if participant.IsFederal && participant.CurrentSalary != nil {
			salaries = append(salaries, *participant.CurrentSalary)
		} else {
			salaries = append(salaries, decimal.Zero)
		}
	}

	// Calculate taxes (simplified - use first two participants' ages and salaries)
	// For households with more than 2 participants, only the first two are considered
	// for age-based deductions (consistent with married filing jointly tax treatment)
	ageParticipant1 := 0
	ageParticipant2 := 0
	salaryParticipant1 := decimal.Zero
	salaryParticipant2 := decimal.Zero

	if len(ages) > 0 {
		ageParticipant1 = ages[0]
		salaryParticipant1 = salaries[0]
	}
	if len(ages) > 1 {
		ageParticipant2 = ages[1]
		salaryParticipant2 = salaries[1]
	}

	// Calculate taxes (excluding FICA for now, will calculate separately)
	currentTaxableIncome := CalculateCurrentTaxableIncome(salaryParticipant1, salaryParticipant2)
	federalTax, stateTax, localTax, _ := ce.TaxCalc.CalculateTotalTaxes(currentTaxableIncome, false, ageParticipant1, ageParticipant2, grossIncome)

	// Calculate FICA taxes for each individual separately
	ficaTax := decimal.Zero
	for _, participant := range household.Participants {
		if participant.IsFederal && participant.CurrentSalary != nil {
			participantFICA := ce.TaxCalc.FICATaxCalc.CalculateFICA(*participant.CurrentSalary, *participant.CurrentSalary)
			ficaTax = ficaTax.Add(participantFICA)
		}
	}

	// Calculate net income: gross - taxes - FEHB - TSP contributions
	netIncome := grossIncome.Sub(federalTax).Sub(stateTax).Sub(localTax).Sub(ficaTax).Sub(fehbPremium).Sub(tspContributions)

	return netIncome
}

// Legacy two-person GenerateAnnualProjection removed; generic projection handled via GenerateAnnualProjectionGeneric

// RunScenarios runs all scenarios and returns a comparison
func (ce *CalculationEngine) RunScenarios(config *domain.Configuration) (*domain.ScenarioComparison, error) {
	ctx := context.Background()
	scenarios := make([]domain.ScenarioSummary, len(config.Scenarios))
	for i, scenario := range config.Scenarios {
		summary, err := ce.RunGenericScenario(ctx, config, &scenario)
		if err != nil {
			return nil, fmt.Errorf("RunScenario failed: %w", err)
		}
		scenarios[i] = *summary
	}

	// Calculate baseline (current net income) - use format-appropriate method
	baselineNetIncome := ce.calculateCurrentNetIncomeGeneric(config.Household)

	comparison := &domain.ScenarioComparison{
		BaselineNetIncome: baselineNetIncome,
		Scenarios:         scenarios,
		Assumptions:       config.GlobalAssumptions.GenerateAssumptions(),
	}

	// Generate impact analysis
	comparison.ImmediateImpact = ce.generateImpactAnalysis(baselineNetIncome, scenarios)
	comparison.LongTermProjection = ce.generateLongTermAnalysis(scenarios)

	return comparison, nil
}

// Calculate is a LEGACY function for backwards compatibility with old integration tests only.
// New code should use calculateCurrentNetIncomeGeneric() which works with any participant names.
// The parameter names 'robert' and 'dawn' are legacy names representing first and second participants.
func (nic *NetIncomeCalculator) Calculate(robert, dawn *domain.Employee, debug bool) decimal.Decimal {
	// Calculate gross income
	grossIncome := robert.CurrentSalary.Add(dawn.CurrentSalary)

	// Calculate FEHB premiums (only first participant pays FEHB in legacy format)
	fehbPremium := robert.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26)) // 26 pay periods per year

	// Calculate TSP contributions (pre-tax)
	tspContributions := robert.TotalAnnualTSPContribution().Add(dawn.TotalAnnualTSPContribution())

	// Calculate taxes - use projection start date for age calculation
	projectionStartYear := ProjectionBaseYear
	projectionStartDate := time.Date(projectionStartYear, 1, 1, 0, 0, 0, 0, time.UTC)
	ageParticipant1 := robert.Age(projectionStartDate)
	ageParticipant2 := dawn.Age(projectionStartDate)

	// Calculate taxes (excluding FICA for now, will calculate separately)
	currentTaxableIncome := CalculateCurrentTaxableIncome(robert.CurrentSalary, dawn.CurrentSalary)
	federalTax, stateTax, localTax, _ := nic.TaxCalc.CalculateTotalTaxes(currentTaxableIncome, false, ageParticipant1, ageParticipant2, grossIncome)

	// Calculate FICA taxes for each individual separately, as SS wage base applies per individual
	participant1FICA := nic.TaxCalc.FICATaxCalc.CalculateFICA(robert.CurrentSalary, robert.CurrentSalary)
	participant2FICA := nic.TaxCalc.FICATaxCalc.CalculateFICA(dawn.CurrentSalary, dawn.CurrentSalary)
	ficaTax := participant1FICA.Add(participant2FICA)

	// Calculate net income: gross - taxes - FEHB - TSP contributions
	netIncome := grossIncome.Sub(federalTax).Sub(stateTax).Sub(localTax).Sub(ficaTax).Sub(fehbPremium).Sub(tspContributions)

	// Debug output for verification (LEGACY output - use generic calculation for new code)
	if debug {
		nic.Logger.Debugf("CURRENT NET INCOME CALCULATION BREAKDOWN (LEGACY):")
		nic.Logger.Debugf("===================================================")
		nic.Logger.Debugf("Participant 1 Salary:   $%s", robert.CurrentSalary.StringFixed(2))
		nic.Logger.Debugf("Participant 2 Salary:   $%s", dawn.CurrentSalary.StringFixed(2))
		nic.Logger.Debugf("Combined Gross Income:  $%s", grossIncome.StringFixed(2))
		nic.Logger.Debugf("")
		nic.Logger.Debugf("DEDUCTIONS:")
		nic.Logger.Debugf("  Federal Tax:          $%s", federalTax.StringFixed(2))
		nic.Logger.Debugf("  State Tax:            $%s", stateTax.StringFixed(2))
		nic.Logger.Debugf("  Local Tax:            $%s", localTax.StringFixed(2))
		nic.Logger.Debugf("  FICA Tax:             $%s", ficaTax.StringFixed(2))
		nic.Logger.Debugf("  FEHB Premium:         $%s", fehbPremium.StringFixed(2))
		nic.Logger.Debugf("  TSP Contributions:    $%s", tspContributions.StringFixed(2))
		nic.Logger.Debugf("  Total Deductions:     $%s", federalTax.Add(stateTax).Add(localTax).Add(ficaTax).Add(fehbPremium).Add(tspContributions).StringFixed(2))
		nic.Logger.Debugf("")
		nic.Logger.Debugf("CURRENT NET TAKE-HOME:  $%s", netIncome.StringFixed(2))
		nic.Logger.Debugf("Monthly Take-Home:      $%s", netIncome.Div(decimal.NewFromInt(12)).StringFixed(2))
		nic.Logger.Debugf("")
	}

	return netIncome
}
