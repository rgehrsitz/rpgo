package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestNewInputParser(t *testing.T) {
	parser := NewInputParser()
	assert.NotNil(t, parser, "Should create input parser")
}

func TestInputParser_LoadFromFile_FileNotFound(t *testing.T) {
	parser := NewInputParser()

	config, err := parser.LoadFromFile("nonexistent.yaml")

	assert.Error(t, err, "Should error for nonexistent file")
	assert.Nil(t, config, "Should return nil config")
	assert.Contains(t, err.Error(), "failed to read file", "Should have specific error message")
}

func TestInputParser_LoadFromFile_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(invalidFile, []byte("invalid: yaml: content: [unclosed"), 0644)
	assert.NoError(t, err)

	parser := NewInputParser()
	config, err := parser.LoadFromFile(invalidFile)

	assert.Error(t, err, "Should error for invalid YAML")
	assert.Nil(t, config, "Should return nil config")
	assert.Contains(t, err.Error(), "failed to parse YAML", "Should have specific error message")
}

func TestInputParser_LoadFromFile_ValidYAML(t *testing.T) {
	// Create a temporary file with valid YAML
	tmpDir := t.TempDir()
	validFile := filepath.Join(tmpDir, "valid.yaml")

	validYAML := `
household:
  filing_status: "single"
  participants:
    - name: "John Doe"
      is_federal: true
      birth_date: "1970-01-01T00:00:00Z"
      hire_date: "2000-01-01T00:00:00Z"
      current_salary: 100000
      high_3_salary: 95000
      tsp_balance_traditional: 400000
      tsp_balance_roth: 50000
      tsp_contribution_percent: 0.15
      ss_benefit_fra: 2500
      ss_benefit_62: 1750
      ss_benefit_70: 3100
      fehb_premium_per_pay_period: 500
      is_primary_fehb_holder: true
      survivor_benefit_election_percent: 0.0

scenarios:
  - name: "Test Scenario"
    participant_scenarios:
      "John Doe":
        participant_name: "John Doe"
        retirement_date: "2030-01-01T00:00:00Z"
        ss_start_age: 62
        tsp_withdrawal_strategy: "4_percent_rule"

global_assumptions:
  inflation_rate: 0.025
  fehb_premium_inflation: 0.06
  tsp_return_pre_retirement: 0.07
  tsp_return_post_retirement: 0.06
  cola_general_rate: 0.025
  projection_years: 25
  current_location:
    state: "TestState"
    county: "TestCounty"
    municipality: "TestMunicipality"
`

	err := os.WriteFile(validFile, []byte(validYAML), 0644)
	assert.NoError(t, err)

	parser := NewInputParser()
	config, err := parser.LoadFromFile(validFile)

	assert.NoError(t, err, "Should not error for valid YAML")
	assert.NotNil(t, config, "Should return config")
	assert.Equal(t, "single", config.Household.FilingStatus, "Should parse filing status")
	assert.Len(t, config.Household.Participants, 1, "Should parse participants")
	assert.Equal(t, "John Doe", config.Household.Participants[0].Name, "Should parse participant name")
	assert.Len(t, config.Scenarios, 1, "Should parse scenarios")
	assert.Equal(t, "Test Scenario", config.Scenarios[0].Name, "Should parse scenario name")
}

func TestInputParser_ValidateConfiguration_NilHousehold(t *testing.T) {
	parser := NewInputParser()

	config := &domain.Configuration{
		Household: nil,
		Scenarios: []domain.GenericScenario{
			{Name: "test"},
		},
	}

	err := parser.ValidateConfiguration(config)
	assert.Error(t, err, "Should error for nil household")
	assert.Contains(t, err.Error(), "household is required", "Should have specific error message")
}

func TestInputParser_ValidateConfiguration_NoScenarios(t *testing.T) {
	parser := NewInputParser()

	config := &domain.Configuration{
		Household: &domain.Household{
			Participants: []domain.Participant{
				{
					Name:         "test",
					BirthDate:    time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
					SSBenefitFRA: decimal.NewFromInt(2500),
					SSBenefit62:  decimal.NewFromInt(1750),
					SSBenefit70:  decimal.NewFromInt(3100),
				},
			},
		},
		Scenarios: []domain.GenericScenario{},
	}

	err := parser.ValidateConfiguration(config)
	assert.Error(t, err, "Should error for no scenarios")
	assert.Contains(t, err.Error(), "no scenarios provided", "Should have specific error message")
}

func TestInputParser_ValidateParticipant_EmptyName(t *testing.T) {
	parser := NewInputParser()

	participant := &domain.Participant{
		Name:         "",
		BirthDate:    time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		SSBenefitFRA: decimal.NewFromInt(2500),
		SSBenefit62:  decimal.NewFromInt(1750),
		SSBenefit70:  decimal.NewFromInt(3100),
	}

	err := parser.validateParticipant(0, participant)
	assert.Error(t, err, "Should error for empty name")
	assert.Contains(t, err.Error(), "name is required", "Should have specific error message")
}

func TestInputParser_ValidateParticipant_ZeroBirthDate(t *testing.T) {
	parser := NewInputParser()

	participant := &domain.Participant{
		Name:         "test",
		BirthDate:    time.Time{},
		SSBenefitFRA: decimal.NewFromInt(2500),
		SSBenefit62:  decimal.NewFromInt(1750),
		SSBenefit70:  decimal.NewFromInt(3100),
	}

	err := parser.validateParticipant(0, participant)
	assert.Error(t, err, "Should error for zero birth date")
	assert.Contains(t, err.Error(), "birth date is required", "Should have specific error message")
}

func TestInputParser_ValidateParticipant_InvalidSSBenefits(t *testing.T) {
	parser := NewInputParser()

	participant := &domain.Participant{
		Name:         "test",
		BirthDate:    time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		SSBenefitFRA: decimal.Zero,
		SSBenefit62:  decimal.NewFromInt(1750),
		SSBenefit70:  decimal.NewFromInt(3100),
	}

	err := parser.validateParticipant(0, participant)
	assert.Error(t, err, "Should error for zero SS benefit FRA")
	assert.Contains(t, err.Error(), "social security benefit at FRA must be positive", "Should have specific error message")
}

func TestInputParser_ValidateParticipant_InvalidSSProgression(t *testing.T) {
	parser := NewInputParser()

	participant := &domain.Participant{
		Name:         "test",
		BirthDate:    time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		SSBenefitFRA: decimal.NewFromInt(2500),
		SSBenefit62:  decimal.NewFromInt(3000), // Higher than FRA
		SSBenefit70:  decimal.NewFromInt(3100),
	}

	err := parser.validateParticipant(0, participant)
	assert.Error(t, err, "Should error for invalid SS progression")
	assert.Contains(t, err.Error(), "SS benefit at 62 cannot be greater than at FRA", "Should have specific error message")
}

func TestInputParser_ValidateFederalParticipant_MissingRequiredFields(t *testing.T) {
	parser := NewInputParser()

	participant := &domain.Participant{
		Name:         "test",
		BirthDate:    time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		IsFederal:    true,
		SSBenefitFRA: decimal.NewFromInt(2500),
		SSBenefit62:  decimal.NewFromInt(1750),
		SSBenefit70:  decimal.NewFromInt(3100),
		// Missing required federal fields
	}

	err := parser.validateFederalParticipant(participant)
	assert.Error(t, err, "Should error for missing hire date")
	assert.Contains(t, err.Error(), "hire date is required for federal employees", "Should have specific error message")
}

func TestInputParser_ValidateFederalParticipant_InvalidValues(t *testing.T) {
	parser := NewInputParser()

	hireDate := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	participant := &domain.Participant{
		Name:                           "test",
		BirthDate:                      time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC),
		IsFederal:                      true,
		HireDate:                       &hireDate,
		CurrentSalary:                  decimalPtr(decimal.Zero), // Invalid
		High3Salary:                    decimalPtr(decimal.NewFromInt(95000)),
		TSPBalanceTraditional:          decimalPtr(decimal.NewFromInt(400000)),
		TSPBalanceRoth:                 decimalPtr(decimal.NewFromInt(50000)),
		TSPContributionPercent:         decimalPtr(decimal.NewFromFloat(0.15)),
		SSBenefitFRA:                   decimal.NewFromInt(2500),
		SSBenefit62:                    decimal.NewFromInt(1750),
		SSBenefit70:                    decimal.NewFromInt(3100),
		SurvivorBenefitElectionPercent: decimalPtr(decimal.Zero),
	}

	err := parser.validateFederalParticipant(participant)
	assert.Error(t, err, "Should error for invalid current salary")
	assert.Contains(t, err.Error(), "current salary must be positive", "Should have specific error message")
}

func TestInputParser_ValidateExternalPension_InvalidValues(t *testing.T) {
	parser := NewInputParser()

	pension := &domain.ExternalPension{
		MonthlyBenefit:  decimal.Zero, // Invalid
		StartAge:        50,
		COLAAdjustment:  decimal.NewFromFloat(0.02),
		SurvivorBenefit: decimal.NewFromFloat(0.5),
	}

	err := parser.validateExternalPension(pension)
	assert.Error(t, err, "Should error for invalid monthly benefit")
	assert.Contains(t, err.Error(), "monthly benefit must be positive", "Should have specific error message")
}

func TestInputParser_ValidateExternalPension_InvalidAge(t *testing.T) {
	parser := NewInputParser()

	pension := &domain.ExternalPension{
		MonthlyBenefit:  decimal.NewFromInt(2000),
		StartAge:        40, // Invalid
		COLAAdjustment:  decimal.NewFromFloat(0.02),
		SurvivorBenefit: decimal.NewFromFloat(0.5),
	}

	err := parser.validateExternalPension(pension)
	assert.Error(t, err, "Should error for invalid start age")
	assert.Contains(t, err.Error(), "start age must be between 50 and 75", "Should have specific error message")
}

func TestInputParser_ValidateGenericScenario_EmptyName(t *testing.T) {
	parser := NewInputParser()

	household := &domain.Household{
		Participants: []domain.Participant{
			{Name: "test"},
		},
	}

	scenario := &domain.GenericScenario{
		Name: "",
		ParticipantScenarios: map[string]domain.ParticipantScenario{
			"test": {
				ParticipantName: "test",
				SSStartAge:      62,
			},
		},
	}

	err := parser.validateGenericScenario(0, scenario, household)
	assert.Error(t, err, "Should error for empty scenario name")
	assert.Contains(t, err.Error(), "scenario name is required", "Should have specific error message")
}

func TestInputParser_ValidateGenericScenario_UnknownParticipant(t *testing.T) {
	parser := NewInputParser()

	household := &domain.Household{
		Participants: []domain.Participant{
			{Name: "test"},
		},
	}

	scenario := &domain.GenericScenario{
		Name: "test scenario",
		ParticipantScenarios: map[string]domain.ParticipantScenario{
			"unknown": {
				ParticipantName: "unknown",
				SSStartAge:      62,
			},
		},
	}

	err := parser.validateGenericScenario(0, scenario, household)
	assert.Error(t, err, "Should error for unknown participant")
	assert.Contains(t, err.Error(), "participant scenario references unknown participant", "Should have specific error message")
}

func TestInputParser_ValidateGlobalAssumptions_InvalidInflationRate(t *testing.T) {
	parser := NewInputParser()

	assumptions := &domain.GlobalAssumptions{
		InflationRate:           decimal.NewFromFloat(-0.15), // Invalid
		FEHBPremiumInflation:    decimal.NewFromFloat(0.06),
		TSPReturnPreRetirement:  decimal.NewFromFloat(0.07),
		TSPReturnPostRetirement: decimal.NewFromFloat(0.06),
		COLAGeneralRate:         decimal.NewFromFloat(0.025),
		ProjectionYears:         25,
		CurrentLocation: domain.Location{
			State: "TestState",
		},
	}

	err := parser.validateGlobalAssumptions(assumptions)
	assert.Error(t, err, "Should error for invalid inflation rate")
	assert.Contains(t, err.Error(), "inflation rate cannot be less than -10%", "Should have specific error message")
}

func TestInputParser_ValidateGlobalAssumptions_InvalidProjectionYears(t *testing.T) {
	parser := NewInputParser()

	assumptions := &domain.GlobalAssumptions{
		InflationRate:           decimal.NewFromFloat(0.025),
		FEHBPremiumInflation:    decimal.NewFromFloat(0.06),
		TSPReturnPreRetirement:  decimal.NewFromFloat(0.07),
		TSPReturnPostRetirement: decimal.NewFromFloat(0.06),
		COLAGeneralRate:         decimal.NewFromFloat(0.025),
		ProjectionYears:         0, // Invalid
		CurrentLocation: domain.Location{
			State: "TestState",
		},
	}

	err := parser.validateGlobalAssumptions(assumptions)
	assert.Error(t, err, "Should error for invalid projection years")
	assert.Contains(t, err.Error(), "projection years must be between 1 and 50", "Should have specific error message")
}

func TestInputParser_ValidateGlobalAssumptions_MissingState(t *testing.T) {
	parser := NewInputParser()

	assumptions := &domain.GlobalAssumptions{
		InflationRate:           decimal.NewFromFloat(0.025),
		FEHBPremiumInflation:    decimal.NewFromFloat(0.06),
		TSPReturnPreRetirement:  decimal.NewFromFloat(0.07),
		TSPReturnPostRetirement: decimal.NewFromFloat(0.06),
		COLAGeneralRate:         decimal.NewFromFloat(0.025),
		ProjectionYears:         25,
		CurrentLocation: domain.Location{
			State: "", // Invalid
		},
	}

	err := parser.validateGlobalAssumptions(assumptions)
	assert.Error(t, err, "Should error for missing state")
	assert.Contains(t, err.Error(), "state is required", "Should have specific error message")
}

func TestInputParser_LoadRegulatoryConfig_FileNotFound(t *testing.T) {
	parser := NewInputParser()

	config, err := parser.LoadRegulatoryConfig("nonexistent.yaml")

	assert.Error(t, err, "Should error for nonexistent file")
	assert.Nil(t, config, "Should return nil config")
	assert.Contains(t, err.Error(), "failed to read regulatory config file", "Should have specific error message")
}

func TestInputParser_LoadRegulatoryConfig_InvalidYAML(t *testing.T) {
	// Create a temporary file with invalid YAML
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.yaml")

	err := os.WriteFile(invalidFile, []byte("invalid: yaml: content: [unclosed"), 0644)
	assert.NoError(t, err)

	parser := NewInputParser()
	config, err := parser.LoadRegulatoryConfig(invalidFile)

	assert.Error(t, err, "Should error for invalid YAML")
	assert.Nil(t, config, "Should return nil config")
	assert.Contains(t, err.Error(), "failed to parse regulatory YAML", "Should have specific error message")
}

func TestInputParser_ValidateRegulatoryConfig_InvalidDataYear(t *testing.T) {
	parser := NewInputParser()

	regConfig := &domain.RegulatoryConfig{
		Metadata: domain.RegulatoryMetadata{
			DataYear: 2010, // Invalid
		},
		FederalTax: domain.FederalTaxRules{
			BracketsMFJ: []domain.TaxBracket{
				{Min: decimal.Zero, Max: decimal.NewFromInt(100000), Rate: decimal.NewFromFloat(0.10)},
			},
		},
		FICA: domain.FICARules{
			SocialSecurity: domain.SocialSecurityFICA{
				Rate: decimal.NewFromFloat(0.062),
			},
			Medicare: domain.MedicareFICA{
				Rate: decimal.NewFromFloat(0.0145),
			},
		},
	}

	err := parser.validateRegulatoryConfig(regConfig)
	assert.Error(t, err, "Should error for invalid data year")
	assert.Contains(t, err.Error(), "regulatory data year 2010 seems invalid", "Should have specific error message")
}

func TestInputParser_ValidateRegulatoryConfig_MissingTaxBrackets(t *testing.T) {
	parser := NewInputParser()

	regConfig := &domain.RegulatoryConfig{
		Metadata: domain.RegulatoryMetadata{
			DataYear: 2025,
		},
		FederalTax: domain.FederalTaxRules{
			BracketsMFJ: []domain.TaxBracket{}, // Empty
		},
		FICA: domain.FICARules{
			SocialSecurity: domain.SocialSecurityFICA{
				Rate: decimal.NewFromFloat(0.062),
			},
			Medicare: domain.MedicareFICA{
				Rate: decimal.NewFromFloat(0.0145),
			},
		},
	}

	err := parser.validateRegulatoryConfig(regConfig)
	assert.Error(t, err, "Should error for missing tax brackets")
	assert.Contains(t, err.Error(), "federal tax brackets are required", "Should have specific error message")
}

func TestInputParser_ValidateRegulatoryConfig_InvalidFICARates(t *testing.T) {
	parser := NewInputParser()

	regConfig := &domain.RegulatoryConfig{
		Metadata: domain.RegulatoryMetadata{
			DataYear: 2025,
		},
		FederalTax: domain.FederalTaxRules{
			BracketsMFJ: []domain.TaxBracket{
				{Min: decimal.Zero, Max: decimal.NewFromInt(100000), Rate: decimal.NewFromFloat(0.10)},
			},
		},
		FICA: domain.FICARules{
			SocialSecurity: domain.SocialSecurityFICA{
				Rate: decimal.Zero, // Invalid
			},
			Medicare: domain.MedicareFICA{
				Rate: decimal.NewFromFloat(0.0145),
			},
		},
	}

	err := parser.validateRegulatoryConfig(regConfig)
	assert.Error(t, err, "Should error for invalid Social Security rate")
	assert.Contains(t, err.Error(), "Social Security rate must be positive", "Should have specific error message")
}

// Helper function for creating decimal pointers
func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}
