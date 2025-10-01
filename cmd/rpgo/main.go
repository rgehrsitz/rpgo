package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/rgehrsitz/rpgo/internal/compare"
	"github.com/rgehrsitz/rpgo/internal/config"
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/output"
	"github.com/rgehrsitz/rpgo/internal/transform"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

// simpleCLILogger implements calculation.Logger using the standard log package
type simpleCLILogger struct{}

func (simpleCLILogger) Debugf(format string, args ...any) { log.Printf("DEBUG: "+format, args...) }
func (simpleCLILogger) Infof(format string, args ...any)  { log.Printf("INFO: "+format, args...) }
func (simpleCLILogger) Warnf(format string, args ...any)  { log.Printf("WARN: "+format, args...) }
func (simpleCLILogger) Errorf(format string, args ...any) { log.Printf("ERROR: "+format, args...) }

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "rpgo %s (commit %s, built %s)\n", version, commit, date)
			if info := buildInfo(); info != "" {
				fmt.Fprintln(os.Stdout, info)
			}
		},
	}
}

func buildInfo() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi != nil {
		return bi.String()
	}
	return ""
}

// fileExists checks if a file exists
func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return !os.IsNotExist(err)
}

var rootCmd = &cobra.Command{
	Use:   "rpgo",
	Short: "FERS Retirement Calculator CLI",
	Long:  "Comprehensive retirement planning calculator for federal employees",
}

var calculateCmd = &cobra.Command{
	Use:   "calculate [input-file]",
	Short: "Calculate retirement scenarios",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]

		// Parse input - try regulatory config first, fall back to original
		parser := config.NewInputParser()
		regulatoryFile, _ := cmd.Flags().GetString("regulatory-config")

		var configData *domain.Configuration
		var err error

		// Try to load with regulatory config if specified or if regulatory.yaml exists
		if regulatoryFile != "" || fileExists("regulatory.yaml") {
			if regulatoryFile == "" {
				regulatoryFile = "regulatory.yaml"
			}

			fmt.Printf("Loading regulatory config from: %s\n", regulatoryFile)
			configData, err = parser.LoadFromFileWithRegulatory(inputFile, regulatoryFile)
			if err != nil {
				fmt.Printf("Failed to load with regulatory config: %v\n", err)
				fmt.Printf("Falling back to standalone config loading...\n")
				// Fall back to original loading
				configData, err = parser.LoadFromFile(inputFile)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				fmt.Printf("✅ Successfully loaded config with regulatory data\n")
			}
		} else {
			// Original loading method
			configData, err = parser.LoadFromFile(inputFile)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Load historical data if available
		var hdm *calculation.HistoricalDataManager
		dataPath := "data" // Default path, could be made configurable
		if _, err := os.Stat(dataPath); err == nil {
			hdm = calculation.NewHistoricalDataManager(dataPath)
			if loadErr := hdm.LoadAllData(); loadErr != nil {
				fmt.Printf("Warning: Could not load historical data from %s: %v\n", dataPath, loadErr)
				fmt.Printf("Falling back to statistical models...\n")
				hdm = nil
			}
		}

		// Run calculations
		engine := calculation.NewCalculationEngineWithConfig(configData.GlobalAssumptions.FederalRules)
		engine.HistoricalData = hdm // Set the historical data manager
		debugMode, _ := cmd.Flags().GetBool("debug")
		if debugMode {
			engine.SetLogger(simpleCLILogger{})
		}
		engine.Debug = debugMode
		results, err := engine.RunScenarios(configData)
		if err != nil {
			log.Fatal(err)
		}

		// Generate output
		outputFormat, _ := cmd.Flags().GetString("format")

		// Get the formatter and write to stdout instead of file
		if f := output.GetFormatterByName(outputFormat); f != nil {
			data, err := f.Format(results)
			if err != nil {
				log.Fatal(err)
			}
			fmt.Print(string(data))
		} else {
			// Fallback to original GenerateReport for unsupported formats
			if err := output.GenerateReport(results, outputFormat); err != nil {
				log.Fatal(err)
			}
		}
	},
}

// Example config command removed (legacy)

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

		// Load historical data if available
		var hdm *calculation.HistoricalDataManager
		dataPath := "data" // Default path, could be made configurable
		if _, err := os.Stat(dataPath); err == nil {
			hdm = calculation.NewHistoricalDataManager(dataPath)
			if loadErr := hdm.LoadAllData(); loadErr != nil {
				fmt.Printf("Warning: Could not load historical data from %s: %v\n", dataPath, loadErr)
				fmt.Printf("Falling back to statistical models...\n")
				hdm = nil
			}
		}

		// Run break-even analysis
		engine := calculation.NewCalculationEngineWithConfig(config.GlobalAssumptions.FederalRules)
		engine.HistoricalData = hdm // Set the historical data manager
		debugMode, _ := cmd.Flags().GetBool("debug")
		if debugMode {
			engine.SetLogger(simpleCLILogger{})
		}
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

var compareCmd = &cobra.Command{
	Use:   "compare [input-file]",
	Short: "Compare retirement scenarios using built-in strategy templates",
	Long: `Compare a base retirement scenario against alternative strategies.

Examples:
  ./rpgo compare config.yaml --base Base --with postpone_1yr,delay_ss_70
  ./rpgo compare config.yaml --base Base --with conservative,aggressive --format csv
  ./rpgo compare config.yaml --list-templates  # Show all available templates
`,
	Args: cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		// Handle --list-templates flag
		listTemplates, _ := cmd.Flags().GetBool("list-templates")
		if listTemplates {
			// Get participant name for templates (or use default)
			participantName, _ := cmd.Flags().GetString("participant")
			if participantName == "" {
				participantName = "Participant" // default placeholder
			}

			registry := transform.CreateBuiltInTemplates(participantName)
			fmt.Print(transform.GetTemplateHelp(registry))
			return
		}

		// Require input file for actual comparison
		if len(args) == 0 {
			log.Fatal("input file required for comparison (use --list-templates to see available templates)")
		}

		inputFile := args[0]

		// Parse input
		parser := config.NewInputParser()
		regulatoryFile, _ := cmd.Flags().GetString("regulatory-config")

		var configData *domain.Configuration
		var err error

		// Try to load with regulatory config if specified or if regulatory.yaml exists
		if regulatoryFile != "" || fileExists("regulatory.yaml") {
			if regulatoryFile == "" {
				regulatoryFile = "regulatory.yaml"
			}

			configData, err = parser.LoadFromFileWithRegulatory(inputFile, regulatoryFile)
			if err != nil {
				fmt.Printf("Failed to load with regulatory config: %v\n", err)
				fmt.Printf("Falling back to standalone config loading...\n")
				configData, err = parser.LoadFromFile(inputFile)
				if err != nil {
					log.Fatal(err)
				}
			}
		} else {
			configData, err = parser.LoadFromFile(inputFile)
			if err != nil {
				log.Fatal(err)
			}
		}

		// Get comparison options
		baseScenarioName, _ := cmd.Flags().GetString("base")
		templatesStr, _ := cmd.Flags().GetString("with")
		participantName, _ := cmd.Flags().GetString("participant")
		outputFormat, _ := cmd.Flags().GetString("format")

		if baseScenarioName == "" {
			log.Fatal("--base flag is required to specify the base scenario name")
		}

		if templatesStr == "" {
			log.Fatal("--with flag is required to specify templates to compare (or use --list-templates)")
		}

		// Parse template list
		templateNames := transform.ParseTemplateList(templatesStr)
		if len(templateNames) == 0 {
			log.Fatal("no valid templates specified in --with flag")
		}

		// Determine participant name
		if participantName == "" {
			// Try to get first participant from household
			if len(configData.Household.Participants) > 0 {
				participantName = configData.Household.Participants[0].Name
			} else {
				log.Fatal("--participant flag required when household has multiple participants")
			}
		}

		// Create calculation engine
		engine := calculation.NewCalculationEngineWithConfig(configData.GlobalAssumptions.FederalRules)
		debugMode, _ := cmd.Flags().GetBool("debug")
		if debugMode {
			engine.SetLogger(simpleCLILogger{})
			engine.Debug = true
		}

		// Create comparison engine
		compareEngine := compare.NewCompareEngine(engine)

		// Run comparison
		ctx := context.Background()
		comparisonSet, err := compareEngine.Compare(ctx, configData, compare.CompareOptions{
			BaseScenarioName: baseScenarioName,
			Templates:        templateNames,
			ParticipantName:  participantName,
		})
		if err != nil {
			log.Fatalf("Comparison failed: %v", err)
		}

		// Set config path for display
		comparisonSet.ConfigPath = inputFile

		// Format and output results
		switch strings.ToLower(outputFormat) {
		case "csv":
			formatter := &compare.CSVFormatter{}
			output, err := formatter.Format(comparisonSet)
			if err != nil {
				log.Fatalf("Failed to format CSV: %v", err)
			}
			fmt.Print(output)

		case "json":
			formatter := &compare.JSONFormatter{Pretty: true}
			output, err := formatter.Format(comparisonSet)
			if err != nil {
				log.Fatalf("Failed to format JSON: %v", err)
			}
			fmt.Print(output)

		case "table", "console", "":
			formatter := &compare.TableFormatter{}
			output := formatter.Format(comparisonSet)
			fmt.Print(output)

		default:
			log.Fatalf("Unknown output format: %s (valid: table, csv, json)", outputFormat)
		}
	},
}

// Monte Carlo command removed (legacy)

func init() {
	calculateCmd.Flags().StringP("format", "f", "console", "Output format (console, html, json, csv)")
	calculateCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	calculateCmd.Flags().Bool("debug", false, "Enable debug output for detailed calculations")
	calculateCmd.Flags().String("regulatory-config", "", "Path to regulatory config file (default: regulatory.yaml if it exists)")

	// Break-even command flags
	breakEvenCmd.Flags().Bool("debug", false, "Enable debug output for detailed calculations")

	// Compare command flags
	compareCmd.Flags().String("base", "", "Base scenario name to compare against (required)")
	compareCmd.Flags().String("with", "", "Comma-separated list of templates to compare (required)")
	compareCmd.Flags().StringP("format", "f", "table", "Output format (table, csv, json)")
	compareCmd.Flags().String("participant", "", "Participant name for template application (auto-detected if not specified)")
	compareCmd.Flags().Bool("list-templates", false, "List all available scenario templates")
	compareCmd.Flags().Bool("debug", false, "Enable debug output for detailed calculations")
	compareCmd.Flags().String("regulatory-config", "", "Path to regulatory config file (default: regulatory.yaml if it exists)")

	// FERS Monte Carlo command flags
	rootCmd.AddCommand(calculateCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(breakEvenCmd)
	rootCmd.AddCommand(compareCmd)
	rootCmd.AddCommand(versionCmd())

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
			riskLevel, _ := calculateRiskLevel(result.SuccessRate)
			fmt.Printf("  %s\n", riskLevel)

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
	monteCarloCmd.Flags().Float64P("withdrawal", "w", 40000, "Annual withdrawal amount (or percentage as decimal for fixed_percentage strategy, e.g., 0.04 for 4%)")
	monteCarloCmd.Flags().StringP("strategy", "t", "fixed_amount", "Withdrawal strategy: fixed_amount (constant $), fixed_percentage (% of balance), inflation_adjusted ($ + inflation), guardrails (dynamic)")

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

// Legacy Monte Carlo helper functions removed

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
