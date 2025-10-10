package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/rgehrsitz/rpgo/internal/config"
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/output"
	"github.com/shopspring/decimal"
	"github.com/spf13/cobra"
)

var sensitivityCmd = &cobra.Command{
	Use:   "sensitivity [input-file]",
	Short: "Perform sensitivity analysis on retirement scenarios",
	Long: `Perform sensitivity analysis to test how robust retirement plans are to parameter changes.

Examples:
  # Single parameter sweep
  ./rpgo sensitivity config.yaml --parameter inflation_rate --range 0.015-0.040 --steps 6

  # Multiple parameter sweep
  ./rpgo sensitivity config.yaml --parameter inflation_rate:0.015-0.040:6 --parameter tsp_return_post_retirement:0.03-0.06:4

  # Matrix analysis
  ./rpgo sensitivity config.yaml --parameter inflation_rate:0.015-0.040:6 --parameter tsp_return_post_retirement:0.03-0.06:4 --output matrix

  # Use predefined parameter sets
  ./rpgo sensitivity config.yaml --parameter-set common --base-scenario "Baseline Scenario"`,
	Args: cobra.ExactArgs(1),
	Run:  runSensitivityAnalysis,
}

var (
	sensitivityParameter    []string
	sensitivityRange        string
	sensitivitySteps        int
	sensitivityBaseScenario string
	sensitivityOutputFormat string
	sensitivityParameterSet string
	sensitivityAnalysisType string
)

func init() {
	sensitivityCmd.Flags().StringSliceVar(&sensitivityParameter, "parameter", []string{}, "Parameter to analyze (format: name:min-max:steps)")
	sensitivityCmd.Flags().StringVar(&sensitivityRange, "range", "", "Range for single parameter analysis (format: min-max)")
	sensitivityCmd.Flags().IntVar(&sensitivitySteps, "steps", 5, "Number of steps for parameter sweep")
	sensitivityCmd.Flags().StringVar(&sensitivityBaseScenario, "base-scenario", "", "Base scenario name for analysis")
	sensitivityCmd.Flags().StringVar(&sensitivityOutputFormat, "output", "table", "Output format (table, csv, json)")
	sensitivityCmd.Flags().StringVar(&sensitivityParameterSet, "parameter-set", "", "Use predefined parameter set (common, critical)")
	sensitivityCmd.Flags().StringVar(&sensitivityAnalysisType, "analysis-type", "single", "Analysis type (single, multi, matrix)")

	rootCmd.AddCommand(sensitivityCmd)
}

func runSensitivityAnalysis(cmd *cobra.Command, args []string) {
	inputFile := args[0]

	// Load configuration
	parser := config.NewInputParser()
	config, err := parser.LoadFromFileWithRegulatory(inputFile, "regulatory.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	// Determine base scenario
	baseScenario := sensitivityBaseScenario
	if baseScenario == "" {
		if len(config.Scenarios) > 0 {
			baseScenario = config.Scenarios[0].Name
		} else {
			fmt.Fprintf(os.Stderr, "No scenarios found in configuration\n")
			os.Exit(1)
		}
	}

	// Create sensitivity analyzer
	analyzer := calculation.NewSensitivityAnalyzer()

	// Determine parameters to analyze
	var parameters []domain.SensitivityParameter

	if sensitivityParameterSet != "" {
		// Use predefined parameter set
		parameters = getPredefinedParameterSet(sensitivityParameterSet)
	} else if len(sensitivityParameter) > 0 {
		// Parse custom parameters
		parameters = parseCustomParameters(sensitivityParameter)
	} else {
		// Use single parameter with range
		if sensitivityRange == "" {
			fmt.Fprintf(os.Stderr, "Must specify either --parameter, --parameter-set, or --range\n")
			os.Exit(1)
		}

		paramName := "inflation_rate" // Default parameter
		if len(sensitivityParameter) > 0 {
			paramName = sensitivityParameter[0]
		}

		param, err := parseSingleParameter(paramName, sensitivityRange, sensitivitySteps)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing parameter: %v\n", err)
			os.Exit(1)
		}
		parameters = []domain.SensitivityParameter{param}
	}

	// Perform analysis
	var analysis interface{}
	var analysisErr error

	if len(parameters) == 1 {
		// Single parameter analysis
		analysis, analysisErr = analyzer.AnalyzeSingleParameter(config, parameters[0], baseScenario)
	} else if len(parameters) == 2 && sensitivityAnalysisType == "matrix" {
		// Matrix analysis
		analysis, analysisErr = analyzer.AnalyzeParameterMatrix(config, parameters[0], parameters[1], baseScenario)
	} else {
		// Multi-parameter analysis
		analysis, analysisErr = analyzer.AnalyzeMultipleParameters(config, parameters, baseScenario)
	}

	if analysisErr != nil {
		fmt.Fprintf(os.Stderr, "Error performing sensitivity analysis: %v\n", analysisErr)
		os.Exit(1)
	}

	// Format and output results
	formatter := output.NewSensitivityFormatter(sensitivityOutputFormat)
	output, err := formatter.FormatSensitivityAnalysis(analysis)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
		os.Exit(1)
	}

	fmt.Print(output)
}

func getPredefinedParameterSet(setName string) []domain.SensitivityParameter {
	switch setName {
	case "common":
		return domain.GetCommonParameters()
	case "critical":
		return []domain.SensitivityParameter{
			domain.InflationRateParam,
			domain.TSPReturnPostRetirementParam,
			domain.COLABateParam,
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown parameter set: %s\n", setName)
		os.Exit(1)
		return nil
	}
}

func parseCustomParameters(paramStrings []string) []domain.SensitivityParameter {
	parameters := make([]domain.SensitivityParameter, 0, len(paramStrings))

	for _, paramStr := range paramStrings {
		param, err := parseParameterString(paramStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing parameter '%s': %v\n", paramStr, err)
			os.Exit(1)
		}
		parameters = append(parameters, param)
	}

	return parameters
}

func parseParameterString(paramStr string) (domain.SensitivityParameter, error) {
	// Format: name:min-max:steps
	parts := strings.Split(paramStr, ":")
	if len(parts) != 3 {
		return domain.SensitivityParameter{}, fmt.Errorf("invalid parameter format: %s (expected name:min-max:steps)", paramStr)
	}

	name := parts[0]
	rangeStr := parts[1]
	stepsStr := parts[2]

	// Check if this is a common parameter
	for _, commonParam := range domain.GetCommonParameters() {
		if commonParam.Name == name {
			// Use the common parameter but override the range and steps
			minMax := strings.Split(rangeStr, "-")
			if len(minMax) != 2 {
				return domain.SensitivityParameter{}, fmt.Errorf("invalid range format: %s (expected min-max)", rangeStr)
			}

			minValue, err := parseDecimal(minMax[0])
			if err != nil {
				return domain.SensitivityParameter{}, fmt.Errorf("invalid min value: %v", err)
			}

			maxValue, err := parseDecimal(minMax[1])
			if err != nil {
				return domain.SensitivityParameter{}, fmt.Errorf("invalid max value: %v", err)
			}

			steps, err := strconv.Atoi(stepsStr)
			if err != nil {
				return domain.SensitivityParameter{}, fmt.Errorf("invalid steps value: %v", err)
			}

			// Return the common parameter with updated range and steps
			param := commonParam
			param.MinValue = minValue
			param.MaxValue = maxValue
			param.Steps = steps
			return param, nil
		}
	}

	// Parse range
	minMax := strings.Split(rangeStr, "-")
	if len(minMax) != 2 {
		return domain.SensitivityParameter{}, fmt.Errorf("invalid range format: %s (expected min-max)", rangeStr)
	}

	minValue, err := parseDecimal(minMax[0])
	if err != nil {
		return domain.SensitivityParameter{}, fmt.Errorf("invalid min value: %v", err)
	}

	maxValue, err := parseDecimal(minMax[1])
	if err != nil {
		return domain.SensitivityParameter{}, fmt.Errorf("invalid max value: %v", err)
	}

	// Parse steps
	steps, err := strconv.Atoi(stepsStr)
	if err != nil {
		return domain.SensitivityParameter{}, fmt.Errorf("invalid steps value: %v", err)
	}

	// Get the actual base value from the configuration
	var baseValue decimal.Decimal
	switch name {
	case "inflation_rate":
		baseValue = decimal.NewFromFloat(0.025)
	case "tsp_return_pre_retirement":
		baseValue = decimal.NewFromFloat(0.08)
	case "tsp_return_post_retirement":
		baseValue = decimal.NewFromFloat(0.065)
	case "cola_rate":
		baseValue = decimal.NewFromFloat(0.025)
	case "fehb_inflation":
		baseValue = decimal.NewFromFloat(0.065)
	default:
		// Calculate base value (midpoint) for unknown parameters
		baseValue = minValue.Add(maxValue).Div(decimal.NewFromInt(2))
	}

	// Get description from common parameters
	description := "Custom parameter"
	for _, commonParam := range domain.GetCommonParameters() {
		if commonParam.Name == name {
			description = commonParam.Description
			break
		}
	}

	return domain.SensitivityParameter{
		Name:        name,
		MinValue:    minValue,
		MaxValue:    maxValue,
		Steps:       steps,
		BaseValue:   baseValue,
		Unit:        "percent",
		Description: description,
	}, nil
}

func parseSingleParameter(name, rangeStr string, steps int) (domain.SensitivityParameter, error) {
	// Parse range
	minMax := strings.Split(rangeStr, "-")
	if len(minMax) != 2 {
		return domain.SensitivityParameter{}, fmt.Errorf("invalid range format: %s (expected min-max)", rangeStr)
	}

	minValue, err := parseDecimal(minMax[0])
	if err != nil {
		return domain.SensitivityParameter{}, fmt.Errorf("invalid min value: %v", err)
	}

	maxValue, err := parseDecimal(minMax[1])
	if err != nil {
		return domain.SensitivityParameter{}, fmt.Errorf("invalid max value: %v", err)
	}

	// Get the actual base value from the configuration
	var baseValue decimal.Decimal
	switch name {
	case "inflation_rate":
		baseValue = decimal.NewFromFloat(0.025)
	case "tsp_return_pre_retirement":
		baseValue = decimal.NewFromFloat(0.08)
	case "tsp_return_post_retirement":
		baseValue = decimal.NewFromFloat(0.065)
	case "cola_rate":
		baseValue = decimal.NewFromFloat(0.025)
	case "fehb_inflation":
		baseValue = decimal.NewFromFloat(0.065)
	default:
		// Calculate base value (midpoint) for unknown parameters
		baseValue = minValue.Add(maxValue).Div(decimal.NewFromInt(2))
	}

	// Get description from common parameters
	description := "Custom parameter"
	for _, commonParam := range domain.GetCommonParameters() {
		if commonParam.Name == name {
			description = commonParam.Description
			break
		}
	}

	return domain.SensitivityParameter{
		Name:        name,
		MinValue:    minValue,
		MaxValue:    maxValue,
		Steps:       steps,
		BaseValue:   baseValue,
		Unit:        "percent",
		Description: description,
	}, nil
}
