package calculation

import (
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// CalculateFERSSupplementYear calculates the FERS Special Retirement Supplement for a given year offset
func CalculateFERSSupplementYear(employee *domain.Employee, retirementDate time.Time, yearsSinceRetirement int, inflationRate decimal.Decimal) decimal.Decimal {
	if yearsSinceRetirement < 0 {
		return decimal.Zero
	}

	projectionDate := retirementDate.AddDate(yearsSinceRetirement, 0, 0)
	age := employee.Age(projectionDate)
	if age >= 62 {
		return decimal.Zero // SRS stops at age 62
	}

	serviceYears := employee.YearsOfService(retirementDate)
	srs := CalculateFERSSpecialRetirementSupplement(employee.SSBenefit62, serviceYears, age)

	for y := 0; y < yearsSinceRetirement; y++ {
		srs = srs.Mul(decimal.NewFromFloat(1).Add(inflationRate))
	}
	return srs
}

// CalculateFEHBPremium calculates FEHB premium for a given year
func CalculateFEHBPremium(employee *domain.Employee, year int, premiumInflation decimal.Decimal, fehbConfig domain.FEHBConfig) decimal.Decimal {
	inflationFactor := decimal.NewFromFloat(1).Add(premiumInflation)
	adjustedPremium := employee.FEHBPremiumPerPayPeriod.Mul(inflationFactor.Pow(decimal.NewFromInt(int64(year))))
	return adjustedPremium.Mul(decimal.NewFromInt(int64(fehbConfig.PayPeriodsPerYear)))
}

// CalculateRMD wraps RMD calculation with birth year
func CalculateRMD(balance decimal.Decimal, birthYear, age int) decimal.Decimal {
	rmdCalc := NewRMDCalculator(birthYear)
	return rmdCalc.CalculateRMD(balance, age)
}
