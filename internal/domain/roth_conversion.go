package domain

import (
	"fmt"

	"github.com/shopspring/decimal"
)

// RothConversion represents a single Roth conversion event
type RothConversion struct {
	Year   int             `yaml:"year" json:"year"`     // Calendar year of conversion
	Amount decimal.Decimal `yaml:"amount" json:"amount"` // Amount to convert from Traditional to Roth
	Source string          `yaml:"source" json:"source"` // Source account: "traditional_tsp" or "traditional_ira"
}

// RothConversionSchedule represents a series of Roth conversions for a participant
type RothConversionSchedule struct {
	Conversions []RothConversion `yaml:"conversions" json:"conversions"`
}

// RothConversionPlan represents the complete analysis and recommendations
type RothConversionPlan struct {
	Participant      string                `json:"participant"`
	ConversionWindow YearRange             `json:"conversionWindow"`
	TargetBracket    int                   `json:"targetBracket"`
	Objective        OptimizationObjective `json:"objective"`

	Baseline     *ScenarioSummary    `json:"baseline"`
	Recommended  *ConversionOutcome  `json:"recommended"`
	Alternatives []ConversionOutcome `json:"alternatives"`
	Analysis     *ConversionAnalysis `json:"analysis"`
}

// YearRange represents a range of years for conversion planning
type YearRange struct {
	Start int `json:"start"`
	End   int `json:"end"`
}

// OptimizationObjective defines what to optimize for
type OptimizationObjective int

const (
	MinimizeLifetimeTax OptimizationObjective = iota
	MinimizeLifetimeIRMAA
	MinimizeCombined
	MaximizeEstate
	MaximizeNetIncome
)

func (o OptimizationObjective) String() string {
	switch o {
	case MinimizeLifetimeTax:
		return "minimize_lifetime_tax"
	case MinimizeLifetimeIRMAA:
		return "minimize_lifetime_irmaa"
	case MinimizeCombined:
		return "minimize_combined"
	case MaximizeEstate:
		return "maximize_estate"
	case MaximizeNetIncome:
		return "maximize_net_income"
	default:
		return "unknown"
	}
}

// ConversionStrategy represents a specific conversion strategy
type ConversionStrategy struct {
	Year   int             `json:"year"`
	Amount decimal.Decimal `json:"amount"`
}

// ConversionOutcome represents the results of applying a conversion strategy
type ConversionOutcome struct {
	Strategy      ConversionStrategy `json:"strategy"`
	Projection    *ScenarioSummary   `json:"projection"`
	LifetimeTax   decimal.Decimal    `json:"lifetimeTax"`
	LifetimeIRMAA decimal.Decimal    `json:"lifetimeIrmaa"`
	FinalBalances FinalBalances      `json:"finalBalances"`
	NetBenefit    decimal.Decimal    `json:"netBenefit"`
	ROI           decimal.Decimal    `json:"roi"`
}

// FinalBalances represents account balances at the end of projection
type FinalBalances struct {
	TraditionalTSP decimal.Decimal `json:"traditionalTsp"`
	RothTSP        decimal.Decimal `json:"rothTsp"`
	TaxableAccount decimal.Decimal `json:"taxableAccount"`
}

// ConversionAnalysis provides detailed analysis of conversion strategies
type ConversionAnalysis struct {
	TotalConversions    decimal.Decimal     `json:"totalConversions"`
	TotalTaxPaid        decimal.Decimal     `json:"totalTaxPaid"`
	IRMAASavings        decimal.Decimal     `json:"irmaaSavings"`
	RMDTaxReduction     decimal.Decimal     `json:"rmdTaxReduction"`
	NetBenefit          decimal.Decimal     `json:"netBenefit"`
	ROI                 decimal.Decimal     `json:"roi"`
	Recommendation      string              `json:"recommendation"`
	SensitivityAnalysis SensitivityAnalysis `json:"sensitivityAnalysis"`
}

// SensitivityAnalysis shows how results change with different conversion amounts
type SensitivityAnalysis struct {
	Plus20Percent  decimal.Decimal `json:"plus20Percent"`
	Minus20Percent decimal.Decimal `json:"minus20Percent"`
	OptimalRange   string          `json:"optimalRange"`
}

// BracketRoom represents available room in a tax bracket
type BracketRoom struct {
	BracketNumber int             `json:"bracketNumber"`
	RoomAmount    decimal.Decimal `json:"roomAmount"`
	CurrentIncome decimal.Decimal `json:"currentIncome"`
	BracketEdge   decimal.Decimal `json:"bracketEdge"`
}

// TaxBracketInfo represents information about a tax bracket
type TaxBracketInfo struct {
	BracketNumber int             `json:"bracketNumber"`
	MinIncome     decimal.Decimal `json:"minIncome"`
	MaxIncome     decimal.Decimal `json:"maxIncome"`
	Rate          decimal.Decimal `json:"rate"`
}

// RothConversionConstraints defines limits and constraints for conversion planning
type RothConversionConstraints struct {
	MinConversionAmount decimal.Decimal `json:"minConversionAmount"`
	MaxConversionAmount decimal.Decimal `json:"maxConversionAmount"`
	MaxTotalConversions decimal.Decimal `json:"maxTotalConversions"`
	MinYearsBetween     int             `json:"minYearsBetween"`
	MaxYearsBetween     int             `json:"maxYearsBetween"`
	Participant         string          `json:"participant"`
}

// DefaultRothConversionConstraints returns sensible defaults for conversion planning
func DefaultRothConversionConstraints(participant string) RothConversionConstraints {
	return RothConversionConstraints{
		MinConversionAmount: decimal.NewFromInt(1000),    // $1,000 minimum
		MaxConversionAmount: decimal.NewFromInt(200000),  // $200,000 maximum per year
		MaxTotalConversions: decimal.NewFromInt(1000000), // $1M maximum total
		MinYearsBetween:     0,                           // No minimum gap
		MaxYearsBetween:     5,                           // Max 5 years between conversions
		Participant:         participant,
	}
}

// Validate checks if the constraints are valid
func (c RothConversionConstraints) Validate() error {
	if c.Participant == "" {
		return fmt.Errorf("participant name cannot be empty")
	}

	if c.MinConversionAmount.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("min conversion amount must be positive")
	}

	if c.MaxConversionAmount.LessThanOrEqual(c.MinConversionAmount) {
		return fmt.Errorf("max conversion amount must be greater than min")
	}

	if c.MaxTotalConversions.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("max total conversions must be positive")
	}

	if c.MinYearsBetween < 0 {
		return fmt.Errorf("min years between cannot be negative")
	}

	if c.MaxYearsBetween < c.MinYearsBetween {
		return fmt.Errorf("max years between must be >= min years between")
	}

	return nil
}

// IsValidYear checks if a year is within the conversion window
func (yr YearRange) IsValidYear(year int) bool {
	return year >= yr.Start && year <= yr.End
}

// Years returns all years in the range
func (yr YearRange) Years() []int {
	years := make([]int, 0, yr.End-yr.Start+1)
	for year := yr.Start; year <= yr.End; year++ {
		years = append(years, year)
	}
	return years
}

// String returns a string representation of the year range
func (yr YearRange) String() string {
	return fmt.Sprintf("%d-%d", yr.Start, yr.End)
}
