package sequencing

import (
	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// CreateStrategy creates a sequencing strategy based on the configuration
func CreateStrategy(config *domain.WithdrawalSequencingConfig) SequencingStrategy {
	if config == nil {
		return NewStandardStrategy()
	}

	switch config.Strategy {
	case "standard":
		return NewStandardStrategy()
	case "tax_efficient":
		return NewTaxEfficientStrategy()
	case "bracket_fill":
		return NewBracketFillStrategy()
	case "custom":
		return NewCustomStrategy(config.CustomSequence)
	default:
		// Fallback to standard if unknown strategy
		return NewStandardStrategy()
	}
}

// CreateStrategyContext creates a StrategyContext from the current projection state
func CreateStrategyContext(
	needAmount decimal.Decimal,
	currentOrdinaryIncome decimal.Decimal,
	magiCurrent decimal.Decimal,
	isRMDYear bool,
	config *domain.WithdrawalSequencingConfig,
) StrategyContext {
	ctx := StrategyContext{
		NeedAmount:            needAmount,
		CurrentOrdinaryIncome: currentOrdinaryIncome,
		MAGICurrent:           magiCurrent,
		IsRMDYear:             isRMDYear,
	}

	// Add bracket-fill specific parameters if strategy is bracket_fill
	if config != nil && config.Strategy == "bracket_fill" {
		ctx.TargetBracketPercent = config.TargetBracket
		ctx.BracketBufferAmount = config.BracketBuffer

		// TODO: Add marginal bracket edges calculation
		// For now, we'll use placeholder values
		ctx.MarginalBracketEdges = []decimal.Decimal{
			decimal.NewFromInt(11000),  // 10% bracket
			decimal.NewFromInt(44725),  // 12% bracket
			decimal.NewFromInt(95375),  // 22% bracket
			decimal.NewFromInt(182050), // 24% bracket
			decimal.NewFromInt(231250), // 32% bracket
			decimal.NewFromInt(578125), // 35% bracket
		}
	}

	return ctx
}

// CreateWithdrawalSources creates WithdrawalSource slice from participant data
func CreateWithdrawalSources(
	participant *domain.Participant,
	traditionalBalance decimal.Decimal,
	rothBalance decimal.Decimal,
	isRMDYear bool,
	rmdAmount decimal.Decimal,
) []WithdrawalSource {
	sources := []WithdrawalSource{}

	// Add taxable account if present
	if participant.TaxableAccountBalance != nil && participant.TaxableAccountBalance.GreaterThan(decimal.Zero) {
		basis := decimal.Zero
		if participant.TaxableAccountBasis != nil {
			basis = *participant.TaxableAccountBasis
		}
		sources = append(sources, WithdrawalSource{
			Name:         "taxable",
			Balance:      *participant.TaxableAccountBalance,
			Basis:        basis,
			TaxTreatment: CapitalGains,
			RMDRequired:  false,
			Priority:     1,
		})
	}

	// Add traditional TSP
	if traditionalBalance.GreaterThan(decimal.Zero) {
		pendingRMD := decimal.Zero
		if isRMDYear {
			pendingRMD = rmdAmount
		}
		sources = append(sources, WithdrawalSource{
			Name:         "traditional",
			Balance:      traditionalBalance,
			Basis:        decimal.Zero, // Traditional has no basis
			TaxTreatment: OrdinaryIncome,
			RMDRequired:  isRMDYear,
			Priority:     2,
			PendingRMD:   pendingRMD,
		})
	}

	// Add Roth TSP
	if rothBalance.GreaterThan(decimal.Zero) {
		sources = append(sources, WithdrawalSource{
			Name:         "roth",
			Balance:      rothBalance,
			Basis:        decimal.Zero, // Roth has no basis for qualified distributions
			TaxTreatment: TaxFree,
			RMDRequired:  false,
			Priority:     3,
		})
	}

	return sources
}
