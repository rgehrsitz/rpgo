package calculation

import (
	"context"
	"fmt"
	"sort"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// RothConversionPlanner provides comprehensive Roth conversion planning and optimization
type RothConversionPlanner struct {
	calcEngine *CalculationEngine
}

// NewRothConversionPlanner creates a new Roth conversion planner
func NewRothConversionPlanner(calcEngine *CalculationEngine) *RothConversionPlanner {
	return &RothConversionPlanner{
		calcEngine: calcEngine,
	}
}

// PlanRothConversions performs comprehensive Roth conversion planning
func (rcp *RothConversionPlanner) PlanRothConversions(
	ctx context.Context,
	config *domain.Configuration,
	participant string,
	window domain.YearRange,
	targetBracket int,
	objective domain.OptimizationObjective,
	constraints domain.RothConversionConstraints,
) (*domain.RothConversionPlan, error) {

	// Validate inputs
	if err := constraints.Validate(); err != nil {
		return nil, fmt.Errorf("invalid constraints: %w", err)
	}

	if !window.IsValidYear(window.Start) || !window.IsValidYear(window.End) {
		return nil, fmt.Errorf("invalid year range: %s", window.String())
	}

	if targetBracket < 10 || targetBracket > 37 {
		return nil, fmt.Errorf("invalid target bracket: %d (must be 10-37)", targetBracket)
	}

	// Find the base scenario (first scenario in config)
	if len(config.Scenarios) == 0 {
		return nil, fmt.Errorf("no scenarios found in configuration")
	}
	baseScenario := config.Scenarios[0]

	// Validate participant exists
	if _, exists := baseScenario.ParticipantScenarios[participant]; !exists {
		return nil, fmt.Errorf("participant %s not found in scenario", participant)
	}

	// 1. Run baseline scenario (no conversions)
	baseline, err := rcp.calcEngine.RunGenericScenario(ctx, config, &baseScenario)
	if err != nil {
		return nil, fmt.Errorf("failed to run baseline scenario: %w", err)
	}

	// 2. Generate candidate conversion strategies
	candidates, err := rcp.generateConversionStrategies(baseline, window, targetBracket, constraints)
	if err != nil {
		return nil, fmt.Errorf("failed to generate conversion strategies: %w", err)
	}

	// Debug: Print candidate strategies
	fmt.Printf("DEBUG: Generated %d candidate strategies for window %s\n", len(candidates), window.String())
	fmt.Printf("DEBUG: Baseline projection starts at year %d, has %d years\n", baseline.Projection[0].Year, len(baseline.Projection))
	fmt.Printf("DEBUG: Available years in projection: ")
	for i, year := range baseline.Projection {
		if i < 10 { // Only show first 10 years
			fmt.Printf("%d ", year.Year)
		}
	}
	fmt.Printf("\n")
	for i, candidate := range candidates {
		fmt.Printf("DEBUG: Candidate %d: Year %d, Amount %s\n", i+1, candidate.Year, candidate.Amount.String())
	}

	// 3. Evaluate each candidate strategy
	results := make([]domain.ConversionOutcome, 0, len(candidates))
	for _, strategy := range candidates {
		outcome, err := rcp.evaluateConversionStrategy(ctx, config, &baseScenario, participant, strategy, baseline)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate strategy for year %d: %w", strategy.Year, err)
		}
		results = append(results, outcome)
	}

	// 4. Find optimal strategy based on objective
	optimal, err := rcp.selectOptimalStrategy(baseline, results, objective)
	if err != nil {
		return nil, fmt.Errorf("failed to select optimal strategy: %w", err)
	}

	// 5. Generate analysis
	analysis := rcp.generateConversionAnalysis(baseline, results, optimal, objective)

	// 6. Create the plan
	plan := &domain.RothConversionPlan{
		Participant:      participant,
		ConversionWindow: window,
		TargetBracket:    targetBracket,
		Objective:        objective,
		Baseline:         baseline,
		Recommended:      optimal,
		Alternatives:     results,
		Analysis:         analysis,
	}

	return plan, nil
}

// generateConversionStrategies creates candidate conversion strategies based on bracket room
func (rcp *RothConversionPlanner) generateConversionStrategies(
	baseline *domain.ScenarioSummary,
	window domain.YearRange,
	targetBracket int,
	constraints domain.RothConversionConstraints,
) ([]domain.ConversionStrategy, error) {

	var strategies []domain.ConversionStrategy

	for year := window.Start; year <= window.End; year++ {
		// Find the projection year by searching for the matching year in the Date field
		var yearData *domain.AnnualCashFlow
		for i, projectionYear := range baseline.Projection {
			if projectionYear.Date.Year() == year {
				yearData = &baseline.Projection[i]
				break
			}
		}

		if yearData == nil {
			fmt.Printf("DEBUG: Skipping year %d (not found in projection)\n", year)
			continue // Skip years outside projection
		}
		fmt.Printf("DEBUG: Year %d: FederalTaxableIncome = %s\n", year, yearData.FederalTaxableIncome.String())

		// Calculate bracket room
		bracketRoom, err := rcp.calculateBracketRoom(*yearData, targetBracket)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate bracket room for year %d: %w", year, err)
		}

		fmt.Printf("DEBUG: Year %d: Bracket room = %s (current income = %s, bracket edge = %s)\n",
			year, bracketRoom.RoomAmount.String(), bracketRoom.CurrentIncome.String(), bracketRoom.BracketEdge.String())

		if bracketRoom.RoomAmount.GreaterThan(constraints.MinConversionAmount) {
			// Cap the conversion amount by constraints
			conversionAmount := decimal.Min(bracketRoom.RoomAmount, constraints.MaxConversionAmount)

			strategies = append(strategies, domain.ConversionStrategy{
				Year:   year,
				Amount: conversionAmount,
			})
			fmt.Printf("DEBUG: Added strategy for year %d: %s\n", year, conversionAmount.String())
		} else {
			fmt.Printf("DEBUG: No room in year %d (room = %s, min = %s)\n", year, bracketRoom.RoomAmount.String(), constraints.MinConversionAmount.String())
		}
	}

	return strategies, nil
}

// calculateBracketRoom determines how much room is available in a target tax bracket
func (rcp *RothConversionPlanner) calculateBracketRoom(
	yearData domain.AnnualCashFlow,
	targetBracket int,
) (domain.BracketRoom, error) {

	// Get current taxable income
	currentIncome := yearData.FederalTaxableIncome

	// Define 2025 tax brackets (simplified)
	brackets := []domain.TaxBracketInfo{
		{BracketNumber: 10, MinIncome: decimal.Zero, MaxIncome: decimal.NewFromInt(23200), Rate: decimal.NewFromFloat(0.10)},
		{BracketNumber: 12, MinIncome: decimal.NewFromInt(23201), MaxIncome: decimal.NewFromInt(94300), Rate: decimal.NewFromFloat(0.12)},
		{BracketNumber: 22, MinIncome: decimal.NewFromInt(94301), MaxIncome: decimal.NewFromInt(201050), Rate: decimal.NewFromFloat(0.22)},
		{BracketNumber: 24, MinIncome: decimal.NewFromInt(201051), MaxIncome: decimal.NewFromInt(383900), Rate: decimal.NewFromFloat(0.24)},
		{BracketNumber: 32, MinIncome: decimal.NewFromInt(383901), MaxIncome: decimal.NewFromInt(487450), Rate: decimal.NewFromFloat(0.32)},
		{BracketNumber: 35, MinIncome: decimal.NewFromInt(487451), MaxIncome: decimal.NewFromInt(731200), Rate: decimal.NewFromFloat(0.35)},
		{BracketNumber: 37, MinIncome: decimal.NewFromInt(731201), MaxIncome: decimal.NewFromInt(999999999), Rate: decimal.NewFromFloat(0.37)},
	}

	// Find the target bracket
	var targetBracketInfo *domain.TaxBracketInfo
	for _, bracket := range brackets {
		if bracket.BracketNumber == targetBracket {
			targetBracketInfo = &bracket
			break
		}
	}

	if targetBracketInfo == nil {
		return domain.BracketRoom{}, fmt.Errorf("invalid target bracket: %d", targetBracket)
	}

	// Calculate room in the target bracket
	var roomAmount decimal.Decimal
	if currentIncome.LessThan(targetBracketInfo.MinIncome) {
		// Current income is below the bracket, room is the entire bracket
		roomAmount = targetBracketInfo.MaxIncome.Sub(targetBracketInfo.MinIncome)
	} else if currentIncome.LessThan(targetBracketInfo.MaxIncome) {
		// Current income is within the bracket, room is remaining space
		roomAmount = targetBracketInfo.MaxIncome.Sub(currentIncome)
	} else {
		// Current income is above the bracket, no room
		roomAmount = decimal.Zero
	}

	return domain.BracketRoom{
		BracketNumber: targetBracket,
		RoomAmount:    roomAmount,
		CurrentIncome: currentIncome,
		BracketEdge:   targetBracketInfo.MaxIncome,
	}, nil
}

// evaluateConversionStrategy runs a full projection with a specific conversion strategy
func (rcp *RothConversionPlanner) evaluateConversionStrategy(
	ctx context.Context,
	config *domain.Configuration,
	baseScenario *domain.GenericScenario,
	participant string,
	strategy domain.ConversionStrategy,
	baseline *domain.ScenarioSummary,
) (domain.ConversionOutcome, error) {

	// Create a modified scenario with the conversion
	modifiedScenario := baseScenario.DeepCopy()

	// Add the conversion to the participant's scenario
	ps := modifiedScenario.ParticipantScenarios[participant]
	if ps.RothConversions == nil {
		ps.RothConversions = &domain.RothConversionSchedule{
			Conversions: make([]domain.RothConversion, 0),
		}
	}

	ps.RothConversions.Conversions = append(ps.RothConversions.Conversions, domain.RothConversion{
		Year:   strategy.Year,
		Amount: strategy.Amount,
		Source: "traditional_tsp",
	})

	modifiedScenario.ParticipantScenarios[participant] = ps

	// Run the modified scenario
	projection, err := rcp.calcEngine.RunGenericScenario(ctx, config, modifiedScenario)
	if err != nil {
		return domain.ConversionOutcome{}, fmt.Errorf("failed to run modified scenario: %w", err)
	}

	// Calculate metrics
	lifetimeTax := rcp.calculateLifetimeTax(projection)
	lifetimeIRMAA := rcp.calculateLifetimeIRMAA(projection)
	finalBalances := rcp.calculateFinalBalances(projection, participant)

	// Calculate net benefit vs baseline
	baselineTax := rcp.calculateLifetimeTax(baseline)
	baselineIRMAA := rcp.calculateLifetimeIRMAA(baseline)

	taxDifference := lifetimeTax.Sub(baselineTax)
	irmaaDifference := lifetimeIRMAA.Sub(baselineIRMAA)
	netBenefit := taxDifference.Add(irmaaDifference).Neg() // Negative because we want to minimize costs

	// Calculate ROI (simplified)
	roi := decimal.Zero
	if !taxDifference.IsZero() {
		roi = netBenefit.Div(taxDifference.Abs()).Mul(decimal.NewFromInt(100))
	}

	return domain.ConversionOutcome{
		Strategy:      strategy,
		Projection:    projection,
		LifetimeTax:   lifetimeTax,
		LifetimeIRMAA: lifetimeIRMAA,
		FinalBalances: finalBalances,
		NetBenefit:    netBenefit,
		ROI:           roi,
	}, nil
}

// selectOptimalStrategy chooses the best strategy based on the optimization objective
func (rcp *RothConversionPlanner) selectOptimalStrategy(
	baseline *domain.ScenarioSummary,
	results []domain.ConversionOutcome,
	objective domain.OptimizationObjective,
) (*domain.ConversionOutcome, error) {

	if len(results) == 0 {
		return nil, fmt.Errorf("no conversion strategies to evaluate")
	}

	// Sort results based on objective
	sort.Slice(results, func(i, j int) bool {
		switch objective {
		case domain.MinimizeLifetimeTax:
			return results[i].LifetimeTax.LessThan(results[j].LifetimeTax)
		case domain.MinimizeLifetimeIRMAA:
			return results[i].LifetimeIRMAA.LessThan(results[j].LifetimeIRMAA)
		case domain.MinimizeCombined:
			combinedI := results[i].LifetimeTax.Add(results[i].LifetimeIRMAA)
			combinedJ := results[j].LifetimeTax.Add(results[j].LifetimeIRMAA)
			return combinedI.LessThan(combinedJ)
		case domain.MaximizeEstate:
			return results[i].FinalBalances.RothTSP.GreaterThan(results[j].FinalBalances.RothTSP)
		case domain.MaximizeNetIncome:
			return results[i].Projection.TotalLifetimeIncome.GreaterThan(results[j].Projection.TotalLifetimeIncome)
		default:
			return results[i].NetBenefit.GreaterThan(results[j].NetBenefit)
		}
	})

	return &results[0], nil
}

// generateConversionAnalysis creates detailed analysis of conversion strategies
func (rcp *RothConversionPlanner) generateConversionAnalysis(
	baseline *domain.ScenarioSummary,
	results []domain.ConversionOutcome,
	optimal *domain.ConversionOutcome,
	objective domain.OptimizationObjective,
) *domain.ConversionAnalysis {

	if optimal == nil || len(results) == 0 {
		return &domain.ConversionAnalysis{
			Recommendation: "No optimal strategy found",
		}
	}

	// Calculate totals
	totalConversions := decimal.Zero
	for _, result := range results {
		totalConversions = totalConversions.Add(result.Strategy.Amount)
	}

	totalTaxPaid := optimal.LifetimeTax.Sub(rcp.calculateLifetimeTax(baseline))
	irmaaSavings := rcp.calculateLifetimeIRMAA(baseline).Sub(optimal.LifetimeIRMAA)

	// Calculate RMD tax reduction (simplified)
	rmdTaxReduction := decimal.Zero // TODO: Implement RMD tax reduction calculation

	netBenefit := irmaaSavings.Add(rmdTaxReduction).Sub(totalTaxPaid)

	// Calculate ROI
	roi := decimal.Zero
	if !totalTaxPaid.IsZero() {
		roi = netBenefit.Div(totalTaxPaid.Abs()).Mul(decimal.NewFromInt(100))
	}

	// Generate recommendation
	recommendation := rcp.generateRecommendation(optimal, netBenefit, roi, objective)

	// Sensitivity analysis (simplified)
	sensitivity := domain.SensitivityAnalysis{
		Plus20Percent:  netBenefit.Mul(decimal.NewFromFloat(1.2)),
		Minus20Percent: netBenefit.Mul(decimal.NewFromFloat(0.8)),
		OptimalRange: fmt.Sprintf("$%s - $%s",
			optimal.Strategy.Amount.Mul(decimal.NewFromFloat(0.8)).StringFixed(0),
			optimal.Strategy.Amount.Mul(decimal.NewFromFloat(1.2)).StringFixed(0)),
	}

	return &domain.ConversionAnalysis{
		TotalConversions:    totalConversions,
		TotalTaxPaid:        totalTaxPaid,
		IRMAASavings:        irmaaSavings,
		RMDTaxReduction:     rmdTaxReduction,
		NetBenefit:          netBenefit,
		ROI:                 roi,
		Recommendation:      recommendation,
		SensitivityAnalysis: sensitivity,
	}
}

// generateRecommendation creates a human-readable recommendation
func (rcp *RothConversionPlanner) generateRecommendation(
	optimal *domain.ConversionOutcome,
	netBenefit decimal.Decimal,
	roi decimal.Decimal,
	objective domain.OptimizationObjective,
) string {

	if netBenefit.GreaterThan(decimal.Zero) {
		return fmt.Sprintf("✓ Execute Roth conversion strategy - Net benefit: $%s (%.1f%% ROI)",
			netBenefit.StringFixed(0), roi.InexactFloat64())
	} else if netBenefit.LessThan(decimal.Zero) {
		return fmt.Sprintf("⚠ Consider alternative strategies - Net cost: $%s",
			netBenefit.Abs().StringFixed(0))
	} else {
		return "→ Roth conversion provides neutral benefit - consider other factors"
	}
}

// Helper functions for calculating lifetime metrics

func (rcp *RothConversionPlanner) calculateLifetimeTax(projection *domain.ScenarioSummary) decimal.Decimal {
	total := decimal.Zero
	for _, year := range projection.Projection {
		total = total.Add(year.FederalTax)
	}
	return total
}

func (rcp *RothConversionPlanner) calculateLifetimeIRMAA(projection *domain.ScenarioSummary) decimal.Decimal {
	total := decimal.Zero
	for _, year := range projection.Projection {
		// IRMAA surcharge is monthly, convert to annual
		total = total.Add(year.IRMAASurcharge.Mul(decimal.NewFromInt(12)))
	}
	return total
}

func (rcp *RothConversionPlanner) calculateFinalBalances(projection *domain.ScenarioSummary, participant string) domain.FinalBalances {
	if len(projection.Projection) == 0 {
		return domain.FinalBalances{}
	}

	lastYear := projection.Projection[len(projection.Projection)-1]

	// Extract participant-specific balances (simplified - assumes single participant)
	totalTSP := lastYear.TSPBalances[participant]

	// For now, assume 50/50 split between Traditional and Roth
	// TODO: Track Traditional vs Roth balances separately
	traditionalTSP := totalTSP.Mul(decimal.NewFromFloat(0.5))
	rothTSP := totalTSP.Mul(decimal.NewFromFloat(0.5))

	return domain.FinalBalances{
		TraditionalTSP: traditionalTSP,
		RothTSP:        rothTSP,
		TaxableAccount: decimal.Zero, // TODO: Track taxable account balances
	}
}
