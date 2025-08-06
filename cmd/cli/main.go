package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/rpgo/retirement-calculator/internal/output"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "fers-calc",
	Short: "FERS Retirement Calculator",
	Long:  "Comprehensive retirement planning calculator for federal employees",
}

var calculateCmd = &cobra.Command{
	Use:   "calculate [input-file]",
	Short: "Calculate retirement scenarios",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]

		// Parse input
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile(inputFile)
		if err != nil {
			log.Fatal(err)
		}

		// Run calculations
		engine := calculation.NewCalculationEngineWithConfig(config.GlobalAssumptions.FederalRules)
		debugMode, _ := cmd.Flags().GetBool("debug")
		engine.Debug = debugMode
		results, err := engine.RunScenarios(config)
		if err != nil {
			log.Fatal(err)
		}

		// Generate output
		outputFormat, _ := cmd.Flags().GetString("format")
		output.GenerateReport(results, outputFormat)
	},
}

var exampleCmd = &cobra.Command{
	Use:   "example [output-file]",
	Short: "Generate an example configuration file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		outputFile := args[0]

		parser := config.NewInputParser()
		exampleConfig := parser.CreateExampleConfiguration()

		err := output.SaveConfiguration(exampleConfig, outputFile)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Example configuration saved to %s\n", outputFile)
	},
}

var validateCmd = &cobra.Command{
	Use:   "validate [input-file]",
	Short: "Validate a configuration file",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]

		parser := config.NewInputParser()
		_, err := parser.LoadFromFile(inputFile)
		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("Configuration file %s is valid\n", inputFile)
	},
}

var breakEvenCmd = &cobra.Command{
	Use:   "break-even [input-file]",
	Short: "Calculate break-even TSP withdrawal rates to match current net income",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]

		// Parse input
		parser := config.NewInputParser()
		config, err := parser.LoadFromFile(inputFile)
		if err != nil {
			log.Fatal(err)
		}

		// Run break-even analysis
		engine := calculation.NewCalculationEngineWithConfig(config.GlobalAssumptions.FederalRules)
		debugMode, _ := cmd.Flags().GetBool("debug")
		engine.Debug = debugMode
		analysis, err := engine.CalculateBreakEvenAnalysis(config)
		if err != nil {
			log.Fatal(err)
		}

		// Display results
		fmt.Println("BREAK-EVEN TSP WITHDRAWAL RATE ANALYSIS")
		fmt.Println("========================================")
		fmt.Printf("Target Net Income (Current): $%s\n\n", analysis.TargetNetIncome.StringFixed(2))

		for _, result := range analysis.Results {
			fmt.Printf("SCENARIO: %s\n", result.ScenarioName)
			fmt.Println(strings.Repeat("-", 50))
			fmt.Printf("Break-Even TSP Withdrawal Rate: %s%%\n", result.BreakEvenWithdrawalRate.Mul(decimal.NewFromInt(100)).StringFixed(2))
			fmt.Printf("Analysis Year: %d (first full retirement year)\n", result.ProjectedYear)
			fmt.Printf("Projected Net Income: $%s\n", result.ProjectedNetIncome.StringFixed(2))
			fmt.Printf("Total TSP Withdrawal: $%s\n", result.TSPWithdrawalAmount.StringFixed(2))
			fmt.Printf("Remaining TSP Balance: $%s\n", result.TotalTSPBalance.StringFixed(2))
			diff := result.CurrentVsBreakEvenDiff
			if diff.Abs().LessThan(decimal.NewFromInt(1000)) {
				fmt.Printf("Income Match: Within $1,000 (difference: $%s)\n", diff.StringFixed(2))
			} else {
				fmt.Printf("Income Difference: $%s\n", diff.StringFixed(2))
			}
			fmt.Println()
		}

		fmt.Println("INTERPRETATION:")
		fmt.Println("• These withdrawal rates would provide the same net income as your current working situation")
		fmt.Println("• Lower rates mean you could withdraw less and still maintain your lifestyle")
		fmt.Println("• Consider starting with a lower rate (like 2-3%) and adjusting as needed")
		fmt.Println("• Remember that these rates will grow your TSP if they're below the investment return rate")
	},
}

var fersMonteCarloCmd = &cobra.Command{
	Use:   "monte-carlo [config-file] [data-path]",
	Short: "Run FERS Monte Carlo simulations using configuration file",
	Long:  "Run comprehensive FERS Monte Carlo simulations that model all retirement components (pension, SS, TSP, taxes, FEHB) with variable market conditions.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		configFile := args[0]
		dataPath := args[1]

		// Parse configuration
		parser := config.NewInputParser()
		baseConfig, err := parser.LoadFromFile(configFile)
		if err != nil {
			log.Fatal(err)
		}

		// Load historical data
		hdm := calculation.NewHistoricalDataManager(dataPath)
		if err := hdm.LoadAllData(); err != nil {
			fmt.Printf("Error loading historical data: %v\n", err)
			os.Exit(1)
		}

		// Get Monte Carlo parameters
		numSimulations, _ := cmd.Flags().GetInt("simulations")
		useHistorical, _ := cmd.Flags().GetBool("historical")
		seed, _ := cmd.Flags().GetInt64("seed")
		debugMode, _ := cmd.Flags().GetBool("debug")

		// Create FERS Monte Carlo engine
		engine := calculation.NewFERSMonteCarloEngine(baseConfig, hdm)
		engine.SetDebug(debugMode)

		// Configure Monte Carlo simulation
		mcConfig := calculation.FERSMonteCarloConfig{
			BaseConfig:     baseConfig,
			NumSimulations: numSimulations,
			UseHistorical:  useHistorical,
			Seed:           seed,
		}

		fmt.Printf("\n🎲 FERS MONTE CARLO SIMULATION\n")
		fmt.Printf("==============================\n")
		fmt.Printf("Configuration: %s\n", configFile)
		fmt.Printf("Historical Data: %s\n", dataPath)
		fmt.Printf("Simulations: %d\n", numSimulations)
		fmt.Printf("Data Source: %s\n", map[bool]string{true: "Historical", false: "Statistical"}[useHistorical])
		if seed != 0 {
			fmt.Printf("Seed: %d\n", seed)
		}

		// Run simulation with progress indication
		fmt.Printf("\n🔄 Running %d simulations...\n", numSimulations)
		fmt.Printf("   Progress: [")
		startTime := time.Now()

		// Show progress for large simulations
		if numSimulations > 100 {
			for i := 0; i < 10; i++ {
				fmt.Printf(" ")
			}
			fmt.Printf("] 0%%")
		}

		result, err := engine.RunFERSMonteCarlo(mcConfig)
		if err != nil {
			log.Fatal(err)
		}
		duration := time.Since(startTime)

		// Clear progress line and show completion
		if numSimulations > 100 {
			fmt.Printf("\r   Progress: [")
			for i := 0; i < 10; i++ {
				fmt.Printf("█")
			}
			fmt.Printf("] 100%%\n")
		}

		// Display results
		fmt.Printf("\n📊 SIMULATION RESULTS\n")
		fmt.Printf("====================\n")
		fmt.Printf("Execution Time: %v\n", duration)
		fmt.Printf("Simulations Completed: %d\n", result.NumSimulations)

		// Success metrics
		fmt.Printf("\n✅ Success Metrics:\n")
		successRatePercent := result.SuccessRate.Mul(decimal.NewFromInt(100))
		fmt.Printf("  Success Rate: %s%%\n", successRatePercent.StringFixed(2))
		fmt.Printf("  Median Net Income: $%s\n", result.MedianNetIncome.StringFixed(2))
		fmt.Printf("  Income Volatility: $%s\n", result.IncomeVolatility.StringFixed(2))

		// TSP metrics
		fmt.Printf("\n💰 TSP Analysis:\n")
		tspDepletionPercent := result.TSPDepletionRate.Mul(decimal.NewFromInt(100))
		fmt.Printf("  TSP Depletion Rate: %s%%\n", tspDepletionPercent.StringFixed(2))
		fmt.Printf("  Median TSP Longevity: %s years\n", result.TSPLongevityPercentiles.P50.StringFixed(0))

		// Percentile analysis
		fmt.Printf("\n📈 Net Income Percentiles:\n")
		fmt.Printf("  10th Percentile: $%s\n", result.NetIncomePercentiles.P10.StringFixed(2))
		fmt.Printf("  25th Percentile: $%s\n", result.NetIncomePercentiles.P25.StringFixed(2))
		fmt.Printf("  50th Percentile: $%s\n", result.NetIncomePercentiles.P50.StringFixed(2))
		fmt.Printf("  75th Percentile: $%s\n", result.NetIncomePercentiles.P75.StringFixed(2))
		fmt.Printf("  90th Percentile: $%s\n", result.NetIncomePercentiles.P90.StringFixed(2))

		// Risk assessment
		fmt.Printf("\n⚠️  Risk Assessment:\n")
		riskLevel, riskDescription := calculateRiskLevel(result.SuccessRate)
		fmt.Printf("  Risk Level: %s\n", riskLevel)
		fmt.Printf("  Description: %s\n", riskDescription)

		// Recommendations
		fmt.Printf("\n💡 Recommendations:\n")
		recommendations := generateRecommendations(result)
		for _, rec := range recommendations {
			fmt.Printf("  • %s\n", rec)
		}

		// Market conditions summary
		if len(result.Simulations) > 0 {
			fmt.Printf("\n📊 Market Conditions Summary:\n")
			fmt.Printf("  Average Inflation Rate: %s%%\n", calculateAverageInflation(result).Mul(decimal.NewFromInt(100)).StringFixed(2))
			fmt.Printf("  Average COLA Rate: %s%%\n", calculateAverageCOLA(result).Mul(decimal.NewFromInt(100)).StringFixed(2))
		}

		// Generate HTML report
		htmlReport := &output.MonteCarloHTMLReport{
			Result: result,
			Config: mcConfig,
		}

		htmlOutputPath := "monte_carlo_report.html"
		if err := htmlReport.GenerateHTMLReport(htmlOutputPath); err != nil {
			fmt.Printf("\n⚠️  Warning: Could not generate HTML report: %v\n", err)
		} else {
			fmt.Printf("\n📄 HTML Report Generated: %s\n", htmlOutputPath)
			fmt.Printf("   Open this file in your web browser for interactive charts and detailed analysis\n")
		}

		// Generate CSV reports
		csvReport := &output.MonteCarloCSVReport{
			Result: result,
			Config: mcConfig,
		}

		csvOutputDir := "monte_carlo_csv"
		if err := csvReport.GenerateAllCSVReports(csvOutputDir); err != nil {
			fmt.Printf("\n⚠️  Warning: Could not generate CSV reports: %v\n", err)
		} else {
			fmt.Printf("\n📊 CSV Reports Generated in: %s/\n", csvOutputDir)
			fmt.Printf("   - monte_carlo_summary.csv: Aggregate statistics\n")
			fmt.Printf("   - monte_carlo_detailed.csv: Individual simulation results\n")
			fmt.Printf("   - monte_carlo_percentiles.csv: Percentile analysis\n")
		}

		// Summary with color-coded success rate
		successRateFloat, _ := successRatePercent.Float64()

		fmt.Printf("\n" + strings.Repeat("=", 60) + "\n")
		fmt.Printf("🎯 QUICK SUMMARY\n")
		fmt.Printf("Success Rate: ")
		if successRateFloat >= 90 {
			fmt.Printf("🟢 %s%% (Excellent)\n", successRatePercent.StringFixed(1))
		} else if successRateFloat >= 70 {
			fmt.Printf("🟡 %s%% (Good)\n", successRatePercent.StringFixed(1))
		} else {
			fmt.Printf("🔴 %s%% (Needs Attention)\n", successRatePercent.StringFixed(1))
		}
		fmt.Printf("Median Income: $%s\n", result.MedianNetIncome.StringFixed(0))
		fmt.Printf("Median Final TSP Balance: $%s\n", result.MedianFinalTSPBalance.StringFixed(0))
		fmt.Printf("Risk Level: %s\n", riskLevel)
		fmt.Printf(strings.Repeat("=", 60) + "\n")
	},
}

func init() {
	calculateCmd.Flags().StringP("format", "f", "console", "Output format (console, html, json, csv)")
	calculateCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	calculateCmd.Flags().Bool("debug", false, "Enable debug output for detailed calculations")

	// Break-even command flags
	breakEvenCmd.Flags().Bool("debug", false, "Enable debug output for detailed calculations")

	// FERS Monte Carlo command flags
	fersMonteCarloCmd.Flags().IntP("simulations", "s", 1000, "Number of simulations to run")
	fersMonteCarloCmd.Flags().BoolP("historical", "d", true, "Use historical data (false for statistical)")
	fersMonteCarloCmd.Flags().Int64P("seed", "r", 0, "Random seed (0 for auto-generated)")
	fersMonteCarloCmd.Flags().Bool("debug", false, "Enable debug output for detailed calculations")

	rootCmd.AddCommand(calculateCmd)
	rootCmd.AddCommand(exampleCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(breakEvenCmd)
	rootCmd.AddCommand(fersMonteCarloCmd)

	// Initialize historical command
	initHistoricalCommand()
}

func initHistoricalCommand() {
	// Historical command
	historicalCmd := &cobra.Command{
		Use:   "historical",
		Short: "Manage and analyze historical financial data",
		Long:  "Historical data management for TSP returns, inflation, and COLA rates.",
	}

	// Load subcommand
	loadCmd := &cobra.Command{
		Use:   "load [data-path]",
		Short: "Load historical data from the specified path",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dataPath := args[0]

			// Check if data path exists
			if _, err := os.Stat(dataPath); os.IsNotExist(err) {
				fmt.Printf("Error: Data path '%s' does not exist\n", dataPath)
				os.Exit(1)
			}

			// Create historical data manager
			hdm := calculation.NewHistoricalDataManager(dataPath)

			fmt.Printf("Loading historical data from: %s\n", dataPath)

			// Load all data
			if err := hdm.LoadAllData(); err != nil {
				fmt.Printf("Error loading data: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("✅ Historical data loaded successfully!")

			// Display summary
			minYear, maxYear, err := hdm.GetAvailableYears()
			if err != nil {
				fmt.Printf("Error getting year range: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("\n📊 Data Summary:\n")
			fmt.Printf("  Year Range: %d - %d (%d years)\n", minYear, maxYear, maxYear-minYear+1)
			fmt.Printf("  TSP Funds: C, S, I, F, G\n")
			fmt.Printf("  Inflation: CPI-U rates\n")
			fmt.Printf("  COLA: Social Security rates\n")

			// Validate data quality
			issues, err := hdm.ValidateDataQuality()
			if err != nil {
				fmt.Printf("Error validating data quality: %v\n", err)
				os.Exit(1)
			}

			if len(issues) == 0 {
				fmt.Println("  ✅ Data quality: No issues found")
			} else {
				fmt.Printf("  ⚠️  Data quality issues found:\n")
				for _, issue := range issues {
					fmt.Printf("    - %s\n", issue)
				}
			}
		},
	}

	// Stats subcommand
	statsCmd := &cobra.Command{
		Use:   "stats [data-path]",
		Short: "Display statistical summaries of historical data",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dataPath := args[0]

			hdm := calculation.NewHistoricalDataManager(dataPath)
			if err := hdm.LoadAllData(); err != nil {
				fmt.Printf("Error loading data: %v\n", err)
				os.Exit(1)
			}

			fmt.Println("📈 Historical Data Statistics")

			// TSP Fund Statistics
			fmt.Println("TSP Fund Returns (Annual):")
			funds := map[string]*calculation.HistoricalDataSet{
				"C Fund (S&P 500)":         hdm.TSPFunds.CFund,
				"S Fund (Small Cap)":       hdm.TSPFunds.SFund,
				"I Fund (International)":   hdm.TSPFunds.IFund,
				"F Fund (Bonds)":           hdm.TSPFunds.FFund,
				"G Fund (Govt Securities)": hdm.TSPFunds.GFund,
			}

			for name, dataset := range funds {
				if dataset == nil {
					continue
				}
				stats := dataset.Statistics
				fmt.Printf("  %s:\n", name)
				fmt.Printf("    Mean: %s%%\n", stats.Mean.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("    Std Dev: %s%%\n", stats.StdDev.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("    Min: %s%%\n", stats.Min.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("    Max: %s%%\n", stats.Max.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("    Years: %d\n", stats.Count)
				fmt.Println()
			}

			// Inflation Statistics
			if hdm.Inflation != nil {
				stats := hdm.Inflation.Statistics
				fmt.Println("Inflation (CPI-U):")
				fmt.Printf("  Mean: %s%%\n", stats.Mean.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("  Std Dev: %s%%\n", stats.StdDev.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("  Min: %s%%\n", stats.Min.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("  Max: %s%%\n", stats.Max.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("  Years: %d\n", stats.Count)
				fmt.Println()
			}

			// COLA Statistics
			if hdm.COLA != nil {
				stats := hdm.COLA.Statistics
				fmt.Println("Social Security COLA:")
				fmt.Printf("  Mean: %s%%\n", stats.Mean.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("  Std Dev: %s%%\n", stats.StdDev.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("  Min: %s%%\n", stats.Min.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("  Max: %s%%\n", stats.Max.Mul(decimal.NewFromInt(100)).StringFixed(3))
				fmt.Printf("  Years: %d\n", stats.Count)
				fmt.Println()
			}
		},
	}

	// Query subcommand
	queryCmd := &cobra.Command{
		Use:   "query [data-path] [year] [fund-type]",
		Short: "Query specific historical data",
		Long:  "Query specific historical data for a given year and fund type.\n\nFund types: C, S, I, F, G, inflation, cola\nExample: historical query ./data 2020 C",
		Args:  cobra.ExactArgs(3),
		Run: func(cmd *cobra.Command, args []string) {
			dataPath := args[0]
			yearStr := args[1]
			fundType := args[2]

			// Parse year
			var year int
			if _, err := fmt.Sscanf(yearStr, "%d", &year); err != nil {
				fmt.Printf("Error: Invalid year '%s'\n", yearStr)
				os.Exit(1)
			}

			hdm := calculation.NewHistoricalDataManager(dataPath)
			if err := hdm.LoadAllData(); err != nil {
				fmt.Printf("Error loading data: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("🔍 Querying %s data for year %d\n\n", fundType, year)

			switch fundType {
			case "C", "S", "I", "F", "G":
				result, err := hdm.GetTSPReturn(fundType, year)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("TSP %s Fund Return: %s%%\n", fundType, result.Mul(decimal.NewFromInt(100)).StringFixed(3))

			case "inflation":
				result, err := hdm.GetInflationRate(year)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("Inflation Rate: %s%%\n", result.Mul(decimal.NewFromInt(100)).StringFixed(3))

			case "cola":
				result, err := hdm.GetCOLARate(year)
				if err != nil {
					fmt.Printf("Error: %v\n", err)
					os.Exit(1)
				}
				fmt.Printf("COLA Rate: %s%%\n", result.Mul(decimal.NewFromInt(100)).StringFixed(3))

			default:
				fmt.Printf("Error: Unknown fund type '%s'. Valid types: C, S, I, F, G, inflation, cola\n", fundType)
				os.Exit(1)
			}
		},
	}

	// Monte Carlo subcommand
	monteCarloCmd := &cobra.Command{
		Use:   "monte-carlo [data-path]",
		Short: "Run Monte Carlo retirement simulations",
		Long:  "Run Monte Carlo simulations to analyze retirement portfolio sustainability using historical or statistical market data.",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			dataPath := args[0]

			// Load historical data
			hdm := calculation.NewHistoricalDataManager(dataPath)
			if err := hdm.LoadAllData(); err != nil {
				fmt.Printf("Error loading historical data: %v\n", err)
				os.Exit(1)
			}

			// Get simulation parameters
			numSimulations, _ := cmd.Flags().GetInt("simulations")
			projectionYears, _ := cmd.Flags().GetInt("years")
			useHistorical, _ := cmd.Flags().GetBool("historical")
			initialBalance, _ := cmd.Flags().GetFloat64("balance")
			annualWithdrawal, _ := cmd.Flags().GetFloat64("withdrawal")
			withdrawalStrategy, _ := cmd.Flags().GetString("strategy")

			// Default asset allocation (60% C, 20% S, 10% I, 10% F)
			assetAllocation := map[string]decimal.Decimal{
				"C": decimal.NewFromFloat(0.6),
				"S": decimal.NewFromFloat(0.2),
				"I": decimal.NewFromFloat(0.1),
				"F": decimal.NewFromFloat(0.1),
			}

			// Create Monte Carlo configuration
			config := calculation.MonteCarloConfig{
				NumSimulations:     numSimulations,
				ProjectionYears:    projectionYears,
				Seed:               time.Now().UnixNano(),
				UseHistorical:      useHistorical,
				AssetAllocation:    assetAllocation,
				WithdrawalStrategy: withdrawalStrategy,
				InitialBalance:     decimal.NewFromFloat(initialBalance),
				AnnualWithdrawal:   decimal.NewFromFloat(annualWithdrawal),
			}

			// Create and run simulator
			simulator := calculation.NewMonteCarloSimulator(hdm, config)
			result, err := simulator.RunSimulation(config)
			if err != nil {
				fmt.Printf("Error running Monte Carlo simulation: %v\n", err)
				os.Exit(1)
			}

			// Display results
			fmt.Println("🎲 MONTE CARLO SIMULATION RESULTS")
			fmt.Println("==================================")
			fmt.Printf("Simulations: %d\n", result.NumSimulations)
			fmt.Printf("Projection Years: %d\n", result.ProjectionYears)
			fmt.Printf("Data Source: %s\n", map[bool]string{true: "Historical", false: "Statistical"}[useHistorical])
			fmt.Printf("Withdrawal Strategy: %s\n", withdrawalStrategy)
			fmt.Printf("Initial Balance: $%s\n", result.InitialBalance.StringFixed(2))
			fmt.Printf("Annual Withdrawal: $%s\n", result.AnnualWithdrawal.StringFixed(2))
			fmt.Println()

			// Asset allocation
			fmt.Println("Asset Allocation:")
			for fund, allocation := range result.AssetAllocation {
				fmt.Printf("  %s Fund: %s%%\n", fund, allocation.Mul(decimal.NewFromInt(100)).StringFixed(1))
			}
			fmt.Println()

			// Success metrics
			fmt.Println("Success Metrics:")
			fmt.Printf("  Success Rate: %s%%\n", result.SuccessRate.Mul(decimal.NewFromInt(100)).StringFixed(2))
			fmt.Printf("  Median Ending Balance: $%s\n", result.MedianEndingBalance.StringFixed(2))
			fmt.Println()

			// Percentile ranges
			fmt.Println("Ending Balance Percentiles:")
			fmt.Printf("  10th Percentile: $%s\n", result.PercentileRanges.P10.StringFixed(2))
			fmt.Printf("  25th Percentile: $%s\n", result.PercentileRanges.P25.StringFixed(2))
			fmt.Printf("  50th Percentile: $%s\n", result.PercentileRanges.P50.StringFixed(2))
			fmt.Printf("  75th Percentile: $%s\n", result.PercentileRanges.P75.StringFixed(2))
			fmt.Printf("  90th Percentile: $%s\n", result.PercentileRanges.P90.StringFixed(2))
			fmt.Println()

			// Risk assessment
			fmt.Println("Risk Assessment:")
			if result.SuccessRate.GreaterThanOrEqual(decimal.NewFromFloat(0.95)) {
				fmt.Println("  🟢 LOW RISK: 95%+ success rate")
			} else if result.SuccessRate.GreaterThanOrEqual(decimal.NewFromFloat(0.85)) {
				fmt.Println("  🟡 MODERATE RISK: 85-95% success rate")
			} else if result.SuccessRate.GreaterThanOrEqual(decimal.NewFromFloat(0.75)) {
				fmt.Println("  🟠 HIGH RISK: 75-85% success rate")
			} else {
				fmt.Println("  🔴 VERY HIGH RISK: <75% success rate")
			}

			// Recommendations
			fmt.Println("\nRecommendations:")
			if result.SuccessRate.LessThan(decimal.NewFromFloat(0.85)) {
				fmt.Println("  • Consider reducing withdrawal amount")
				fmt.Println("  • Increase allocation to bonds (F/G funds)")
				fmt.Println("  • Consider working longer or saving more")
			} else if result.SuccessRate.GreaterThan(decimal.NewFromFloat(0.95)) {
				fmt.Println("  • Current plan appears sustainable")
				fmt.Println("  • Consider increasing withdrawal or taking more risk")
			} else {
				fmt.Println("  • Monitor plan regularly")
				fmt.Println("  • Consider guardrails withdrawal strategy")
			}
		},
	}

	// Add flags to Monte Carlo command
	monteCarloCmd.Flags().IntP("simulations", "s", 1000, "Number of simulations to run")
	monteCarloCmd.Flags().IntP("years", "y", 25, "Number of years to project")
	monteCarloCmd.Flags().BoolP("historical", "d", true, "Use historical data (false for statistical)")
	monteCarloCmd.Flags().Float64P("balance", "b", 1000000, "Initial portfolio balance")
	monteCarloCmd.Flags().Float64P("withdrawal", "w", 40000, "Annual withdrawal amount")
	monteCarloCmd.Flags().StringP("strategy", "t", "fixed_amount", "Withdrawal strategy (fixed_amount, fixed_percentage, inflation_adjusted, guardrails)")

	historicalCmd.AddCommand(loadCmd)
	historicalCmd.AddCommand(statsCmd)
	historicalCmd.AddCommand(queryCmd)
	historicalCmd.AddCommand(monteCarloCmd)
	rootCmd.AddCommand(historicalCmd)
}

// Helper functions for FERS Monte Carlo

func calculateRiskLevel(successRate decimal.Decimal) (string, string) {
	if successRate.GreaterThanOrEqual(decimal.NewFromFloat(0.95)) {
		return "🟢 LOW RISK", "95%+ success rate indicates sustainable retirement plan"
	} else if successRate.GreaterThanOrEqual(decimal.NewFromFloat(0.85)) {
		return "🟡 MODERATE RISK", "85-95% success rate suggests plan needs monitoring"
	} else if successRate.GreaterThanOrEqual(decimal.NewFromFloat(0.75)) {
		return "🟠 HIGH RISK", "75-85% success rate indicates plan may need adjustment"
	} else {
		return "🔴 VERY HIGH RISK", "<75% success rate suggests plan needs significant changes"
	}
}

func generateRecommendations(result *calculation.FERSMonteCarloResult) []string {
	var recommendations []string

	if result.SuccessRate.LessThan(decimal.NewFromFloat(0.85)) {
		recommendations = append(recommendations, "Consider reducing retirement spending or working longer")
		recommendations = append(recommendations, "Increase allocation to bonds (F/G funds) for stability")
		recommendations = append(recommendations, "Consider delaying Social Security benefits")
		recommendations = append(recommendations, "Review FEHB plan options to reduce costs")
	} else if result.SuccessRate.GreaterThan(decimal.NewFromFloat(0.95)) {
		recommendations = append(recommendations, "Current plan appears sustainable")
		recommendations = append(recommendations, "Consider increasing retirement spending")
		recommendations = append(recommendations, "May be able to retire earlier or take more investment risk")
	} else {
		recommendations = append(recommendations, "Monitor plan regularly and adjust as needed")
		recommendations = append(recommendations, "Consider guardrails withdrawal strategy")
		recommendations = append(recommendations, "Review asset allocation for optimal risk/return")
	}

	// Add TSP-specific recommendations
	if result.TSPDepletionRate.GreaterThan(decimal.NewFromFloat(0.1)) {
		recommendations = append(recommendations, "TSP may deplete early - consider reducing withdrawal rate")
	}

	return recommendations
}

func calculateAverageInflation(result *calculation.FERSMonteCarloResult) decimal.Decimal {
	if len(result.Simulations) == 0 {
		return decimal.Zero
	}

	var total decimal.Decimal
	count := 0
	for _, sim := range result.Simulations {
		if !sim.MarketConditions.InflationRate.IsZero() {
			total = total.Add(sim.MarketConditions.InflationRate)
			count++
		}
	}

	if count == 0 {
		return decimal.Zero
	}
	return total.Div(decimal.NewFromInt(int64(count)))
}

func calculateAverageCOLA(result *calculation.FERSMonteCarloResult) decimal.Decimal {
	if len(result.Simulations) == 0 {
		return decimal.Zero
	}

	var total decimal.Decimal
	count := 0
	for _, sim := range result.Simulations {
		if !sim.MarketConditions.COLARate.IsZero() {
			total = total.Add(sim.MarketConditions.COLARate)
			count++
		}
	}

	if count == 0 {
		return decimal.Zero
	}
	return total.Div(decimal.NewFromInt(int64(count)))
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
