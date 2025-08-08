package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// AnnualCashFlow represents the complete cash flow for a single year
type AnnualCashFlow struct {
	Year      int       `json:"year"`
	Date      time.Time `json:"date"`
	AgeRobert int       `json:"age_robert"`
	AgeDawn   int       `json:"age_dawn"`

	// Income Sources
	SalaryRobert         decimal.Decimal `json:"salary_robert"`
	SalaryDawn           decimal.Decimal `json:"salary_dawn"`
	PensionRobert        decimal.Decimal `json:"pension_robert"`
	PensionDawn          decimal.Decimal `json:"pension_dawn"`
	TSPWithdrawalRobert  decimal.Decimal `json:"tsp_withdrawal_robert"`
	TSPWithdrawalDawn    decimal.Decimal `json:"tsp_withdrawal_dawn"`
	SSBenefitRobert      decimal.Decimal `json:"ss_benefit_robert"`
	SSBenefitDawn        decimal.Decimal `json:"ss_benefit_dawn"`
	FERSSupplementRobert decimal.Decimal `json:"fers_supplement_robert"`
	FERSSupplementDawn   decimal.Decimal `json:"fers_supplement_dawn"`
	TotalGrossIncome     decimal.Decimal `json:"total_gross_income"`

	// Deductions and Taxes
	FederalTax       decimal.Decimal `json:"federal_tax"`
	StateTax         decimal.Decimal `json:"state_tax"`
	LocalTax         decimal.Decimal `json:"local_tax"`
	FICATax          decimal.Decimal `json:"fica_tax"`
	TSPContributions decimal.Decimal `json:"tsp_contributions"`
	FEHBPremium      decimal.Decimal `json:"fehb_premium"`
	MedicarePremium  decimal.Decimal `json:"medicare_premium"`
	NetIncome        decimal.Decimal `json:"net_income"`

	// TSP Balances (end of year)
	TSPBalanceRobert      decimal.Decimal `json:"tsp_balance_robert"`
	TSPBalanceDawn        decimal.Decimal `json:"tsp_balance_dawn"`
	TSPBalanceTraditional decimal.Decimal `json:"tsp_balance_traditional"`
	TSPBalanceRoth        decimal.Decimal `json:"tsp_balance_roth"`

	// Additional Information
	IsRetired          bool            `json:"is_retired"`
	IsMedicareEligible bool            `json:"is_medicare_eligible"`
	IsRMDYear          bool            `json:"is_rmd_year"`
	RMDAmount          decimal.Decimal `json:"rmd_amount"`
}

// ScenarioSummary provides a summary of key metrics for a retirement scenario
type ScenarioSummary struct {
	Name                string           `json:"name"`
	FirstYearNetIncome  decimal.Decimal  `json:"first_year_net_income"`
	Year5NetIncome      decimal.Decimal  `json:"year_5_net_income"`
	Year10NetIncome     decimal.Decimal  `json:"year_10_net_income"`
	TotalLifetimeIncome decimal.Decimal  `json:"total_lifetime_income"`
	TSPLongevity        int              `json:"tsp_longevity"`
	SuccessRate         decimal.Decimal  `json:"success_rate"` // From Monte Carlo
	InitialTSPBalance   decimal.Decimal  `json:"initial_tsp_balance"`
	FinalTSPBalance     decimal.Decimal  `json:"final_tsp_balance"`
	Projection          []AnnualCashFlow `json:"projection"`
	
	// Absolute calendar year comparisons for apples-to-apples analysis
	NetIncome2030       decimal.Decimal  `json:"net_income_2030"`
	NetIncome2035       decimal.Decimal  `json:"net_income_2035"`
	NetIncome2040       decimal.Decimal  `json:"net_income_2040"`
	PreRetirementNet2030 decimal.Decimal `json:"pre_retirement_net_2030"` // What current net would be with COLA growth
	PreRetirementNet2035 decimal.Decimal `json:"pre_retirement_net_2035"`
	PreRetirementNet2040 decimal.Decimal `json:"pre_retirement_net_2040"`
}

// ScenarioComparison provides a comparison of all scenarios
type ScenarioComparison struct {
	BaselineNetIncome  decimal.Decimal   `json:"baseline_net_income"`
	Scenarios          []ScenarioSummary `json:"scenarios"`
	ImmediateImpact    ImpactAnalysis    `json:"immediate_impact"`
	LongTermProjection LongTermAnalysis  `json:"long_term_projection"`
	Assumptions        []string          `json:"assumptions"` // Dynamic assumptions from config
}

// ImpactAnalysis provides analysis of the immediate impact of retirement
type ImpactAnalysis struct {
	CurrentToFirstYear   IncomeChange `json:"current_to_first_year"`
	CurrentToSteadyState IncomeChange `json:"current_to_steady_state"`
	RecommendedScenario  string       `json:"recommended_scenario"`
	KeyConsiderations    []string     `json:"key_considerations"`
}

// LongTermAnalysis provides analysis of long-term projections
type LongTermAnalysis struct {
	BestScenarioForIncome    string   `json:"best_scenario_for_income"`
	BestScenarioForLongevity string   `json:"best_scenario_for_longevity"`
	RiskAssessment           string   `json:"risk_assessment"`
	Recommendations          []string `json:"recommendations"`
}

// IncomeChange represents the change in income between two periods
type IncomeChange struct {
	ScenarioName     string          `json:"scenario_name"`
	NetIncomeChange  decimal.Decimal `json:"net_income_change"`
	PercentageChange decimal.Decimal `json:"percentage_change"`
	MonthlyChange    decimal.Decimal `json:"monthly_change"`
}

// TSPProjection represents a single year's TSP projection
type TSPProjection struct {
	Year             int             `json:"year"`
	BeginningBalance decimal.Decimal `json:"beginning_balance"`
	Growth           decimal.Decimal `json:"growth"`
	Withdrawal       decimal.Decimal `json:"withdrawal"`
	RMD              decimal.Decimal `json:"rmd"`
	EndingBalance    decimal.Decimal `json:"ending_balance"`
	TraditionalPct   decimal.Decimal `json:"traditional_pct"`
	RothPct          decimal.Decimal `json:"roth_pct"`
}

// MonteCarloResults represents the results of Monte Carlo simulation
type MonteCarloResults struct {
	Simulations         []SimulationOutcome `json:"simulations"`
	SuccessRate         decimal.Decimal     `json:"success_rate"`
	MedianEndingBalance decimal.Decimal     `json:"median_ending_balance"`
	PercentileRanges    PercentileRanges    `json:"percentile_ranges"`
	NumSimulations      int                 `json:"num_simulations"`
}

// SimulationOutcome represents a single Monte Carlo simulation outcome
type SimulationOutcome struct {
	YearOutcomes    []YearOutcome   `json:"year_outcomes"`
	PortfolioLasted int             `json:"portfolio_lasted"`
	EndingBalance   decimal.Decimal `json:"ending_balance"`
	Success         bool            `json:"success"`
}

// YearOutcome represents a single year's outcome in a Monte Carlo simulation
type YearOutcome struct {
	Year       int             `json:"year"`
	Balance    decimal.Decimal `json:"balance"`
	Withdrawal decimal.Decimal `json:"withdrawal"`
	Return     decimal.Decimal `json:"return"`
}

// PercentileRanges represents percentile ranges for Monte Carlo results
type PercentileRanges struct {
	P10 decimal.Decimal `json:"p10"`
	P25 decimal.Decimal `json:"p25"`
	P50 decimal.Decimal `json:"p50"`
	P75 decimal.Decimal `json:"p75"`
	P90 decimal.Decimal `json:"p90"`
}

// TaxableIncome represents various income components for tax calculation
type TaxableIncome struct {
	Salary             decimal.Decimal `json:"salary"`
	FERSPension        decimal.Decimal `json:"fers_pension"`
	TSPWithdrawalsTrad decimal.Decimal `json:"tsp_withdrawals_trad"`
	TaxableSSBenefits  decimal.Decimal `json:"taxable_ss_benefits"`
	OtherTaxableIncome decimal.Decimal `json:"other_taxable_income"`
	WageIncome         decimal.Decimal `json:"wage_income"`
	InterestIncome     decimal.Decimal `json:"interest_income"`
}

// CalculateTotalIncome calculates the total gross income for the year
func (acf *AnnualCashFlow) CalculateTotalIncome() decimal.Decimal {
	return acf.SalaryRobert.Add(acf.SalaryDawn).
		Add(acf.PensionRobert).Add(acf.PensionDawn).
		Add(acf.TSPWithdrawalRobert).Add(acf.TSPWithdrawalDawn).
		Add(acf.SSBenefitRobert).Add(acf.SSBenefitDawn).
		Add(acf.FERSSupplementRobert).Add(acf.FERSSupplementDawn)
}

// CalculateTotalDeductions calculates the total deductions for the year
func (acf *AnnualCashFlow) CalculateTotalDeductions() decimal.Decimal {
	return acf.FederalTax.Add(acf.StateTax).Add(acf.LocalTax).Add(acf.FICATax).
		Add(acf.TSPContributions).Add(acf.FEHBPremium).Add(acf.MedicarePremium)
}

// CalculateNetIncome calculates the net income for the year
func (acf *AnnualCashFlow) CalculateNetIncome() decimal.Decimal {
	acf.NetIncome = acf.TotalGrossIncome.Sub(acf.CalculateTotalDeductions())
	return acf.NetIncome
}

// TotalTSPBalance returns the combined TSP balance for both employees
func (acf *AnnualCashFlow) TotalTSPBalance() decimal.Decimal {
	return acf.TSPBalanceRobert.Add(acf.TSPBalanceDawn)
}

// IsTSPDepleted returns true if TSP balances are zero or negative
func (acf *AnnualCashFlow) IsTSPDepleted() bool {
	return acf.TotalTSPBalance().LessThanOrEqual(decimal.Zero)
}
