package sequencing

import (
	"github.com/shopspring/decimal"
)

// TaxTreatment represents tax characteristics of a withdrawal source
// Ordinary: fully taxable as ordinary income (traditional, pensions)
// TaxFree: no current year tax impact (Roth principal / qualified dist.)
// CapitalGains: only gains portion taxed (approx via basis tracking)
type TaxTreatment int

const (
	TaxFree TaxTreatment = iota
	OrdinaryIncome
	CapitalGains
)

func (tt TaxTreatment) String() string {
	switch tt {
	case TaxFree:
		return "tax_free"
	case OrdinaryIncome:
		return "ordinary"
	case CapitalGains:
		return "capital_gains"
	default:
		return "unknown"
	}
}

// WithdrawalSource represents an available pool for withdrawals
// Name: semantic identifier (taxable | traditional | roth)
// Balance: current available balance
// Basis: for taxable accounts to approximate gains (optional / zero for non-taxable)
// TaxTreatment: how withdrawals impact taxes/MAGI
// RMDRequired: if true, must satisfy required distribution (traditional accounts in RMD years)
// Priority: used for ordered/custom strategies
// PendingRMD: amount that must be withdrawn before other logic (set externally per year)
type WithdrawalSource struct {
	Name         string
	Balance      decimal.Decimal
	Basis        decimal.Decimal
	TaxTreatment TaxTreatment
	RMDRequired  bool
	Priority     int
	PendingRMD   decimal.Decimal
}

// WithdrawalAllocation captures actual withdrawal from a source and its tax/MAGI decomposition
// Gross: total dollars withdrawn from the source
// OrdinaryPortion: amount treated as ordinary income (for taxes & MAGI)
// CapitalGainsPortion: capital gains recognized (taxable portion of taxable account withdrawal)
// TaxFreePortion: tax-free amount (Roth, basis recovery, etc.)
// MAGIImpact: portion contributing to MAGI (usually ordinary + capital gains)
type WithdrawalAllocation struct {
	Source              string
	Gross               decimal.Decimal
	OrdinaryPortion     decimal.Decimal
	CapitalGainsPortion decimal.Decimal
	TaxFreePortion      decimal.Decimal
	MAGIImpact          decimal.Decimal
}

// WithdrawalPlan aggregates the full plan for meeting a target amount
// Requested: target net/gross withdrawal requested (strategy interprets as amount to source from accounts)
// Allocations: slice of per-source allocations
// TotalSourced: sum of Gross across allocations
// RemainingNeed: unmet portion if insufficient balances
// EstimatedOrdinaryIncome: sum of ordinary portions
// EstimatedCapitalGains: sum of capital gains portions
// EstimatedMAGIImpact: total MAGI impact
// Notes: strategy-specific notes or warnings
// StrategyUsed: resolved strategy after fallbacks
// BracketFilled: indicates if bracket-fill target reached (for bracket_fill strategy)
// TraditionalUsed / RothUsed / TaxableUsed: convenience totals
// RMDSatisfied: whether all pending RMD amounts were satisfied
// BracketTarget / BracketBuffer: echo inputs for transparency
type WithdrawalPlan struct {
	Requested               decimal.Decimal
	Allocations             []WithdrawalAllocation
	TotalSourced            decimal.Decimal
	RemainingNeed           decimal.Decimal
	EstimatedOrdinaryIncome decimal.Decimal
	EstimatedCapitalGains   decimal.Decimal
	EstimatedMAGIImpact     decimal.Decimal
	Notes                   []string
	StrategyUsed            string
	BracketFilled           bool
	TraditionalUsed         decimal.Decimal
	RothUsed                decimal.Decimal
	TaxableUsed             decimal.Decimal
	RMDSatisfied            bool
	BracketTarget           *int
	BracketBuffer           *int
}

// StrategyContext provides inputs required by sequencing strategies
// NeedAmount: amount strategy should attempt to source (gross concept for now)
// CurrentOrdinaryIncome: income already accumulated before sequencing (for bracket logic)
// MarginalBracketEdges: ascending slice of bracket upper bounds (for bracket_fill)
// TargetBracketPercent: desired marginal bracket (e.g. 22) for bracket_fill
// BracketBufferAmount: stay this many dollars below bracket edge
// IsRMDYear: whether RMD rules apply to traditional accounts this year
// IRMAAThreshold: first IRMAA threshold (future IRMAA-aware logic)
// MAGICurrent: MAGI accumulated so far before withdrawals.
type StrategyContext struct {
	NeedAmount            decimal.Decimal
	CurrentOrdinaryIncome decimal.Decimal
	MAGICurrent           decimal.Decimal
	MarginalBracketEdges  []decimal.Decimal
	TargetBracketPercent  *int
	BracketBufferAmount   *int
	IsRMDYear             bool
	IRMAAThreshold        *decimal.Decimal
}

// SequencingStrategy defines interface for all withdrawal sequencing algorithms
type SequencingStrategy interface {
	Name() string
	Plan(sources []WithdrawalSource, ctx StrategyContext) WithdrawalPlan
}
