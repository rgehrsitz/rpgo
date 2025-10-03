package sequencing

import "github.com/shopspring/decimal"

// TaxEfficientStrategy: roth -> traditional -> taxable (variant rationale: reduce future RMDs while still using taxable last)
// However common advice often spends taxable first; this variant intentionally prioritizes Roth to illustrate contrast.
// We'll adapt: actual definition from planning doc: Roth first then Traditional; taxable not modeled fully yet for RMD impact difference.
// Here we keep order: roth -> traditional -> taxable.

type TaxEfficientStrategy struct{}

func NewTaxEfficientStrategy() *TaxEfficientStrategy { return &TaxEfficientStrategy{} }

func (s *TaxEfficientStrategy) Name() string { return "tax_efficient" }

func (s *TaxEfficientStrategy) Plan(sources []WithdrawalSource, ctx StrategyContext) WithdrawalPlan {
	plan := WithdrawalPlan{Requested: ctx.NeedAmount, StrategyUsed: s.Name(), Allocations: []WithdrawalAllocation{}}
	remaining := ctx.NeedAmount

	order := []string{"roth", "traditional", "taxable"}
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
		case CapitalGains:
			// Gains approximation same as standard
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
			alloc.MAGIImpact = gain
		}

		plan.Allocations = append(plan.Allocations, alloc)
		plan.TotalSourced = plan.TotalSourced.Add(withdraw)
		remaining = remaining.Sub(withdraw)
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
