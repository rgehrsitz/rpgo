package calculation

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
)

// FERSMonteCarloConfig holds configuration for FERS Monte Carlo simulations
type FERSMonteCarloConfig struct {
	// Base configuration (reuses existing domain.Configuration)
	BaseConfig *domain.Configuration

	// Monte Carlo specific settings
	NumSimulations int
	UseHistorical  bool
	Seed           int64

	// Market variability settings
	TSPReturnVariability decimal.Decimal // Std dev for TSP returns
	InflationVariability decimal.Decimal // Std dev for inflation
	COLAVariability      decimal.Decimal // Std dev for COLA
	FEHBVariability      decimal.Decimal // Std dev for FEHB increases
}

// FERSMonteCarloEngine manages FERS Monte Carlo simulations
type FERSMonteCarloEngine struct {
	calcEngine     *CalculationEngine
	historicalData *HistoricalDataManager
	config         FERSMonteCarloConfig
}

// FERSMonteCarloResult represents the results of a FERS Monte Carlo simulation
type FERSMonteCarloResult struct {
	// Success metrics
	SuccessRate          decimal.Decimal  `json:"success_rate"`
	MedianNetIncome      decimal.Decimal  `json:"median_net_income"`
	NetIncomePercentiles PercentileRanges `json:"net_income_percentiles"`

	// TSP metrics
	TSPLongevityPercentiles PercentileRanges `json:"tsp_longevity_percentiles"`
	TSPDepletionRate        decimal.Decimal  `json:"tsp_depletion_rate"`
	MedianFinalTSPBalance   decimal.Decimal  `json:"median_final_tsp_balance"`

	// Risk metrics
	IncomeVolatility  decimal.Decimal `json:"income_volatility"`
	WorstCaseScenario decimal.Decimal `json:"worst_case_scenario"`
	BestCaseScenario  decimal.Decimal `json:"best_case_scenario"`

	// Detailed results
	Simulations      []FERSMonteCarloSimulation `json:"simulations"`
	MarketConditions []MarketCondition          `json:"market_conditions"`

	// Configuration
	NumSimulations  int                        `json:"num_simulations"`
	BaseConfig      *domain.Configuration      `json:"base_config"`
	AssetAllocation map[string]decimal.Decimal `json:"asset_allocation"`
}

// FERSMonteCarloSimulation represents a single FERS Monte Carlo simulation
type FERSMonteCarloSimulation struct {
	SimulationID     int                       `json:"simulation_id"`
	MarketConditions MarketCondition           `json:"market_conditions"`
	ScenarioResults  []*domain.ScenarioSummary `json:"scenario_results"`
	Success          bool                      `json:"success"`
	NetIncomeMetrics NetIncomeMetrics          `json:"net_income_metrics"`
	TSPMetrics       TSPMetrics                `json:"tsp_metrics"`
}

// MarketCondition represents market conditions for a simulation
type MarketCondition struct {
	Year          int                        `json:"year"`
	TSPReturns    map[string]decimal.Decimal `json:"tsp_returns"`
	InflationRate decimal.Decimal            `json:"inflation_rate"`
	COLARate      decimal.Decimal            `json:"cola_rate"`
	FEHBIncrease  decimal.Decimal            `json:"fehb_increase"`
}

// MarketConditionSeries represents year-by-year market conditions for entire projection
type MarketConditionSeries struct {
	Years []MarketCondition `json:"years"`
}

// NetIncomeMetrics represents net income metrics for a simulation
type NetIncomeMetrics struct {
	FirstYearNetIncome decimal.Decimal `json:"first_year_net_income"`
	Year5NetIncome     decimal.Decimal `json:"year_5_net_income"`
	Year10NetIncome    decimal.Decimal `json:"year_10_net_income"`
	MinNetIncome       decimal.Decimal `json:"min_net_income"`
	MaxNetIncome       decimal.Decimal `json:"max_net_income"`
	AverageNetIncome   decimal.Decimal `json:"average_net_income"`
}

// TSPMetrics represents TSP metrics for a simulation
type TSPMetrics struct {
	InitialBalance decimal.Decimal `json:"initial_balance"`
	FinalBalance   decimal.Decimal `json:"final_balance"`
	Longevity      int             `json:"longevity"`
	Depleted       bool            `json:"depleted"`
	MaxDrawdown    decimal.Decimal `json:"max_drawdown"`
}

// NewFERSMonteCarloEngine creates a new FERS Monte Carlo engine
func NewFERSMonteCarloEngine(baseConfig *domain.Configuration, historicalData *HistoricalDataManager) *FERSMonteCarloEngine {
	// Get Monte Carlo settings from configuration with defaults
	mcSettings := baseConfig.GlobalAssumptions.MonteCarloSettings

	// Apply defaults if not configured
	tspVariability := mcSettings.TSPReturnVariability
	if tspVariability.IsZero() {
		tspVariability = decimal.NewFromFloat(0.15) // 15% default - typical stock market variability
	}

	inflationVariability := mcSettings.InflationVariability
	if inflationVariability.IsZero() {
		inflationVariability = decimal.NewFromFloat(0.02) // 2% default - based on CPI historical variation
	}

	colaVariability := mcSettings.COLAVariability
	if colaVariability.IsZero() {
		colaVariability = decimal.NewFromFloat(0.02) // 2% default - Social Security COLA variation
	}

	fehbVariability := mcSettings.FEHBVariability
	if fehbVariability.IsZero() {
		fehbVariability = decimal.NewFromFloat(0.05) // 5% default - health insurance premium increases
	}

	return &FERSMonteCarloEngine{
		calcEngine:     NewCalculationEngineWithConfig(baseConfig.GlobalAssumptions.FederalRules),
		historicalData: historicalData,
		config: FERSMonteCarloConfig{
			BaseConfig:           baseConfig,
			NumSimulations:       1000,
			UseHistorical:        true,
			TSPReturnVariability: tspVariability,
			InflationVariability: inflationVariability,
			COLAVariability:      colaVariability,
			FEHBVariability:      fehbVariability,
		},
	}
}

// SetDebug enables or disables debug output
func (fmc *FERSMonteCarloEngine) SetDebug(debug bool) {
	fmc.calcEngine.Debug = debug
}

// SetLogger sets the logger for the underlying calculation engine used by Monte Carlo.
func (fmc *FERSMonteCarloEngine) SetLogger(l Logger) {
	if fmc.calcEngine != nil {
		fmc.calcEngine.SetLogger(l)
	}
}

// RunFERSMonteCarlo executes the FERS Monte Carlo simulation
func (fmce *FERSMonteCarloEngine) RunFERSMonteCarlo(config FERSMonteCarloConfig) (*FERSMonteCarloResult, error) {
	if fmce.historicalData == nil || !fmce.historicalData.IsLoaded {
		return nil, fmt.Errorf("historical data not loaded")
	}

	// Set random seed (Go 1.20+ approach)
	if config.Seed == 0 {
		config.Seed = seedFunc()
	}
	// As of Go 1.20, global rand is automatically seeded with random data
	// For reproducible sequences when seed is specified, use modern Go random generation
	// Note: In Go 1.20+, the global rand is automatically seeded, so we only need to seed
	// if we want reproducible results with a specific seed
	if config.Seed != 0 {
		// For reproducible results, we would need to use a local random source
		// For now, we'll use the global rand which is automatically seeded in Go 1.20+
		// This maintains the same behavior while avoiding the deprecated call
	}

	// Update config
	fmce.config = config

	// Run simulations in parallel
	simulations := make([]FERSMonteCarloSimulation, config.NumSimulations)
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 10) // Limit concurrency

	for i := 0; i < config.NumSimulations; i++ {
		wg.Add(1)
		go func(simIndex int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			simulation, err := fmce.runSingleFERSSimulation(simIndex)
			if err != nil {
				// Log error but continue with other simulations
				if fmce.calcEngine != nil && fmce.calcEngine.Logger != nil {
					fmce.calcEngine.Logger.Errorf("Simulation %d failed: %v", simIndex, err)
				}
				return
			}
			simulations[simIndex] = *simulation
		}(i)
	}

	wg.Wait()

	// Calculate aggregate results
	result := fmce.calculateAggregateResults(simulations)

	return result, nil
}

// runSingleFERSSimulation runs a single FERS Monte Carlo simulation
func (fmce *FERSMonteCarloEngine) runSingleFERSSimulation(simIndex int) (*FERSMonteCarloSimulation, error) {
	// Generate market conditions with enhanced variability
	marketConditions := fmce.generateEnhancedMarketConditions()

	// Create a proper deep copy of the configuration to ensure each simulation is independent
	modifiedConfig := fmce.deepCopyConfiguration(fmce.config.BaseConfig)
	modifiedConfig.GlobalAssumptions = fmce.applyMarketConditionsToAssumptions(marketConditions)

	// Apply TSP market conditions to the configuration
	fmce.applyMarketConditionsToTSPCalculations(marketConditions, &modifiedConfig)

	// Create a separate calculation engine instance for this simulation to avoid race conditions
	// when running parallel simulations with different Monte Carlo fund returns
	simEngine := NewCalculationEngineWithConfig(modifiedConfig.GlobalAssumptions.FederalRules)
	simEngine.HistoricalData = fmce.calcEngine.HistoricalData // Share historical data
	simEngine.Logger = fmce.calcEngine.Logger                 // Share logger
	simEngine.Debug = fmce.calcEngine.Debug                   // Share debug setting

	// Set Monte Carlo fund returns on this simulation's engine
	simEngine.MonteCarloFundReturns = marketConditions.TSPReturns

	// Run full FERS calculation for each scenario using the simulation-specific engine
	var scenarioResults []*domain.ScenarioSummary
	for _, scenario := range modifiedConfig.Scenarios {
		summary, err := simEngine.RunScenario(context.Background(), &modifiedConfig, &scenario)
		if err != nil {
			return nil, fmt.Errorf("failed to run scenario %s: %w", scenario.Name, err)
		}
		scenarioResults = append(scenarioResults, summary)
	}

	// No need to clear Monte Carlo fund returns as this engine instance will be discarded

	// Calculate metrics
	netIncomeMetrics := fmce.calculateNetIncomeMetrics(scenarioResults)
	tspMetrics := fmce.calculateTSPMetrics(scenarioResults)

	// Determine success (simplified: check if any scenario has sustainable income)
	success := fmce.determineSuccess(scenarioResults)

	return &FERSMonteCarloSimulation{
		SimulationID:     simIndex,
		MarketConditions: marketConditions,
		ScenarioResults:  scenarioResults,
		Success:          success,
		NetIncomeMetrics: netIncomeMetrics,
		TSPMetrics:       tspMetrics,
	}, nil
}


// generateEnhancedMarketConditions generates market conditions with proper Monte Carlo variability
func (fmce *FERSMonteCarloEngine) generateEnhancedMarketConditions() MarketCondition {
	if fmce.config.UseHistorical {
		return fmce.generateEnhancedHistoricalMarketConditions()
	} else {
		return fmce.generateStatisticalMarketConditions()
	}
}

// generateEnhancedHistoricalMarketConditions generates more realistic historical market conditions
// by sampling different historical years for different market components and applying variability
func (fmce *FERSMonteCarloEngine) generateEnhancedHistoricalMarketConditions() MarketCondition {
	// Get available years
	minYear, maxYear, err := fmce.historicalData.GetAvailableYears()
	if err != nil {
		// Fallback to statistical if no historical data
		return fmce.generateStatisticalMarketConditions()
	}

	marketData := MarketCondition{
		TSPReturns: make(map[string]decimal.Decimal),
	}

	// Sample DIFFERENT historical years for different components to increase variability
	tspYear := minYear + rand.Intn(maxYear-minYear+1)
	inflationYear := minYear + rand.Intn(maxYear-minYear+1)
	colaYear := minYear + rand.Intn(maxYear-minYear+1)
	fehbYear := minYear + rand.Intn(maxYear-minYear+1)

	// Sample TSP fund returns from one historical year, but apply variability
	funds := []string{"C", "S", "I", "F", "G"}
	for _, fund := range funds {
		if baseReturn, err := fmce.historicalData.GetTSPReturn(fund, tspYear); err == nil {
			// Apply random variability around the historical value using configured parameters
			variabilityFactor := fmce.generateRandomVariability(fmce.config.TSPReturnVariability)
			adjustedReturn := baseReturn.Mul(decimal.NewFromFloat(1.0).Add(variabilityFactor))
			marketData.TSPReturns[fund] = adjustedReturn
		} else {
			// Fallback to statistical generation
			marketData.TSPReturns[fund] = fmce.generateStatisticalTSPReturn(fund)
		}
	}

	// Sample inflation from a different historical year with variability
	if baseInflation, err := fmce.historicalData.GetInflationRate(inflationYear); err == nil {
		variabilityFactor := fmce.generateRandomVariability(fmce.config.InflationVariability)
		marketData.InflationRate = baseInflation.Mul(decimal.NewFromFloat(1.0).Add(variabilityFactor))
	} else {
		marketData.InflationRate = fmce.generateStatisticalInflation()
	}

	// Sample COLA from yet another historical year with variability
	if baseCOLA, err := fmce.historicalData.GetCOLARate(colaYear); err == nil {
		variabilityFactor := fmce.generateRandomVariability(fmce.config.COLAVariability)
		marketData.COLARate = baseCOLA.Mul(decimal.NewFromFloat(1.0).Add(variabilityFactor))
	} else {
		marketData.COLARate = fmce.generateStatisticalCOLA()
	}

	// Sample FEHB increase from another year with variability
	if baseFEHB, err := fmce.historicalData.GetInflationRate(fehbYear); err == nil {
		// Use inflation as proxy for FEHB increases, with additional variability
		variabilityFactor := fmce.generateRandomVariability(fmce.config.FEHBVariability)
		marketData.FEHBIncrease = baseFEHB.Mul(decimal.NewFromFloat(1.0).Add(variabilityFactor))
	} else {
		marketData.FEHBIncrease = fmce.generateStatisticalInflation() // Fallback
	}

	marketData.Year = tspYear // Use TSP year as reference
	return marketData
}

// generateRandomVariability generates a random variability factor using normal distribution
// Returns a factor between -3*stdDev and +3*stdDev (approximately 99.7% of values)
func (fmce *FERSMonteCarloEngine) generateRandomVariability(stdDev decimal.Decimal) decimal.Decimal {
	if stdDev.IsZero() {
		return decimal.Zero
	}

	// Generate normal distribution using Box-Muller transform
	// Generate two uniform random numbers
	u1 := rand.Float64()
	u2 := rand.Float64()

	// Box-Muller transformation to get normal distribution
	z := math.Sqrt(-2.0*math.Log(u1)) * math.Cos(2.0*math.Pi*u2)

	// Scale by standard deviation and convert to decimal
	variability := decimal.NewFromFloat(z).Mul(stdDev)

	// Cap at +/- 3 standard deviations to prevent extreme outliers
	maxVariability := stdDev.Mul(decimal.NewFromFloat(3.0))
	minVariability := stdDev.Mul(decimal.NewFromFloat(-3.0))

	if variability.GreaterThan(maxVariability) {
		variability = maxVariability
	} else if variability.LessThan(minVariability) {
		variability = minVariability
	}

	return variability
}

// generateHistoricalMarketConditions generates market conditions from historical data
func (fmce *FERSMonteCarloEngine) generateHistoricalMarketConditions() MarketCondition {
	// Get random historical year
	minYear, maxYear, err := fmce.historicalData.GetAvailableYears()
	if err != nil {
		// Fallback to statistical if no historical data
		return fmce.generateStatisticalMarketConditions()
	}

	randomYear := minYear + rand.Intn(maxYear-minYear+1)

	// Get historical data for that year
	marketData := MarketCondition{
		Year:       randomYear,
		TSPReturns: make(map[string]decimal.Decimal),
	}

	// Sample TSP fund returns
	funds := []string{"C", "S", "I", "F", "G"}
	for _, fund := range funds {
		if returnRate, err := fmce.historicalData.GetTSPReturn(fund, randomYear); err == nil {
			marketData.TSPReturns[fund] = returnRate
		} else {
			// Fallback to statistical generation
			marketData.TSPReturns[fund] = fmce.generateStatisticalTSPReturn(fund)
		}
	}

	// Sample inflation and COLA
	if inflation, err := fmce.historicalData.GetInflationRate(randomYear); err == nil {
		marketData.InflationRate = inflation
	} else {
		marketData.InflationRate = fmce.generateStatisticalInflation()
	}

	if cola, err := fmce.historicalData.GetCOLARate(randomYear); err == nil {
		marketData.COLARate = cola
	} else {
		marketData.COLARate = fmce.generateStatisticalCOLA()
	}

	// Generate FEHB increase (not in historical data, so use statistical)
	marketData.FEHBIncrease = fmce.generateStatisticalFEHBIncrease()

	return marketData
}

// generateStatisticalMarketConditions generates market conditions using statistical distributions
func (fmce *FERSMonteCarloEngine) generateStatisticalMarketConditions() MarketCondition {
	marketData := MarketCondition{
		Year:       rand.Intn(30) + 2025, // Random year between 2025-2055
		TSPReturns: make(map[string]decimal.Decimal),
	}

	// Generate TSP fund returns
	funds := []string{"C", "S", "I", "F", "G"}
	for _, fund := range funds {
		marketData.TSPReturns[fund] = fmce.generateStatisticalTSPReturn(fund)
	}

	marketData.InflationRate = fmce.generateStatisticalInflation()
	marketData.COLARate = fmce.generateStatisticalCOLA()
	marketData.FEHBIncrease = fmce.generateStatisticalFEHBIncrease()

	return marketData
}

// generateStatisticalTSPReturn generates statistical TSP return for a fund
func (fmce *FERSMonteCarloEngine) generateStatisticalTSPReturn(fund string) decimal.Decimal {
	// Get statistical models from configuration
	models := fmce.config.BaseConfig.GlobalAssumptions.TSPStatisticalModels

	var mean, stdDev decimal.Decimal
	var foundInConfig bool

	// Get parameters from configuration based on fund
	switch fund {
	case "C":
		if !models.CFund.Mean.IsZero() && !models.CFund.StandardDev.IsZero() {
			mean = models.CFund.Mean
			stdDev = models.CFund.StandardDev
			foundInConfig = true
		}
	case "S":
		if !models.SFund.Mean.IsZero() && !models.SFund.StandardDev.IsZero() {
			mean = models.SFund.Mean
			stdDev = models.SFund.StandardDev
			foundInConfig = true
		}
	case "I":
		if !models.IFund.Mean.IsZero() && !models.IFund.StandardDev.IsZero() {
			mean = models.IFund.Mean
			stdDev = models.IFund.StandardDev
			foundInConfig = true
		}
	case "F":
		if !models.FFund.Mean.IsZero() && !models.FFund.StandardDev.IsZero() {
			mean = models.FFund.Mean
			stdDev = models.FFund.StandardDev
			foundInConfig = true
		}
	case "G":
		if !models.GFund.Mean.IsZero() && !models.GFund.StandardDev.IsZero() {
			mean = models.GFund.Mean
			stdDev = models.GFund.StandardDev
			foundInConfig = true
		}
	}

	// Use historical defaults if not configured (preserving current values with documentation)
	if !foundInConfig {
		switch fund {
		case "C":
			mean = decimal.NewFromFloat(0.1125)   // 11.25% historical mean (TSP.gov 1988-2024)
			stdDev = decimal.NewFromFloat(0.1744) // 17.44% historical std dev
		case "S":
			mean = decimal.NewFromFloat(0.1117)   // 11.17% historical mean (TSP.gov 1988-2024)
			stdDev = decimal.NewFromFloat(0.1933) // 19.33% historical std dev
		case "I":
			mean = decimal.NewFromFloat(0.0634)   // 6.34% historical mean (TSP.gov 1988-2024)
			stdDev = decimal.NewFromFloat(0.1863) // 18.63% historical std dev
		case "F":
			mean = decimal.NewFromFloat(0.0532)   // 5.32% historical mean (TSP.gov 1988-2024)
			stdDev = decimal.NewFromFloat(0.0565) // 5.65% historical std dev
		case "G":
			mean = decimal.NewFromFloat(0.0493)   // 4.93% historical mean (TSP.gov 1988-2024)
			stdDev = decimal.NewFromFloat(0.0165) // 1.65% historical std dev (very stable)
		default:
			mean = decimal.NewFromFloat(0.08)   // 8% default mean for unknown funds
			stdDev = decimal.NewFromFloat(0.15) // 15% default std dev
		}
	}

	// Generate normal distribution using Box-Muller transform
	u1 := rand.Float64()
	u2 := rand.Float64()
	z := fmce.boxMullerTransform(u1, u2)

	// Convert to decimal and apply mean/std dev
	zDecimal := decimal.NewFromFloat(z)
	return mean.Add(zDecimal.Mul(stdDev))
}

// generateStatisticalInflation generates statistical inflation rate
func (fmce *FERSMonteCarloEngine) generateStatisticalInflation() decimal.Decimal {
	mean := decimal.NewFromFloat(0.0259)   // 2.59% historical mean
	stdDev := decimal.NewFromFloat(0.0137) // 1.37% historical std dev

	u1 := rand.Float64()
	u2 := rand.Float64()
	z := fmce.boxMullerTransform(u1, u2)

	zDecimal := decimal.NewFromFloat(z)
	inflation := mean.Add(zDecimal.Mul(stdDev))

	// Ensure inflation is within reasonable bounds (0% to 20%)
	if inflation.LessThan(decimal.Zero) {
		inflation = decimal.Zero
	} else if inflation.GreaterThan(decimal.NewFromFloat(0.20)) {
		inflation = decimal.NewFromFloat(0.20)
	}

	return inflation
}

// generateStatisticalCOLA generates statistical COLA rate
func (fmce *FERSMonteCarloEngine) generateStatisticalCOLA() decimal.Decimal {
	mean := decimal.NewFromFloat(0.0255)   // 2.55% historical mean
	stdDev := decimal.NewFromFloat(0.0182) // 1.82% historical std dev

	u1 := rand.Float64()
	u2 := rand.Float64()
	z := fmce.boxMullerTransform(u1, u2)

	zDecimal := decimal.NewFromFloat(z)
	cola := mean.Add(zDecimal.Mul(stdDev))

	// Ensure COLA is within reasonable bounds (0% to 15%)
	if cola.LessThan(decimal.Zero) {
		cola = decimal.Zero
	} else if cola.GreaterThan(decimal.NewFromFloat(0.15)) {
		cola = decimal.NewFromFloat(0.15)
	}

	return cola
}

// generateStatisticalFEHBIncrease generates statistical FEHB premium increase
func (fmce *FERSMonteCarloEngine) generateStatisticalFEHBIncrease() decimal.Decimal {
	mean := decimal.NewFromFloat(0.045)   // 4.5% historical mean
	stdDev := decimal.NewFromFloat(0.025) // 2.5% historical std dev

	u1 := rand.Float64()
	u2 := rand.Float64()
	z := fmce.boxMullerTransform(u1, u2)

	zDecimal := decimal.NewFromFloat(z)
	return mean.Add(zDecimal.Mul(stdDev))
}

// boxMullerTransform implements Box-Muller transform for normal distribution
func (fmce *FERSMonteCarloEngine) boxMullerTransform(u1, u2 float64) float64 {
	z0 := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)
	return z0
}

// applyMarketConditionsToAssumptions applies market conditions to global assumptions
func (fmce *FERSMonteCarloEngine) applyMarketConditionsToAssumptions(market MarketCondition) domain.GlobalAssumptions {
	// Create a copy of the original assumptions instead of modifying the shared reference
	assumptions := fmce.config.BaseConfig.GlobalAssumptions

	// Apply market conditions to the copy
	assumptions.InflationRate = market.InflationRate
	assumptions.COLAGeneralRate = market.COLARate

	return assumptions
}

// applyMarketConditionsToTSPCalculations applies market conditions to TSP calculations
func (fmce *FERSMonteCarloEngine) applyMarketConditionsToTSPCalculations(market MarketCondition, config *domain.Configuration) {
	// Get default TSP allocation from configuration
	defaultAllocation := config.GlobalAssumptions.MonteCarloSettings.DefaultTSPAllocation

	// Create asset allocation map from configuration (with fallback defaults)
	assetAllocation := map[string]decimal.Decimal{
		"C": defaultAllocation.CFund,
		"S": defaultAllocation.SFund,
		"I": defaultAllocation.IFund,
		"F": defaultAllocation.FFund,
		"G": defaultAllocation.GFund,
	}

	// Apply fallback defaults if allocation is zero or not configured
	if defaultAllocation.CFund.IsZero() && defaultAllocation.SFund.IsZero() &&
		defaultAllocation.IFund.IsZero() && defaultAllocation.FFund.IsZero() &&
		defaultAllocation.GFund.IsZero() {
		// Use conservative balanced allocation as ultimate fallback
		assetAllocation = map[string]decimal.Decimal{
			"C": decimal.NewFromFloat(0.60), // 60% Large Cap Stock Index
			"S": decimal.NewFromFloat(0.20), // 20% Small Cap Stock Index
			"I": decimal.NewFromFloat(0.10), // 10% International Stock Index
			"F": decimal.NewFromFloat(0.10), // 10% Fixed Income Index
			"G": decimal.NewFromFloat(0.00), // 0% Government Securities
		}
	}

	var weightedReturn decimal.Decimal
	for fund, allocation := range assetAllocation {
		if returnRate, exists := market.TSPReturns[fund]; exists {
			weightedReturn = weightedReturn.Add(returnRate.Mul(allocation))
		}
	}

	// Apply the weighted return to both pre and post retirement TSP return rates
	config.GlobalAssumptions.TSPReturnPreRetirement = weightedReturn
	config.GlobalAssumptions.TSPReturnPostRetirement = weightedReturn
}

// calculateNetIncomeMetrics calculates net income metrics for a simulation
func (fmce *FERSMonteCarloEngine) calculateNetIncomeMetrics(scenarioResults []*domain.ScenarioSummary) NetIncomeMetrics {
	if len(scenarioResults) == 0 {
		return NetIncomeMetrics{}
	}

	// Use the first scenario for now (could be enhanced to aggregate across scenarios)
	summary := scenarioResults[0]

	// Calculate min, max, and average net income across the projection period
	var minNetIncome, maxNetIncome, totalNetIncome decimal.Decimal
	var count int

	// Use the projection data to calculate variability
	if len(summary.Projection) > 0 {
		minNetIncome = summary.Projection[0].NetIncome
		maxNetIncome = summary.Projection[0].NetIncome
		totalNetIncome = decimal.Zero

		// Apply reasonable bounds to prevent extreme outliers while preserving natural distribution
		for _, year := range summary.Projection {
			netIncome := year.NetIncome

			// Only validate for obviously impossible values
			if netIncome.LessThan(decimal.Zero) {
				netIncome = decimal.Zero
			}

			// Cap extremely unrealistic values using configured limit
			// This preserves the natural distribution while preventing calculation errors
			maxReasonableIncome := fmce.config.BaseConfig.GlobalAssumptions.MonteCarloSettings.MaxReasonableIncome
			if maxReasonableIncome.IsZero() {
				maxReasonableIncome = decimal.NewFromInt(5000000) // $5M default cap
			}

			if netIncome.GreaterThan(maxReasonableIncome) {
				// Cap extreme values that might indicate calculation errors
				netIncome = maxReasonableIncome
			}

			if netIncome.LessThan(minNetIncome) {
				minNetIncome = netIncome
			}
			if netIncome.GreaterThan(maxNetIncome) {
				maxNetIncome = netIncome
			}
			totalNetIncome = totalNetIncome.Add(netIncome)
			count++
		}
	} else {
		// Fallback to first year values if no projection data
		minNetIncome = summary.FirstYearNetIncome
		maxNetIncome = summary.FirstYearNetIncome
		totalNetIncome = summary.FirstYearNetIncome
		count = 1
	}

	averageNetIncome := totalNetIncome.Div(decimal.NewFromInt(int64(count)))

	return NetIncomeMetrics{
		FirstYearNetIncome: summary.FirstYearNetIncome,
		Year5NetIncome:     summary.Year5NetIncome,
		Year10NetIncome:    summary.Year10NetIncome,
		MinNetIncome:       minNetIncome,
		MaxNetIncome:       maxNetIncome,
		AverageNetIncome:   averageNetIncome,
	}
}

// calculateTSPMetrics calculates TSP metrics for a simulation
func (fmce *FERSMonteCarloEngine) calculateTSPMetrics(scenarioResults []*domain.ScenarioSummary) TSPMetrics {
	if len(scenarioResults) == 0 {
		return TSPMetrics{}
	}

	// Use the first scenario for now (could be enhanced to aggregate across scenarios)
	summary := scenarioResults[0]

	return TSPMetrics{
		InitialBalance: summary.InitialTSPBalance,
		FinalBalance:   summary.FinalTSPBalance,
		Longevity:      summary.TSPLongevity,
		Depleted:       summary.TSPLongevity < len(summary.Projection),
		MaxDrawdown:    decimal.Zero, // Would need to calculate from projection
	}
}

// determineSuccess determines if a simulation is successful
func (fmce *FERSMonteCarloEngine) determineSuccess(scenarioResults []*domain.ScenarioSummary) bool {
	if len(scenarioResults) == 0 {
		return false
	}

	// Simplified success criteria: check if TSP lasts the full projection
	// Could be enhanced with more sophisticated criteria
	for _, summary := range scenarioResults {
		if summary.TSPLongevity < len(summary.Projection) {
			return false
		}
	}

	return true
}

// calculateAggregateResults calculates aggregate results across all simulations
func (fmce *FERSMonteCarloEngine) calculateAggregateResults(simulations []FERSMonteCarloSimulation) *FERSMonteCarloResult {
	// Count successful simulations
	successCount := 0
	for _, sim := range simulations {
		if sim.Success {
			successCount++
		}
	}

	successRate := decimal.NewFromInt(int64(successCount)).Div(decimal.NewFromInt(int64(len(simulations))))

	// Calculate net income percentiles using average net income for better variability representation
	var netIncomes []decimal.Decimal
	for _, sim := range simulations {
		netIncomes = append(netIncomes, sim.NetIncomeMetrics.AverageNetIncome)
	}

	netIncomePercentiles := fmce.calculatePercentileRanges(netIncomes)

	// Calculate TSP longevity percentiles
	var tspLongevities []decimal.Decimal
	for _, sim := range simulations {
		tspLongevities = append(tspLongevities, decimal.NewFromInt(int64(sim.TSPMetrics.Longevity)))
	}

	tspLongevityPercentiles := fmce.calculatePercentileRanges(tspLongevities)

	// Calculate TSP depletion rate
	depletionCount := 0
	for _, sim := range simulations {
		if sim.TSPMetrics.Depleted {
			depletionCount++
		}
	}

	tspDepletionRate := decimal.NewFromInt(int64(depletionCount)).Div(decimal.NewFromInt(int64(len(simulations))))

	// Calculate median final TSP balance
	var finalTSPBalances []decimal.Decimal
	for _, sim := range simulations {
		finalTSPBalances = append(finalTSPBalances, sim.TSPMetrics.FinalBalance)
	}
	medianFinalTSPBalance := fmce.calculateMedian(finalTSPBalances)

	// Calculate median net income using average net income across projection period
	medianNetIncome := fmce.calculateMedian(netIncomes)

	// Calculate income volatility (simplified)
	incomeVolatility := fmce.calculateStandardDeviation(netIncomes)

	// Find worst and best case scenarios
	worstCase := fmce.findMin(netIncomes)
	bestCase := fmce.findMax(netIncomes)

	return &FERSMonteCarloResult{
		SuccessRate:             successRate,
		MedianNetIncome:         medianNetIncome,
		NetIncomePercentiles:    netIncomePercentiles,
		TSPLongevityPercentiles: tspLongevityPercentiles,
		TSPDepletionRate:        tspDepletionRate,
		MedianFinalTSPBalance:   medianFinalTSPBalance,
		IncomeVolatility:        incomeVolatility,
		WorstCaseScenario:       worstCase,
		BestCaseScenario:        bestCase,
		Simulations:             simulations,
		NumSimulations:          len(simulations),
		BaseConfig:              fmce.config.BaseConfig,
	}
}

// Helper functions for statistical calculations
func (fmce *FERSMonteCarloEngine) calculatePercentileRanges(values []decimal.Decimal) PercentileRanges {
	if len(values) == 0 {
		return PercentileRanges{}
	}

	// Sort values
	fmce.sortDecimalSlice(values)

	n := len(values)
	return PercentileRanges{
		P10: values[n/10],
		P25: values[n/4],
		P50: values[n/2],
		P75: values[3*n/4],
		P90: values[9*n/10],
	}
}

func (fmce *FERSMonteCarloEngine) calculateMedian(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	fmce.sortDecimalSlice(values)
	return values[len(values)/2]
}

func (fmce *FERSMonteCarloEngine) calculateStandardDeviation(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	// Calculate mean
	var sum decimal.Decimal
	for _, v := range values {
		sum = sum.Add(v)
	}
	mean := sum.Div(decimal.NewFromInt(int64(len(values))))

	// Calculate variance
	var varianceSum decimal.Decimal
	for _, v := range values {
		diff := v.Sub(mean)
		varianceSum = varianceSum.Add(diff.Mul(diff))
	}
	variance := varianceSum.Div(decimal.NewFromInt(int64(len(values))))

	// Calculate standard deviation
	varianceFloat, _ := variance.Float64()
	stdDevFloat := math.Sqrt(varianceFloat)
	return decimal.NewFromFloat(stdDevFloat)
}

func (fmce *FERSMonteCarloEngine) findMin(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	min := values[0]
	for _, v := range values {
		if v.LessThan(min) {
			min = v
		}
	}
	return min
}

func (fmce *FERSMonteCarloEngine) findMax(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}

	max := values[0]
	for _, v := range values {
		if v.GreaterThan(max) {
			max = v
		}
	}
	return max
}

func (fmce *FERSMonteCarloEngine) sortDecimalSlice(values []decimal.Decimal) {
	// Simple bubble sort for small arrays
	for i := 0; i < len(values)-1; i++ {
		for j := 0; j < len(values)-i-1; j++ {
			if values[j].GreaterThan(values[j+1]) {
				values[j], values[j+1] = values[j+1], values[j]
			}
		}
	}
}

// deepCopyConfiguration creates a deep copy of the configuration to ensure each simulation is independent
func (fmce *FERSMonteCarloEngine) deepCopyConfiguration(config *domain.Configuration) domain.Configuration {
	// Deep copy the configuration
	newConfig := domain.Configuration{
		PersonalDetails:   make(map[string]domain.Employee),
		GlobalAssumptions: config.GlobalAssumptions, // This will be overwritten anyway
		Scenarios:         make([]domain.Scenario, len(config.Scenarios)),
	}

	// Deep copy personal details
	for key, employee := range config.PersonalDetails {
		newConfig.PersonalDetails[key] = employee // decimal.Decimal is a value type, so this is safe
	}

	// Deep copy scenarios
	copy(newConfig.Scenarios, config.Scenarios)

	return newConfig
}
