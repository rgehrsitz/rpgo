// Debug test to compare 0% vs 4% TSP withdrawal scenarios
package main

import (
	"fmt"
	"log"

	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/internal/config"
)

func main() {
	// Load 4% scenario
	config4pct, err := config.LoadConfiguration("comprehensive_single_4pct.yaml")
	if err != nil {
		log.Fatal("Error loading 4% config:", err)
	}

	// Load 0% scenario
	config0pct, err := config.LoadConfiguration("comprehensive_single_0pct.yaml")
	if err != nil {
		log.Fatal("Error loading 0% config:", err)
	}

	// Create calculation engine
	engine := calculation.NewCalculationEngine()

	// Run both scenarios
	scenario4pct := &config4pct.Scenarios[0]
	scenario0pct := &config0pct.Scenarios[0]

	result4pct, err := engine.RunScenario(config4pct, scenario4pct)
	if err != nil {
		log.Fatal("Error running 4% scenario:", err)
	}

	result0pct, err := engine.RunScenario(config0pct, scenario0pct)
	if err != nil {
		log.Fatal("Error running 0% scenario:", err)
	}

	// Compare first 10 years
	fmt.Println("Year-by-Year Comparison (First 10 Years):")
	fmt.Println("Year | 4% TSP Withdrawal                    | 0% TSP Withdrawal")
	fmt.Println("     | Gross    | TSP W/D | Net      | Gross    | TSP W/D | Net")
	fmt.Println("-----|----------|---------|----------|----------|---------|----------")

	for i := 0; i < 10 && i < len(result4pct.Projection) && i < len(result0pct.Projection); i++ {
		year4 := result4pct.Projection[i]
		year0 := result0pct.Projection[i]

		tspWithdrawal4 := year4.TSPWithdrawalRobert.Add(year4.TSPWithdrawalDawn)
		tspWithdrawal0 := year0.TSPWithdrawalRobert.Add(year0.TSPWithdrawalDawn)

		fmt.Printf("%4d | $%7s | $%6s | $%7s | $%7s | $%6s | $%7s\n",
			year4.Year,
			year4.TotalGrossIncome.StringFixed(0),
			tspWithdrawal4.StringFixed(0),
			year4.NetIncome.StringFixed(0),
			year0.TotalGrossIncome.StringFixed(0),
			tspWithdrawal0.StringFixed(0),
			year0.NetIncome.StringFixed(0),
		)
	}

	// Calculate averages
	var total4, total0 int64
	var count int
	for i := 0; i < len(result4pct.Projection) && i < len(result0pct.Projection); i++ {
		total4 += result4pct.Projection[i].NetIncome.IntPart()
		total0 += result0pct.Projection[i].NetIncome.IntPart()
		count++
	}

	avg4 := total4 / int64(count)
	avg0 := total0 / int64(count)

	fmt.Println("\nAverage Net Income over", count, "years:")
	fmt.Printf("4%% TSP Withdrawal: $%d\n", avg4)
	fmt.Printf("0%% TSP Withdrawal: $%d\n", avg0)
	fmt.Printf("Difference: $%d (0%% - 4%%)\n", avg0-avg4)
}
