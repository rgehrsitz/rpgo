package sequencing

import (
	"testing"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

func TestCreateStrategy(t *testing.T) {
	tests := []struct {
		name     string
		config   *domain.WithdrawalSequencingConfig
		expected string
	}{
		{
			name: "Standard Strategy",
			config: &domain.WithdrawalSequencingConfig{
				Strategy: "standard",
			},
			expected: "standard",
		},
		{
			name: "Tax Efficient Strategy",
			config: &domain.WithdrawalSequencingConfig{
				Strategy: "tax_efficient",
			},
			expected: "tax_efficient",
		},
		{
			name: "Bracket Fill Strategy",
			config: &domain.WithdrawalSequencingConfig{
				Strategy: "bracket_fill",
			},
			expected: "bracket_fill",
		},
		{
			name: "Custom Strategy",
			config: &domain.WithdrawalSequencingConfig{
				Strategy: "custom",
			},
			expected: "custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strategy := CreateStrategy(tt.config)
			if strategy == nil {
				t.Fatal("Expected strategy to be created, got nil")
			}

			// Test that the strategy can be executed
			sources := CreateWithdrawalSources(
				&domain.Participant{Name: "Test"},
				decimal.NewFromInt(100000), // traditional
				decimal.NewFromInt(50000),  // roth
				false,                      // not RMD year
				decimal.Zero,               // no RMD amount
			)

			ctx := CreateStrategyContext(
				decimal.NewFromInt(10000), // withdrawal amount
				decimal.Zero,              // current ordinary income
				decimal.Zero,              // current MAGI
				false,                     // not RMD year
				tt.config,
			)

			plan := strategy.Plan(sources, ctx)
			if plan.Allocations == nil {
				t.Fatal("Expected allocations to be created")
			}

			if len(plan.Allocations) == 0 {
				t.Fatal("Expected allocations to be created")
			}
		})
	}
}

func TestCreateWithdrawalSources(t *testing.T) {
	participant := &domain.Participant{
		Name:                  "Test",
		TaxableAccountBalance: decimalPtr(decimal.NewFromInt(75000)),
		TaxableAccountBasis:   decimalPtr(decimal.NewFromInt(60000)),
	}

	traditionalBalance := decimal.NewFromInt(100000)
	rothBalance := decimal.NewFromInt(50000)
	isRMDYear := true
	rmdAmount := decimal.NewFromInt(5000)

	sources := CreateWithdrawalSources(
		participant,
		traditionalBalance,
		rothBalance,
		isRMDYear,
		rmdAmount,
	)

	if sources == nil {
		t.Fatal("Expected sources to be created, got nil")
	}

	// Check that all sources are present
	foundTaxable := false
	foundTraditional := false
	foundRoth := false

	for _, source := range sources {
		switch source.Name {
		case "taxable":
			foundTaxable = true
			if !source.Balance.Equal(*participant.TaxableAccountBalance) {
				t.Errorf("Expected taxable balance %v, got %v", *participant.TaxableAccountBalance, source.Balance)
			}
		case "traditional":
			foundTraditional = true
			if !source.Balance.Equal(traditionalBalance) {
				t.Errorf("Expected traditional balance %v, got %v", traditionalBalance, source.Balance)
			}
		case "roth":
			foundRoth = true
			if !source.Balance.Equal(rothBalance) {
				t.Errorf("Expected roth balance %v, got %v", rothBalance, source.Balance)
			}
		}
	}

	if !foundTaxable {
		t.Error("Expected taxable source to be found")
	}
	if !foundTraditional {
		t.Error("Expected traditional source to be found")
	}
	if !foundRoth {
		t.Error("Expected roth source to be found")
	}
}

func TestCreateStrategyContext(t *testing.T) {
	config := &domain.WithdrawalSequencingConfig{
		Strategy: "standard",
	}

	withdrawalAmount := decimal.NewFromInt(10000)
	currentOrdinaryIncome := decimal.NewFromInt(50000)
	currentMAGI := decimal.NewFromInt(75000)
	isRMDYear := false

	ctx := CreateStrategyContext(
		withdrawalAmount,
		currentOrdinaryIncome,
		currentMAGI,
		isRMDYear,
		config,
	)

	if ctx.NeedAmount.IsZero() {
		t.Fatal("Expected context to be created")
	}

	if !ctx.NeedAmount.Equal(withdrawalAmount) {
		t.Errorf("Expected withdrawal amount %v, got %v", withdrawalAmount, ctx.NeedAmount)
	}

	if !ctx.CurrentOrdinaryIncome.Equal(currentOrdinaryIncome) {
		t.Errorf("Expected current ordinary income %v, got %v", currentOrdinaryIncome, ctx.CurrentOrdinaryIncome)
	}

	if !ctx.MAGICurrent.Equal(currentMAGI) {
		t.Errorf("Expected current MAGI %v, got %v", currentMAGI, ctx.MAGICurrent)
	}

	if ctx.IsRMDYear != isRMDYear {
		t.Errorf("Expected RMD year %v, got %v", isRMDYear, ctx.IsRMDYear)
	}
}

func TestStandardStrategy(t *testing.T) {
	strategy := &StandardStrategy{}

	sources := []WithdrawalSource{
		{Name: "taxable", Balance: decimal.NewFromInt(50000), Basis: decimal.NewFromInt(40000)},
		{Name: "traditional", Balance: decimal.NewFromInt(100000), Basis: decimal.Zero},
		{Name: "roth", Balance: decimal.NewFromInt(30000), Basis: decimal.Zero},
	}

	ctx := StrategyContext{
		NeedAmount:            decimal.NewFromInt(20000),
		CurrentOrdinaryIncome: decimal.Zero,
		MAGICurrent:           decimal.Zero,
		IsRMDYear:             false,
	}

	plan := strategy.Plan(sources, ctx)

	if plan.Allocations == nil {
		t.Fatal("Expected plan to be created, got nil")
	}

	if len(plan.Allocations) == 0 {
		t.Fatal("Expected allocations to be created")
	}

	// Standard strategy should prioritize taxable first
	totalAllocated := decimal.Zero
	for _, allocation := range plan.Allocations {
		totalAllocated = totalAllocated.Add(allocation.Gross)
	}

	if !totalAllocated.Equal(ctx.NeedAmount) {
		t.Errorf("Expected total allocation %v, got %v", ctx.NeedAmount, totalAllocated)
	}

	// First allocation should be from taxable
	if plan.Allocations[0].Source != "taxable" {
		t.Errorf("Expected first allocation from taxable, got %s", plan.Allocations[0].Source)
	}
}

func TestTaxEfficientStrategy(t *testing.T) {
	strategy := &TaxEfficientStrategy{}

	sources := []WithdrawalSource{
		{Name: "taxable", Balance: decimal.NewFromInt(50000), Basis: decimal.NewFromInt(40000)},
		{Name: "traditional", Balance: decimal.NewFromInt(100000), Basis: decimal.Zero},
		{Name: "roth", Balance: decimal.NewFromInt(30000), Basis: decimal.Zero},
	}

	ctx := StrategyContext{
		NeedAmount:            decimal.NewFromInt(20000),
		CurrentOrdinaryIncome: decimal.Zero,
		MAGICurrent:           decimal.Zero,
		IsRMDYear:             false,
	}

	plan := strategy.Plan(sources, ctx)

	if plan.Allocations == nil {
		t.Fatal("Expected plan to be created, got nil")
	}

	if len(plan.Allocations) == 0 {
		t.Fatal("Expected allocations to be created")
	}

	// Tax efficient strategy should prioritize Roth first
	totalAllocated := decimal.Zero
	for _, allocation := range plan.Allocations {
		totalAllocated = totalAllocated.Add(allocation.Gross)
	}

	if !totalAllocated.Equal(ctx.NeedAmount) {
		t.Errorf("Expected total allocation %v, got %v", ctx.NeedAmount, totalAllocated)
	}

	// First allocation should be from Roth
	if plan.Allocations[0].Source != "roth" {
		t.Errorf("Expected first allocation from roth, got %s", plan.Allocations[0].Source)
	}
}

func TestBracketFillStrategy(t *testing.T) {
	strategy := &BracketFillStrategy{}

	sources := []WithdrawalSource{
		{Name: "taxable", Balance: decimal.NewFromInt(50000), Basis: decimal.NewFromInt(40000)},
		{Name: "traditional", Balance: decimal.NewFromInt(100000), Basis: decimal.Zero},
		{Name: "roth", Balance: decimal.NewFromInt(30000), Basis: decimal.Zero},
	}

	targetBracket := 22
	bracketBuffer := 5000

	ctx := StrategyContext{
		NeedAmount:            decimal.NewFromInt(20000),
		CurrentOrdinaryIncome: decimal.Zero,
		MAGICurrent:           decimal.Zero,
		IsRMDYear:             false,
		TargetBracketPercent:  &targetBracket,
		BracketBufferAmount:   &bracketBuffer,
	}

	plan := strategy.Plan(sources, ctx)

	if plan.Allocations == nil {
		t.Fatal("Expected plan to be created, got nil")
	}

	if len(plan.Allocations) == 0 {
		t.Fatal("Expected allocations to be created")
	}

	// Bracket fill strategy should prioritize traditional first (to fill bracket)
	totalAllocated := decimal.Zero
	for _, allocation := range plan.Allocations {
		totalAllocated = totalAllocated.Add(allocation.Gross)
	}

	if !totalAllocated.Equal(ctx.NeedAmount) {
		t.Errorf("Expected total allocation %v, got %v", ctx.NeedAmount, totalAllocated)
	}

	// First allocation should be from traditional (to fill bracket)
	if plan.Allocations[0].Source != "traditional" {
		t.Errorf("Expected first allocation from traditional, got %s", plan.Allocations[0].Source)
	}
}

func TestCustomStrategy(t *testing.T) {
	strategy := &CustomStrategy{
		Sequence: []string{"roth", "taxable", "traditional"},
	}

	sources := []WithdrawalSource{
		{Name: "taxable", Balance: decimal.NewFromInt(50000), Basis: decimal.NewFromInt(40000)},
		{Name: "traditional", Balance: decimal.NewFromInt(100000), Basis: decimal.Zero},
		{Name: "roth", Balance: decimal.NewFromInt(30000), Basis: decimal.Zero},
	}

	ctx := StrategyContext{
		NeedAmount:            decimal.NewFromInt(20000),
		CurrentOrdinaryIncome: decimal.Zero,
		MAGICurrent:           decimal.Zero,
		IsRMDYear:             false,
	}

	plan := strategy.Plan(sources, ctx)

	if plan.Allocations == nil {
		t.Fatal("Expected plan to be created, got nil")
	}

	if len(plan.Allocations) == 0 {
		t.Fatal("Expected allocations to be created")
	}

	// Custom strategy should follow the specified sequence
	totalAllocated := decimal.Zero
	for _, allocation := range plan.Allocations {
		totalAllocated = totalAllocated.Add(allocation.Gross)
	}

	if !totalAllocated.Equal(ctx.NeedAmount) {
		t.Errorf("Expected total allocation %v, got %v", ctx.NeedAmount, totalAllocated)
	}

	// First allocation should be from Roth (as specified in custom sequence)
	if plan.Allocations[0].Source != "roth" {
		t.Errorf("Expected first allocation from roth, got %s", plan.Allocations[0].Source)
	}
}

// Helper function to create decimal pointer
func decimalPtr(d decimal.Decimal) *decimal.Decimal {
	return &d
}
