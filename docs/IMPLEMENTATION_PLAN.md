# RPGO Enhancement Implementation Plan

**Status:** Phase 3.1 Complete - Advanced Features In Progress
**Last Updated:** December 2024
**Target:** Open Source Release

## Executive Summary

This document outlines the comprehensive enhancement plan for RPGO (FERS Retirement Planning Calculator) based on collaborative analysis and user requirements. The focus is on building a robust, user-friendly, terminal-based retirement planning tool with sophisticated analysis capabilities.

## Core Principles

1. **Terminal-Based Excellence** - Rich TUI experience using Bubble Tea, not mobile/web
2. **Accuracy Over Approximation** - Full calculations with feedback loops, especially for high-stakes decisions
3. **Open Source Quality** - Tests written as we go, inline documentation, clean architecture
4. **Extensible Foundation** - Transform pipeline architecture enables future features
5. **No Backward Compatibility Constraints** - Pre-release, can evolve config format freely

## Architecture Decisions

### ✅ Approved Technologies

- **Bubble Tea** - TUI framework (inspired by Charm's Crush project)
- **Transform Pipeline** - Foundation for all interactive/comparative features
- **Test-Driven** - Write tests alongside implementation
- **Inline Documentation** - Document as we code

### ❌ Rejected Approaches

- **Mobile-friendly HTML** - Unnecessary for terminal-based tool (desktop viewing sufficient)
- **Approximate Roth calculations** - Full modeling required for accuracy
- **Guided menu prompts** - TUI provides richer experience

## Implementation Phases

---

## PHASE 1: FOUNDATION (Weeks 1-4)

**Goal:** Build extensible architecture that enables all future features

### 1.1 Transform Pipeline Architecture (3-4 days) ✅ COMPLETE

**Priority:** CRITICAL - Unlocks everything else

**Status:** Complete (October 2025). See earlier conversation summary for details.

**Scope:**

```go
// Core abstraction
type ScenarioTransform interface {
    Apply(base *domain.GenericScenario) (*domain.GenericScenario, error)
    Name() string
    Description() string
    Validate() error
}

// Transform composition
func ApplyTransforms(base *domain.GenericScenario, transforms []ScenarioTransform) (*domain.GenericScenario, error)

// Transform registry for built-in transforms
type TransformRegistry struct {
    transforms map[string]func(params map[string]interface{}) (ScenarioTransform, error)
}
```

**Built-in Transforms to Implement:**

1. `PostponeRetirement` - Delay retirement date by N months for participant
2. `DelaySSClaim` - Change Social Security start age
3. `ModifyTSPStrategy` - Switch withdrawal strategy
4. `AdjustTSPRate` - Change withdrawal percentage
5. `ModifyTSPAllocation` - Change fund allocation percentages
6. `SetTSPBalance` - Override starting TSP balance
7. `ChangeCOLA` - Adjust COLA assumptions
8. `ModifyInflation` - Change inflation rate
9. `AdjustFEHBInflation` - Change healthcare inflation
10. `SetRetirementDate` - Absolute date (not relative)
11. `EnableRothConversion` - Add conversion schedule
12. `ModifyMortalityDate` - Change death date for mortality scenarios
13. `AdjustSurvivorSpending` - Change survivor spending factor
14. `ChangeFilingStatus` - Switch between joint/single
15. `SetExternalPension` - Add/modify external pension

**Deliverables:**

- [ ] `internal/transform/` package created
- [ ] Core interfaces defined
- [ ] 15 built-in transforms implemented
- [ ] Unit tests for each transform (table-driven)
- [ ] Transform composition tests
- [ ] Validation logic (e.g., can't retire before hire date)
- [ ] README in `internal/transform/` explaining usage

**Testing Strategy:**

```go
func TestPostponeRetirement(t *testing.T) {
    tests := []struct {
        name           string
        baseDate       time.Time
        postponeMonths int
        expectedDate   time.Time
        expectError    bool
    }{
        {"Postpone 6 months", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), 6, time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC), false},
        {"Postpone 0 months", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), 0, time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), false},
        {"Negative months error", time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC), -6, time.Time{}, true},
    }
    // ... test implementation
}
```

**Dependencies Enabled:**

- Scenario compare command
- Break-even solver
- TUI interactive mode
- Sensitivity analysis
- Roth conversion planner

---

### 1.2 Scenario Compare Command (2-3 days) ✅ COMPLETE

**Priority:** HIGH - Quick win that proves transform architecture

**Status:** Complete (October 2025). See earlier conversation summary for details.

**Scope:**

```bash
# Compare with built-in scenario templates
./rpgo compare base.yaml --with postpone_1yr,postpone_2yr,delay_ss_67,delay_ss_70

# Compare with explicit transforms
./rpgo compare base.yaml --transform "postpone_retirement:months=12,participant=Alice" \
                         --transform "delay_ss:age=67,participant=Alice"

# Export comparison matrix
./rpgo compare base.yaml --with postpone_1yr,postpone_2yr --format csv > comparison.csv
./rpgo compare base.yaml --with postpone_1yr,postpone_2yr --format json > comparison.json
```

**Output Format (Console):**

```text
SCENARIO COMPARISON MATRIX
=================================================================
Scenario              First Year    Year 5      Year 10    TSP Longevity    Lifetime Income
-----------------------------------------------------------------
Baseline              $156,234      $168,891    $175,420   28 years         $4,234,567
Postpone 1 Year       $158,123      $171,234    $178,901   30 years         $4,456,789
Postpone 2 Years      $160,456      $173,890    $182,456   32 years         $4,678,912
Delay SS to 67        $149,890      $182,345    $189,234   25 years         $4,567,890
Delay SS to 70        $145,678      $195,678    $201,234   23 years         $4,789,123
-----------------------------------------------------------------
RECOMMENDED: Postpone 2 Years
  • Highest TSP longevity (32 years)
  • Strong long-term income ($182,456 at year 10)
  • Highest lifetime income ($4,678,912)
```

**Implementation:**

- [ ] Create `cmd/compare.go`
- [ ] Built-in scenario template library
- [ ] Transform parser for `--transform` flag
- [ ] Matrix generation logic
- [ ] Console formatter (table)
- [ ] CSV exporter
- [ ] JSON exporter
- [ ] Recommendation engine (simple heuristic)
- [ ] Tests for each output format
- [ ] Integration test with sample config

**Deliverables:**

- [ ] `./rpgo compare` command works
- [ ] At least 10 built-in scenario templates
- [ ] All output formats functional
- [ ] Documentation in README

---

### 1.3 Enhanced Break-Even Solver (4-5 days) ✅ COMPLETE

**Priority:** HIGH - Answers critical "when can I retire?" question

**Status:** Complete (October 2025). See [Phase 1.3 Complete](PHASE_1.3_COMPLETE.md) and [Optimize Command Documentation](OPTIMIZE_COMMAND.md) for details.

**Scope:**

**Multi-Dimensional Solver Modes:**

1. **Solve for Retirement Date:**

```bash
./rpgo break-even config.yaml \
  --solve retirement_date \
  --participant "Alice Johnson" \
  --target-net-income 180000 \
  --max-date 2035-12-31
```

1. **Solve for TSP Balance:**

```bash
./rpgo break-even config.yaml \
  --solve tsp_balance \
  --participant "Alice Johnson" \
  --goal-annual-income 150000 \
  --retirement-date 2027-06-30
```

1. **Solve for TSP Withdrawal Rate:**

```bash
./rpgo break-even config.yaml \
  --solve tsp_withdrawal_rate \
  --target-net-income 175000
```

1. **Solve for SS Start Age:**

```bash
./rpgo break-even config.yaml \
  --solve ss_start_age \
  --participant "Alice Johnson" \
  --optimize lifetime_income  # or: first_year_income, tsp_longevity
```

**Architecture:**

```go
// Goal solver framework
type GoalSolver struct {
    Variable   SolverVariable        // What to solve for
    Objective  ObjectiveFunction     // What to optimize
    Constraint ConstraintFunction    // What must be satisfied
    Search     SearchStrategy        // Binary, gradient, etc.
    Tolerance  decimal.Decimal
}

type SolverVariable int
const (
    RetirementDate SolverVariable = iota
    TSPBalance
    TSPWithdrawalRate
    SSStartAge
)

type ObjectiveFunction func(*domain.ScenarioComparison) decimal.Decimal

type SearchStrategy interface {
    Solve(config *domain.Configuration,
          variable SolverVariable,
          objective ObjectiveFunction,
          constraints []ConstraintFunction) (Solution, error)
}
```

**Search Strategies:**

- **Binary Search** - For monotonic objectives (retirement date, TSP balance)
- **Grid Search** - For discrete variables (SS start age 62-70)
- **Golden Section** - For unimodal continuous objectives
- **Constraint Propagation** - For multi-constraint problems

**Output:**

```text
BREAK-EVEN ANALYSIS: Earliest Retirement Date
=================================================================
Target Net Income:    $180,000/year
Participant:          Alice Johnson
Constraints:
  • SS Start Age:     62 (from config)
  • TSP Strategy:     4% Rule (from config)
  • TSP Balance:      $1,966,168.86 (from config)

SOLUTION:
✓ Earliest Retirement Date: June 30, 2029

PROJECTION FOR SOLUTION:
  First Year Net Income:  $181,234  (meets target)
  Year 5 Net Income:      $189,567
  TSP Longevity:          27 years
  IRMAA Risk:             Low

SENSITIVITY:
  Retire 3 months earlier:  Net = $177,890  ❌ Below target
  Retire 3 months later:    Net = $183,456  ✓ Above target

RECOMMENDATION: Retire June 30, 2029 or later to maintain $180K target.
```

**Implementation:**

- [ ] Create `internal/solver/` package
- [ ] Goal solver framework
- [ ] Binary search strategy
- [ ] Grid search strategy
- [ ] Solver for retirement date
- [ ] Solver for TSP balance
- [ ] Solver for TSP withdrawal rate
- [ ] Solver for SS start age
- [ ] Convergence criteria & iteration limits
- [ ] Sensitivity analysis (±1 step)
- [ ] Integration with transform pipeline
- [ ] Console output formatter
- [ ] Unit tests for each solver mode
- [ ] Integration tests with sample configs
- [ ] Performance tests (should solve in <10 seconds)

**Deliverables:**

- [ ] `./rpgo break-even` with 4 solver modes
- [ ] Documentation with examples
- [ ] Tests achieving 80%+ coverage

---

### 1.4 Bubble Tea TUI Foundation (5-7 days) ✅ FOUNDATION COMPLETE

**Priority:** HIGH - Transforms user experience

**Status:** Foundation complete (October 2025). Core architecture and scene navigation implemented. Components and interactive features in progress.

**Inspiration:** Charm's Crush (<https://github.com/charmbracelet/crush>)

**Completed (Foundation):**

- ✅ Created `cmd/rpgo-tui/main.go` - TUI entry point
- ✅ Basic TUI scaffolding (Model-Update-View pattern)
- ✅ Scene navigation system (7 scenes: Home, Scenarios, Parameters, Compare, Optimize, Results, Help)
- ✅ Keyboard shortcuts (h, s, p, c, o, r, ?, ESC, q)
- ✅ Message-driven architecture with comprehensive message types
- ✅ Application chrome (title bar, status bar, error/loading states)
- ✅ Lipgloss styling system with professional color palette
- ✅ Help screen with keyboard documentation
- ✅ See [Phase 1.4 Foundation Complete](PHASE_1.4_FOUNDATION_COMPLETE.md) for details

**Initial TUI Features:**

1. **Config Loading & Scenario Selection**
2. **Live Parameter Adjustment** (arrow keys, number input)
3. **Real-time Recalculation** (with loading spinner)
4. **Multi-Pane View** (parameters | results | comparison)
5. **Keyboard Shortcuts** (hjkl, arrow keys, tab)
6. **Save Modified Scenario**

**UI Layout:**

```text
┌─ RPGO: FERS Retirement Planning ─────────────────────────────────────────────────┐
│ Scenario: Early Retirement 2027                          [R]ecalc [S]ave [Q]uit │
├────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                    │
│ ┌─ Household ──────────────┐  ┌─ Key Metrics ─────────────────────────────────┐ │
│ │                          │  │                                                │ │
│ │ > Alice Johnson          │  │ Current Net Income:    $179,773               │ │
│ │   • Age: 60              │  │ First Retirement Year: $156,234  (-$23,539)   │ │
│ │   • Federal Employee     │  │ Year 5 Net Income:     $168,891  (-$10,882)   │ │
│ │   • TSP: $1,966,169      │  │ Year 10 Net Income:    $175,420  (-$4,353)    │ │
│ │                          │  │                                                │ │
│ │   Bob Smith              │  │ TSP Longevity:         28 years               │ │
│ │   • Age: 62              │  │ TSP Final Balance:     $234,567               │ │
│ │   • Private Sector       │  │ Lifetime PV Income:    $4,234,567             │ │
│ │   • External Pension     │  │                                                │ │
│ │                          │  │ IRMAA Risk:            🟡 Medium (Year 15+)   │ │
│ └──────────────────────────┘  └────────────────────────────────────────────────┘ │
│                                                                                    │
│ ┌─ Alice's Scenario ───────────────────────────────────────────────────────────┐ │
│ │                                                                              │ │
│ │ Retirement Date       ⬅️  2027-06-30  ➡️   [Enter to edit]                 │ │
│ │ SS Start Age          ⬅️      62      ➡️   (age 62-70)                      │ │
│ │ TSP Withdrawal        ⬅️   4% Rule    ➡️   [variable_pct | need_based]     │ │
│ │ TSP Withdrawal Rate   ⬅️     4.0%     ➡️   (if variable_pct)               │ │
│ │                                                                              │ │
│ └──────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                    │
│ ┌─ Bob's Scenario ─────────────────────────────────────────────────────────────┐ │
│ │                                                                              │ │
│ │ SS Start Age          ⬅️      65      ➡️   (age 62-70)                      │ │
│ │                                                                              │ │
│ └──────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                    │
│ ┌─ Quick Actions ──────────────────────────────────────────────────────────────┐ │
│ │ [C]ompare with Scenarios  [B]reak-even Analysis  [I]RMM Analysis            │ │
│ └──────────────────────────────────────────────────────────────────────────────┘ │
│                                                                                    │
└────────────────────────────────────────────────────────────────────────────────────┘
```

**Key Libraries:**

- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - UI components (spinner, table, input)
- `github.com/muesli/termenv` - Terminal capabilities detection

**Implementation:**

- [ ] Create `cmd/interactive.go`
- [ ] Basic TUI scaffolding (model, update, view)
- [ ] Config loading in TUI
- [ ] Multi-pane layout (lipgloss)
- [ ] Participant list navigation
- [ ] Parameter adjustment UI (arrow keys)
- [ ] Number input component
- [ ] Date picker component
- [ ] Enum selector (TSP strategy, filing status)
- [ ] Calculation trigger (with spinner)
- [ ] Results display pane
- [ ] Save scenario dialog
- [ ] Keyboard help overlay (?)
- [ ] Error display (toast/banner)
- [ ] Status bar with hints
- [ ] Integration tests (harder for TUI, may defer)

**Deliverables:**

- [ ] `./rpgo interactive config.yaml` launches TUI
- [ ] Can adjust all key parameters
- [ ] Live recalculation works
- [ ] Can save modified scenarios
- [ ] Responsive to terminal resize
- [ ] Keyboard shortcuts documented

**Future TUI Enhancements (Tier 2+):**

- Side-by-side scenario comparison pane
- Inline sparkline charts (income over time)
- IRMAA risk visualization
- Drill-down to year-by-year breakdown
- Diff view (base vs. modified)

---

## PHASE 2: CORE FEATURES (Weeks 5-8)

### 2.1 IRMAA Threshold Alerts (2-3 days)

**Priority:** HIGH - Quick win, high perceived value

**Scope:**

Medicare IRMAA (Income-Related Monthly Adjustment Amount) adds surcharges based on MAGI (Modified Adjusted Gross Income). This is a **tax cliff** situation that retirees need to avoid.

**2025 IRMAA Thresholds (Married Filing Jointly):**

- $206,000 - $258,000: +$69.90/month Part B (+$12.90 Part D)
- $258,000 - $322,000: +$174.70/month Part B (+$33.20 Part D)
- $322,000 - $386,000: +$279.50/month Part B (+$53.50 Part D)
- $386,000 - $750,000: +$384.30/month Part B (+$73.80 Part D)
- $750,000+: +$419.30/month Part B (+$81.90 Part D)

**MAGI Calculation:**

```go
// MAGI = AGI + tax-exempt interest + excluded foreign income + non-taxable SS
func CalculateMAGI(cashFlow *domain.AnnualCashFlow) decimal.Decimal {
    agi := cashFlow.GetTotalSalary().
        Add(cashFlow.GetTotalPension()).
        Add(cashFlow.GetTotalTSPWithdrawal()).  // Traditional only
        Add(cashFlow.GetTotalSSBenefit()).      // Full SS, not just taxable portion
        Sub(cashFlow.FederalStandardDeduction)

    // Add back tax-exempt interest if tracked
    // For now, assume negligible

    return agi
}
```

**Alert Thresholds:**

- 🟢 **Safe:** More than $10,000 below next threshold
- 🟡 **Caution:** Within $10,000 of next threshold
- 🔴 **Risk:** Within $5,000 of next threshold
- ❌ **Breach:** Crossed into higher tier

**Output Format:**

```text
IRMAA RISK ANALYSIS
=================================================================
Filing Status: Married Filing Jointly

Year  MAGI       Threshold   Margin    Risk     Part B    Part D    Annual Impact
---------------------------------------------------------------------------------
2027  $187,500   $206,000   +$18,500   🟢 Safe  $2,096    $558     Baseline
2028  $198,000   $206,000    +$8,000   🟡 Warn  $2,096    $558     Baseline
2029  $204,500   $206,000    +$1,500   🔴 Risk  $2,096    $558     Baseline
2030  $211,000   $206,000    -$5,000   ❌ BREACH $2,935   $713     +$994/year
2031  $219,000   $206,000   -$13,000   ❌ BREACH $2,935   $713     +$994/year
...

RECOMMENDATIONS:
⚠️  Years 2030-2035 breach first IRMAA tier (+$994/year each)
💡 Consider Roth conversions in low-MAGI years (2027-2029)
💡 Defer TSP withdrawals in 2030 if possible
💡 Time capital gains to avoid threshold breaches
```

**Integration Points:**

- Add `IRMAARiskAnalysis` to `ScenarioSummary`
- Extend `AnnualCashFlow` with MAGI calculation
- New output section in console formatter
- New pane in TUI showing IRMAA timeline
- HTML report gets IRMAA risk chart

**Implementation:**

- [ ] MAGI calculation function
- [ ] IRMAA tier lookup (2025 thresholds)
- [ ] Risk classification logic
- [ ] Annual surcharge calculation
- [ ] Risk timeline generator
- [ ] Console formatter section
- [ ] TUI IRMAA pane
- [ ] HTML chart integration
- [ ] Recommendation engine (simple heuristics)
- [ ] Unit tests for MAGI calculation
- [ ] Tests for each risk tier
- [ ] Integration test with sample projection

**Deliverables:**

- [ ] IRMAA analysis in all output formats
- [ ] Actionable recommendations
- [ ] Tests covering all tiers

---

### 2.2 Tax-Smart Withdrawal Sequencing (7-10 days)

**Priority:** HIGH - Material impact on longevity and taxes

**Context:**
The order in which you withdraw from different account types affects:

- Current year taxes
- Future RMDs
- IRMAA eligibility
- Estate planning

**Common Sequencing Strategies:**

1. **Standard:** Taxable → Traditional TSP → Roth TSP
2. **Tax-Efficient:** Roth → Traditional (minimize RMDs)
3. **Bracket-Fill:** Fill current tax bracket from Traditional, rest from Roth
4. **IRMAA-Aware:** Minimize Traditional withdrawals in high-income years

**Config Schema Addition:**

```yaml
household:
  filing_status: "married_filing_jointly"
  participants:
    - name: "Alice Johnson"
      # ... existing fields ...

      # NEW: Taxable account tracking
      taxable_account_balance: "250000"
      taxable_account_basis: "200000"  # For capital gains calculation

      # NEW: Withdrawal sequencing
      withdrawal_sequencing:
        strategy: "tax_efficient"  # standard | tax_efficient | bracket_fill | custom
        custom_sequence: ["taxable", "roth", "traditional"]  # if strategy=custom

        # Advanced: bracket-fill parameters
        target_bracket: 22  # Fill up to 22% bracket
        bracket_buffer: 5000  # Stay $5k below bracket edge
```

**Withdrawal Sequencing Engine:**

```go
type WithdrawalSource struct {
    Name         string
    Balance      decimal.Decimal
    TaxTreatment TaxTreatment  // TaxFree, Ordinary, CapitalGains
    Priority     int            // For custom sequencing
}

type TaxTreatment int
const (
    TaxFree TaxTreatment = iota      // Roth
    OrdinaryIncome                    // Traditional TSP, pensions
    CapitalGains                      // Taxable account
    PreTax                            // Salary (don't withdraw, but track)
)

type WithdrawalSequencer interface {
    CalculateWithdrawals(
        needAmount decimal.Decimal,
        sources []WithdrawalSource,
        currentIncome decimal.Decimal,
        taxContext TaxContext,
    ) WithdrawalPlan
}

type WithdrawalPlan struct {
    Withdrawals map[string]decimal.Decimal  // source -> amount
    TaxImpact   decimal.Decimal
    MAGIImpact  decimal.Decimal
    Remaining   decimal.Decimal  // Unmet need (if sources depleted)
}
```

**Implementation:**

- [ ] Add taxable account fields to domain
- [ ] Withdrawal source abstraction
- [ ] Standard sequencing strategy
- [ ] Tax-efficient sequencing strategy
- [ ] Bracket-fill sequencing strategy
- [ ] Custom sequencing strategy
- [ ] IRMAA-aware sequencing (stretch)
- [ ] Capital gains calculation (for taxable)
- [ ] Integrate into projection engine
- [ ] Add withdrawal breakdown to AnnualCashFlow
- [ ] Console output showing source breakdown
- [ ] TUI visualization of withdrawal sources
- [ ] Tests for each strategy
- [ ] Comparison tests (strategy A vs B)
- [ ] Integration tests with full projections

**Output Enhancement:**

```text
YEAR 2028 WITHDRAWAL BREAKDOWN
=================================================================
Net Income Need:          $45,000  (after pensions, SS)

Withdrawal Sequence (Tax-Efficient Strategy):
  1. Roth TSP:           $25,000  (Tax-Free)
  2. Traditional TSP:    $20,000  (Ordinary Income)

Tax Impact:
  Federal Tax on $20K:    $2,400  (12% bracket)
  MAGI Addition:          $20,000
  IRMAA Impact:           None (below threshold)

Account Balances After Withdrawal:
  Taxable:            $250,000  (untouched)
  Traditional TSP:    $918,456  (↓ $20,000)
  Roth TSP:           $142,789  (↓ $25,000)
```

**Deliverables:**

- [ ] Full withdrawal sequencing support
- [ ] 3+ sequencing strategies
- [ ] Integration with projection
- [ ] Enhanced output showing breakdown
- [ ] Tests with 80%+ coverage

---

### 2.3 Roth Conversion Planner (10-12 days)

**Priority:** HIGH - Sophisticated tax optimization, full calculations required

**Context:**
Roth conversions involve paying tax now to avoid:

- Future RMDs on Traditional TSP
- Higher tax rates in retirement
- IRMAA surcharges
- Estate tax for heirs

**Key Principle:** Convert in **low-income years** (before SS, after retirement, before RMDs).

**Config Schema Addition:**

```yaml
scenarios:
  - name: "Roth Conversion Strategy"
    participant_scenarios:
      "Alice Johnson":
        retirement_date: "2027-06-30"
        ss_start_age: 67  # Delay SS to create conversion window

        # NEW: Roth conversion schedule
        roth_conversions:
          - year: 2028
            amount: "50000"
            source: "traditional_tsp"  # or "traditional_ira" if supported
          - year: 2029
            amount: "75000"
            source: "traditional_tsp"
          - year: 2030
            amount: "50000"
            source: "traditional_tsp"
```

**Roth Conversion Planner Command:**

```bash
# Find optimal conversion strategy
./rpgo plan-roth config.yaml \
  --participant "Alice Johnson" \
  --window 2028-2032 \
  --target-bracket 22 \
  --objective minimize_lifetime_tax

# Output: Recommended conversion amounts per year
```

**Conversion Impact Modeling:**

When converting $X from Traditional to Roth in Year Y:

1. **Immediate Tax:** $X added to ordinary income → increases tax by $X * marginal_rate
2. **Future RMD Reduction:** Lower Traditional balance → lower RMDs at age 73+
3. **Future IRMAA Impact:** Lower RMDs → lower MAGI → may avoid IRMAA surcharges
4. **Estate Benefit:** Roth passes tax-free to heirs (not modeled initially)
5. **Opportunity Cost:** Tax paid now vs. invested

**Full Calculation Algorithm:**

```go
func PlanRothConversions(
    config *domain.Configuration,
    participant string,
    window YearRange,
    targetBracket int,
    objective OptimizationObjective,
) *RothConversionPlan {
    // 1. Run baseline scenario (no conversions)
    baseline := runFullProjection(config)

    // 2. Generate candidate conversion strategies
    // Strategy: Convert up to bracket edge each year in window
    candidates := []ConversionStrategy{}
    for year := window.Start; year <= window.End; year++ {
        // Calculate available "room" in target bracket
        yearIncome := baseline.Projection[year].TotalGrossIncome
        bracketRoom := calculateBracketRoom(yearIncome, targetBracket)

        if bracketRoom.GreaterThan(decimal.Zero) {
            candidates = append(candidates, ConversionStrategy{
                Year: year,
                Amount: bracketRoom,
            })
        }
    }

    // 3. For each candidate, run FULL projection with conversion
    results := []ConversionOutcome{}
    for _, strategy := range candidates {
        modifiedConfig := applyConversions(config, strategy)
        projection := runFullProjection(modifiedConfig)

        outcome := ConversionOutcome{
            Strategy: strategy,
            Projection: projection,
            LifetimeTax: calculateLifetimeTax(projection),
            LifetimeIRMAA: calculateLifetimeIRMAA(projection),
            FinalBalances: projection.FinalBalances,
        }
        results = append(results, outcome)
    }

    // 4. Compare outcomes based on objective
    optimal := selectOptimal(baseline, results, objective)

    return &RothConversionPlan{
        Baseline: baseline,
        Recommended: optimal,
        Alternatives: results,
        Analysis: compareOutcomes(baseline, results),
    }
}
```

**Objectives:**

- `minimize_lifetime_tax` - Total federal tax paid over projection
- `minimize_lifetime_irmaa` - Total IRMAA surcharges paid
- `minimize_combined` - Tax + IRMAA combined
- `maximize_estate` - Final Roth balance (for heirs)
- `maximize_net_income` - Average annual net income

**Output:**

```text
ROTH CONVERSION ANALYSIS
=================================================================
Participant: Alice Johnson
Conversion Window: 2028-2032 (after retirement, before SS at 67)
Target Bracket: 22%
Objective: Minimize Lifetime Tax + IRMAA

BASELINE (No Conversions):
  Lifetime Federal Tax:     $487,234
  Lifetime IRMAA:           $18,450
  Total Cost:               $505,684
  Final Traditional TSP:    $234,567
  Final Roth TSP:           $89,123

RECOMMENDED STRATEGY:
  2028: Convert $45,000  (fills 22% bracket, tax = $9,900)
  2029: Convert $52,000  (fills 22% bracket, tax = $11,440)
  2030: Convert $48,000  (fills 22% bracket, tax = $10,560)

  Total Converted: $145,000
  Total Tax Paid:  $31,900 (22% effective on conversions)

OUTCOME WITH CONVERSIONS:
  Lifetime Federal Tax:     $519,134  (+$31,900 conversion tax)
  Lifetime IRMAA:           $9,200    (-$9,250 saved!)
  Total Cost:               $528,334  (+$22,650 vs baseline)

  BUT: Lower RMDs starting 2038
       Avoid IRMAA breaches in years 2040-2045

  Final Traditional TSP:    $89,567   (↓ $145,000 converted)
  Final Roth TSP:           $234,123  (↑ $145,000)

NET BENEFIT OVER 30 YEARS:
  Tax Cost:      +$31,900 (paid upfront)
  IRMAA Savings: -$9,250  (years 2040-2045)
  RMD Reduction: -$15,400 (lower taxes on smaller RMDs)

  Total Benefit: -$24,650 (NET SAVINGS)
  ROI: 77% return on conversion tax paid

RECOMMENDATION: ✓ Execute Roth conversion strategy
  • Converts during low-income window
  • Avoids IRMAA surcharges
  • Reduces future RMD tax burden
  • Provides tax-free income flexibility

SENSITIVITY:
  If convert 20% more:  Net benefit = -$28,123 (better)
  If convert 20% less:  Net benefit = -$19,890 (worse)

  Optimal range: $130,000 - $160,000 total conversions
```

**Implementation:**

- [ ] Roth conversion transform
- [ ] Conversion impact in tax calculation
- [ ] Traditional → Roth balance transfer
- [ ] RMD recalculation with lower Traditional balance
- [ ] MAGI recalculation
- [ ] Conversion planner command
- [ ] Bracket room calculator
- [ ] Multi-year optimization
- [ ] Objective functions (5 types)
- [ ] Comparison engine
- [ ] NPV calculation (optional, for opportunity cost)
- [ ] Sensitivity analysis
- [ ] Output formatter
- [ ] Unit tests for conversion logic
- [ ] Tests for each objective
- [ ] Integration tests with full scenarios
- [ ] Performance tests (should complete in <30 seconds)

**Deliverables:**

- [ ] `./rpgo plan-roth` command
- [ ] Full calculation with feedback loops
- [ ] Multiple optimization objectives
- [ ] Comprehensive output
- [ ] Tests with 80%+ coverage
- [ ] Documentation with examples

---

### 2.4 Healthcare Cost Expansion (5-7 days)

**Priority:** HIGH - Major real-world cost driver

**Current State:** Only FEHB premiums and Medicare Part B with IRMAA modeled.

**Gaps:**

1. Pre-65 healthcare coverage (retire before Medicare eligibility)
2. Medicare Part D (prescription drug coverage)
3. Medigap/Medicare Supplement
4. FEHB → Medicare transition
5. Per-person vs. household costs

**Config Schema Addition:**

```yaml
household:
  participants:
    - name: "Alice Johnson"
      # ... existing fields ...

      # NEW: Healthcare configuration
      healthcare:
        # Pre-Medicare (before age 65)
        pre_medicare_coverage: "fehb"  # fehb | cobra | marketplace | retiree_plan
        pre_medicare_monthly_premium: "0"  # If not FEHB (FEHB uses participant's fehb_premium)

        # Medicare (age 65+)
        medicare_part_b: true  # Default true
        medicare_part_d: true  # Default true
        medicare_part_d_plan: "standard"  # standard | enhanced
        medigap_plan: "G"  # A-N, or none

        # Transition
        drop_fehb_at_65: true  # Stop FEHB when Medicare eligible

global_assumptions:
  healthcare_inflation:
    fehb: "0.06"           # 6% annual increase
    medicare_b: "0.05"     # 5% annual increase
    medicare_d: "0.05"     # 5% annual increase
    medigap: "0.04"        # 4% annual increase
    marketplace: "0.07"    # 7% annual increase
```

**Medicare Part D Costs (2025):**

- Standard: ~$35/month base premium + IRMAA
- Enhanced: ~$50/month for better coverage

**Medicare Part D IRMAA (same thresholds as Part B):**

- < $206,000 MAGI: $0 surcharge
- $206,000-$258,000: +$12.90/month
- $258,000-$322,000: +$33.20/month
- $322,000-$386,000: +$53.50/month
- $386,000-$750,000: +$73.80/month
- $750,000+: +$81.90/month

**Medigap Plan G (typical):**

- ~$150-250/month depending on age and location
- Covers Medicare copays, coinsurance, deductibles

**Healthcare Cost Calculation:**

```go
func (ce *CalculationEngine) CalculateHealthcareCosts(
    participant *domain.Participant,
    age int,
    year int,
    magi decimal.Decimal,
    filingStatus string,
) HealthcareCostBreakdown {

    breakdown := HealthcareCostBreakdown{}

    if age < 65 {
        // Pre-Medicare
        switch participant.Healthcare.PreMedicareCoverage {
        case "fehb":
            premium := participant.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26))
            breakdown.FEHBPremium = inflateFromBase(premium, year, fehbInflationRate)
        case "marketplace":
            premium := participant.Healthcare.PreMedicareMonthlyPremium.Mul(decimal.NewFromInt(12))
            breakdown.MarketplacePremium = inflateFromBase(premium, year, marketplaceInflationRate)
        // ... other cases
        }
    } else {
        // Medicare (age 65+)

        // Part B
        if participant.Healthcare.MedicarePartB {
            basePremium := decimal.NewFromFloat(174.70)  // 2025 standard
            partBPremium := inflateFromBase(basePremium.Mul(decimal.NewFromInt(12)), year, medicareInflationRate)

            // Add IRMAA
            irmaa := ce.MedicareCalc.CalculatePartBPremium(magi, filingStatus, year)
            breakdown.MedicarePartB = partBPremium.Add(irmaa.Sub(basePremium.Mul(decimal.NewFromInt(12))))
        }

        // Part D
        if participant.Healthcare.MedicarePartD {
            var basePremium decimal.Decimal
            switch participant.Healthcare.MedicarePartDPlan {
            case "standard":
                basePremium = decimal.NewFromFloat(35.0)
            case "enhanced":
                basePremium = decimal.NewFromFloat(50.0)
            }

            partDPremium := inflateFromBase(basePremium.Mul(decimal.NewFromInt(12)), year, medicareInflationRate)

            // Add Part D IRMAA
            partDIRMAA := calculatePartDIRMAA(magi, filingStatus)
            breakdown.MedicarePartD = partDPremium.Add(partDIRMAA)
        }

        // Medigap
        if participant.Healthcare.MedigapPlan != "" {
            baseCost := getMedigapBaseCost(participant.Healthcare.MedigapPlan, age)
            breakdown.Medigap = inflateFromBase(baseCost, year, medigapInflationRate)
        }

        // FEHB drops if configured
        if !participant.Healthcare.DropFEHBAt65 {
            // Keep FEHB as secondary (rare but possible)
            premium := participant.FEHBPremiumPerPayPeriod.Mul(decimal.NewFromInt(26))
            breakdown.FEHBPremium = inflateFromBase(premium, year, fehbInflationRate)
        }
    }

    breakdown.Total = breakdown.FEHBPremium.
        Add(breakdown.MarketplacePremium).
        Add(breakdown.MedicarePartB).
        Add(breakdown.MedicarePartD).
        Add(breakdown.Medigap)

    return breakdown
}
```

**Implementation:**

- [ ] Healthcare config schema
- [ ] Pre-Medicare coverage options
- [ ] Medicare Part D premium calculation
- [ ] Medicare Part D IRMAA calculation
- [ ] Medigap cost lookup table (by plan + age)
- [ ] Healthcare inflation by type
- [ ] FEHB → Medicare transition logic
- [ ] Per-person healthcare calculation
- [ ] Aggregate household healthcare costs
- [ ] Add HealthcareCostBreakdown to AnnualCashFlow
- [ ] Console output section
- [ ] TUI healthcare detail pane
- [ ] HTML chart for healthcare costs over time
- [ ] Tests for each coverage type
- [ ] Tests for transition at age 65
- [ ] Tests for IRMAA impact on Part D
- [ ] Integration tests with full projection

**Deliverables:**

- [ ] Full healthcare cost modeling
- [ ] Pre-65 coverage options
- [ ] Medicare Part D + IRMAA
- [ ] Medigap integration
- [ ] Enhanced output showing breakdown
- [ ] Tests with 80%+ coverage

---

## PHASE 3: ADVANCED FEATURES (Weeks 9-12)

### 3.1 Survivor Viability Analysis (3-5 days) ✅ COMPLETE

**Priority:** HIGH - Critical for couples' retirement planning

**Status:** Complete (December 2024). Comprehensive survivor viability analysis implemented.

**Scope:** Model financial impact when one spouse dies

**Config Enhancement:**

```yaml
scenarios:
  - name: "Mortality Stress Test"
    mortality:
      participants:
        "Alice Johnson":
          death_date: "2034-06-30"  # Age 69
      assumptions:
        survivor_spending_factor: "0.75"  # Survivor needs 75% of couple's spending
        tsp_spousal_transfer: "merge"     # Combine TSP balances
        filing_status_switch: "next_year"  # Switch to single filing
```

**Output:**

```text
SURVIVOR VIABILITY ANALYSIS
=================================================================
Scenario: Alice passes in 2034 (age 69)
Survivor: Bob (age 71 at time of death)

PRE-DEATH (Married, 2033):
  Combined Net Income:    $175,420/year
  Monthly:                $14,618
  Healthcare Costs:       $18,450/year (both on Medicare)

POST-DEATH (Survivor, 2035+):
  Bob's Net Income:       $118,234/year  (-32.6%)
  Monthly:                $9,853         (-32.6%)
  Healthcare Costs:       $9,225/year    (single person)

  Income Sources:
    Bob's Pension:        $42,000
    Bob's SS:             $38,400
    Alice's Survivor SS:  $19,200  (50% of Alice's benefit)
    TSP Withdrawal:       $18,634  (merged balances)

  Filing Status:        Single (starting 2035)
  Tax Impact:           +$8,450/year (single filer, narrower brackets)

VIABILITY ASSESSMENT:
  Survivor Income vs. Target (75% of couple):
    Target:    $131,565  (75% of $175,420)
    Actual:    $118,234
    Shortfall: -$13,331  (-10.1%)  🟡 CAUTION

  TSP Longevity:
    Before death:  28 years (couple)
    After death:   19 years (survivor)  ⚠️ Reduced by 9 years

  IRMAA Risk:
    Before death:  Low
    After death:   Moderate (single filer thresholds lower)

RECOMMENDATION:
  ⚠️  Survivor income falls short of 75% spending target by $13,331/year
  💡 Consider increasing life insurance to cover gap
  💡 Review survivor benefit elections on pensions
  💡 Build up Roth TSP (tax-free withdrawals help single filer)

LIFE INSURANCE NEEDS:
  To bridge $13,331/year gap for 19 years:
    Present Value (4% discount): $178,456
    Recommended coverage: $200,000
```

**Implementation:**

- [x] Survivor income calculation
- [x] Tax impact of filing status change
- [x] TSP balance merge logic
- [x] Survivor SS benefit calculation
- [x] Viability score calculation
- [x] Life insurance needs estimator
- [x] Output formatter
- [x] Tests for survivor scenarios
- [x] Integration with existing mortality code

**Deliverables:**

- [x] `./rpgo analyze-survivor` command
- [x] Comprehensive pre-death vs post-death analysis
- [x] Viability assessment with scoring (EXCELLENT/GOOD/CAUTION/RISK/CRITICAL)
- [x] Life insurance needs calculation with present value
- [x] Alternative strategies and recommendations
- [x] Flexible configuration options
- [x] Console output formatter
- [x] Integration with existing projection engine

---

### 3.2 Part-Time Work Modeling (5-7 days) ✅ COMPLETE

**Priority:** HIGH - Enables phased retirement scenarios

**Status:** Complete (December 2024). Comprehensive part-time work modeling implemented.

**Scope:** Model phased retirement with reduced hours/salary

**Config Enhancement:**

```yaml
scenarios:
  - name: "Phased Retirement"
    participant_scenarios:
      "Alice Johnson":
        retirement_date: "2030-06-30"  # Full retirement

        # NEW: Part-time work before full retirement
        part_time_work:
          start_date: "2027-01-01"
          end_date: "2030-06-30"

          schedule:
            - period_start: "2027-01-01"
              period_end: "2028-12-31"
              annual_salary: "95000"  # 50% time
              tsp_contribution_percent: "0.15"
              work_type: "w2"  # w2 | 1099

            - period_start: "2029-01-01"
              period_end: "2030-06-30"
              annual_salary: "60000"  # 33% time
              tsp_contribution_percent: "0.10"
              work_type: "w2"

        # FERS Supplement earnings test applies during part-time
        # (Supplement reduced/eliminated if earnings > $23,400)
```

**Implementation:**

- [x] Part-time work schedule parser
- [x] Salary proration by period
- [x] TSP contribution calculation for part-time
- [x] FICA calculation for part-time
- [x] FERS supplement earnings test
- [x] W-2 vs 1099 distinction (basic, defer self-employment tax)
- [x] Integration into projection engine
- [x] Output showing part-time periods
- [x] Tests for various schedules
- [x] Tests for earnings test impact

**Deliverables:**

- [x] `PartTimeWorkSchedule` domain model with validation
- [x] `PartTimeWorkCalculator` with comprehensive calculations
- [x] FERS supplement earnings test implementation
- [x] Self-employment tax calculation for 1099 work
- [x] Integration with projection engine
- [x] Console output showing part-time work details
- [x] Flexible configuration with multiple periods
- [x] Support for both W-2 and 1099 work types

---

### 3.3 Inflation Sensitivity Analysis (3-4 days)

**Scope:** Parameter sweep to test robustness

**Command:**

```bash
# Sweep single parameter
./rpgo sensitivity config.yaml \
  --parameter inflation_rate \
  --range 0.015-0.040 \
  --steps 6

# Sweep multiple parameters (2D matrix)
./rpgo sensitivity config.yaml \
  --parameter inflation_rate:0.015-0.040:6 \
  --parameter tsp_return_post_retirement:0.03-0.06:4 \
  --output matrix
```

**Output:**

```text
SENSITIVITY ANALYSIS: Inflation Rate
=================================================================
Base Case: inflation_rate = 2.5%
Range: 1.5% to 4.0% (6 steps)

Inflation  Year 5 Net   Year 10 Net   TSP Longevity   Lifetime Income
------------------------------------------------------------------------
1.5%       $172,890     $181,234      35 years        $4,567,890
2.0%       $170,123     $177,890      32 years        $4,450,123
2.5%       $168,891     $175,420      28 years        $4,234,567  ← BASE
3.0%       $165,234     $170,890      25 years        $4,012,345
3.5%       $162,890     $167,123      23 years        $3,890,234
4.0%       $159,456     $162,789      20 years        $3,678,901

SENSITIVITY:
  +1% inflation → -$8,333 Year 5 income (-4.9%)
  +1% inflation → -8 years TSP longevity (-28.6%)  ⚠️ HIGH SENSITIVITY

RECOMMENDATION: Plan conservatively. TSP longevity highly sensitive to inflation.
```

**Implementation:**

- [ ] Parameter sweep framework
- [ ] Single-parameter sweep
- [ ] Multi-parameter matrix
- [ ] Output formatters (table, CSV, JSON)
- [ ] Sensitivity metrics calculation
- [ ] Tests for sweep logic

---

### 3.4 Enhanced HTML Reports (5-7 days)

**Scope:** Desktop-optimized, information-dense reports with drill-down

**New Features:**

1. **Interactive Charts:**
   - Income over time (stacked: salary, pension, TSP, SS)
   - TSP balance trajectory
   - Tax breakdown by year
   - Healthcare costs over time
   - IRMAA risk timeline

2. **Drill-Down Tables:**
   - Click year → see detailed cash flow for that year
   - Click "taxes" → see bracket utilization
   - Click "healthcare" → see cost breakdown

3. **Comparison Views:**
   - Side-by-side scenario comparison
   - Diff highlighting (what changed?)
   - Break-even visualization

4. **Risk Indicators:**
   - IRMAA breach warnings
   - TSP depletion risk
   - Tax bracket jumps

**Implementation:**

- [ ] Enhanced HTML templates
- [ ] JavaScript for interactivity
- [ ] Chart.js integration (or similar)
- [ ] Collapsible sections
- [ ] Scenario comparison tables
- [ ] Export to standalone HTML (no external deps)
- [ ] Tests for HTML generation

---

## PHASE 4: POLISH & STABILITY (Ongoing)

### 4.1 Testing & Quality (Ongoing)

**Test Coverage Goals:**

- Unit tests: 80%+ coverage
- Integration tests: All major commands
- Golden tests: Stable output validation
- Performance tests: Commands complete in < 30 seconds

**Testing Strategy:**

- Table-driven tests for all calculations
- Property-based tests for solver convergence
- Snapshot tests for output formats
- Benchmark tests for performance regression

---

### 4.2 Documentation (Ongoing)

**Inline Documentation:**

- All public functions have godoc comments
- Complex algorithms explained with comments
- Financial formulas cited with sources

**User Documentation:**

- README with examples for each command
- Configuration guide with all options
- FAQ for common questions
- Troubleshooting guide

**Developer Documentation:**

- Architecture overview
- Contributing guide
- Testing guide
- Release process

---

### 4.3 Open Source Preparation (Before Release)

**Legal & Licensing:**

- [ ] Choose license (MIT recommended for broad adoption)
- [ ] Add LICENSE file
- [ ] Copyright notices in source files
- [ ] Dependency audit (all licenses compatible)

**Project Metadata:**

- [ ] Clear README with badges
- [ ] CONTRIBUTING.md
- [ ] CODE_OF_CONDUCT.md
- [ ] Issue templates
- [ ] PR templates
- [ ] CI/CD setup (GitHub Actions)

**Release Preparation:**

- [ ] Semantic versioning (start at 0.1.0)
- [ ] Changelog
- [ ] Binary releases (multi-platform)
- [ ] Docker image (optional)
- [ ] Homebrew formula (optional)

---

## Technical Debt & Future Considerations

### Known Limitations (Document, Don't Fix Yet)

1. **State/Local Tax:** Only Pennsylvania implemented
2. **External Pensions:** Schema exists, not fully integrated
3. **RMD:** Basic logic present, could be enhanced
4. **Monte Carlo + Tax:** Separate tools, not integrated
5. **Stochastic Mortality:** Not planned for initial release

### Architecture Improvements (Later)

1. **Plugin System:** Allow custom transforms/strategies
2. **Scripting:** Lua/JavaScript for advanced users
3. **API Server:** REST API for integration with other tools
4. **Web UI:** Separate project, consumes API
5. **Database Backend:** Store scenarios, history, comparisons

---

## Success Metrics

### Phase 1 Success

- [ ] `./rpgo compare` works with 10+ built-in templates
- [ ] `./rpgo break-even` solves for 4 dimensions
- [ ] `./rpgo interactive` launches TUI with parameter adjustment
- [ ] Transform pipeline enables easy scenario exploration
- [ ] Tests pass with 80%+ coverage

### Phase 2 Success

- [ ] IRMAA alerts show in all output formats
- [ ] Tax-smart withdrawal sequencing operational
- [ ] Roth conversion planner provides actionable recommendations
- [ ] Healthcare costs fully modeled (pre-65, Part D, Medigap)
- [ ] All features have comprehensive tests

### Phase 3 Success

- [ ] Survivor analysis provides viability scores
- [ ] Part-time work modeling integrated
- [ ] Sensitivity analysis reveals parameter impacts
- [ ] HTML reports are information-dense and interactive

### Open Source Launch Success

- [ ] GitHub repository public with clear README
- [ ] CI/CD passing
- [ ] Binary releases for Mac/Linux/Windows
- [ ] 5+ GitHub stars in first month
- [ ] 1+ external contributor

---

## Timeline Summary

| Phase | Duration | Key Deliverables |
|-------|----------|-----------------|
| Phase 1 | 4 weeks | Transform pipeline, compare, break-even, TUI foundation |
| Phase 2 | 4 weeks | IRMAA alerts, tax sequencing, Roth planner, healthcare |
| Phase 3 | 4 weeks | Survivor analysis, part-time work, sensitivity, HTML |
| Phase 4 | Ongoing | Testing, docs, polish, open source prep |

**Total Estimated Time:** 12-16 weeks of focused development

---

## Next Steps

1. **Review & Approve** this implementation plan
2. **Start Phase 1.1:** Transform Pipeline Architecture (3-4 days)
3. **Daily Progress Updates** (via todo list)
4. **Weekly Milestone Reviews**

---

## Questions / Decisions Needed

### Immediate (Phase 1)

- [ ] Approve transform pipeline design
- [ ] Confirm 15 built-in transforms sufficient
- [ ] Agree on Bubble Tea dependencies

### Near-Term (Phase 2)

- [ ] Healthcare: Support multiple external pensions per participant?
- [ ] Roth planner: Include NPV discount rate for opportunity cost?
- [ ] Tax sequencing: Implement IRMAA-aware strategy in Phase 2 or defer?

### Long-Term (Phase 3+)

- [ ] HTML reports: Self-hosted vs. static file generation?
- [ ] Open source: GitHub organization or personal repo?
- [ ] Marketing: Write blog post about tool? Submit to aggregators?

---

## Revision History

| Date | Version | Changes | Author |
|------|---------|---------|--------|
| 2025-10-01 | 1.0 | Initial comprehensive plan created | Claude |

---

## End of implementation plan
