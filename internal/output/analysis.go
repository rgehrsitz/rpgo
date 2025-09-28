package output

import (
	"sort"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// Recommendation encapsulates the selection result of the best scenario.
type Recommendation struct {
	ScenarioName       string
	FirstRetirementNet decimal.Decimal
	NetIncomeChange    decimal.Decimal
	PercentageChange   decimal.Decimal
}

// AnalyzeScenarios determines the scenario with highest first-year retirement net income.
// Extracted from embedded console logic for testability.
func AnalyzeScenarios(results *domain.ScenarioComparison) Recommendation {
	baseline := results.BaselineNetIncome
	type ranked struct {
		name   string
		income decimal.Decimal
	}
	var ranks []ranked
	for _, sc := range results.Scenarios {
		var yrIncome decimal.Decimal
		for _, y := range sc.Projection {
			if y.IsRetired {
				yrIncome = y.NetIncome
				break
			}
		}
		ranks = append(ranks, ranked{sc.Name, yrIncome})
	}
	if len(ranks) == 0 {
		return Recommendation{}
	}
	sort.Slice(ranks, func(i, j int) bool { return ranks[i].income.GreaterThan(ranks[j].income) })
	best := ranks[0]
	delta := best.income.Sub(baseline)
	pct := decimal.Zero
	if !baseline.IsZero() {
		pct = delta.Div(baseline).Mul(decimal.NewFromInt(100))
	}
	return Recommendation{ScenarioName: best.name, FirstRetirementNet: best.income, NetIncomeChange: delta, PercentageChange: pct}
}
