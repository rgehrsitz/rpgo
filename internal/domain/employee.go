package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Employee represents a federal employee with all necessary information for retirement planning
type Employee struct {
	Name                          string          `yaml:"name" json:"name"`
	BirthDate                     time.Time       `yaml:"birth_date" json:"birth_date"`
	HireDate                      time.Time       `yaml:"hire_date" json:"hire_date"`
	CurrentSalary                 decimal.Decimal `yaml:"current_salary" json:"current_salary"`
	High3Salary                   decimal.Decimal `yaml:"high_3_salary" json:"high_3_salary"`
	TSPBalanceTraditional         decimal.Decimal `yaml:"tsp_balance_traditional" json:"tsp_balance_traditional"`
	TSPBalanceRoth                decimal.Decimal `yaml:"tsp_balance_roth" json:"tsp_balance_roth"`
	TSPContributionPercent        decimal.Decimal `yaml:"tsp_contribution_percent" json:"tsp_contribution_percent"`
	SSBenefitFRA                  decimal.Decimal `yaml:"ss_benefit_fra" json:"ss_benefit_fra"`       // Monthly at Full Retirement Age
	SSBenefit62                   decimal.Decimal `yaml:"ss_benefit_62" json:"ss_benefit_62"`        // Monthly at age 62
	SSBenefit70                   decimal.Decimal `yaml:"ss_benefit_70" json:"ss_benefit_70"`        // Monthly at age 70
	FEHBPremiumMonthly            decimal.Decimal `yaml:"fehb_premium_monthly" json:"fehb_premium_monthly"`
	SurvivorBenefitElectionPercent decimal.Decimal `yaml:"survivor_benefit_election_percent" json:"survivor_benefit_election_percent"`
	
	// Optional fields for additional context (not used in calculations)
	PayPlanGrade                  string          `yaml:"pay_plan_grade,omitempty" json:"pay_plan_grade,omitempty"`
	SSNLast4                      string          `yaml:"ssn_last4,omitempty" json:"ssn_last4,omitempty"`
}



// RetirementScenario represents a specific retirement scenario for an employee
type RetirementScenario struct {
	EmployeeName              string          `yaml:"employee_name" json:"employee_name"`
	RetirementDate            time.Time       `yaml:"retirement_date" json:"retirement_date"`
	SSStartAge                int             `yaml:"ss_start_age" json:"ss_start_age"`
	TSPWithdrawalStrategy     string          `yaml:"tsp_withdrawal_strategy" json:"tsp_withdrawal_strategy"`
	TSPWithdrawalTargetMonthly *decimal.Decimal `yaml:"tsp_withdrawal_target_monthly,omitempty" json:"tsp_withdrawal_target_monthly,omitempty"`
	TSPWithdrawalRate         *decimal.Decimal `yaml:"tsp_withdrawal_rate,omitempty" json:"tsp_withdrawal_rate,omitempty"`
}

// Scenario represents a complete retirement scenario for both employees
type Scenario struct {
	Name    string                    `yaml:"name" json:"name"`
	Robert  RetirementScenario        `yaml:"robert" json:"robert"`
	Dawn    RetirementScenario        `yaml:"dawn" json:"dawn"`
}

// GlobalAssumptions contains all the global parameters for calculations
type GlobalAssumptions struct {
	InflationRate              decimal.Decimal `yaml:"inflation_rate" json:"inflation_rate"`
	FEHBPremiumInflation       decimal.Decimal `yaml:"fehb_premium_inflation" json:"fehb_premium_inflation"`
	TSPReturnPreRetirement     decimal.Decimal `yaml:"tsp_return_pre_retirement" json:"tsp_return_pre_retirement"`
	TSPReturnPostRetirement    decimal.Decimal `yaml:"tsp_return_post_retirement" json:"tsp_return_post_retirement"`
	COLAGeneralRate            decimal.Decimal `yaml:"cola_general_rate" json:"cola_general_rate"`
	ProjectionYears            int             `yaml:"projection_years" json:"projection_years"`
	CurrentLocation            Location        `yaml:"current_location" json:"current_location"`
}

// Location represents the geographic location for tax calculations
type Location struct {
	State        string `yaml:"state" json:"state"`
	County       string `yaml:"county" json:"county"`
	Municipality string `yaml:"municipality" json:"municipality"`
}

// Configuration represents the complete input configuration
type Configuration struct {
	PersonalDetails     map[string]Employee     `yaml:"personal_details" json:"personal_details"`
	GlobalAssumptions   GlobalAssumptions       `yaml:"global_assumptions" json:"global_assumptions"`
	Scenarios           []Scenario              `yaml:"scenarios" json:"scenarios"`
}

// Age calculates the age of the employee at a given date
func (e *Employee) Age(atDate time.Time) int {
	age := atDate.Year() - e.BirthDate.Year()
	if atDate.YearDay() < e.BirthDate.YearDay() {
		age--
	}
	return age
}

// YearsOfService calculates the years of service at a given date
func (e *Employee) YearsOfService(atDate time.Time) decimal.Decimal {
	serviceDuration := atDate.Sub(e.HireDate)
	years := decimal.NewFromFloat(serviceDuration.Hours() / 24 / 365.25)
	return years.Round(4) // Round to 4 decimal places for precision
}

// FullRetirementAge calculates the Social Security Full Retirement Age based on birth year
func (e *Employee) FullRetirementAge() int {
	birthYear := e.BirthDate.Year()
	
	switch {
	case birthYear <= 1937:
		return 65
	case birthYear == 1938:
		return 65 + 2 // 65 years and 2 months
	case birthYear == 1939:
		return 65 + 4 // 65 years and 4 months
	case birthYear == 1940:
		return 65 + 6 // 65 years and 6 months
	case birthYear == 1941:
		return 65 + 8 // 65 years and 8 months
	case birthYear == 1942:
		return 65 + 10 // 65 years and 10 months
	case birthYear >= 1943 && birthYear <= 1954:
		return 66
	case birthYear == 1955:
		return 66 + 2 // 66 years and 2 months
	case birthYear == 1956:
		return 66 + 4 // 66 years and 4 months
	case birthYear == 1957:
		return 66 + 6 // 66 years and 6 months
	case birthYear == 1958:
		return 66 + 8 // 66 years and 8 months
	case birthYear == 1959:
		return 66 + 10 // 66 years and 10 months
	default: // 1960 and later
		return 67
	}
}

// MinimumRetirementAge calculates the FERS Minimum Retirement Age
func (e *Employee) MinimumRetirementAge() int {
	birthYear := e.BirthDate.Year()
	
	switch {
	case birthYear <= 1947:
		return 55
	case birthYear == 1948:
		return 55 + 2 // 55 years and 2 months
	case birthYear == 1949:
		return 55 + 4 // 55 years and 4 months
	case birthYear == 1950:
		return 55 + 6 // 55 years and 6 months
	case birthYear == 1951:
		return 55 + 8 // 55 years and 8 months
	case birthYear == 1952:
		return 55 + 10 // 55 years and 10 months
	case birthYear >= 1953 && birthYear <= 1964:
		return 56
	case birthYear == 1965:
		return 56 + 2 // 56 years and 2 months
	case birthYear == 1966:
		return 56 + 4 // 56 years and 4 months
	case birthYear == 1967:
		return 56 + 6 // 56 years and 6 months
	case birthYear == 1968:
		return 56 + 8 // 56 years and 8 months
	case birthYear == 1969:
		return 56 + 10 // 56 years and 10 months
	case birthYear >= 1970:
		return 57
	default:
		return 57
	}
}

// TotalTSPBalance returns the combined traditional and Roth TSP balance
func (e *Employee) TotalTSPBalance() decimal.Decimal {
	return e.TSPBalanceTraditional.Add(e.TSPBalanceRoth)
}

// AnnualTSPContribution calculates the annual TSP contribution amount
func (e *Employee) AnnualTSPContribution() decimal.Decimal {
	return e.CurrentSalary.Mul(e.TSPContributionPercent)
}

// AgencyMatch calculates the annual agency match (5% of salary if contributing at least 5%)
func (e *Employee) AgencyMatch() decimal.Decimal {
	if e.TSPContributionPercent.GreaterThanOrEqual(decimal.NewFromFloat(0.05)) {
		return e.CurrentSalary.Mul(decimal.NewFromFloat(0.05))
	}
	return decimal.Zero
}

// TotalAnnualTSPContribution returns the combined employee and agency contributions
func (e *Employee) TotalAnnualTSPContribution() decimal.Decimal {
	return e.AnnualTSPContribution().Add(e.AgencyMatch())
} 