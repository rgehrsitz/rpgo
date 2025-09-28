package config

import (
	"fmt"
	"os"

	"github.com/rgehrsitz/rpgo/internal/domain"
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
	// Generic-only validation path
	if err := ip.validateGenericConfiguration(config); err != nil {
		return fmt.Errorf("configuration validation failed: %w", err)
	}
	if err := ip.validateGlobalAssumptions(&config.GlobalAssumptions); err != nil {
		return fmt.Errorf("global assumptions validation failed: %w", err)
	}
	return nil
}

// validateLegacyConfiguration validates the legacy robert/dawn format
// (Removed legacy validation logic)

// validateNewConfiguration validates the new generic household format
func (ip *InputParser) validateGenericConfiguration(config *domain.Configuration) error {
	// Validate household
	if config.Household == nil {
		return fmt.Errorf("household is required")
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
	if len(config.Scenarios) == 0 {
		return fmt.Errorf("no scenarios provided")
	}

	for i, scenario := range config.Scenarios {
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
