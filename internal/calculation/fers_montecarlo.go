package calculation

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// FERSMonteCarloEngine provides full FERS retirement planning with Monte Carlo market variability
type FERSMonteCarloEngine struct {
	baseConfig        *domain.Configuration
	historicalData    *HistoricalDataManager
	calculationEngine *CalculationEngine
	config            FERSMonteCarloConfig
}

// FERSMonteCarloConfig holds configuration for FERS Monte Carlo simulations
type FERSMonteCarloConfig struct {
	NumSimulations  int
	ProjectionYears int
	Seed            int64
	UseHistorical   bool

	// Market variability settings
	TSPReturnVariability decimal.Decimal // Standard deviation for TSP returns
	InflationVariability decimal.Decimal // Standard deviation for inflation
	COLAVariability      decimal.Decimal // Standard deviation for COLA
	FEHBVariability      decimal.Decimal // Standard deviation for FEHB premiums

	// Monte Carlo specific settings
	MaxReasonableIncome  decimal.Decimal // Cap for unrealistic income scenarios
	DefaultTSPAllocation domain.TSPAllocation
}

// FERSMonteCarloResult represents comprehensive results from FERS Monte Carlo simulation
type FERSMonteCarloResult struct {
	BaseScenarioName     string                     `json:"baseScenarioName"`
	NumSimulations       int                        `json:"numSimulations"`
	ProjectionYears      int                        `json:"projectionYears"`
	SuccessRate          decimal.Decimal            `json:"successRate"`
	MedianLifetimeIncome decimal.Decimal            `json:"medianLifetimeIncome"`
	MedianTSPLongevity   int                        `json:"medianTSPLongevity"`
	PercentileRanges     FERSPercentileRanges       `json:"percentileRanges"`
	Simulations          []FERSMonteCarloSimulation `json:"simulations"`
	MarketConditions     []MarketCondition          `json:"marketConditions"`
}

// FERSMonteCarloSimulation represents a single FERS Monte Carlo simulation outcome
type FERSMonteCarloSimulation struct {
	SimulationID    int                    `json:"simulationId"`
	ScenarioSummary domain.ScenarioSummary `json:"scenarioSummary"`
	MarketCondition MarketCondition        `json:"marketCondition"`
	Success         bool                   `json:"success"`
	FailureYear     int                    `json:"failureYear,omitempty"`
	FailureReason   string                 `json:"failureReason,omitempty"`
}

// MarketCondition represents the market conditions for a single simulation
type MarketCondition struct {
	TSPReturns    map[string]decimal.Decimal `json:"tspReturns"` // Annual returns by fund
	InflationRate decimal.Decimal            `json:"inflationRate"`
	COLARate      decimal.Decimal            `json:"colaRate"`
	FEHBInflation decimal.Decimal            `json:"fehbInflation"`
}

// FERSPercentileRanges represents percentile ranges for FERS Monte Carlo results
type FERSPercentileRanges struct {
	LifetimeIncome map[string]decimal.Decimal `json:"lifetimeIncome"` // 10th, 25th, 50th, 75th, 90th percentiles
	TSPLongevity   map[string]int             `json:"tspLongevity"`   // 10th, 25th, 50th, 75th, 90th percentiles
	Year5Income    map[string]decimal.Decimal `json:"year5Income"`    // 10th, 25th, 50th, 75th, 90th percentiles
	Year10Income   map[string]decimal.Decimal `json:"year10Income"`   // 10th, 25th, 50th, 75th, 90th percentiles
}

// NewFERSMonteCarloEngine creates a new FERS Monte Carlo engine
func NewFERSMonteCarloEngine(baseConfig *domain.Configuration, historicalData *HistoricalDataManager) *FERSMonteCarloEngine {
	return &FERSMonteCarloEngine{
		baseConfig:        baseConfig,
		historicalData:    historicalData,
		calculationEngine: NewCalculationEngineWithConfig(baseConfig.GlobalAssumptions.FederalRules),
		config: FERSMonteCarloConfig{
			NumSimulations:       1000,
			ProjectionYears:      30,
			Seed:                 time.Now().UnixNano(),
			UseHistorical:        true,
			TSPReturnVariability: decimal.NewFromFloat(0.05),     // 5% standard deviation (reduced from 15%)
			InflationVariability: decimal.NewFromFloat(0.01),     // 1% standard deviation (reduced from 2%)
			COLAVariability:      decimal.NewFromFloat(0.005),    // 0.5% standard deviation (reduced from 1%)
			FEHBVariability:      decimal.NewFromFloat(0.02),     // 2% standard deviation (reduced from 5%)
			MaxReasonableIncome:  decimal.NewFromFloat(10000000), // $10M cap (increased from $500K)
			DefaultTSPAllocation: domain.TSPAllocation{
				CFund: decimal.NewFromFloat(0.6),
				SFund: decimal.NewFromFloat(0.2),
				IFund: decimal.NewFromFloat(0.1),
				FFund: decimal.NewFromFloat(0.1),
				GFund: decimal.Zero,
			},
		},
	}
}

// RunFERSMonteCarlo runs a comprehensive FERS Monte Carlo simulation
func (fmce *FERSMonteCarloEngine) RunFERSMonteCarlo(ctx context.Context, baseScenarioName string) (*FERSMonteCarloResult, error) {
	// Find base scenario
	var baseScenario *domain.GenericScenario
	for i := range fmce.baseConfig.Scenarios {
		if fmce.baseConfig.Scenarios[i].Name == baseScenarioName {
			baseScenario = &fmce.baseConfig.Scenarios[i]
			break
		}
	}
	if baseScenario == nil {
		return nil, fmt.Errorf("base scenario '%s' not found", baseScenarioName)
	}

	// Set random seed
	rand.Seed(fmce.config.Seed)

	// Run simulations in parallel
	simulations := make([]FERSMonteCarloSimulation, fmce.config.NumSimulations)
	marketConditions := make([]MarketCondition, fmce.config.NumSimulations)

	var wg sync.WaitGroup
	simChan := make(chan FERSMonteCarloSimulation, fmce.config.NumSimulations)
	marketChan := make(chan MarketCondition, fmce.config.NumSimulations)

	// Generate simulations concurrently
	for i := 0; i < fmce.config.NumSimulations; i++ {
		wg.Add(1)
		go func(simID int) {
			defer wg.Done()

			// Generate market conditions for this simulation
			marketCondition := fmce.generateMarketConditions()
			marketChan <- marketCondition

			// Run single FERS simulation
			simulation, err := fmce.runSingleFERSSimulation(ctx, baseScenario, marketCondition, simID)
			if err != nil {
				// Create failed simulation
				simulation = &FERSMonteCarloSimulation{
					SimulationID:    simID,
					MarketCondition: marketCondition,
					Success:         false,
					FailureReason:   err.Error(),
				}
			}

			simChan <- *simulation
		}(i)
	}

	// Wait for all simulations to complete
	go func() {
		wg.Wait()
		close(simChan)
		close(marketChan)
	}()

	// Collect results
	for i := 0; i < fmce.config.NumSimulations; i++ {
		simulations[i] = <-simChan
		marketConditions[i] = <-marketChan
	}

	// Calculate summary statistics
	result := fmce.calculateFERSSummary(simulations, marketConditions, baseScenarioName)

	return result, nil
}

// generateMarketConditions creates market conditions for a single simulation
func (fmce *FERSMonteCarloEngine) generateMarketConditions() MarketCondition {
	condition := MarketCondition{
		TSPReturns:    make(map[string]decimal.Decimal),
		InflationRate: fmce.generateInflationRate(),
		COLARate:      fmce.generateCOLARate(),
		FEHBInflation: fmce.generateFEHBInflation(),
	}

	// Generate TSP fund returns
	funds := []string{"C", "S", "I", "F", "G"}
	for _, fund := range funds {
		if fmce.config.UseHistorical {
			condition.TSPReturns[fund] = fmce.generateHistoricalTSPReturn(fund)
		} else {
			condition.TSPReturns[fund] = fmce.generateStatisticalTSPReturn(fund)
		}
	}

	return condition
}

// generateInflationRate generates a random inflation rate
func (fmce *FERSMonteCarloEngine) generateInflationRate() decimal.Decimal {
	baseRate := fmce.baseConfig.GlobalAssumptions.InflationRate
	if fmce.config.UseHistorical {
		// Use historical inflation data if available
		return fmce.generateHistoricalInflationRate()
	}

	// Generate using normal distribution around base rate
	variability := fmce.config.InflationVariability
	randomFactor := decimal.NewFromFloat(rand.NormFloat64()).Mul(variability)
	result := baseRate.Add(randomFactor)
	if result.LessThan(decimal.NewFromFloat(-0.05)) {
		result = decimal.NewFromFloat(-0.05) // Cap at -5%
	}
	return result
}

// generateCOLARate generates a random COLA rate
func (fmce *FERSMonteCarloEngine) generateCOLARate() decimal.Decimal {
	baseRate := fmce.baseConfig.GlobalAssumptions.COLAGeneralRate
	if fmce.config.UseHistorical {
		return fmce.generateHistoricalCOLARate()
	}

	variability := fmce.config.COLAVariability
	randomFactor := decimal.NewFromFloat(rand.NormFloat64()).Mul(variability)
	result := baseRate.Add(randomFactor)
	if result.LessThan(decimal.NewFromFloat(-0.02)) {
		result = decimal.NewFromFloat(-0.02) // Cap at -2%
	}
	return result
}

// generateFEHBInflation generates a random FEHB inflation rate
func (fmce *FERSMonteCarloEngine) generateFEHBInflation() decimal.Decimal {
	baseRate := fmce.baseConfig.GlobalAssumptions.FEHBPremiumInflation
	variability := fmce.config.FEHBVariability
	randomFactor := decimal.NewFromFloat(rand.NormFloat64()).Mul(variability)
	result := baseRate.Add(randomFactor)
	if result.LessThan(decimal.NewFromFloat(0.0)) {
		result = decimal.NewFromFloat(0.0) // Cap at 0%
	}
	return result
}

// generateHistoricalTSPReturn generates TSP return using historical data
func (fmce *FERSMonteCarloEngine) generateHistoricalTSPReturn(fund string) decimal.Decimal {
	if fmce.historicalData == nil {
		return fmce.generateStatisticalTSPReturn(fund)
	}

	// Get a random historical year
	year, err := fmce.historicalData.GetRandomHistoricalYear()
	if err != nil {
		return fmce.generateStatisticalTSPReturn(fund)
	}

	// Get TSP return for that year
	returnRate, err := fmce.historicalData.GetTSPReturn(fund, year)
	if err != nil {
		return fmce.generateStatisticalTSPReturn(fund)
	}

	return returnRate
}

// generateStatisticalTSPReturn generates TSP return using statistical distribution
func (fmce *FERSMonteCarloEngine) generateStatisticalTSPReturn(fund string) decimal.Decimal {
	// Base returns by fund (long-term averages)
	baseReturns := map[string]decimal.Decimal{
		"C": decimal.NewFromFloat(0.10), // 10% average
		"S": decimal.NewFromFloat(0.12), // 12% average
		"I": decimal.NewFromFloat(0.08), // 8% average
		"F": decimal.NewFromFloat(0.05), // 5% average
		"G": decimal.NewFromFloat(0.03), // 3% average
	}

	baseReturn := baseReturns[fund]
	variability := fmce.config.TSPReturnVariability
	randomFactor := decimal.NewFromFloat(rand.NormFloat64()).Mul(variability)
	result := baseReturn.Add(randomFactor)
	if result.LessThan(decimal.NewFromFloat(-0.5)) {
		result = decimal.NewFromFloat(-0.5) // Cap at -50%
	}
	return result
}

// generateHistoricalInflationRate generates inflation rate using historical data
func (fmce *FERSMonteCarloEngine) generateHistoricalInflationRate() decimal.Decimal {
	if fmce.historicalData == nil {
		// Fall back to statistical generation
		baseRate := fmce.baseConfig.GlobalAssumptions.InflationRate
		variability := fmce.config.InflationVariability
		randomFactor := decimal.NewFromFloat(rand.NormFloat64()).Mul(variability)
		result := baseRate.Add(randomFactor)
		if result.LessThan(decimal.NewFromFloat(-0.05)) {
			result = decimal.NewFromFloat(-0.05) // Cap at -5%
		}
		return result
	}

	// Get a random historical year
	year, err := fmce.historicalData.GetRandomHistoricalYear()
	if err != nil {
		// Fall back to statistical generation
		baseRate := fmce.baseConfig.GlobalAssumptions.InflationRate
		variability := fmce.config.InflationVariability
		randomFactor := decimal.NewFromFloat(rand.NormFloat64()).Mul(variability)
		result := baseRate.Add(randomFactor)
		if result.LessThan(decimal.NewFromFloat(-0.05)) {
			result = decimal.NewFromFloat(-0.05) // Cap at -5%
		}
		return result
	}

	// Get inflation rate for that year
	inflationRate, err := fmce.historicalData.GetInflationRate(year)
	if err != nil {
		// Fall back to statistical generation
		baseRate := fmce.baseConfig.GlobalAssumptions.InflationRate
		variability := fmce.config.InflationVariability
		randomFactor := decimal.NewFromFloat(rand.NormFloat64()).Mul(variability)
		result := baseRate.Add(randomFactor)
		if result.LessThan(decimal.NewFromFloat(-0.05)) {
			result = decimal.NewFromFloat(-0.05) // Cap at -5%
		}
		return result
	}

	return inflationRate
}

// generateHistoricalCOLARate generates COLA rate using historical data
func (fmce *FERSMonteCarloEngine) generateHistoricalCOLARate() decimal.Decimal {
	if fmce.historicalData == nil {
		// Fall back to statistical generation
		baseRate := fmce.baseConfig.GlobalAssumptions.COLAGeneralRate
		variability := fmce.config.COLAVariability
		randomFactor := decimal.NewFromFloat(rand.NormFloat64()).Mul(variability)
		result := baseRate.Add(randomFactor)
		if result.LessThan(decimal.NewFromFloat(-0.02)) {
			result = decimal.NewFromFloat(-0.02) // Cap at -2%
		}
		return result
	}

	// Get a random historical year
	year, err := fmce.historicalData.GetRandomHistoricalYear()
	if err != nil {
		// Fall back to statistical generation
		baseRate := fmce.baseConfig.GlobalAssumptions.COLAGeneralRate
		variability := fmce.config.COLAVariability
		randomFactor := decimal.NewFromFloat(rand.NormFloat64()).Mul(variability)
		result := baseRate.Add(randomFactor)
		if result.LessThan(decimal.NewFromFloat(-0.02)) {
			result = decimal.NewFromFloat(-0.02) // Cap at -2%
		}
		return result
	}

	// Get COLA rate for that year
	colaRate, err := fmce.historicalData.GetCOLARate(year)
	if err != nil {
		// Fall back to statistical generation
		baseRate := fmce.baseConfig.GlobalAssumptions.COLAGeneralRate
		variability := fmce.config.COLAVariability
		randomFactor := decimal.NewFromFloat(rand.NormFloat64()).Mul(variability)
		result := baseRate.Add(randomFactor)
		if result.LessThan(decimal.NewFromFloat(-0.02)) {
			result = decimal.NewFromFloat(-0.02) // Cap at -2%
		}
		return result
	}

	return colaRate
}

// runSingleFERSSimulation runs a single FERS simulation with given market conditions
func (fmce *FERSMonteCarloEngine) runSingleFERSSimulation(ctx context.Context, baseScenario *domain.GenericScenario, marketCondition MarketCondition, simID int) (*FERSMonteCarloSimulation, error) {
	// Create modified configuration with market conditions
	modifiedConfig := fmce.createModifiedConfig(marketCondition)

	// Run the scenario using the calculation engine
	summary, err := fmce.calculationEngine.RunGenericScenario(ctx, modifiedConfig, baseScenario)
	if err != nil {
		return nil, fmt.Errorf("failed to run scenario: %w", err)
	}

	// Determine success/failure
	success := true
	failureYear := 0
	failureReason := ""

	// Check for unrealistic income (too high)
	if summary.TotalLifetimeIncome.GreaterThan(fmce.config.MaxReasonableIncome) {
		success = false
		failureReason = "Unrealistic income projection"
	}

	// Check for TSP depletion too early
	if summary.TSPLongevity < 5 {
		success = false
		failureYear = summary.TSPLongevity
		failureReason = "TSP depleted too early"
	}

	// Check for negative net income in early years
	if summary.Year5NetIncome.LessThan(decimal.NewFromFloat(-50000)) {
		success = false
		failureYear = 5
		failureReason = "Severely negative net income in early retirement"
	}

	return &FERSMonteCarloSimulation{
		SimulationID:    simID,
		ScenarioSummary: *summary,
		MarketCondition: marketCondition,
		Success:         success,
		FailureYear:     failureYear,
		FailureReason:   failureReason,
	}, nil
}

// createModifiedConfig creates a configuration with modified market conditions
func (fmce *FERSMonteCarloEngine) createModifiedConfig(marketCondition MarketCondition) *domain.Configuration {
	// Deep copy the base configuration
	modifiedConfig := *fmce.baseConfig

	// Modify market assumptions
	modifiedConfig.GlobalAssumptions.InflationRate = marketCondition.InflationRate
	modifiedConfig.GlobalAssumptions.COLAGeneralRate = marketCondition.COLARate
	modifiedConfig.GlobalAssumptions.FEHBPremiumInflation = marketCondition.FEHBInflation

	// Modify TSP returns (this would require extending the domain model)
	// For now, we'll use the base returns and let the calculation engine handle variability

	return &modifiedConfig
}

// calculateFERSSummary calculates summary statistics for FERS Monte Carlo results
func (fmce *FERSMonteCarloEngine) calculateFERSSummary(simulations []FERSMonteCarloSimulation, marketConditions []MarketCondition, baseScenarioName string) *FERSMonteCarloResult {
	// Calculate success rate
	successCount := 0
	for _, sim := range simulations {
		if sim.Success {
			successCount++
		}
	}
	successRate := decimal.NewFromFloat(float64(successCount)).Div(decimal.NewFromFloat(float64(len(simulations))))

	// Calculate percentiles for lifetime income
	lifetimeIncomes := make([]decimal.Decimal, 0, len(simulations))
	tspLongevities := make([]int, 0, len(simulations))
	year5Incomes := make([]decimal.Decimal, 0, len(simulations))
	year10Incomes := make([]decimal.Decimal, 0, len(simulations))

	for _, sim := range simulations {
		if sim.Success {
			lifetimeIncomes = append(lifetimeIncomes, sim.ScenarioSummary.TotalLifetimeIncome)
			tspLongevities = append(tspLongevities, sim.ScenarioSummary.TSPLongevity)
			year5Incomes = append(year5Incomes, sim.ScenarioSummary.Year5NetIncome)
			year10Incomes = append(year10Incomes, sim.ScenarioSummary.Year10NetIncome)
		}
	}

	// Calculate median values
	medianLifetimeIncome := decimal.Zero
	medianTSPLongevity := 0
	if len(lifetimeIncomes) > 0 {
		medianLifetimeIncome = calculateMedian(lifetimeIncomes)
		medianTSPLongevity = calculateMedianInt(tspLongevities)
	}

	// Calculate percentile ranges
	percentileRanges := FERSPercentileRanges{
		LifetimeIncome: calculatePercentiles(lifetimeIncomes),
		TSPLongevity:   calculatePercentilesInt(tspLongevities),
		Year5Income:    calculatePercentiles(year5Incomes),
		Year10Income:   calculatePercentiles(year10Incomes),
	}

	return &FERSMonteCarloResult{
		BaseScenarioName:     baseScenarioName,
		NumSimulations:       fmce.config.NumSimulations,
		ProjectionYears:      fmce.config.ProjectionYears,
		SuccessRate:          successRate,
		MedianLifetimeIncome: medianLifetimeIncome,
		MedianTSPLongevity:   medianTSPLongevity,
		PercentileRanges:     percentileRanges,
		Simulations:          simulations,
		MarketConditions:     marketConditions,
	}
}

// Helper functions for statistical calculations
func calculateMedian(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	// Sort values (simple bubble sort for small datasets)
	for i := 0; i < len(values)-1; i++ {
		for j := 0; j < len(values)-i-1; j++ {
			if values[j].GreaterThan(values[j+1]) {
				values[j], values[j+1] = values[j+1], values[j]
			}
		}
	}

	mid := len(values) / 2
	if len(values)%2 == 0 {
		return values[mid-1].Add(values[mid]).Div(decimal.NewFromInt(2))
	}
	return values[mid]
}

func calculateMedianInt(values []int) int {
	if len(values) == 0 {
		return 0
	}

	// Sort values
	for i := 0; i < len(values)-1; i++ {
		for j := 0; j < len(values)-i-1; j++ {
			if values[j] > values[j+1] {
				values[j], values[j+1] = values[j+1], values[j]
			}
		}
	}

	mid := len(values) / 2
	if len(values)%2 == 0 {
		return (values[mid-1] + values[mid]) / 2
	}
	return values[mid]
}

func calculatePercentiles(values []decimal.Decimal) map[string]decimal.Decimal {
	if len(values) == 0 {
		return map[string]decimal.Decimal{
			"10th": decimal.Zero, "25th": decimal.Zero, "50th": decimal.Zero,
			"75th": decimal.Zero, "90th": decimal.Zero,
		}
	}

	// Sort values
	for i := 0; i < len(values)-1; i++ {
		for j := 0; j < len(values)-i-1; j++ {
			if values[j].GreaterThan(values[j+1]) {
				values[j], values[j+1] = values[j+1], values[j]
			}
		}
	}

	percentiles := map[string]decimal.Decimal{
		"10th": getPercentile(values, 0.1),
		"25th": getPercentile(values, 0.25),
		"50th": getPercentile(values, 0.5),
		"75th": getPercentile(values, 0.75),
		"90th": getPercentile(values, 0.9),
	}

	return percentiles
}

func calculatePercentilesInt(values []int) map[string]int {
	if len(values) == 0 {
		return map[string]int{
			"10th": 0, "25th": 0, "50th": 0, "75th": 0, "90th": 0,
		}
	}

	// Sort values
	for i := 0; i < len(values)-1; i++ {
		for j := 0; j < len(values)-i-1; j++ {
			if values[j] > values[j+1] {
				values[j], values[j+1] = values[j+1], values[j]
			}
		}
	}

	percentiles := map[string]int{
		"10th": getPercentileInt(values, 0.1),
		"25th": getPercentileInt(values, 0.25),
		"50th": getPercentileInt(values, 0.5),
		"75th": getPercentileInt(values, 0.75),
		"90th": getPercentileInt(values, 0.9),
	}

	return percentiles
}

func getPercentile(values []decimal.Decimal, percentile float64) decimal.Decimal {
	index := percentile * float64(len(values)-1)
	if index == float64(int(index)) {
		return values[int(index)]
	}

	lower := values[int(index)]
	upper := values[int(index)+1]
	fraction := decimal.NewFromFloat(index - float64(int(index)))

	return lower.Add(upper.Sub(lower).Mul(fraction))
}

func getPercentileInt(values []int, percentile float64) int {
	index := percentile * float64(len(values)-1)
	if index == float64(int(index)) {
		return values[int(index)]
	}

	lower := values[int(index)]
	upper := values[int(index)+1]
	fraction := index - float64(int(index))

	return lower + int(float64(upper-lower)*fraction)
}
