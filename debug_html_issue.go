package main

import (
	"fmt"
	"log"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
	"github.com/shopspring/decimal"
)

func main() {
	// Reproduce the exact CLI sequence
	parser := config.NewInputParser()
	baseConfig, err := parser.LoadFromFile("example_config_comprehensive.yaml")
	if err != nil {
		log.Fatal(err)
	}

	hdm := calculation.NewHistoricalDataManager("./data")
	if err := hdm.LoadAllData(); err != nil {
		log.Fatal(err)
	}

	engine := calculation.NewFERSMonteCarloEngine(baseConfig, hdm)

	mcConfig := calculation.FERSMonteCarloConfig{
		BaseConfig:     baseConfig,
		NumSimulations: 10,
		UseHistorical:  true,
		Seed:           12345,
	}

	result, err := engine.RunFERSMonteCarlo(mcConfig)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("=== ANALYZING MONTE CARLO RESULT STRUCTURE ===\n")
	fmt.Printf("Number of simulations: %d\n", len(result.Simulations))

	// Check first few simulations in detail
	for i := 0; i < min(3, len(result.Simulations)); i++ {
		sim := result.Simulations[i]
		fmt.Printf("\nSimulation %d:\n", i+1)
		fmt.Printf("  Number of scenario results: %d\n", len(sim.ScenarioResults))

		if len(sim.ScenarioResults) > 0 {
			scenario := sim.ScenarioResults[0]
			fmt.Printf("  Scenario name: %s\n", scenario.Name)
			fmt.Printf("  Projection length: %d years\n", len(scenario.Projection))

			// Check first few years of this simulation
			for year := 0; year < min(3, len(scenario.Projection)); year++ {
				yearData := scenario.Projection[year]
				totalTSP := yearData.TSPBalanceRobert.Add(yearData.TSPBalanceDawn)
				fmt.Printf("    Year %d: TSP=$%s, NetIncome=$%s\n",
					year+1, totalTSP.StringFixed(0), yearData.NetIncome.StringFixed(0))
			}
		}
	}

	// Now let's manually trace what the HTML generation SHOULD do
	fmt.Printf("\n=== MANUAL TIME SERIES EXTRACTION ===\n")

	if len(result.Simulations) == 0 {
		fmt.Printf("❌ No simulations in result!\n")
		return
	}

	firstSim := result.Simulations[0]
	if len(firstSim.ScenarioResults) == 0 {
		fmt.Printf("❌ No scenario results in first simulation!\n")
		return
	}

	projectionLength := len(firstSim.ScenarioResults[0].Projection)
	fmt.Printf("Projection length: %d years\n", projectionLength)

	// Extract TSP balances for each year across all simulations (like the HTML code does)
	yearlyTSPBalances := make([][]decimal.Decimal, projectionLength)

	for _, sim := range result.Simulations {
		if len(sim.ScenarioResults) > 0 {
			scenario := sim.ScenarioResults[0]
			for yearIdx, yearData := range scenario.Projection {
				if yearIdx < projectionLength {
					if yearlyTSPBalances[yearIdx] == nil {
						yearlyTSPBalances[yearIdx] = make([]decimal.Decimal, 0, len(result.Simulations))
					}
					totalTSP := yearData.TSPBalanceRobert.Add(yearData.TSPBalanceDawn)
					yearlyTSPBalances[yearIdx] = append(yearlyTSPBalances[yearIdx], totalTSP)
				}
			}
		}
	}

	// Check what we extracted for year 6 (index 5)
	year6Index := 5
	if year6Index < len(yearlyTSPBalances) && len(yearlyTSPBalances[year6Index]) > 0 {
		fmt.Printf("\nYear 6 TSP balances across all simulations:\n")
		for i, balance := range yearlyTSPBalances[year6Index] {
			fmt.Printf("  Sim %d: $%s\n", i+1, balance.StringFixed(0))
		}

		// Check if they're all identical
		allSame := true
		if len(yearlyTSPBalances[year6Index]) > 1 {
			first := yearlyTSPBalances[year6Index][0]
			for _, balance := range yearlyTSPBalances[year6Index][1:] {
				if !balance.Equal(first) {
					allSame = false
					break
				}
			}
		}

		if allSame {
			fmt.Printf("❌ PROBLEM: All Year 6 TSP values are identical: $%s\n", yearlyTSPBalances[year6Index][0].StringFixed(0))
			fmt.Printf("This explains why HTML percentiles are the same!\n")
		} else {
			fmt.Printf("✅ Year 6 TSP values show variation as expected\n")
		}
	}

	// Let's also check if there's an issue with the scenario results vs individual simulations
	fmt.Printf("\n=== CHECKING FOR DATA CORRUPTION ===\n")

	// Compare individual simulation TSP values vs what we extract for HTML
	for i := 0; i < min(3, len(result.Simulations)); i++ {
		sim := result.Simulations[i]
		if len(sim.ScenarioResults) > 0 && len(sim.ScenarioResults[0].Projection) > year6Index {
			directTSP := sim.ScenarioResults[0].Projection[year6Index].TSPBalanceRobert.Add(sim.ScenarioResults[0].Projection[year6Index].TSPBalanceDawn)
			extractedTSP := yearlyTSPBalances[year6Index][i]

			fmt.Printf("Sim %d Year 6: Direct=$%s, Extracted=$%s, Match=%v\n",
				i+1, directTSP.StringFixed(0), extractedTSP.StringFixed(0), directTSP.Equal(extractedTSP))
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
