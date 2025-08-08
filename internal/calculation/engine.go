package calculation

import (
	"fmt"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// CalculationEngine orchestrates all retirement calculations
type CalculationEngine struct {
	TaxCalc             *ComprehensiveTaxCalculator
	MedicareCalc        *MedicareCalculator
	LifecycleFundLoader *LifecycleFundLoader
	Debug               bool // Enable debug output for detailed calculations
	Logger              Logger
}

// NewCalculationEngine creates a new calculation engine
func NewCalculationEngine() *CalculationEngine {
	return &CalculationEngine{
		TaxCalc:      NewComprehensiveTaxCalculator(),
		MedicareCalc: NewMedicareCalculator(),
		Logger:       NopLogger{},
	}
}

// NewCalculationEngineWithConfig creates a new calculation engine with configurable tax settings
func NewCalculationEngineWithConfig(federalRules domain.FederalRules) *CalculationEngine {
	engine := &CalculationEngine{
		TaxCalc:             NewComprehensiveTaxCalculatorWithConfig(federalRules),
		MedicareCalc:        NewMedicareCalculatorWithConfig(federalRules.MedicareConfig),
		LifecycleFundLoader: NewLifecycleFundLoader("data"),
		Logger:              NopLogger{},
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

// RunScenario calculates a complete retirement scenario
func (ce *CalculationEngine) RunScenario(config *domain.Configuration, scenario *domain.Scenario) (*domain.ScenarioSummary, error) {
	robert := config.PersonalDetails["robert"]
	dawn := config.PersonalDetails["dawn"]

	// Validate retirement dates are after hire dates
	if scenario.Robert.RetirementDate.Before(robert.HireDate) {
		return nil, fmt.Errorf("robert's retirement date (%s) cannot be before hire date (%s)",
			scenario.Robert.RetirementDate.Format("2006-01-02"), robert.HireDate.Format("2006-01-02"))
	}
	if scenario.Dawn.RetirementDate.Before(dawn.HireDate) {
		return nil, fmt.Errorf("dawn's retirement date (%s) cannot be before hire date (%s)",
			scenario.Dawn.RetirementDate.Format("2006-01-02"), dawn.HireDate.Format("2006-01-02"))
	}

	// Validate inflation and return rates are reasonable (allow deflation but cap extreme values)
	if config.GlobalAssumptions.InflationRate.LessThan(decimal.NewFromFloat(-0.10)) || config.GlobalAssumptions.InflationRate.GreaterThan(decimal.NewFromFloat(0.20)) {
		return nil, fmt.Errorf("inflation rate must be between -10%% and 20%%, got %s%%",
			config.GlobalAssumptions.InflationRate.Mul(decimal.NewFromInt(100)).StringFixed(2))
	}

	// Generate annual projections
	projection := ce.GenerateAnnualProjection(&robert, &dawn, scenario, &config.GlobalAssumptions, config.GlobalAssumptions.FederalRules)

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
	summary := &domain.ScenarioSummary{
		Name:               scenario.Name,
		FirstYearNetIncome: first,
		Year5NetIncome:     year5,
		Year10NetIncome:    year10,
		Projection:         projection,
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

	// Set initial and final TSP balances
	if len(projection) > 0 {
		summary.InitialTSPBalance = projection[0].TSPBalanceRobert.Add(projection[0].TSPBalanceDawn)
		summary.FinalTSPBalance = projection[len(projection)-1].TSPBalanceRobert.Add(projection[len(projection)-1].TSPBalanceDawn)
	}

	return summary, nil
}

// GenerateAnnualProjection generates annual cash flow projections for a scenario
// GenerateAnnualProjection is implemented in projection.go

// calculateMedicarePremium moved to medicare.go

// RunScenarios runs all scenarios and returns a comparison
func (ce *CalculationEngine) RunScenarios(config *domain.Configuration) (*domain.ScenarioComparison, error) {
	scenarios := make([]domain.ScenarioSummary, len(config.Scenarios))

	for i, scenario := range config.Scenarios {
		summary, err := ce.RunScenario(config, &scenario)
		if err != nil {
			return nil, err
		}
		scenarios[i] = *summary
	}

	// Calculate baseline (current net income)
	robert := config.PersonalDetails["robert"]
	dawn := config.PersonalDetails["dawn"]
	baselineNetIncome := ce.calculateCurrentNetIncome(&robert, &dawn, &config.GlobalAssumptions)

	comparison := &domain.ScenarioComparison{
		BaselineNetIncome: baselineNetIncome,
		Scenarios:         scenarios,
	}

	// Generate impact analysis
	comparison.ImmediateImpact = ce.generateImpactAnalysis(baselineNetIncome, scenarios)
	comparison.LongTermProjection = ce.generateLongTermAnalysis(scenarios)

	return comparison, nil
}

// calculateCurrentNetIncome calculates current net income
func (ce *CalculationEngine) calculateCurrentNetIncome(robert, dawn *domain.Employee, _ *domain.GlobalAssumptions) decimal.Decimal {
	// Calculate gross income
	grossIncome := robert.CurrentSalary.Add(dawn.CurrentSalary)

	// Calculate FEHB premiums (only Robert pays FEHB, Dawn has FSA-HC)
	fehbPremium := robert.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26)) // 26 pay periods per year

	// Calculate TSP contributions (pre-tax)
	tspContributions := robert.TotalAnnualTSPContribution().Add(dawn.TotalAnnualTSPContribution())

	// Calculate taxes - use projection start date for age calculation
	projectionStartYear := ProjectionBaseYear
	projectionStartDate := time.Date(projectionStartYear, 1, 1, 0, 0, 0, 0, time.UTC)
	ageRobert := robert.Age(projectionStartDate)
	ageDawn := dawn.Age(projectionStartDate)

	// Calculate taxes (excluding FICA for now, will calculate separately)
	currentTaxableIncome := CalculateCurrentTaxableIncome(robert.CurrentSalary, dawn.CurrentSalary)
	federalTax, stateTax, localTax, _ := ce.TaxCalc.CalculateTotalTaxes(currentTaxableIncome, false, ageRobert, ageDawn, grossIncome)

	// Calculate FICA taxes for each individual separately, as SS wage base applies per individual
	robertFICA := ce.TaxCalc.FICATaxCalc.CalculateFICA(robert.CurrentSalary, robert.CurrentSalary)
	dawnFICA := ce.TaxCalc.FICATaxCalc.CalculateFICA(dawn.CurrentSalary, dawn.CurrentSalary)
	ficaTax := robertFICA.Add(dawnFICA)

	// Calculate net income: gross - taxes - FEHB - TSP contributions
	netIncome := grossIncome.Sub(federalTax).Sub(stateTax).Sub(localTax).Sub(ficaTax).Sub(fehbPremium).Sub(tspContributions)

	// Debug output for verification
	if ce.Debug {
		ce.Logger.Debugf("CURRENT NET INCOME CALCULATION BREAKDOWN:")
		ce.Logger.Debugf("=========================================")
		ce.Logger.Debugf("Robert's Salary:        $%s", robert.CurrentSalary.StringFixed(2))
		ce.Logger.Debugf("Dawn's Salary:          $%s", dawn.CurrentSalary.StringFixed(2))
		ce.Logger.Debugf("Combined Gross Income:  $%s", grossIncome.StringFixed(2))
		ce.Logger.Debugf("")
		ce.Logger.Debugf("DEDUCTIONS:")
		ce.Logger.Debugf("  Federal Tax:          $%s", federalTax.StringFixed(2))
		ce.Logger.Debugf("  State Tax:            $%s", stateTax.StringFixed(2))
		ce.Logger.Debugf("  Local Tax:            $%s", localTax.StringFixed(2))
		ce.Logger.Debugf("  FICA Tax:             $%s", ficaTax.StringFixed(2))
		ce.Logger.Debugf("  FEHB Premium (Robert): $%s", fehbPremium.StringFixed(2))
		ce.Logger.Debugf("  TSP Contributions:    $%s", tspContributions.StringFixed(2))
		ce.Logger.Debugf("  Total Deductions:     $%s", federalTax.Add(stateTax).Add(localTax).Add(ficaTax).Add(fehbPremium).Add(tspContributions).StringFixed(2))
		ce.Logger.Debugf("")
		ce.Logger.Debugf("CURRENT NET TAKE-HOME:  $%s", netIncome.StringFixed(2))
		ce.Logger.Debugf("Monthly Take-Home:      $%s", netIncome.Div(decimal.NewFromInt(12)).StringFixed(2))
		ce.Logger.Debugf("")
	}

	return netIncome
}
