package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// AnnualCashFlow represents the complete cash flow for a single year
type AnnualCashFlow struct {
	Year      int       `json:"year"`
	Date      time.Time `json:"date"`
	AgeRobert int       `json:"ageRobert"`
	AgeDawn   int       `json:"ageDawn"`

	// Income Sources
	SalaryRobert          decimal.Decimal `json:"salaryRobert"`
	SalaryDawn            decimal.Decimal `json:"salaryDawn"`
	PensionRobert         decimal.Decimal `json:"pensionRobert"`
	PensionDawn           decimal.Decimal `json:"pensionDawn"`
	SurvivorPensionRobert decimal.Decimal `json:"survivorPensionRobert"`
	SurvivorPensionDawn   decimal.Decimal `json:"survivorPensionDawn"`
	TSPWithdrawalRobert   decimal.Decimal `json:"tspWithdrawalRobert"`
	TSPWithdrawalDawn     decimal.Decimal `json:"tspWithdrawalDawn"`
	SSBenefitRobert       decimal.Decimal `json:"ssBenefitRobert"`
	SSBenefitDawn         decimal.Decimal `json:"ssBenefitDawn"`
	FERSSupplementRobert  decimal.Decimal `json:"fersSupplementRobert"`
	FERSSupplementDawn    decimal.Decimal `json:"fersSupplementDawn"`
	TotalGrossIncome      decimal.Decimal `json:"totalGrossIncome"`

	// Deductions and Taxes
	FederalTax               decimal.Decimal `json:"federalTax"`
	FederalTaxableIncome     decimal.Decimal `json:"federalTaxableIncome"`
	FederalStandardDeduction decimal.Decimal `json:"federalStandardDeduction"`
	FederalFilingStatus      string          `json:"federalFilingStatus"`
	FederalSeniors65Plus     int             `json:"federalSeniors65Plus"`
	StateTax                 decimal.Decimal `json:"stateTax"`
	LocalTax                 decimal.Decimal `json:"localTax"`
	FICATax                  decimal.Decimal `json:"ficaTax"`
	TSPContributions         decimal.Decimal `json:"tspContributions"`
	FEHBPremium              decimal.Decimal `json:"fehbPremium"`
	MedicarePremium          decimal.Decimal `json:"medicarePremium"`
	NetIncome                decimal.Decimal `json:"netIncome"`

	// TSP Balances (end of year)
	TSPBalanceRobert      decimal.Decimal `json:"tspBalanceRobert"`
	TSPBalanceDawn        decimal.Decimal `json:"tspBalanceDawn"`
	TSPBalanceTraditional decimal.Decimal `json:"tspBalanceTraditional"`
	TSPBalanceRoth        decimal.Decimal `json:"tspBalanceRoth"`

	// Additional Information
	IsRetired          bool            `json:"isRetired"`
	IsMedicareEligible bool            `json:"isMedicareEligible"`
	IsRMDYear          bool            `json:"isRmdYear"`
	RMDAmount          decimal.Decimal `json:"rmdAmount"`

	// Mortality / survivor tracking (Phase 1 deterministic death modeling)
	RobertDeceased     bool `json:"robertDeceased"`
	DawnDeceased       bool `json:"dawnDeceased"`
	FilingStatusSingle bool `json:"filingStatusSingle"` // true once survivor filing status applies
}

// ScenarioSummary provides a summary of key metrics for a retirement scenario
type ScenarioSummary struct {
	Name                string           `json:"name"`
	FirstYearNetIncome  decimal.Decimal  `json:"firstYearNetIncome"`
	Year5NetIncome      decimal.Decimal  `json:"year5NetIncome"`
	Year10NetIncome     decimal.Decimal  `json:"year10NetIncome"`
	TotalLifetimeIncome decimal.Decimal  `json:"totalLifetimeIncome"`
	TSPLongevity        int              `json:"tspLongevity"`
	SuccessRate         decimal.Decimal  `json:"successRate"` // From Monte Carlo
	InitialTSPBalance   decimal.Decimal  `json:"initialTspBalance"`
	FinalTSPBalance     decimal.Decimal  `json:"finalTspBalance"`
	Projection          []AnnualCashFlow `json:"projection"`

	// Absolute calendar year comparisons for apples-to-apples analysis
	NetIncome2030        decimal.Decimal `json:"netIncome2030"`
	NetIncome2035        decimal.Decimal `json:"netIncome2035"`
	NetIncome2040        decimal.Decimal `json:"netIncome2040"`
	PreRetirementNet2030 decimal.Decimal `json:"preRetirementNet2030"` // What current net would be with COLA growth
	PreRetirementNet2035 decimal.Decimal `json:"preRetirementNet2035"`
	PreRetirementNet2040 decimal.Decimal `json:"preRetirementNet2040"`
}

// ScenarioComparison provides a comparison of all scenarios
type ScenarioComparison struct {
	BaselineNetIncome  decimal.Decimal   `json:"baselineNetIncome"`
	Scenarios          []ScenarioSummary `json:"scenarios"`
	ImmediateImpact    ImpactAnalysis    `json:"immediateImpact"`
	LongTermProjection LongTermAnalysis  `json:"longTermProjection"`
	Assumptions        []string          `json:"assumptions"` // Dynamic assumptions from config
}

// ImpactAnalysis provides analysis of the immediate impact of retirement
type ImpactAnalysis struct {
	CurrentToFirstYear   IncomeChange `json:"currentToFirstYear"`
	CurrentToSteadyState IncomeChange `json:"currentToSteadyState"`
	RecommendedScenario  string       `json:"recommendedScenario"`
	KeyConsiderations    []string     `json:"keyConsiderations"`
}

// LongTermAnalysis provides analysis of long-term projections
type LongTermAnalysis struct {
	BestScenarioForIncome    string   `json:"bestScenarioForIncome"`
	BestScenarioForLongevity string   `json:"bestScenarioForLongevity"`
	RiskAssessment           string   `json:"riskAssessment"`
	Recommendations          []string `json:"recommendations"`
}

// IncomeChange represents the change in income between two periods
type IncomeChange struct {
	ScenarioName     string          `json:"scenarioName"`
	NetIncomeChange  decimal.Decimal `json:"netIncomeChange"`
	PercentageChange decimal.Decimal `json:"percentageChange"`
	MonthlyChange    decimal.Decimal `json:"monthlyChange"`
}

// TSPProjection represents a single year's TSP projection
type TSPProjection struct {
	Year             int             `json:"year"`
	BeginningBalance decimal.Decimal `json:"beginningBalance"`
	Growth           decimal.Decimal `json:"growth"`
	Withdrawal       decimal.Decimal `json:"withdrawal"`
	RMD              decimal.Decimal `json:"rmd"`
	EndingBalance    decimal.Decimal `json:"endingBalance"`
	TraditionalPct   decimal.Decimal `json:"traditionalPct"`
	RothPct          decimal.Decimal `json:"rothPct"`
}

// MonteCarloResults represents the results of Monte Carlo simulation
type MonteCarloResults struct {
	Simulations         []SimulationOutcome `json:"simulations"`
	SuccessRate         decimal.Decimal     `json:"successRate"`
	MedianEndingBalance decimal.Decimal     `json:"medianEndingBalance"`
	PercentileRanges    PercentileRanges    `json:"percentileRanges"`
	NumSimulations      int                 `json:"numSimulations"`
}

// SimulationOutcome represents a single Monte Carlo simulation outcome
type SimulationOutcome struct {
	YearOutcomes    []YearOutcome   `json:"yearOutcomes"`
	PortfolioLasted int             `json:"portfolioLasted"`
	EndingBalance   decimal.Decimal `json:"endingBalance"`
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
	FERSPension        decimal.Decimal `json:"fersPension"`
	TSPWithdrawalsTrad decimal.Decimal `json:"tspWithdrawalsTrad"`
	TaxableSSBenefits  decimal.Decimal `json:"taxableSsBenefits"`
	OtherTaxableIncome decimal.Decimal `json:"otherTaxableIncome"`
	WageIncome         decimal.Decimal `json:"wageIncome"`
	InterestIncome     decimal.Decimal `json:"interestIncome"`
}

// CalculateTotalIncome calculates the total gross income for the year
func (acf *AnnualCashFlow) CalculateTotalIncome() decimal.Decimal {
	return acf.SalaryRobert.Add(acf.SalaryDawn).
		Add(acf.PensionRobert).Add(acf.PensionDawn).
		Add(acf.SurvivorPensionRobert).Add(acf.SurvivorPensionDawn).
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
