package config

import (
	"fmt"
	"os"
	"time"

	"github.com/rpgo/retirement-calculator/internal/domain"
	"github.com/shopspring/decimal"
	"gopkg.in/yaml.v3"
)

// InputParser handles parsing of input configuration files
type InputParser struct{}

// NewInputParser creates a new input parser
func NewInputParser() *InputParser {
	return &InputParser{}
}

// LoadFromFile loads configuration from a YAML or JSON file
func (ip *InputParser) LoadFromFile(filename string) (*domain.Configuration, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	var config domain.Configuration
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Validate the configuration
	if err := ip.ValidateConfiguration(&config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &config, nil
}

// ValidateConfiguration validates the loaded configuration
func (ip *InputParser) ValidateConfiguration(config *domain.Configuration) error {
	// Determine which format is being used
	isLegacy := config.IsLegacyFormat()
	isNew := config.IsNewFormat()

	if !isLegacy && !isNew {
		return fmt.Errorf("configuration must use either legacy format (personal_details) or new format (household)")
	}

	if isLegacy && isNew {
		return fmt.Errorf("configuration cannot use both legacy and new formats simultaneously")
	}

	// Validate based on format
	if isLegacy {
		if err := ip.validateLegacyConfiguration(config); err != nil {
			return fmt.Errorf("legacy configuration validation failed: %w", err)
		}
	} else {
		if err := ip.validateNewConfiguration(config); err != nil {
			return fmt.Errorf("new configuration validation failed: %w", err)
		}
	}

	// Validate global assumptions (common to both formats)
	if err := ip.validateGlobalAssumptions(&config.GlobalAssumptions); err != nil {
		return fmt.Errorf("global assumptions validation failed: %w", err)
	}

	return nil
}

// validateLegacyConfiguration validates the legacy robert/dawn format
func (ip *InputParser) validateLegacyConfiguration(config *domain.Configuration) error {
	// Validate personal details
	if len(config.PersonalDetails) == 0 {
		return fmt.Errorf("no personal details provided")
	}

	// Check for required employees
	if _, exists := config.PersonalDetails["robert"]; !exists {
		return fmt.Errorf("robert employee details are required")
	}
	if _, exists := config.PersonalDetails["dawn"]; !exists {
		return fmt.Errorf("dawn employee details are required")
	}

	// Validate each employee
	for name, employee := range config.PersonalDetails {
		if err := ip.validateEmployee(name, &employee); err != nil {
			return fmt.Errorf("employee %s validation failed: %w", name, err)
		}
	}

	// Validate scenarios
	if len(config.Scenarios) == 0 {
		return fmt.Errorf("no scenarios provided")
	}

	for i, scenario := range config.Scenarios {
		if err := ip.validateScenario(i, &scenario); err != nil {
			return fmt.Errorf("scenario %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateNewConfiguration validates the new generic household format
func (ip *InputParser) validateNewConfiguration(config *domain.Configuration) error {
	// Validate household
	if config.Household == nil {
		return fmt.Errorf("household is required")
	}

	if len(config.Household.Participants) == 0 {
		return fmt.Errorf("household must have at least one participant")
	}

	// Validate filing status
	validFilingStatuses := map[string]bool{
		"single":                 true,
		"married_filing_jointly": true,
	}
	if !validFilingStatuses[config.Household.FilingStatus] {
		return fmt.Errorf("invalid filing status: %s (must be 'single' or 'married_filing_jointly')", config.Household.FilingStatus)
	}

	// Validate each participant
	for i, participant := range config.Household.Participants {
		if err := ip.validateParticipant(i, &participant); err != nil {
			return fmt.Errorf("participant %d (%s) validation failed: %w", i, participant.Name, err)
		}
	}

	// Validate FEHB logic - only one person can be primary holder
	fehbHolders := 0
	for _, p := range config.Household.Participants {
		if p.IsPrimaryFEHBHolder {
			fehbHolders++
		}
	}
	if fehbHolders > 1 {
		return fmt.Errorf("only one participant can be the primary FEHB holder")
	}

	// Validate scenarios
	if len(config.GenericScenarios) == 0 {
		return fmt.Errorf("no scenarios provided")
	}

	for i, scenario := range config.GenericScenarios {
		if err := ip.validateGenericScenario(i, &scenario, config.Household); err != nil {
			return fmt.Errorf("scenario %d validation failed: %w", i, err)
		}
	}

	return nil
}

// validateParticipant validates a single participant
func (ip *InputParser) validateParticipant(index int, participant *domain.Participant) error {
	// Validate required fields
	if participant.Name == "" {
		return fmt.Errorf("name is required")
	}
	if participant.BirthDate.IsZero() {
		return fmt.Errorf("birth date is required")
	}

	// Validate Social Security benefits (required for all participants)
	if participant.SSBenefitFRA.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("social security benefit at FRA must be positive")
	}
	if participant.SSBenefit62.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("social security benefit at 62 must be positive")
	}
	if participant.SSBenefit70.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("social security benefit at 70 must be positive")
	}

	// Validate Social Security benefit progression
	if participant.SSBenefit62.GreaterThan(participant.SSBenefitFRA) {
		return fmt.Errorf("SS benefit at 62 cannot be greater than at FRA")
	}
	if participant.SSBenefitFRA.GreaterThan(participant.SSBenefit70) {
		return fmt.Errorf("SS benefit at FRA cannot be greater than at 70")
	}

	// Federal employee validations
	if participant.IsFederal {
		if err := ip.validateFederalParticipant(participant); err != nil {
			return fmt.Errorf("federal employee validation failed: %w", err)
		}
	}

	// External pension validations
	if participant.ExternalPension != nil {
		if err := ip.validateExternalPension(participant.ExternalPension); err != nil {
			return fmt.Errorf("external pension validation failed: %w", err)
		}
	}

	return nil
}

// validateFederalParticipant validates federal employee specific fields
func (ip *InputParser) validateFederalParticipant(participant *domain.Participant) error {
	// Required fields for federal employees
	if participant.HireDate == nil {
		return fmt.Errorf("hire date is required for federal employees")
	}
	if participant.CurrentSalary == nil {
		return fmt.Errorf("current salary is required for federal employees")
	}
	if participant.High3Salary == nil {
		return fmt.Errorf("high 3 salary is required for federal employees")
	}
	if participant.TSPBalanceTraditional == nil {
		return fmt.Errorf("TSP traditional balance is required for federal employees")
	}
	if participant.TSPBalanceRoth == nil {
		return fmt.Errorf("TSP Roth balance is required for federal employees")
	}
	if participant.TSPContributionPercent == nil {
		return fmt.Errorf("TSP contribution percent is required for federal employees")
	}
	if participant.SurvivorBenefitElectionPercent == nil {
		return fmt.Errorf("survivor benefit election percent is required for federal employees")
	}

	// Validate values
	if participant.CurrentSalary.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("current salary must be positive")
	}
	if participant.High3Salary.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("high 3 salary must be positive")
	}
	if participant.TSPBalanceTraditional.LessThan(decimal.Zero) {
		return fmt.Errorf("TSP traditional balance cannot be negative")
	}
	if participant.TSPBalanceRoth.LessThan(decimal.Zero) {
		return fmt.Errorf("TSP Roth balance cannot be negative")
	}
	if participant.TSPContributionPercent.LessThan(decimal.Zero) || participant.TSPContributionPercent.GreaterThan(decimal.NewFromFloat(1.0)) {
		return fmt.Errorf("TSP contribution percent must be between 0 and 1")
	}
	if participant.SurvivorBenefitElectionPercent.LessThan(decimal.Zero) || participant.SurvivorBenefitElectionPercent.GreaterThan(decimal.NewFromFloat(1.0)) {
		return fmt.Errorf("survivor benefit election percent must be between 0 and 1")
	}

	// Validate date logic
	if participant.BirthDate.After(*participant.HireDate) {
		return fmt.Errorf("birth date cannot be after hire date")
	}

	// FEHB validation
	if participant.IsPrimaryFEHBHolder {
		if participant.FEHBPremiumPerPayPeriod == nil {
			return fmt.Errorf("FEHB premium per pay period is required for primary FEHB holder")
		}
		if participant.FEHBPremiumPerPayPeriod.LessThan(decimal.Zero) {
			return fmt.Errorf("FEHB premium per pay period cannot be negative")
		}
	}

	return nil
}

// validateExternalPension validates external pension details
func (ip *InputParser) validateExternalPension(pension *domain.ExternalPension) error {
	if pension.MonthlyBenefit.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("monthly benefit must be positive")
	}
	if pension.StartAge < 50 || pension.StartAge > 75 {
		return fmt.Errorf("start age must be between 50 and 75")
	}
	if pension.COLAAdjustment.LessThan(decimal.Zero) {
		return fmt.Errorf("COLA adjustment cannot be negative")
	}
	if pension.SurvivorBenefit.LessThan(decimal.Zero) || pension.SurvivorBenefit.GreaterThan(decimal.NewFromFloat(1.0)) {
		return fmt.Errorf("survivor benefit must be between 0 and 1")
	}
	return nil
}

// validateGenericScenario validates a generic scenario
func (ip *InputParser) validateGenericScenario(index int, scenario *domain.GenericScenario, household *domain.Household) error {
	if scenario.Name == "" {
		return fmt.Errorf("scenario name is required")
	}

	if len(scenario.ParticipantScenarios) == 0 {
		return fmt.Errorf("at least one participant scenario is required")
	}

	// Validate each participant scenario
	for name, participantScenario := range scenario.ParticipantScenarios {
		// Check that participant exists in household
		found := false
		for _, p := range household.Participants {
			if p.Name == name {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("participant scenario references unknown participant: %s", name)
		}

		if err := ip.validateParticipantScenario(name, &participantScenario); err != nil {
			return fmt.Errorf("participant scenario %s validation failed: %w", name, err)
		}
	}

	// Validate mortality if present
	if scenario.Mortality != nil {
		for participantName, mortalitySpec := range scenario.Mortality.Participants {
			// Check that participant exists
			found := false
			for _, p := range household.Participants {
				if p.Name == participantName {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("mortality specification references unknown participant: %s", participantName)
			}

			if mortalitySpec.DeathDate != nil && mortalitySpec.DeathAge != nil {
				return fmt.Errorf("mortality for %s: specify either death_date or death_age, not both", participantName)
			}
		}

		if scenario.Mortality.Assumptions != nil {
			if !scenario.Mortality.Assumptions.SurvivorSpendingFactor.IsZero() && (scenario.Mortality.Assumptions.SurvivorSpendingFactor.LessThan(decimal.NewFromFloat(0.4)) || scenario.Mortality.Assumptions.SurvivorSpendingFactor.GreaterThan(decimal.NewFromFloat(1.0))) {
				return fmt.Errorf("survivor_spending_factor must be between 0.4 and 1.0")
			}
			if scenario.Mortality.Assumptions.TSPSpousalTransfer != "" && scenario.Mortality.Assumptions.TSPSpousalTransfer != "merge" && scenario.Mortality.Assumptions.TSPSpousalTransfer != "separate" {
				return fmt.Errorf("tsp_spousal_transfer must be 'merge' or 'separate'")
			}
			if scenario.Mortality.Assumptions.FilingStatusSwitch != "" && scenario.Mortality.Assumptions.FilingStatusSwitch != "next_year" && scenario.Mortality.Assumptions.FilingStatusSwitch != "immediate" {
				return fmt.Errorf("filing_status_switch must be 'next_year' or 'immediate'")
			}
		}
	}

	return nil
}

// validateParticipantScenario validates a participant scenario
func (ip *InputParser) validateParticipantScenario(participantName string, scenario *domain.ParticipantScenario) error {
	if scenario.ParticipantName == "" {
		return fmt.Errorf("participant name is required")
	}
	if scenario.ParticipantName != participantName {
		return fmt.Errorf("participant scenario name mismatch: expected %s, got %s", participantName, scenario.ParticipantName)
	}

	if scenario.SSStartAge < 62 || scenario.SSStartAge > 70 {
		return fmt.Errorf("social security start age must be between 62 and 70")
	}

	// TSP withdrawal validation (only for federal employees)
	if scenario.TSPWithdrawalStrategy != "" {
		validStrategies := map[string]bool{
			"4_percent_rule":      true,
			"need_based":          true,
			"variable_percentage": true,
		}
		if !validStrategies[scenario.TSPWithdrawalStrategy] {
			return fmt.Errorf("TSP withdrawal strategy must be '4_percent_rule', 'need_based', or 'variable_percentage'")
		}

		if scenario.TSPWithdrawalStrategy == "need_based" && scenario.TSPWithdrawalTargetMonthly == nil {
			return fmt.Errorf("TSP withdrawal target monthly is required for need_based strategy")
		}
		if scenario.TSPWithdrawalStrategy == "variable_percentage" && scenario.TSPWithdrawalRate == nil {
			return fmt.Errorf("TSP withdrawal rate is required for variable_percentage strategy")
		}
		if scenario.TSPWithdrawalTargetMonthly != nil && scenario.TSPWithdrawalTargetMonthly.LessThanOrEqual(decimal.Zero) {
			return fmt.Errorf("TSP withdrawal target monthly must be positive")
		}
		if scenario.TSPWithdrawalRate != nil && (scenario.TSPWithdrawalRate.LessThan(decimal.Zero) || scenario.TSPWithdrawalRate.GreaterThan(decimal.NewFromFloat(0.2))) {
			return fmt.Errorf("TSP withdrawal rate must be between 0 and 20%%")
		}
	}

	return nil
}

// validateEmployee validates a single employee's data
func (ip *InputParser) validateEmployee(_ string, employee *domain.Employee) error {
	// Validate required fields
	if employee.BirthDate.IsZero() {
		return fmt.Errorf("birth date is required")
	}
	if employee.HireDate.IsZero() {
		return fmt.Errorf("hire date is required")
	}
	if employee.CurrentSalary.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("current salary must be positive")
	}
	if employee.High3Salary.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("high 3 salary must be positive")
	}
	if employee.TSPBalanceTraditional.LessThan(decimal.Zero) {
		return fmt.Errorf("TSP traditional balance cannot be negative")
	}
	if employee.TSPBalanceRoth.LessThan(decimal.Zero) {
		return fmt.Errorf("TSP Roth balance cannot be negative")
	}
	if employee.TSPContributionPercent.LessThan(decimal.Zero) || employee.TSPContributionPercent.GreaterThan(decimal.NewFromFloat(1.0)) {
		return fmt.Errorf("TSP contribution percent must be between 0 and 1")
	}
	if employee.SSBenefitFRA.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("social security benefit at FRA must be positive")
	}
	if employee.SSBenefit62.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("social security benefit at 62 must be positive")
	}
	if employee.SSBenefit70.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("social security benefit at 70 must be positive")
	}
	if employee.FEHBPremiumPerPayPeriod.LessThan(decimal.Zero) {
		return fmt.Errorf("FEHB premium per pay period cannot be negative")
	}
	if employee.SurvivorBenefitElectionPercent.LessThan(decimal.Zero) || employee.SurvivorBenefitElectionPercent.GreaterThan(decimal.NewFromFloat(1.0)) {
		return fmt.Errorf("survivor benefit election percent must be between 0 and 1")
	}

	// Validate date logic
	if employee.BirthDate.After(employee.HireDate) {
		return fmt.Errorf("birth date cannot be after hire date")
	}

	// Validate Social Security benefit progression
	if employee.SSBenefit62.GreaterThan(employee.SSBenefitFRA) {
		return fmt.Errorf("SS benefit at 62 cannot be greater than at FRA")
	}
	if employee.SSBenefitFRA.GreaterThan(employee.SSBenefit70) {
		return fmt.Errorf("SS benefit at FRA cannot be greater than at 70")
	}

	return nil
}

// validateGlobalAssumptions validates global assumptions
func (ip *InputParser) validateGlobalAssumptions(assumptions *domain.GlobalAssumptions) error {
	if assumptions.InflationRate.LessThan(decimal.NewFromFloat(-0.10)) {
		return fmt.Errorf("inflation rate cannot be less than -10%% (extreme deflation)")
	}
	if assumptions.FEHBPremiumInflation.LessThan(decimal.Zero) {
		return fmt.Errorf("FEHB premium inflation cannot be negative")
	}
	if assumptions.TSPReturnPreRetirement.LessThan(decimal.NewFromFloat(-1.0)) {
		return fmt.Errorf("TSP return pre-retirement cannot be less than -100%%")
	}
	if assumptions.TSPReturnPostRetirement.LessThan(decimal.NewFromFloat(-1.0)) {
		return fmt.Errorf("TSP return post-retirement cannot be less than -100%%")
	}
	if assumptions.COLAGeneralRate.LessThan(decimal.Zero) {
		return fmt.Errorf("COLA general rate cannot be negative")
	}
	if assumptions.ProjectionYears <= 0 || assumptions.ProjectionYears > 50 {
		return fmt.Errorf("projection years must be between 1 and 50")
	}

	// Validate location
	if assumptions.CurrentLocation.State == "" {
		return fmt.Errorf("state is required")
	}

	return nil
}

// validateScenario validates a single scenario
func (ip *InputParser) validateScenario(_ int, scenario *domain.Scenario) error {
	if scenario.Name == "" {
		return fmt.Errorf("scenario name is required")
	}

	// Validate Robert's scenario
	if err := ip.validateRetirementScenario("robert", &scenario.Robert); err != nil {
		return fmt.Errorf("robert scenario validation failed: %w", err)
	}

	// Validate Dawn's scenario
	if err := ip.validateRetirementScenario("dawn", &scenario.Dawn); err != nil {
		return fmt.Errorf("dawn scenario validation failed: %w", err)
	}

	// Validate optional mortality block
	if scenario.Mortality != nil {
		if scenario.Mortality.Robert != nil {
			if scenario.Mortality.Robert.DeathDate != nil && scenario.Mortality.Robert.DeathAge != nil {
				return fmt.Errorf("mortality.robert: specify either death_date or death_age, not both")
			}
		}
		if scenario.Mortality.Dawn != nil {
			if scenario.Mortality.Dawn.DeathDate != nil && scenario.Mortality.Dawn.DeathAge != nil {
				return fmt.Errorf("mortality.dawn: specify either death_date or death_age, not both")
			}
		}
		if scenario.Mortality.Assumptions != nil {
			if !scenario.Mortality.Assumptions.SurvivorSpendingFactor.IsZero() && (scenario.Mortality.Assumptions.SurvivorSpendingFactor.LessThan(decimal.NewFromFloat(0.4)) || scenario.Mortality.Assumptions.SurvivorSpendingFactor.GreaterThan(decimal.NewFromFloat(1.0))) {
				return fmt.Errorf("mortality.assumptions.survivor_spending_factor must be between 0.4 and 1.0")
			}
			if scenario.Mortality.Assumptions.TSPSpousalTransfer != "" && scenario.Mortality.Assumptions.TSPSpousalTransfer != "merge" && scenario.Mortality.Assumptions.TSPSpousalTransfer != "separate" {
				return fmt.Errorf("mortality.assumptions.tsp_spousal_transfer must be 'merge' or 'separate'")
			}
			if scenario.Mortality.Assumptions.FilingStatusSwitch != "" && scenario.Mortality.Assumptions.FilingStatusSwitch != "next_year" && scenario.Mortality.Assumptions.FilingStatusSwitch != "immediate" {
				return fmt.Errorf("mortality.assumptions.filing_status_switch must be 'next_year' or 'immediate'")
			}
		}
	}

	return nil
}

// validateRetirementScenario validates a retirement scenario for an employee
func (ip *InputParser) validateRetirementScenario(_ string, scenario *domain.RetirementScenario) error {
	if scenario.EmployeeName == "" {
		return fmt.Errorf("employee name is required")
	}
	if scenario.RetirementDate.IsZero() {
		return fmt.Errorf("retirement date is required")
	}
	if scenario.SSStartAge < 62 || scenario.SSStartAge > 70 {
		return fmt.Errorf("social security start age must be between 62 and 70")
	}
	if scenario.TSPWithdrawalStrategy != "4_percent_rule" && scenario.TSPWithdrawalStrategy != "need_based" && scenario.TSPWithdrawalStrategy != "variable_percentage" {
		return fmt.Errorf("TSP withdrawal strategy must be '4_percent_rule', 'need_based', or 'variable_percentage'")
	}
	if scenario.TSPWithdrawalStrategy == "need_based" && scenario.TSPWithdrawalTargetMonthly == nil {
		return fmt.Errorf("TSP withdrawal target monthly is required for need_based strategy")
	}
	if scenario.TSPWithdrawalStrategy == "variable_percentage" && scenario.TSPWithdrawalRate == nil {
		return fmt.Errorf("TSP withdrawal rate is required for variable_percentage strategy")
	}
	if scenario.TSPWithdrawalTargetMonthly != nil && scenario.TSPWithdrawalTargetMonthly.LessThanOrEqual(decimal.Zero) {
		return fmt.Errorf("TSP withdrawal target monthly must be positive")
	}
	if scenario.TSPWithdrawalRate != nil && (scenario.TSPWithdrawalRate.LessThan(decimal.Zero) || scenario.TSPWithdrawalRate.GreaterThan(decimal.NewFromFloat(0.2))) {
		return fmt.Errorf("TSP withdrawal rate must be between 0 and 20%%")
	}

	return nil
}

// CreateExampleConfiguration creates an example configuration file
func (ip *InputParser) CreateExampleConfiguration() *domain.Configuration {
	robertBirthDate, _ := time.Parse("2006-01-02", "1963-06-15")
	robertHireDate, _ := time.Parse("2006-01-02", "1985-03-20")
	dawnBirthDate, _ := time.Parse("2006-01-02", "1965-08-22")
	dawnHireDate, _ := time.Parse("2006-01-02", "1988-07-10")

	robertRetirementDate, _ := time.Parse("2006-01-02", "2025-12-31")
	dawnRetirementDate, _ := time.Parse("2006-01-02", "2025-12-31")

	return &domain.Configuration{
		PersonalDetails: map[string]domain.Employee{
			"robert": {
				Name:                           "Robert",
				BirthDate:                      robertBirthDate,
				HireDate:                       robertHireDate,
				CurrentSalary:                  decimal.NewFromInt(95000),
				High3Salary:                    decimal.NewFromInt(93000),
				TSPBalanceTraditional:          decimal.NewFromInt(450000),
				TSPBalanceRoth:                 decimal.NewFromInt(50000),
				TSPContributionPercent:         decimal.NewFromFloat(0.15),
				SSBenefitFRA:                   decimal.NewFromInt(2400),
				SSBenefit62:                    decimal.NewFromInt(1680),
				SSBenefit70:                    decimal.NewFromInt(2976),
				FEHBPremiumPerPayPeriod:        decimal.NewFromInt(875),
				SurvivorBenefitElectionPercent: decimal.Zero,
			},
			"dawn": {
				Name:                           "Dawn",
				BirthDate:                      dawnBirthDate,
				HireDate:                       dawnHireDate,
				CurrentSalary:                  decimal.NewFromInt(87000),
				High3Salary:                    decimal.NewFromInt(85000),
				TSPBalanceTraditional:          decimal.NewFromInt(380000),
				TSPBalanceRoth:                 decimal.NewFromInt(45000),
				TSPContributionPercent:         decimal.NewFromFloat(0.12),
				SSBenefitFRA:                   decimal.NewFromInt(2200),
				SSBenefit62:                    decimal.NewFromInt(1540),
				SSBenefit70:                    decimal.NewFromInt(2728),
				FEHBPremiumPerPayPeriod:        decimal.Zero, // Covered under Robert's FEHB
				SurvivorBenefitElectionPercent: decimal.Zero,
			},
		},
		GlobalAssumptions: domain.GlobalAssumptions{
			InflationRate:           decimal.NewFromFloat(0.025),
			FEHBPremiumInflation:    decimal.NewFromFloat(0.065),
			TSPReturnPreRetirement:  decimal.NewFromFloat(0.055),
			TSPReturnPostRetirement: decimal.NewFromFloat(0.045),
			COLAGeneralRate:         decimal.NewFromFloat(0.025),
			ProjectionYears:         25,
			CurrentLocation: domain.Location{
				State:        "Pennsylvania",
				County:       "Bucks",
				Municipality: "Upper Makefield Township",
			},
		},
		Scenarios: []domain.Scenario{
			{
				Name: "Early Retirement 2025",
				Robert: domain.RetirementScenario{
					EmployeeName:               "robert",
					RetirementDate:             robertRetirementDate,
					SSStartAge:                 62,
					TSPWithdrawalStrategy:      "4_percent_rule",
					TSPWithdrawalTargetMonthly: nil,
				},
				Dawn: domain.RetirementScenario{
					EmployeeName:               "dawn",
					RetirementDate:             dawnRetirementDate,
					SSStartAge:                 62,
					TSPWithdrawalStrategy:      "4_percent_rule",
					TSPWithdrawalTargetMonthly: nil,
				},
			},
			{
				Name: "Delayed Retirement 2028",
				Robert: domain.RetirementScenario{
					EmployeeName:               "robert",
					RetirementDate:             robertRetirementDate.AddDate(3, 0, 0),
					SSStartAge:                 67,
					TSPWithdrawalStrategy:      "need_based",
					TSPWithdrawalTargetMonthly: &[]decimal.Decimal{decimal.NewFromInt(3000)}[0],
				},
				Dawn: domain.RetirementScenario{
					EmployeeName:               "dawn",
					RetirementDate:             dawnRetirementDate.AddDate(3, 0, 0),
					SSStartAge:                 62,
					TSPWithdrawalStrategy:      "4_percent_rule",
					TSPWithdrawalTargetMonthly: nil,
				},
			},
		},
	}
}
