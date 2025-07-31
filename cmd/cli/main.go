package main

import (
	"fmt"
	"log"
	"os"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/rpgo/retirement-calculator/internal/output"
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

func init() {
	calculateCmd.Flags().StringP("format", "f", "console", "Output format (console, html, json, csv)")
	calculateCmd.Flags().BoolP("monte-carlo", "m", false, "Run Monte Carlo simulation")
	calculateCmd.Flags().IntP("simulations", "s", 1000, "Number of Monte Carlo simulations")
	calculateCmd.Flags().BoolP("verbose", "v", false, "Verbose output")
	
	rootCmd.AddCommand(calculateCmd)
	rootCmd.AddCommand(exampleCmd)
	rootCmd.AddCommand(validateCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
} 