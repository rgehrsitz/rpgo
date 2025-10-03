package main

import (
	"context"
	"fmt"
	"os"

	"github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/rgehrsitz/rpgo/internal/config"
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/rgehrsitz/rpgo/internal/output"
	"github.com/spf13/cobra"
)

var analyzeSurvivorCmd = &cobra.Command{
	Use:   "analyze-survivor [input-file]",
	Short: "Analyze survivor viability when a spouse dies",
	Long: `Analyze the financial impact and viability when one spouse dies.

This command performs comprehensive survivor viability analysis including:
- Income replacement analysis
- Tax impact assessment
- TSP longevity changes
- IRMAA risk changes
- Life insurance needs calculation
- Recommendations for improvement

The analysis compares pre-death and post-death financial scenarios to identify
potential shortfalls and recommend strategies to ensure survivor financial security.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		inputFile := args[0]
		format, _ := cmd.Flags().GetString("format")
		regulatoryConfig, _ := cmd.Flags().GetString("regulatory-config")
		scenarioName, _ := cmd.Flags().GetString("scenario")
		deceasedParticipant, _ := cmd.Flags().GetString("deceased")
		survivorSpendingFactor, _ := cmd.Flags().GetString("survivor-spending-factor")
		analysisYears, _ := cmd.Flags().GetInt("analysis-years")

		// Load configuration
		parser := config.NewInputParser()
		var cfg *domain.Configuration
		var err error

		if regulatoryConfig != "" {
			cfg, err = parser.LoadFromFileWithRegulatory(inputFile, regulatoryConfig)
		} else {
			cfg, err = parser.LoadFromFile(inputFile)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Find scenario
		var targetScenario *domain.GenericScenario
		if scenarioName != "" {
			for i := range cfg.Scenarios {
				if cfg.Scenarios[i].Name == scenarioName {
					targetScenario = &cfg.Scenarios[i]
					break
				}
			}
			if targetScenario == nil {
				fmt.Fprintf(os.Stderr, "Error: Scenario '%s' not found\n", scenarioName)
				os.Exit(1)
			}
		} else {
			// Use first scenario with mortality
			for i := range cfg.Scenarios {
				if cfg.Scenarios[i].Mortality != nil {
					targetScenario = &cfg.Scenarios[i]
					break
				}
			}
			if targetScenario == nil {
				fmt.Fprintf(os.Stderr, "Error: No scenario with mortality found\n")
				os.Exit(1)
			}
		}

		// Create survivor configuration
		survivorConfig := domain.DefaultSurvivorScenarioConfig()
		survivorConfig.AnalysisYears = analysisYears

		// Parse survivor spending factor
		if survivorSpendingFactor != "" {
			if factor, err := parseDecimal(survivorSpendingFactor); err == nil {
				survivorConfig.SurvivorSpendingFactor = factor
			}
		}

		// Override deceased participant if specified
		if deceasedParticipant != "" {
			// Find the mortality spec and update it
			if targetScenario.Mortality == nil {
				targetScenario.Mortality = &domain.GenericScenarioMortality{
					Participants: make(map[string]*domain.MortalitySpec),
				}
			}
			if targetScenario.Mortality.Participants == nil {
				targetScenario.Mortality.Participants = make(map[string]*domain.MortalitySpec)
			}

			// Clear existing mortality specs
			for name := range targetScenario.Mortality.Participants {
				delete(targetScenario.Mortality.Participants, name)
			}

			// Add death spec for specified participant
			targetScenario.Mortality.Participants[deceasedParticipant] = &domain.MortalitySpec{
				DeathAge: &[]int{75}[0], // Default death at age 75
			}
		}

		// Create calculation engine and analyzer
		calcEngine := calculation.NewCalculationEngine()
		analyzer := calculation.NewSurvivorViabilityAnalyzer(calcEngine)

		// Perform analysis
		analysis, err := analyzer.AnalyzeSurvivorViability(context.Background(), cfg, targetScenario, survivorConfig)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error analyzing survivor viability: %v\n", err)
			os.Exit(1)
		}

		// Format and output results
		formatter := output.NewSurvivorViabilityFormatter(format)
		output, err := formatter.FormatSurvivorViabilityAnalysis(analysis)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
			os.Exit(1)
		}

		fmt.Print(output)
	},
}

func init() {
	analyzeSurvivorCmd.Flags().StringP("format", "f", "console", "Output format (console, json)")
	analyzeSurvivorCmd.Flags().StringP("regulatory-config", "r", "", "Path to regulatory configuration file")
	analyzeSurvivorCmd.Flags().StringP("scenario", "s", "", "Scenario name to analyze (uses first with mortality if not specified)")
	analyzeSurvivorCmd.Flags().StringP("deceased", "d", "", "Name of participant who dies (creates mortality scenario)")
	analyzeSurvivorCmd.Flags().StringP("survivor-spending-factor", "p", "0.75", "Survivor spending factor (0.75 = 75% of couple's spending)")
	analyzeSurvivorCmd.Flags().IntP("analysis-years", "y", 20, "Number of years to analyze post-death")

	rootCmd.AddCommand(analyzeSurvivorCmd)
}
