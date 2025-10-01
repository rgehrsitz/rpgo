# Transform Package

The transform package provides a composable system for modifying retirement scenarios. Transforms enable scenario comparison, break-even analysis, and interactive user experiences.

## Overview

Transforms are operations that take a base scenario and produce a modified version. They can be:

- **Composed**: Apply multiple transforms in sequence
- **Validated**: Check if a transform can be applied before executing it
- **Reversible**: Original scenarios are never modified (immutable transformations)
- **Type-safe**: Strongly typed with compile-time checks

## Core Interface

```go
type ScenarioTransform interface {
    Apply(base *domain.GenericScenario) (*domain.GenericScenario, error)
    Name() string
    Description() string
    Validate(base *domain.GenericScenario) error
}
```

## Available Transforms

### Retirement Transforms

#### PostponeRetirement

Delays a participant's retirement date by a specified number of months.

```go
transform := &PostponeRetirement{
    Participant: "Alice",
    Months:      12, // Postpone by 12 months
}
```

**Use Cases:**

- "Work one more year" scenarios
- Gradual retirement date exploration
- Break-even analysis for retirement timing

#### SetRetirementDate

Sets a participant's retirement date to an absolute date.

```go
transform := &SetRetirementDate{
    Participant: "Alice",
    Date:        time.Date(2029, 6, 30, 0, 0, 0, 0, time.UTC),
}
```

**Use Cases:**

- Setting specific target retirement dates
- Scenario comparison with fixed dates

### Social Security Transforms

#### DelaySSClaim

Changes the Social Security claiming age for a participant.

```go
transform := &DelaySSClaim{
    Participant: "Alice",
    NewAge:      67, // Must be 62-70
}
```

**Use Cases:**

- Exploring SS claiming strategies
- Optimizing benefit amounts
- Analyzing early vs. delayed claiming

**Note:** Delaying SS increases benefits by ~8% per year from Full Retirement Age to age 70.

### TSP Withdrawal Transforms

#### ModifyTSPStrategy

Changes the TSP withdrawal strategy for a participant.

```go
transform := &ModifyTSPStrategy{
    Participant:   "Alice",
    NewStrategy:   "variable_percentage", // or "4_percent_rule", "need_based", "fixed_amount"
    PreserveRate:  false, // If true, keeps existing withdrawal rate/target
}
```

**Strategies:**

- `4_percent_rule`: Initial 4% withdrawal, adjusted for inflation
- `variable_percentage`: Fixed percentage of current balance
- `need_based`: Withdraw to meet target monthly income
- `fixed_amount`: Fixed dollar amount per year

#### AdjustTSPRate

Changes the TSP withdrawal rate for percentage-based strategies.

```go
transform := &AdjustTSPRate{
    Participant: "Alice",
    NewRate:     decimal.NewFromFloat(0.045), // 4.5%
}
```

**Applicable to:**

- `variable_percentage` strategy
- `4_percent_rule` strategy

**Valid Range:** 0% to 20%

#### SetTSPTargetIncome

Changes the monthly target income for need-based TSP withdrawals.

```go
transform := &SetTSPTargetIncome{
    Participant:   "Alice",
    MonthlyTarget: decimal.NewFromInt(5000), // $5,000/month
}
```

**Applicable to:**

- `need_based` strategy only

### Mortality Transforms

#### SetMortalityDate

Sets or modifies the death date for a participant (for survivor analysis).

```go
transform := &SetMortalityDate{
    Participant: "Alice",
    DeathDate:   time.Date(2034, 6, 30, 0, 0, 0, 0, time.UTC),
}
```

**Use Cases:**

- Survivor income analysis
- Estate planning scenarios
- Life insurance needs assessment

#### SetSurvivorSpendingFactor

Changes the spending adjustment after a spouse's death.

```go
transform := &SetSurvivorSpendingFactor{
    Factor: decimal.NewFromFloat(0.75), // Survivor needs 75% of couple's spending
}
```

**Typical Values:**

- 0.70 - 0.75: Conservative (significant lifestyle reduction)
- 0.75 - 0.85: Moderate (some reduction)
- 0.85 - 0.95: Optimistic (minimal reduction)

#### SetTSPTransferMode

Changes how TSP assets are handled after a spouse's death.

```go
transform := &SetTSPTransferMode{
    Mode: "merge", // or "keep_separate", "survivor_inherits"
}
```

**Modes:**

- `merge`: Combine deceased's TSP with survivor's (most common)
- `keep_separate`: Track separately (complex, rarely used)
- `survivor_inherits`: All TSP goes to survivor

## Using Transforms

### Basic Usage

```go
// Create a base scenario
baseScenario := &domain.GenericScenario{
    Name: "Base Retirement Plan",
    // ... participant scenarios ...
}

// Create a transform
transform := &PostponeRetirement{
    Participant: "Alice",
    Months:      12,
}

// Validate before applying
if err := transform.Validate(baseScenario); err != nil {
    log.Fatalf("Validation failed: %v", err)
}

// Apply the transform
modifiedScenario, err := transform.Apply(baseScenario)
if err != nil {
    log.Fatalf("Transform failed: %v", err)
}

// baseScenario is unchanged, modifiedScenario contains the modification
```

### Composing Multiple Transforms

```go
transforms := []ScenarioTransform{
    &PostponeRetirement{Participant: "Alice", Months: 12},
    &DelaySSClaim{Participant: "Alice", NewAge: 67},
    &AdjustTSPRate{Participant: "Alice", NewRate: decimal.NewFromFloat(0.045)},
}

// Apply all transforms in sequence
modifiedScenario, err := ApplyTransforms(baseScenario, transforms)
if err != nil {
    log.Fatalf("Transform composition failed: %v", err)
}
```

Each transform receives the output of the previous transform, enabling complex scenario modifications.

### Using the Transform Registry

The registry provides a convenient way to create transforms from string parameters:

```go
registry := NewTransformRegistry()

// Parse a transform specification
transform, err := registry.ParseTransformSpec("postpone_retirement:participant=Alice,months=12")
if err != nil {
    log.Fatalf("Parse failed: %v", err)
}

// Apply it
modified, err := transform.Apply(baseScenario)
```

**Spec Format:** `transform_name:param1=value1,param2=value2`

**Examples:**

```text
postpone_retirement:participant=Alice,months=12
delay_ss:participant=Bob,age=67
adjust_tsp_rate:participant=Alice,rate=0.045
set_mortality:participant=Alice,date=2034-06-30
set_survivor_spending:factor=0.75
```

## Transform Registry CLI Usage

The registry enables command-line usage:

```bash
# Compare scenarios with transform specs
./rpgo compare base.yaml \
    --transform "postpone_retirement:participant=Alice,months=12" \
    --transform "delay_ss:participant=Alice,age=67"

# Break-even analysis with transforms
./rpgo break-even base.yaml \
    --transform "adjust_tsp_rate:participant=Alice,rate=0.05"
```

## Error Handling

Transforms use the `TransformError` type for detailed error reporting:

```go
err := transform.Validate(scenario)
if err != nil {
    if transformErr, ok := err.(*TransformError); ok {
        fmt.Printf("Transform: %s\n", transformErr.TransformName)
        fmt.Printf("Operation: %s\n", transformErr.Operation)
        fmt.Printf("Reason: %s\n", transformErr.Reason)
    }
}
```

## Design Principles

### Immutability

Transforms **never modify** the input scenario. They always create and return a new scenario with the modifications applied. This ensures:

- Safe concurrent usage
- Easy undo/redo functionality
- Transparent comparison between base and modified scenarios

### Validation

All transforms implement pre-application validation:

- Check parameter validity (e.g., SS age must be 62-70)
- Verify participant exists in scenario
- Ensure transform is applicable (e.g., TSP rate only for percentage strategies)

Validation failures return descriptive errors **before** any modifications are attempted.

### Composition

Transforms are designed to compose cleanly:

- Each transform is independent
- Output of one transform can be input to another
- No hidden dependencies between transforms
- Deterministic: Same sequence always produces same result

## Testing

All transforms have comprehensive test coverage:

- Validation tests (valid and invalid inputs)
- Application tests (correct modifications)
- Immutability tests (original scenario unchanged)
- Composition tests (multiple transforms)

Run tests:

```bash
go test ./internal/transform -v
```

## Future Enhancements

Planned transforms:

- `ModifyInflation`: Change inflation rate assumptions
- `ModifyCOLA`: Change COLA assumptions
- `SetTSPAllocation`: Modify fund allocation percentages
- `EnableRothConversion`: Add Roth conversion schedule
- `SetExternalPension`: Add/modify external pensions
- `ChangeFilingStatus`: Switch between joint/single filing

These will be added as the compare and solver features are implemented.

## Architecture Notes

### Why Not Modify GlobalAssumptions?

Some parameters (inflation, COLA, TSP returns) are in `GlobalAssumptions`, not `GenericScenario`. Transforms currently only modify scenario-specific parameters.

For assumption modifications, the compare/solver systems will create modified `Configuration` objects with adjusted `GlobalAssumptions` before running projections.

This separation keeps transforms focused on scenario-specific changes while allowing the higher-level commands to handle global assumption variations.

### Performance

Transforms use deep copying for immutability. For typical scenarios with 1-4 participants, this overhead is negligible (<1ms per transform). If performance becomes an issue with many participants or complex scenarios, we can implement copy-on-write optimizations.

## Examples

### Scenario: "Work One More Year"

```go
// Compare retiring now vs. working one more year
baseScenario := loadScenario("retirement_2027.yaml")

oneYearLater := ApplyTransforms(baseScenario, []ScenarioTransform{
    &PostponeRetirement{Participant: "Alice", Months: 12},
    &PostponeRetirement{Participant: "Bob", Months: 12},
})

// Run projections and compare outcomes
```

### Scenario: "Optimize SS Claiming"

```go
// Try different SS claiming ages
scenarios := map[string]*domain.GenericScenario{}

for age := 62; age <= 70; age++ {
    name := fmt.Sprintf("SS at %d", age)
    scenarios[name], _ = ApplyTransforms(baseScenario, []ScenarioTransform{
        &DelaySSClaim{Participant: "Alice", NewAge: age},
    })
}

// Compare lifetime income across all scenarios
```

### Scenario: "Survivor Analysis"

```go
// Analyze financial viability if Alice passes at 75
survivorScenario, _ := ApplyTransforms(baseScenario, []ScenarioTransform{
    &SetMortalityDate{
        Participant: "Alice",
        DeathDate:   time.Date(2040, 6, 30, 0, 0, 0, 0, time.UTC),
    },
    &SetSurvivorSpendingFactor{
        Factor: decimal.NewFromFloat(0.75),
    },
    &SetTSPTransferMode{
        Mode: "merge",
    },
})

// Run projection and analyze survivor income
```

## Contributing

When adding new transforms:

1. **Implement the interface** in a new file or add to existing category file
2. **Add validation logic** to catch invalid parameters early
3. **Write comprehensive tests** (validation, application, edge cases)
4. **Register in registry** if it should be CLI-accessible
5. **Document in this README** with examples

All transforms must:

- Be immutable (never modify input)
- Have clear, descriptive names
- Provide helpful descriptions
- Validate thoroughly before applying
- Have 80%+ test coverage
