package calculation

import (
	"time"

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

// createTSPStrategy creates a TSP withdrawal strategy based on scenario configuration inputs
func (ce *CalculationEngine) createTSPStrategy(strategyName string, targetMonthly *decimal.Decimal, withdrawalRate *decimal.Decimal, initialBalance decimal.Decimal, inflationRate decimal.Decimal) TSPWithdrawalStrategy {
	switch strategyName {
	case "4_percent_rule":
		return NewFourPercentRule(initialBalance, inflationRate)
	case "need_based":
		if targetMonthly != nil {
			return NewNeedBasedWithdrawal(*targetMonthly)
		}
		// Fallback to 4% rule if target not specified
		return NewFourPercentRule(initialBalance, inflationRate)
	case "variable_percentage":
		if withdrawalRate != nil {
			return NewVariablePercentageWithdrawal(initialBalance, *withdrawalRate, inflationRate)
		}
		// Fallback to 4% rule if rate not specified
		return NewFourPercentRule(initialBalance, inflationRate)
	default:
		// Default to 4% rule
		return NewFourPercentRule(initialBalance, inflationRate)
	}
}

// updateTSPBalances updates TSP balances after withdrawal
func (ce *CalculationEngine) updateTSPBalances(traditional, roth, withdrawal, returnRate decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	// Apply growth first
	traditional = traditional.Mul(decimal.NewFromFloat(1).Add(returnRate))
	roth = roth.Mul(decimal.NewFromFloat(1).Add(returnRate))

	// Withdraw from Roth first, then traditional
	if withdrawal.LessThanOrEqual(roth) {
		roth = roth.Sub(withdrawal)
	} else {
		remainingWithdrawal := withdrawal.Sub(roth)
		roth = decimal.Zero
		traditional = traditional.Sub(remainingWithdrawal)
		if traditional.LessThan(decimal.Zero) {
			traditional = decimal.Zero
		}
	}

	// Ensure balances never go negative
	if traditional.LessThan(decimal.Zero) {
		traditional = decimal.Zero
	}
	if roth.LessThan(decimal.Zero) {
		roth = decimal.Zero
	}

	return traditional, roth
}

// growTSPBalance grows a TSP balance with contributions and returns
func (ce *CalculationEngine) growTSPBalance(balance, contribution, returnRate decimal.Decimal) decimal.Decimal {
	return balance.Add(contribution).Mul(decimal.NewFromFloat(1).Add(returnRate))
}

// growTSPBalanceWithAllocation calculates TSP balance growth using lifecycle fund allocation data
func (ce *CalculationEngine) growTSPBalanceWithAllocation(employee *domain.Employee, balance, contribution decimal.Decimal, targetDate time.Time) decimal.Decimal {
	// Get the appropriate allocation for this date
	allocation := ce.getTSPAllocationForEmployee(employee, targetDate)

	// Calculate weighted return based on allocation
	weightedReturn := ce.calculateTSPReturnWithAllocation(allocation, targetDate.Year())

	// Apply growth with the weighted return
	return balance.Add(contribution).Mul(decimal.NewFromFloat(1).Add(weightedReturn))
}

// getTSPAllocationForEmployee returns the TSP allocation for an employee at a specific date
func (ce *CalculationEngine) getTSPAllocationForEmployee(employee *domain.Employee, targetDate time.Time) domain.TSPAllocation {
	// If employee has a lifecycle fund specified, use that
	if employee.TSPLifecycleFund != nil {
		allocation, err := ce.LifecycleFundLoader.GetAllocationAtDate(employee.TSPLifecycleFund.FundName, targetDate)
		if err == nil && allocation != nil {
			return *allocation
		}
		// Fall back to default if lifecycle fund lookup fails
	}

	// If employee has a specific allocation, use that
	if employee.TSPAllocation != nil {
		return *employee.TSPAllocation
	}

	// Use default allocation from global assumptions
	// This would need to be passed in from the configuration
	// For now, return a conservative default
	return domain.TSPAllocation{
		CFund: decimal.NewFromFloat(0.60),
		SFund: decimal.NewFromFloat(0.20),
		IFund: decimal.NewFromFloat(0.10),
		FFund: decimal.NewFromFloat(0.10),
		GFund: decimal.NewFromFloat(0.00),
	}
}

// calculateTSPReturnWithAllocation calculates TSP return using specific allocation and statistical models
func (ce *CalculationEngine) calculateTSPReturnWithAllocation(allocation domain.TSPAllocation, year int) decimal.Decimal {
	// Initialize return values
	var cFundReturn, sFundReturn, iFundReturn, fFundReturn, gFundReturn decimal.Decimal

	// Check if we have Monte Carlo fund returns available (higher priority than historical data)
	usingMonteCarlo := len(ce.MonteCarloFundReturns) > 0

	if usingMonteCarlo {
		// Use Monte Carlo generated fund returns, fall back to historical/statistical for missing funds
		if cReturn, exists := ce.MonteCarloFundReturns["C"]; exists {
			cFundReturn = cReturn
		} else {
			cFundReturn = ce.getFallbackReturn("C", year)
		}

		if sReturn, exists := ce.MonteCarloFundReturns["S"]; exists {
			sFundReturn = sReturn
		} else {
			sFundReturn = ce.getFallbackReturn("S", year)
		}

		if iReturn, exists := ce.MonteCarloFundReturns["I"]; exists {
			iFundReturn = iReturn
		} else {
			iFundReturn = ce.getFallbackReturn("I", year)
		}

		if fReturn, exists := ce.MonteCarloFundReturns["F"]; exists {
			fFundReturn = fReturn
		} else {
			fFundReturn = ce.getFallbackReturn("F", year)
		}

		if gReturn, exists := ce.MonteCarloFundReturns["G"]; exists {
			gFundReturn = gReturn
		} else {
			gFundReturn = ce.getFallbackReturn("G", year)
		}
	} else {
		// Use historical data or statistical models for all funds
		cFundReturn = ce.getFallbackReturn("C", year)
		sFundReturn = ce.getFallbackReturn("S", year)
		iFundReturn = ce.getFallbackReturn("I", year)
		fFundReturn = ce.getFallbackReturn("F", year)
		gFundReturn = ce.getFallbackReturn("G", year)
	}

	// Weighted return calculation using actual allocation
	weightedReturn := decimal.Zero

	// C Fund (Large Cap)
	weightedReturn = weightedReturn.Add(allocation.CFund.Mul(cFundReturn))

	// S Fund (Small Cap)
	weightedReturn = weightedReturn.Add(allocation.SFund.Mul(sFundReturn))

	// I Fund (International)
	weightedReturn = weightedReturn.Add(allocation.IFund.Mul(iFundReturn))

	// F Fund (Bonds)
	weightedReturn = weightedReturn.Add(allocation.FFund.Mul(fFundReturn))

	// G Fund (Government)
	weightedReturn = weightedReturn.Add(allocation.GFund.Mul(gFundReturn))

	return weightedReturn
}

// getFallbackReturn gets historical or statistical fallback return for a fund
func (ce *CalculationEngine) getFallbackReturn(fund string, year int) decimal.Decimal {
	// Try historical data first
	if ce.HistoricalData != nil && ce.HistoricalData.IsLoaded {
		if returnRate, err := ce.HistoricalData.GetTSPReturn(fund, year); err == nil {
			return returnRate
		}
	}

	// Fallback to statistical models
	switch fund {
	case "C":
		return decimal.NewFromFloat(0.1125) // 11.25% historical mean
	case "S":
		return decimal.NewFromFloat(0.1117) // 11.17% historical mean
	case "I":
		return decimal.NewFromFloat(0.0634) // 6.34% historical mean
	case "F":
		return decimal.NewFromFloat(0.0532) // 5.32% historical mean
	case "G":
		return decimal.NewFromFloat(0.0493) // 4.93% historical mean
	default:
		return decimal.NewFromFloat(0.08) // 8% default
	}
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
