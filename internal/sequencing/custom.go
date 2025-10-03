package sequencing

import "github.com/shopspring/decimal"

// CustomStrategy executes withdrawals in a user-specified ordered list of sources.
// Valid source names: taxable, traditional, roth. If sequence invalid, falls back to standard.
type CustomStrategy struct {
	Sequence []string
}

func NewCustomStrategy(sequence []string) *CustomStrategy { return &CustomStrategy{Sequence: sequence} }

func (s *CustomStrategy) Name() string { return "custom" }

func (s *CustomStrategy) Plan(sources []WithdrawalSource, ctx StrategyContext) WithdrawalPlan {
	plan := WithdrawalPlan{Requested: ctx.NeedAmount, StrategyUsed: s.Name(), Allocations: []WithdrawalAllocation{}}
	remaining := ctx.NeedAmount

	// Validate sequence
	allowed := map[string]bool{"taxable": true, "traditional": true, "roth": true}
	seen := map[string]bool{}
	valid := true
	for _, name := range s.Sequence {
		if !allowed[name] || seen[name] {
			valid = false
			break
		}
		seen[name] = true
	}
	if !valid || len(s.Sequence) == 0 {
		plan.Notes = append(plan.Notes, "invalid or empty custom sequence - falling back to standard")
		std := NewStandardStrategy().Plan(sources, ctx)
		std.StrategyUsed = "custom->standard_fallback"
		return std
	}

	// Build lookup
	lookup := map[string]*WithdrawalSource{}
	for i := range sources {
		lookup[sources[i].Name] = &sources[i]
	}

	for _, name := range s.Sequence {
		if remaining.LessThanOrEqual(decimal.Zero) {
			break
		}
		src := lookup[name]
		if src == nil || src.Balance.LessThanOrEqual(decimal.Zero) {
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
