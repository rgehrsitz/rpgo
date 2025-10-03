package sequencing

import "github.com/shopspring/decimal"

// BracketFillStrategy attempts to fill a target marginal tax bracket with ordinary income
// from traditional sources, sourcing remainder from tax-free (Roth) then taxable if needed.
// Order inside logic: satisfy RMD first (if flagged) then fill bracket, then use Roth, then taxable.

type BracketFillStrategy struct{}

func NewBracketFillStrategy() *BracketFillStrategy { return &BracketFillStrategy{} }

func (s *BracketFillStrategy) Name() string { return "bracket_fill" }

func (s *BracketFillStrategy) Plan(sources []WithdrawalSource, ctx StrategyContext) WithdrawalPlan {
	plan := WithdrawalPlan{Requested: ctx.NeedAmount, StrategyUsed: s.Name(), Allocations: []WithdrawalAllocation{}}
	remaining := ctx.NeedAmount

	// Build lookup
	lookup := map[string]*WithdrawalSource{}
	for i := range sources {
		lookup[sources[i].Name] = &sources[i]
	}

	// 1. Handle RMD requirements first from traditional
	trad := lookup["traditional"]
	if trad != nil && trad.RMDRequired && trad.PendingRMD.GreaterThan(decimal.Zero) {
		withdraw := trad.PendingRMD
		if withdraw.GreaterThan(trad.Balance) {
			withdraw = trad.Balance
		}
		if withdraw.GreaterThan(remaining) {
			withdraw = remaining
		}
		if withdraw.GreaterThan(decimal.Zero) {
			alloc := WithdrawalAllocation{Source: "traditional", Gross: withdraw, OrdinaryPortion: withdraw, MAGIImpact: withdraw}
			plan.Allocations = append(plan.Allocations, alloc)
			plan.TotalSourced = plan.TotalSourced.Add(withdraw)
			plan.TraditionalUsed = plan.TraditionalUsed.Add(withdraw)
			plan.EstimatedOrdinaryIncome = plan.EstimatedOrdinaryIncome.Add(withdraw)
			plan.EstimatedMAGIImpact = plan.EstimatedMAGIImpact.Add(withdraw)
			trad.Balance = trad.Balance.Sub(withdraw)
			trad.PendingRMD = decimal.Zero
			remaining = remaining.Sub(withdraw)
		}
	}

	// 2. Fill bracket with additional traditional withdrawals if bracket target defined
	if remaining.GreaterThan(decimal.Zero) && ctx.TargetBracketPercent != nil && trad != nil && trad.Balance.GreaterThan(decimal.Zero) {
		// Determine bracket headroom: targetEdge - buffer - currentOrdinary
		// We approximate targetEdge using bracket edges slice by searching for the edge whose marginal rate == target (not stored directly).
		// For now treat TargetBracketPercent as cap on ordinary income total (CurrentOrdinaryIncome + fill <= pseudoEdge).
		// Simplified placeholder: assume bracketEdges[len-1] is the ceiling, we apply no complex mapping.
		var headroom decimal.Decimal
		if len(ctx.MarginalBracketEdges) > 0 {
			top := ctx.MarginalBracketEdges[len(ctx.MarginalBracketEdges)-1]
			headroom = top.Sub(ctx.CurrentOrdinaryIncome)
			if ctx.BracketBufferAmount != nil {
				buf := decimal.NewFromInt(int64(*ctx.BracketBufferAmount))
				headroom = headroom.Sub(buf)
			}
			if headroom.LessThan(decimal.Zero) {
				headroom = decimal.Zero
			}
		}
		// Withdraw min of remaining, headroom, trad balance
		fill := remaining
		if !headroom.IsZero() && headroom.LessThan(fill) {
			fill = headroom
		}
		if trad.Balance.LessThan(fill) {
			fill = trad.Balance
		}
		if fill.GreaterThan(decimal.Zero) {
			alloc := WithdrawalAllocation{Source: "traditional", Gross: fill, OrdinaryPortion: fill, MAGIImpact: fill}
			plan.Allocations = append(plan.Allocations, alloc)
			plan.TotalSourced = plan.TotalSourced.Add(fill)
			plan.TraditionalUsed = plan.TraditionalUsed.Add(fill)
			plan.EstimatedOrdinaryIncome = plan.EstimatedOrdinaryIncome.Add(fill)
			plan.EstimatedMAGIImpact = plan.EstimatedMAGIImpact.Add(fill)
			trad.Balance = trad.Balance.Sub(fill)
			remaining = remaining.Sub(fill)
			plan.BracketFilled = headroom.Sub(fill).LessThanOrEqual(decimal.Zero)
		}
	}

	// 3. Use Roth for remainder
	if remaining.GreaterThan(decimal.Zero) {
		if roth := lookup["roth"]; roth != nil && roth.Balance.GreaterThan(decimal.Zero) {
			withdraw := roth.Balance
			if withdraw.GreaterThan(remaining) {
				withdraw = remaining
			}
			alloc := WithdrawalAllocation{Source: "roth", Gross: withdraw, TaxFreePortion: withdraw}
			plan.Allocations = append(plan.Allocations, alloc)
			plan.TotalSourced = plan.TotalSourced.Add(withdraw)
			plan.RothUsed = plan.RothUsed.Add(withdraw)
			remaining = remaining.Sub(withdraw)
		}
	}

	// 4. Use taxable last
	if remaining.GreaterThan(decimal.Zero) {
		if taxable := lookup["taxable"]; taxable != nil && taxable.Balance.GreaterThan(decimal.Zero) {
			withdraw := taxable.Balance
			if withdraw.GreaterThan(remaining) {
				withdraw = remaining
			}
			gain := decimal.Zero
			if taxable.Balance.GreaterThan(decimal.Zero) {
				unrealized := taxable.Balance.Sub(taxable.Basis)
				if unrealized.LessThan(decimal.Zero) {
					unrealized = decimal.Zero
				}
				gainRatio := decimal.Zero
				if taxable.Balance.GreaterThan(decimal.Zero) {
					gainRatio = unrealized.Div(taxable.Balance)
				}
				gain = withdraw.Mul(gainRatio)
			}
			alloc := WithdrawalAllocation{Source: "taxable", Gross: withdraw, CapitalGainsPortion: gain, TaxFreePortion: withdraw.Sub(gain), MAGIImpact: gain}
			plan.Allocations = append(plan.Allocations, alloc)
			plan.TotalSourced = plan.TotalSourced.Add(withdraw)
			plan.TaxableUsed = plan.TaxableUsed.Add(withdraw)
			plan.EstimatedCapitalGains = plan.EstimatedCapitalGains.Add(gain)
			plan.EstimatedMAGIImpact = plan.EstimatedMAGIImpact.Add(gain)
			remaining = remaining.Sub(withdraw)
		}
	}

	plan.RemainingNeed = remaining
	if remaining.GreaterThan(decimal.Zero) {
		plan.Notes = append(plan.Notes, "insufficient balances to meet request")
	}
	return plan
}
