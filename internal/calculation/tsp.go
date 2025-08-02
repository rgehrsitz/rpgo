package calculation

import (
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

// TSPWithdrawalStrategy defines the interface for withdrawal strategies
type TSPWithdrawalStrategy interface {
	CalculateWithdrawal(currentBalance decimal.Decimal, year int, targetIncome decimal.Decimal, age int, isRMDYear bool, rmdAmount decimal.Decimal) decimal.Decimal
	GetStrategyName() string
}

// FourPercentRule implements the 4% rule (adjusted for inflation)
type FourPercentRule struct {
	InitialWithdrawalPercent decimal.Decimal
	InflationRate            decimal.Decimal
	InitialBalance           decimal.Decimal
	FirstWithdrawalAmount    decimal.Decimal
}

// NewFourPercentRule creates a new FourPercentRule strategy
func NewFourPercentRule(initialBalance decimal.Decimal, inflationRate decimal.Decimal) *FourPercentRule {
	initialWithdrawal := initialBalance.Mul(decimal.NewFromFloat(0.04))
	return &FourPercentRule{
		InitialWithdrawalPercent: decimal.NewFromFloat(0.04),
		InflationRate:            inflationRate,
		InitialBalance:           initialBalance,
		FirstWithdrawalAmount:    initialWithdrawal,
	}
}

// CalculateWithdrawal calculates the withdrawal amount for a given year
func (fpr *FourPercentRule) CalculateWithdrawal(currentBalance decimal.Decimal, year int, targetIncome decimal.Decimal, age int, isRMDYear bool, rmdAmount decimal.Decimal) decimal.Decimal {
	var withdrawal decimal.Decimal

	if year == 1 {
		withdrawal = fpr.FirstWithdrawalAmount
	} else {
		// Inflate previous year's withdrawal
		inflationFactor := decimal.NewFromFloat(1).Add(fpr.InflationRate)
		withdrawal = fpr.FirstWithdrawalAmount.Mul(inflationFactor.Pow(decimal.NewFromInt(int64(year - 1))))
	}

	// Handle RMD (Required Minimum Distribution)
	if isRMDYear && withdrawal.LessThan(rmdAmount) {
		return rmdAmount
	}

	// Ensure withdrawal doesn't exceed available balance
	if withdrawal.GreaterThan(currentBalance) {
		return currentBalance
	}

	return withdrawal
}

// GetStrategyName returns the name of this strategy
func (fpr *FourPercentRule) GetStrategyName() string {
	return "4_percent_rule"
}

// NeedBasedWithdrawal implements a strategy to withdraw based on a target monthly amount
type NeedBasedWithdrawal struct {
	TargetMonthlyWithdrawal decimal.Decimal
}

// NewNeedBasedWithdrawal creates a new NeedBasedWithdrawal strategy
func NewNeedBasedWithdrawal(targetMonthly decimal.Decimal) *NeedBasedWithdrawal {
	return &NeedBasedWithdrawal{
		TargetMonthlyWithdrawal: targetMonthly,
	}
}

// CalculateWithdrawal calculates the withdrawal amount based on target income
func (nbw *NeedBasedWithdrawal) CalculateWithdrawal(currentBalance decimal.Decimal, year int, targetIncome decimal.Decimal, age int, isRMDYear bool, rmdAmount decimal.Decimal) decimal.Decimal {
	// Calculate annual target withdrawal (this is the amount we want to withdraw)
	annualTarget := nbw.TargetMonthlyWithdrawal.Mul(decimal.NewFromInt(12))

	// The withdrawal should be the target amount, not the gap
	withdrawal := annualTarget

	// Ensure withdrawal is not negative
	if withdrawal.LessThan(decimal.Zero) {
		withdrawal = decimal.Zero
	}

	// Handle RMD
	if isRMDYear && withdrawal.LessThan(rmdAmount) {
		withdrawal = rmdAmount
	}

	// Ensure withdrawal doesn't exceed available balance
	if withdrawal.GreaterThan(currentBalance) {
		return currentBalance
	}

	return withdrawal
}

// GetStrategyName returns the name of this strategy
func (nbw *NeedBasedWithdrawal) GetStrategyName() string {
	return "need_based"
}

// VariablePercentageWithdrawal implements a strategy with a configurable percentage rate of current balance
type VariablePercentageWithdrawal struct {
	WithdrawalRate decimal.Decimal
}

// NewVariablePercentageWithdrawal creates a new VariablePercentageWithdrawal strategy
func NewVariablePercentageWithdrawal(initialBalance decimal.Decimal, withdrawalRate decimal.Decimal, inflationRate decimal.Decimal) *VariablePercentageWithdrawal {
	return &VariablePercentageWithdrawal{
		WithdrawalRate: withdrawalRate,
	}
}

// CalculateWithdrawal calculates the withdrawal amount for a given year using the variable percentage of current balance
func (vpw *VariablePercentageWithdrawal) CalculateWithdrawal(currentBalance decimal.Decimal, year int, targetIncome decimal.Decimal, age int, isRMDYear bool, rmdAmount decimal.Decimal) decimal.Decimal {
	// Calculate withdrawal as percentage of current balance (true percentage-based withdrawal)
	withdrawal := currentBalance.Mul(vpw.WithdrawalRate)

	// Handle RMD (Required Minimum Distribution)
	if isRMDYear && withdrawal.LessThan(rmdAmount) {
		return rmdAmount
	}

	// Ensure withdrawal doesn't exceed available balance
	if withdrawal.GreaterThan(currentBalance) {
		return currentBalance
	}

	return withdrawal
}

// GetStrategyName returns the name of this strategy
func (vpw *VariablePercentageWithdrawal) GetStrategyName() string {
	return "variable_percentage"
}

// RMDCalculator calculates Required Minimum Distributions
type RMDCalculator struct {
	BirthYear int
}

// NewRMDCalculator creates a new RMD calculator
func NewRMDCalculator(birthYear int) *RMDCalculator {
	return &RMDCalculator{
		BirthYear: birthYear,
	}
}

// GetRMDAge returns the age when RMDs start for this birth year
func (rmd *RMDCalculator) GetRMDAge() int {
	return dateutil.GetRMDAge(rmd.BirthYear)
}

// CalculateRMD calculates the Required Minimum Distribution for a given age and balance
func (rmd *RMDCalculator) CalculateRMD(traditionalBalance decimal.Decimal, age int) decimal.Decimal {
	if age < rmd.GetRMDAge() {
		return decimal.Zero
	}

	// IRS Uniform Lifetime Table (simplified version)
	distributionPeriods := map[int]decimal.Decimal{
		72:  decimal.NewFromFloat(27.4),
		73:  decimal.NewFromFloat(26.5),
		74:  decimal.NewFromFloat(25.5),
		75:  decimal.NewFromFloat(24.6),
		76:  decimal.NewFromFloat(23.7),
		77:  decimal.NewFromFloat(22.9),
		78:  decimal.NewFromFloat(22.0),
		79:  decimal.NewFromFloat(21.1),
		80:  decimal.NewFromFloat(20.2),
		81:  decimal.NewFromFloat(19.4),
		82:  decimal.NewFromFloat(18.5),
		83:  decimal.NewFromFloat(17.7),
		84:  decimal.NewFromFloat(16.8),
		85:  decimal.NewFromFloat(16.0),
		86:  decimal.NewFromFloat(15.2),
		87:  decimal.NewFromFloat(14.4),
		88:  decimal.NewFromFloat(13.7),
		89:  decimal.NewFromFloat(12.9),
		90:  decimal.NewFromFloat(12.2),
		91:  decimal.NewFromFloat(11.5),
		92:  decimal.NewFromFloat(10.8),
		93:  decimal.NewFromFloat(10.1),
		94:  decimal.NewFromFloat(9.5),
		95:  decimal.NewFromFloat(8.9),
		96:  decimal.NewFromFloat(8.4),
		97:  decimal.NewFromFloat(7.8),
		98:  decimal.NewFromFloat(7.3),
		99:  decimal.NewFromFloat(6.8),
		100: decimal.NewFromFloat(6.4),
	}

	if period, exists := distributionPeriods[age]; exists {
		return traditionalBalance.Div(period)
	}

	// For ages beyond 100, use a reasonable estimate
	if age > 100 {
		return traditionalBalance.Div(decimal.NewFromFloat(6.0))
	}

	return decimal.Zero
}

// SimulateTSPGrowthPreRetirement simulates TSP growth before retirement
func SimulateTSPGrowthPreRetirement(initialBalance decimal.Decimal, annualContributions decimal.Decimal, annualReturn decimal.Decimal, years int) decimal.Decimal {
	currentBalance := initialBalance
	for i := 0; i < years; i++ {
		currentBalance = currentBalance.Add(annualContributions).Mul(decimal.NewFromFloat(1.0).Add(annualReturn))
	}
	return currentBalance
}

// ProjectTSP projects TSP balances and withdrawals over multiple years
func ProjectTSP(initialBalance decimal.Decimal, strategy TSPWithdrawalStrategy, returnRate decimal.Decimal, years int, birthYear int, targetIncome []decimal.Decimal) []domain.TSPProjection {
	projections := make([]domain.TSPProjection, years)
	currentBalance := initialBalance
	rmdCalc := NewRMDCalculator(birthYear)

	for year := 1; year <= years; year++ {
		// Calculate growth
		growth := currentBalance.Mul(returnRate)

		// Determine if this is an RMD year
		age := birthYear + year - 1
		isRMDYear := age >= rmdCalc.GetRMDAge()
		rmdAmount := rmdCalc.CalculateRMD(currentBalance, age)

		// Calculate withdrawal
		var targetIncomeForYear decimal.Decimal
		if year <= len(targetIncome) {
			targetIncomeForYear = targetIncome[year-1]
		}

		withdrawal := strategy.CalculateWithdrawal(currentBalance, year, targetIncomeForYear, age, isRMDYear, rmdAmount)

		// Ensure withdrawal doesn't exceed balance plus growth
		if withdrawal.GreaterThan(currentBalance.Add(growth)) {
			withdrawal = currentBalance.Add(growth)
		}

		// Calculate ending balance
		endingBalance := currentBalance.Add(growth).Sub(withdrawal)

		projections[year-1] = domain.TSPProjection{
			Year:             year,
			BeginningBalance: currentBalance,
			Growth:           growth,
			Withdrawal:       withdrawal,
			RMD:              rmdAmount,
			EndingBalance:    endingBalance,
		}

		currentBalance = endingBalance
	}

	return projections
}

// ProjectTSPWithTraditionalRoth projects TSP balances separately for Traditional and Roth accounts
func ProjectTSPWithTraditionalRoth(initialTraditional decimal.Decimal, initialRoth decimal.Decimal, strategy TSPWithdrawalStrategy, returnRate decimal.Decimal, years int, birthYear int, targetIncome []decimal.Decimal) ([]decimal.Decimal, []decimal.Decimal, []decimal.Decimal) {
	traditionalBalances := make([]decimal.Decimal, years)
	rothBalances := make([]decimal.Decimal, years)
	withdrawals := make([]decimal.Decimal, years)

	currentTraditional := initialTraditional
	currentRoth := initialRoth
	rmdCalc := NewRMDCalculator(birthYear)

	for year := 1; year <= years; year++ {
		// Calculate growth for both accounts
		traditionalGrowth := currentTraditional.Mul(returnRate)
		rothGrowth := currentRoth.Mul(returnRate)

		// Determine if this is an RMD year (only affects Traditional)
		age := birthYear + year - 1
		isRMDYear := age >= rmdCalc.GetRMDAge()
		rmdAmount := rmdCalc.CalculateRMD(currentTraditional, age)

		// Calculate withdrawal
		var targetIncomeForYear decimal.Decimal
		if year <= len(targetIncome) {
			targetIncomeForYear = targetIncome[year-1]
		}

		totalWithdrawal := strategy.CalculateWithdrawal(currentTraditional.Add(currentRoth), year, targetIncomeForYear, age, isRMDYear, rmdAmount)

		// Prioritize Roth withdrawals first (no RMD requirement)
		var rothWithdrawal, traditionalWithdrawal decimal.Decimal

		if isRMDYear && rmdAmount.GreaterThan(decimal.Zero) {
			// Must take RMD from Traditional first
			traditionalWithdrawal = rmdAmount
			remainingWithdrawal := totalWithdrawal.Sub(rmdAmount)

			if remainingWithdrawal.GreaterThan(decimal.Zero) {
				// Take remaining from Roth
				if remainingWithdrawal.GreaterThan(currentRoth.Add(rothGrowth)) {
					rothWithdrawal = currentRoth.Add(rothGrowth)
				} else {
					rothWithdrawal = remainingWithdrawal
				}
			}
		} else {
			// Take from Roth first, then Traditional
			if totalWithdrawal.GreaterThan(currentRoth.Add(rothGrowth)) {
				rothWithdrawal = currentRoth.Add(rothGrowth)
				traditionalWithdrawal = totalWithdrawal.Sub(rothWithdrawal)
			} else {
				rothWithdrawal = totalWithdrawal
			}
		}

		// Ensure withdrawals don't exceed balances
		if traditionalWithdrawal.GreaterThan(currentTraditional.Add(traditionalGrowth)) {
			traditionalWithdrawal = currentTraditional.Add(traditionalGrowth)
		}
		if rothWithdrawal.GreaterThan(currentRoth.Add(rothGrowth)) {
			rothWithdrawal = currentRoth.Add(rothGrowth)
		}

		// Update balances
		currentTraditional = currentTraditional.Add(traditionalGrowth).Sub(traditionalWithdrawal)
		currentRoth = currentRoth.Add(rothGrowth).Sub(rothWithdrawal)

		// Store results
		traditionalBalances[year-1] = currentTraditional
		rothBalances[year-1] = currentRoth
		withdrawals[year-1] = traditionalWithdrawal.Add(rothWithdrawal)
	}

	return traditionalBalances, rothBalances, withdrawals
}
