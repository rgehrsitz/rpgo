package main

import (
	"context"
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

var planRothCmd = &cobra.Command{
	Use:   "plan-roth [input-file]",
	Short: "Plan optimal Roth conversion strategy",
	Long: `Plan optimal Roth conversion strategy for a participant.

This command analyzes different Roth conversion strategies to find the optimal
approach based on your specified objective (minimize taxes, minimize IRMAA, etc.).

Examples:
  # Find optimal conversions to minimize lifetime tax
  ./rpgo plan-roth config.yaml --participant "Alice Johnson" --window 2028-2032 --target-bracket 22 --objective minimize_lifetime_tax

  # Find optimal conversions to minimize IRMAA surcharges
  ./rpgo plan-roth config.yaml --participant "Alice Johnson" --window 2028-2032 --target-bracket 22 --objective minimize_lifetime_irmaa

  # Find optimal conversions to maximize estate value
  ./rpgo plan-roth config.yaml --participant "Alice Johnson" --window 2028-2032 --target-bracket 22 --objective maximize_estate`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]

		// Parse flags
		participant, _ := cmd.Flags().GetString("participant")
		windowStr, _ := cmd.Flags().GetString("window")
		targetBracket, _ := cmd.Flags().GetInt("target-bracket")
		objectiveStr, _ := cmd.Flags().GetString("objective")
		format, _ := cmd.Flags().GetString("format")
		debug, _ := cmd.Flags().GetBool("debug")
		regulatoryConfig, _ := cmd.Flags().GetString("regulatory-config")

		// Parse window
		window, err := parseYearRange(windowStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing window: %v\n", err)
			os.Exit(1)
		}

		// Parse objective
		objective, err := parseOptimizationObjective(objectiveStr)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing objective: %v\n", err)
			os.Exit(1)
		}

		// Load configuration
		parser := config.NewInputParser()
		var config *domain.Configuration

		if regulatoryConfig != "" {
			config, err = parser.LoadFromFileWithRegulatory(inputFile, regulatoryConfig)
		} else {
			config, err = parser.LoadFromFile(inputFile)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Auto-detect participant if not specified
		if participant == "" {
			participant = autoDetectParticipant(config)
			if participant == "" {
				fmt.Fprintf(os.Stderr, "Error: No participant specified and none found in configuration\n")
				os.Exit(1)
			}
			if debug {
				fmt.Fprintf(os.Stderr, "Auto-detected participant: %s\n", participant)
			}
		}

		// Create calculation engine
		calcEngine := calculation.NewCalculationEngine()

		// Create Roth conversion planner
		planner := calculation.NewRothConversionPlanner(calcEngine)

		// Set up constraints
		constraints := domain.DefaultRothConversionConstraints(participant)

		// Parse additional constraint flags
		if minAmount, _ := cmd.Flags().GetString("min-amount"); minAmount != "" {
			if amount, err := parseDecimal(minAmount); err == nil {
				constraints.MinConversionAmount = amount
			}
		}
		if maxAmount, _ := cmd.Flags().GetString("max-amount"); maxAmount != "" {
			if amount, err := parseDecimal(maxAmount); err == nil {
				constraints.MaxConversionAmount = amount
			}
		}
		if maxTotal, _ := cmd.Flags().GetString("max-total"); maxTotal != "" {
			if amount, err := parseDecimal(maxTotal); err == nil {
				constraints.MaxTotalConversions = amount
			}
		}

		// Run Roth conversion planning
		ctx := context.Background()
		plan, err := planner.PlanRothConversions(ctx, config, participant, window, targetBracket, objective, constraints)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error planning Roth conversions: %v\n", err)
			os.Exit(1)
		}

		// Generate output
		formatter := output.NewFormatter(format)
		output, err := formatter.FormatRothConversionPlan(plan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(output)
	},
}

func init() {
	planRothCmd.Flags().StringP("participant", "p", "", "Participant name (auto-detected if not specified)")
	planRothCmd.Flags().StringP("window", "w", "2028-2032", "Conversion window (e.g., 2028-2032)")
	planRothCmd.Flags().IntP("target-bracket", "b", 22, "Target tax bracket (10, 12, 22, 24, 32, 35, 37)")
	planRothCmd.Flags().StringP("objective", "o", "minimize_combined", "Optimization objective (minimize_lifetime_tax, minimize_lifetime_irmaa, minimize_combined, maximize_estate, maximize_net_income)")
	planRothCmd.Flags().StringP("format", "f", "table", "Output format (table, json)")
	planRothCmd.Flags().BoolP("debug", "d", false, "Enable debug output")
	planRothCmd.Flags().StringP("regulatory-config", "r", "", "Path to regulatory configuration file")

	// Constraint flags
	planRothCmd.Flags().String("min-amount", "", "Minimum conversion amount per year")
	planRothCmd.Flags().String("max-amount", "", "Maximum conversion amount per year")
	planRothCmd.Flags().String("max-total", "", "Maximum total conversions across all years")

	rootCmd.AddCommand(planRothCmd)
}

// parseYearRange parses a year range string like "2028-2032"
func parseYearRange(windowStr string) (domain.YearRange, error) {
	parts := strings.Split(windowStr, "-")
	if len(parts) != 2 {
		return domain.YearRange{}, fmt.Errorf("invalid window format: %s (expected YYYY-YYYY)", windowStr)
	}

	start, err := strconv.Atoi(parts[0])
	if err != nil {
		return domain.YearRange{}, fmt.Errorf("invalid start year: %s", parts[0])
	}

	end, err := strconv.Atoi(parts[1])
	if err != nil {
		return domain.YearRange{}, fmt.Errorf("invalid end year: %s", parts[1])
	}

	if start >= end {
		return domain.YearRange{}, fmt.Errorf("start year must be before end year")
	}

	return domain.YearRange{Start: start, End: end}, nil
}

// parseOptimizationObjective parses an optimization objective string
func parseOptimizationObjective(objectiveStr string) (domain.OptimizationObjective, error) {
	switch objectiveStr {
	case "minimize_lifetime_tax":
		return domain.MinimizeLifetimeTax, nil
	case "minimize_lifetime_irmaa":
		return domain.MinimizeLifetimeIRMAA, nil
	case "minimize_combined":
		return domain.MinimizeCombined, nil
	case "maximize_estate":
		return domain.MaximizeEstate, nil
	case "maximize_net_income":
		return domain.MaximizeNetIncome, nil
	default:
		return domain.MinimizeCombined, fmt.Errorf("invalid objective: %s", objectiveStr)
	}
}

// autoDetectParticipant finds the first federal participant in the configuration
func autoDetectParticipant(config *domain.Configuration) string {
	if config.Household == nil {
		return ""
	}

	for _, participant := range config.Household.Participants {
		if participant.IsFederal {
			return participant.Name
		}
	}

	return ""
}

// parseDecimal parses a decimal string
func parseDecimal(s string) (decimal.Decimal, error) {
	return decimal.NewFromString(s)
}
