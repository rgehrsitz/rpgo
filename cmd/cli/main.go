package main

import (
	"fmt"
	"log"
	"os"
	"strings"

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
		engine := calculation.NewCalculationEngine()
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
		engine := calculation.NewCalculationEngine()
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

func init() {
	calculateCmd.Flags().StringP("format", "f", "console", "Output format (console, html, json, csv)")
	calculateCmd.Flags().BoolP("monte-carlo", "m", false, "Run Monte Carlo simulation")
	calculateCmd.Flags().IntP("simulations", "s", 1000, "Number of Monte Carlo simulations")
	calculateCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	
	rootCmd.AddCommand(calculateCmd)
	rootCmd.AddCommand(exampleCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(breakEvenCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
} 