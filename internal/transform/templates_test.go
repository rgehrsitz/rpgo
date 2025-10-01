package transform

import (
	"strings"
	"testing"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
)

func TestTemplateRegistry_RegisterAndGet(t *testing.T) {
	registry := NewTemplateRegistry()

	template := Template{
		Name:        "test_template",
		Description: "A test template",
		Transforms:  []ScenarioTransform{},
	}

	registry.Register(template)

	// Test exact match
	retrieved, ok := registry.Get("test_template")
	if !ok {
		t.Fatal("Expected to find template")
	}
	if retrieved.Name != template.Name {
		t.Errorf("Expected name %s, got %s", template.Name, retrieved.Name)
	}

	// Test case-insensitive
	retrieved, ok = registry.Get("TEST_TEMPLATE")
	if !ok {
		t.Fatal("Expected case-insensitive lookup to work")
	}

	// Test not found
	_, ok = registry.Get("nonexistent")
	if ok {
		t.Error("Expected not to find nonexistent template")
	}
}

func TestTemplateRegistry_List(t *testing.T) {
	registry := NewTemplateRegistry()

	registry.Register(Template{Name: "template1", Description: "First"})
	registry.Register(Template{Name: "template2", Description: "Second"})

	names := registry.List()
	if len(names) != 2 {
		t.Errorf("Expected 2 templates, got %d", len(names))
	}
}

func TestCreateBuiltInTemplates(t *testing.T) {
	participantName := "Alice"
	registry := CreateBuiltInTemplates(participantName)

	// Test that key templates exist
	expectedTemplates := []string{
		"postpone_1yr",
		"postpone_2yr",
		"delay_ss_67",
		"delay_ss_70",
		"tsp_need_based",
		"tsp_fixed_4pct",
		"conservative",
		"aggressive",
	}

	for _, name := range expectedTemplates {
		template, ok := registry.Get(name)
		if !ok {
			t.Errorf("Expected to find template: %s", name)
			continue
		}
		if len(template.Transforms) == 0 {
			t.Errorf("Template %s has no transforms", name)
		}
		if template.Description == "" {
			t.Errorf("Template %s has no description", name)
		}
	}
}

func TestApplyTemplate(t *testing.T) {
	retirementDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	base := &domain.GenericScenario{
		Name: "Base",
		ParticipantScenarios: map[string]domain.ParticipantScenario{
			"Alice": {
				ParticipantName: "Alice",
				RetirementDate:  &retirementDate,
				SSStartAge:      62,
			},
		},
	}

	template := Template{
		Name:        "test",
		Description: "Test template",
		Transforms: []ScenarioTransform{
			&PostponeRetirement{Participant: "Alice", Months: 12},
			&DelaySSClaim{Participant: "Alice", NewAge: 70},
		},
	}

	result, err := ApplyTemplate(base, template)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	// Verify retirement date was postponed
	resultPS := result.ParticipantScenarios["Alice"]
	expectedDate := retirementDate.AddDate(0, 12, 0)
	if !resultPS.RetirementDate.Equal(expectedDate) {
		t.Errorf("Expected retirement date %v, got %v", expectedDate, *resultPS.RetirementDate)
	}

	// Verify SS age was changed
	if resultPS.SSStartAge != 70 {
		t.Errorf("Expected SS start age 70, got %d", resultPS.SSStartAge)
	}

	// Verify base scenario was not modified
	basePS := base.ParticipantScenarios["Alice"]
	if !basePS.RetirementDate.Equal(retirementDate) {
		t.Error("Base scenario was modified (should be immutable)")
	}
	if basePS.SSStartAge != 62 {
		t.Error("Base scenario SS age was modified")
	}
}

func TestApplyTemplate_EmptyTransforms(t *testing.T) {
	retirementDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	base := &domain.GenericScenario{
		Name: "Base",
		ParticipantScenarios: map[string]domain.ParticipantScenario{
			"Alice": {
				ParticipantName: "Alice",
				RetirementDate:  &retirementDate,
			},
		},
	}

	template := Template{
		Name:        "empty",
		Description: "Empty template",
		Transforms:  []ScenarioTransform{},
	}

	result, err := ApplyTemplate(base, template)
	if err != nil {
		t.Fatalf("Failed to apply empty template: %v", err)
	}

	if result == base {
		t.Error("Expected a copy, got same reference")
	}
}

func TestParseTemplateList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "Single template",
			input:    "postpone_1yr",
			expected: []string{"postpone_1yr"},
		},
		{
			name:     "Multiple templates",
			input:    "postpone_1yr,delay_ss_70,tsp_fixed_4pct",
			expected: []string{"postpone_1yr", "delay_ss_70", "tsp_fixed_4pct"},
		},
		{
			name:     "With spaces",
			input:    "postpone_1yr, delay_ss_70 , tsp_fixed_4pct",
			expected: []string{"postpone_1yr", "delay_ss_70", "tsp_fixed_4pct"},
		},
		{
			name:     "Empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "Only spaces",
			input:    "  ,  ,  ",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseTemplateList(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d templates, got %d", len(tt.expected), len(result))
				return
			}
			for i, expected := range tt.expected {
				if result[i] != expected {
					t.Errorf("Expected template[%d] = %s, got %s", i, expected, result[i])
				}
			}
		})
	}
}

func TestGetTemplateHelp(t *testing.T) {
	registry := CreateBuiltInTemplates("Alice")
	help := GetTemplateHelp(registry)

	// Verify help contains expected sections
	if !strings.Contains(help, "Available Templates") {
		t.Error("Help should contain 'Available Templates' header")
	}

	if !strings.Contains(help, "Retirement Timing") {
		t.Error("Help should contain 'Retirement Timing' category")
	}

	if !strings.Contains(help, "Social Security") {
		t.Error("Help should contain 'Social Security' category")
	}

	if !strings.Contains(help, "TSP Strategies") {
		t.Error("Help should contain 'TSP Strategies' category")
	}

	if !strings.Contains(help, "Combination Strategies") {
		t.Error("Help should contain 'Combination Strategies' category")
	}

	if !strings.Contains(help, "postpone_1yr") {
		t.Error("Help should contain postpone_1yr template")
	}

	if !strings.Contains(help, "Usage:") {
		t.Error("Help should contain usage examples")
	}
}

func TestGetTemplateHelp_EmptyRegistry(t *testing.T) {
	registry := NewTemplateRegistry()
	help := GetTemplateHelp(registry)

	if help != "No templates registered" {
		t.Errorf("Expected 'No templates registered', got: %s", help)
	}
}

func TestBuiltInTemplate_Postpone1Year(t *testing.T) {
	registry := CreateBuiltInTemplates("Alice")
	template, ok := registry.Get("postpone_1yr")
	if !ok {
		t.Fatal("Template not found")
	}

	retirementDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	base := &domain.GenericScenario{
		Name: "Base",
		ParticipantScenarios: map[string]domain.ParticipantScenario{
			"Alice": {
				ParticipantName: "Alice",
				RetirementDate:  &retirementDate,
			},
		},
	}

	result, err := ApplyTemplate(base, template)
	if err != nil {
		t.Fatalf("Failed to apply template: %v", err)
	}

	expectedDate := retirementDate.AddDate(0, 12, 0)
	resultDate := result.ParticipantScenarios["Alice"].RetirementDate
	if !resultDate.Equal(expectedDate) {
		t.Errorf("Expected retirement date %v, got %v", expectedDate, *resultDate)
	}
}

func TestBuiltInTemplate_Conservative(t *testing.T) {
	registry := CreateBuiltInTemplates("Alice")
	template, ok := registry.Get("conservative")
	if !ok {
		t.Fatal("Template not found")
	}

	retirementDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	base := &domain.GenericScenario{
		Name: "Base",
		ParticipantScenarios: map[string]domain.ParticipantScenario{
			"Alice": {
				ParticipantName:        "Alice",
				RetirementDate:         &retirementDate,
				SSStartAge:             62,
				TSPWithdrawalStrategy:  "need_based",
				TSPWithdrawalRate:      nil,
			},
		},
	}

	result, err := ApplyTemplate(base, template)
	if err != nil {
		t.Fatalf("Failed to apply conservative template: %v", err)
	}

	resultPS := result.ParticipantScenarios["Alice"]

	// Check retirement postponed by 2 years
	expectedDate := retirementDate.AddDate(0, 24, 0)
	if !resultPS.RetirementDate.Equal(expectedDate) {
		t.Errorf("Expected retirement date %v, got %v", expectedDate, *resultPS.RetirementDate)
	}

	// Check SS delayed to 70
	if resultPS.SSStartAge != 70 {
		t.Errorf("Expected SS start age 70, got %d", resultPS.SSStartAge)
	}

	// Check TSP strategy is variable_percentage
	if resultPS.TSPWithdrawalStrategy != "variable_percentage" {
		t.Errorf("Expected TSP strategy 'variable_percentage', got %s", resultPS.TSPWithdrawalStrategy)
	}

	// Check TSP rate is 3%
	if resultPS.TSPWithdrawalRate == nil {
		t.Fatal("Expected TSP withdrawal rate to be set")
	}
	expectedRate := 0.03
	if resultPS.TSPWithdrawalRate.InexactFloat64() != expectedRate {
		t.Errorf("Expected TSP rate %f, got %f", expectedRate, resultPS.TSPWithdrawalRate.InexactFloat64())
	}
}

func TestBuiltInTemplate_Aggressive(t *testing.T) {
	registry := CreateBuiltInTemplates("Alice")
	template, ok := registry.Get("aggressive")
	if !ok {
		t.Fatal("Template not found")
	}

	retirementDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	base := &domain.GenericScenario{
		Name: "Base",
		ParticipantScenarios: map[string]domain.ParticipantScenario{
			"Alice": {
				ParticipantName:        "Alice",
				RetirementDate:         &retirementDate,
				SSStartAge:             62,
				TSPWithdrawalStrategy:  "need_based",
			},
		},
	}

	result, err := ApplyTemplate(base, template)
	if err != nil {
		t.Fatalf("Failed to apply aggressive template: %v", err)
	}

	resultPS := result.ParticipantScenarios["Alice"]

	// Check retirement date unchanged
	if !resultPS.RetirementDate.Equal(retirementDate) {
		t.Errorf("Retirement date should not change in aggressive template")
	}

	// Check SS delayed to 70
	if resultPS.SSStartAge != 70 {
		t.Errorf("Expected SS start age 70, got %d", resultPS.SSStartAge)
	}

	// Check TSP rate is 4%
	if resultPS.TSPWithdrawalRate == nil {
		t.Fatal("Expected TSP withdrawal rate to be set")
	}
	expectedRate := 0.04
	if resultPS.TSPWithdrawalRate.InexactFloat64() != expectedRate {
		t.Errorf("Expected TSP rate %f, got %f", expectedRate, resultPS.TSPWithdrawalRate.InexactFloat64())
	}
}
