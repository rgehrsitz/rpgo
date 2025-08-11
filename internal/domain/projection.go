package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// AnnualCashFlow represents the complete cash flow for a single year with generic participant support
type AnnualCashFlow struct {
	Year int       `json:"year"`
	Date time.Time `json:"date"`

	// Participant-based data using maps (participant name -> value)
	Ages                        map[string]int             `json:"ages"`                        // participantName -> age
	Salaries                    map[string]decimal.Decimal `json:"salaries"`                    // participantName -> salary
	Pensions                    map[string]decimal.Decimal `json:"pensions"`                    // participantName -> pension
	SurvivorPensions            map[string]decimal.Decimal `json:"survivorPensions"`            // participantName -> survivor pension
	TSPWithdrawals              map[string]decimal.Decimal `json:"tspWithdrawals"`              // participantName -> TSP withdrawal
	SSBenefits                  map[string]decimal.Decimal `json:"ssBenefits"`                  // participantName -> Social Security benefits
	FERSSupplements             map[string]decimal.Decimal `json:"fersSupplements"`             // participantName -> FERS supplement
	TSPBalances                 map[string]decimal.Decimal `json:"tspBalances"`                 // participantName -> total TSP balance
	ParticipantTSPContributions map[string]decimal.Decimal `json:"participantTspContributions"` // participantName -> TSP contributions
	IsDeceased                  map[string]bool            `json:"isDeceased"`                  // participantName -> deceased status

	// Household-level totals and taxes
	TotalGrossIncome         decimal.Decimal `json:"totalGrossIncome"`
	FederalTax               decimal.Decimal `json:"federalTax"`
	FederalTaxableIncome     decimal.Decimal `json:"federalTaxableIncome"`
	FederalStandardDeduction decimal.Decimal `json:"federalStandardDeduction"`
	FederalFilingStatus      string          `json:"federalFilingStatus"`
	FederalSeniors65Plus     int             `json:"federalSeniors65Plus"`
	StateTax                 decimal.Decimal `json:"stateTax"`
	LocalTax                 decimal.Decimal `json:"localTax"`
	FICATax                  decimal.Decimal `json:"ficaTax"`
	TotalTSPContributions    decimal.Decimal `json:"totalTspContributions"` // Sum of all participant TSP contributions
	FEHBPremium              decimal.Decimal `json:"fehbPremium"`
	MedicarePremium          decimal.Decimal `json:"medicarePremium"`
	NetIncome                decimal.Decimal `json:"netIncome"`

	// Additional Information
	IsRetired          bool            `json:"isRetired"`
	IsMedicareEligible bool            `json:"isMedicareEligible"`
	IsRMDYear          bool            `json:"isRmdYear"`
	RMDAmount          decimal.Decimal `json:"rmdAmount"`
	FilingStatusSingle bool            `json:"filingStatusSingle"` // true once survivor filing status applies
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

// NewAnnualCashFlow creates a new AnnualCashFlow with initialized participant maps
func NewAnnualCashFlow(year int, date time.Time, participantNames []string) *AnnualCashFlow {
	acf := &AnnualCashFlow{
		Year:                        year,
		Date:                        date,
		Ages:                        make(map[string]int),
		Salaries:                    make(map[string]decimal.Decimal),
		Pensions:                    make(map[string]decimal.Decimal),
		SurvivorPensions:            make(map[string]decimal.Decimal),
		TSPWithdrawals:              make(map[string]decimal.Decimal),
		SSBenefits:                  make(map[string]decimal.Decimal),
		FERSSupplements:             make(map[string]decimal.Decimal),
		TSPBalances:                 make(map[string]decimal.Decimal),
		ParticipantTSPContributions: make(map[string]decimal.Decimal),
		IsDeceased:                  make(map[string]bool),
	}

	// Initialize all participants with zero values
	for _, name := range participantNames {
		acf.Ages[name] = 0
		acf.Salaries[name] = decimal.Zero
		acf.Pensions[name] = decimal.Zero
		acf.SurvivorPensions[name] = decimal.Zero
		acf.TSPWithdrawals[name] = decimal.Zero
		acf.SSBenefits[name] = decimal.Zero
		acf.FERSSupplements[name] = decimal.Zero
		acf.TSPBalances[name] = decimal.Zero
		acf.ParticipantTSPContributions[name] = decimal.Zero
		acf.IsDeceased[name] = false
	}

	return acf
}

// SyncLegacyFields syncs the participant maps to legacy fields for backward compatibility
// This method assumes "robert" and "dawn" are the first two participants if they exist
// (Legacy SyncLegacyFields removed: scalar convenience fields deprecated.)

// GetParticipantNames returns all participant names from the cash flow
func (acf *AnnualCashFlow) GetParticipantNames() []string {
	names := make([]string, 0, len(acf.Ages))
	for name := range acf.Ages {
		names = append(names, name)
	}
	return names
}

// GetTotalSalary returns the sum of all participant salaries
func (acf *AnnualCashFlow) GetTotalSalary() decimal.Decimal {
	total := decimal.Zero
	for _, salary := range acf.Salaries {
		total = total.Add(salary)
	}
	return total
}

// GetTotalPension returns the sum of all participant pensions
func (acf *AnnualCashFlow) GetTotalPension() decimal.Decimal {
	total := decimal.Zero
	for _, pension := range acf.Pensions {
		total = total.Add(pension)
	}
	return total
}

// GetTotalTSPWithdrawal returns the sum of all participant TSP withdrawals
func (acf *AnnualCashFlow) GetTotalTSPWithdrawal() decimal.Decimal {
	total := decimal.Zero
	for _, withdrawal := range acf.TSPWithdrawals {
		total = total.Add(withdrawal)
	}
	return total
}

// GetTotalSSBenefit returns the sum of all participant Social Security benefits
func (acf *AnnualCashFlow) GetTotalSSBenefit() decimal.Decimal {
	total := decimal.Zero
	for _, benefit := range acf.SSBenefits {
		total = total.Add(benefit)
	}
	return total
}

// GetTotalFERSSupplement returns the sum of all participant FERS supplements
func (acf *AnnualCashFlow) GetTotalFERSSupplement() decimal.Decimal {
	total := decimal.Zero
	for _, supplement := range acf.FERSSupplements {
		total = total.Add(supplement)
	}
	return total
}

// GetTotalTSPBalance returns the sum of all participant TSP balances
func (acf *AnnualCashFlow) GetTotalTSPBalance() decimal.Decimal {
	total := decimal.Zero
	for _, balance := range acf.TSPBalances {
		total = total.Add(balance)
	}
	return total
}

// GetLivingParticipants returns a list of living participants
func (acf *AnnualCashFlow) GetLivingParticipants() []string {
	living := make([]string, 0)
	for name, deceased := range acf.IsDeceased {
		if !deceased {
			living = append(living, name)
		}
	}
	return living
}

// GetDeceasedParticipants returns a list of deceased participants
func (acf *AnnualCashFlow) GetDeceasedParticipants() []string {
	deceased := make([]string, 0)
	for name, isDead := range acf.IsDeceased {
		if isDead {
			deceased = append(deceased, name)
		}
	}
	return deceased
}

// CalculateTotalIncome calculates the total gross income for the year
func (acf *AnnualCashFlow) CalculateTotalIncome() decimal.Decimal {
	return acf.GetTotalSalary().
		Add(acf.GetTotalPension()).
		Add(acf.GetTotalTSPWithdrawal()).
		Add(acf.GetTotalSSBenefit()).
		Add(acf.GetTotalFERSSupplement())
}

// CalculateTotalDeductions calculates the total deductions for the year
func (acf *AnnualCashFlow) CalculateTotalDeductions() decimal.Decimal {
	return acf.FederalTax.Add(acf.StateTax).Add(acf.LocalTax).Add(acf.FICATax).
		Add(acf.TotalTSPContributions).Add(acf.FEHBPremium).Add(acf.MedicarePremium)
}

// CalculateNetIncome calculates the net income for the year
func (acf *AnnualCashFlow) CalculateNetIncome() decimal.Decimal {
	acf.NetIncome = acf.TotalGrossIncome.Sub(acf.CalculateTotalDeductions())
	return acf.NetIncome
}

// TotalTSPBalance returns the combined TSP balance for all participants
func (acf *AnnualCashFlow) TotalTSPBalance() decimal.Decimal {
	return acf.GetTotalTSPBalance()
}

// IsTSPDepleted returns true if TSP balances are zero or negative
func (acf *AnnualCashFlow) IsTSPDepleted() bool {
	return acf.TotalTSPBalance().LessThanOrEqual(decimal.Zero)
}
