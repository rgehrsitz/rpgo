package sequencing

import "github.com/shopspring/decimal"

// StandardStrategy: taxable -> traditional -> roth
// Prioritizes spending taxable assets first (common for tax deferral) then traditional, preserving Roth for last.
type StandardStrategy struct{}

func NewStandardStrategy() *StandardStrategy { return &StandardStrategy{} }

func (s *StandardStrategy) Name() string { return "standard" }

func (s *StandardStrategy) Plan(sources []WithdrawalSource, ctx StrategyContext) WithdrawalPlan {
	plan := WithdrawalPlan{Requested: ctx.NeedAmount, StrategyUsed: s.Name(), Allocations: []WithdrawalAllocation{}}
	remaining := ctx.NeedAmount

	// Order sources by implied priority: taxable (1), traditional (2), roth (3)
	order := []string{"taxable", "traditional", "roth"}
	lookup := map[string]*WithdrawalSource{}
	for i := range sources {
		lookup[sources[i].Name] = &sources[i]
	}

	for _, name := range order {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		src, ok := lookup[name]
		if !ok || src.Balance.LessThanOrEqual(decimal.Zero) {
			continue
		}

		withdraw := src.Balance
		if withdraw.GreaterThan(remaining) {
			withdraw = remaining
		}

		alloc := WithdrawalAllocation{Source: name, Gross: withdraw}
		switch src.TaxTreatment {
		case OrdinaryIncome:
			alloc.OrdinaryPortion = withdraw
			alloc.MAGIImpact = withdraw
		case TaxFree:
			alloc.TaxFreePortion = withdraw
			alloc.MAGIImpact = decimal.Zero
		case CapitalGains:
			// Approximate gain portion = (Balance - Basis)/Balance * withdraw
			gain := decimal.Zero
			if src.Balance.GreaterThan(decimal.Zero) {
				unrealized := src.Balance.Sub(src.Basis)
				if unrealized.LessThan(decimal.Zero) {
					unrealized = decimal.Zero
				}
				gainRatio := decimal.Zero
				if src.Balance.GreaterThan(decimal.Zero) {
					gainRatio = unrealized.Div(src.Balance)
				}
				gain = withdraw.Mul(gainRatio)
			}
			alloc.CapitalGainsPortion = gain
			alloc.TaxFreePortion = withdraw.Sub(gain)
			alloc.MAGIImpact = gain // Only gains hit MAGI
		}

		plan.Allocations = append(plan.Allocations, alloc)
		plan.TotalSourced = plan.TotalSourced.Add(withdraw)
		remaining = remaining.Sub(withdraw)
		// Track convenience totals
		switch name {
		case "traditional":
			plan.TraditionalUsed = plan.TraditionalUsed.Add(withdraw)
		case "roth":
			plan.RothUsed = plan.RothUsed.Add(withdraw)
		case "taxable":
			plan.TaxableUsed = plan.TaxableUsed.Add(withdraw)
		}
		plan.EstimatedOrdinaryIncome = plan.EstimatedOrdinaryIncome.Add(alloc.OrdinaryPortion)
		plan.EstimatedCapitalGains = plan.EstimatedCapitalGains.Add(alloc.CapitalGainsPortion)
		plan.EstimatedMAGIImpact = plan.EstimatedMAGIImpact.Add(alloc.MAGIImpact)
	}

	plan.RemainingNeed = remaining
	if remaining.GreaterThan(decimal.Zero) {
		plan.Notes = append(plan.Notes, "insufficient balances to meet request")
	}
	return plan
}
