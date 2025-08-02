package main

import (
	"fmt"
	"time"
	
	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/rpgo/retirement-calculator/internal/calculation"
	"github.com/rpgo/retirement-calculator/pkg/dateutil"
	"github.com/shopspring/decimal"
)

func main() {
	birthDate := time.Date(1967, 1, 1, 0, 0, 0, 0, time.UTC)
	hireDate := time.Date(1995, 1, 1, 0, 0, 0, 0, time.UTC)
	retirementDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	
	employee := &domain.Employee{
		BirthDate: birthDate,
		HireDate:  hireDate,
	}
	
	age := dateutil.Age(birthDate, retirementDate)
	serviceYears := dateutil.YearsOfService(hireDate, retirementDate)
	mra := dateutil.MinimumRetirementAge(birthDate)
	
	fmt.Printf("Birth Date: %s\n", birthDate.Format("2006-01-02"))
	fmt.Printf("Retirement Date: %s\n", retirementDate.Format("2006-01-02"))
	fmt.Printf("Age at retirement: %d\n", age)
	fmt.Printf("Years of service: %f\n", serviceYears)
	fmt.Printf("MRA: %d\n", mra)
	fmt.Printf("Age >= MRA: %t\n", age >= mra)
	fmt.Printf("Service >= 10: %t\n", decimal.NewFromFloat(serviceYears).GreaterThanOrEqual(decimal.NewFromInt(10)))
	
	valid, reason := calculation.ValidateFERSEligibility(employee, retirementDate)
	fmt.Printf("Valid: %t\n", valid)
	fmt.Printf("Reason: %s\n", reason)
}