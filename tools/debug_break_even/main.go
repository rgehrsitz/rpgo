package main

import (
	"fmt"
	"os"

	calc "github.com/rgehrsitz/rpgo/internal/calculation"
	"github.com/rgehrsitz/rpgo/internal/config"
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage: debug_break_even <config-file>")
		return
	}
	f := os.Args[1]
	p := config.NewInputParser()
	cfg, err := p.LoadFromFile(f)
	if err != nil {
		panic(err)
	}
	engine := calc.NewCalculationEngineWithConfig(cfg.GlobalAssumptions.FederalRules)
	res, err := engine.RunScenarios(cfg)
	if err != nil {
		panic(err)
	}
	if len(res.Scenarios) < 1 {
		fmt.Println("no scenarios")
		return
	}

	// Find the minimum projection length across scenarios
	minLen := -1
	for _, s := range res.Scenarios {
		if minLen == -1 || len(s.Projection) < minLen {
			minLen = len(s.Projection)
		}
	}
	if minLen <= 0 {
		fmt.Println("no projection data")
		return
	}

	// Header
	header := "Index,Date,Year"
	for i := range res.Scenarios {
		header += fmt.Sprintf(",S%d_Salary,S%d_Pension,S%d_TSP,S%d_SS,S%d_Net", i+1, i+1, i+1, i+1, i+1)
	}
	fmt.Println(header)

	// Iterate years and print components
	for idx := 0; idx < minLen; idx++ {
		yearData := res.Scenarios[0].Projection[idx]
		row := fmt.Sprintf("%d,%s,%d", idx, yearData.Date.Format("2006-01-02"), yearData.Date.Year())
		for sidx := range res.Scenarios {
			projection := res.Scenarios[sidx].Projection[idx]
			row += fmt.Sprintf(",%s,%s,%s,%s,%s",
				sumValues(projection.Salaries).StringFixed(0),
				sumValues(projection.Pensions).StringFixed(0),
				sumValues(projection.TSPWithdrawals).StringFixed(0),
				sumValues(projection.SSBenefits).StringFixed(0),
				projection.NetIncome.StringFixed(0),
			)
		}
		fmt.Println(row)
	}

	// If at least two scenarios, compute cumulative diffs for first two
	if len(res.Scenarios) >= 2 {
		a := res.Scenarios[0].Projection
		b := res.Scenarios[1].Projection
		cumA := decimal.Zero
		cumB := decimal.Zero
		for i := 0; i < len(a) && i < len(b); i++ {
			cumA = cumA.Add(a[i].NetIncome)
			cumB = cumB.Add(b[i].NetIncome)
			diff := cumA.Sub(cumB)
			fmt.Printf("Cumulative Year %d: cumA=%s cumB=%s diff=%s\n",
				a[i].Date.Year(),
				cumA.StringFixed(0),
				cumB.StringFixed(0),
				diff.StringFixed(0),
			)
		}
		if be := calculateCumulativeBreakEven(a, b); be != nil {
			fmt.Printf("\nBreakEven Year: %d (cumA=%s cumB=%s diff=%s)\n",
				be.Year,
				be.CumulativeA.StringFixed(0),
				be.CumulativeB.StringFixed(0),
				be.Difference.StringFixed(0),
			)
		} else {
			fmt.Println("\nNo break-even point found within projection horizon")
		}
	}
}

func sumValues(values map[string]decimal.Decimal) decimal.Decimal {
	total := decimal.Zero
	for _, v := range values {
		total = total.Add(v)
	}
	return total
}

type cumulativeBreakEven struct {
	Year        int
	CumulativeA decimal.Decimal
	CumulativeB decimal.Decimal
	Difference  decimal.Decimal
}

func calculateCumulativeBreakEven(a, b []domain.AnnualCashFlow) *cumulativeBreakEven {
	length := len(a)
	if len(b) < length {
		length = len(b)
	}
	if length == 0 {
		return nil
	}

	prevDiff := decimal.Zero
	cumA := decimal.Zero
	cumB := decimal.Zero

	for i := 0; i < length; i++ {
		cumA = cumA.Add(a[i].NetIncome)
		cumB = cumB.Add(b[i].NetIncome)
		diff := cumA.Sub(cumB)
		if i == 0 {
			prevDiff = diff
			if diff.IsZero() {
				return &cumulativeBreakEven{Year: a[i].Date.Year(), CumulativeA: cumA, CumulativeB: cumB, Difference: diff}
			}
			continue
		}

		if diff.IsZero() || diff.Sign() != prevDiff.Sign() {
			return &cumulativeBreakEven{Year: a[i].Date.Year(), CumulativeA: cumA, CumulativeB: cumB, Difference: diff}
		}

		prevDiff = diff
	}

	return nil
}
