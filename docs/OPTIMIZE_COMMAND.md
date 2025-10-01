# Optimize Command Documentation

The `optimize` command uses advanced break-even solvers to automatically find optimal retirement parameters that achieve your specific goals. Instead of manually testing different scenarios, the optimizer searches the parameter space to find the best configuration.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Optimization Targets](#optimization-targets)
- [Optimization Goals](#optimization-goals)
- [Command Usage](#command-usage)
- [Constraints](#constraints)
- [Examples](#examples)
- [Understanding Results](#understanding-results)
- [Technical Details](#technical-details)

## Overview

The optimize command solves questions like:

- **"What TSP withdrawal rate will give me $120,000 per year?"** → `--target tsp_rate --goal match_income --target-income 120000`
- **"When should I retire to maximize lifetime income?"** → `--target retirement_date --goal maximize_income`
- **"What Social Security age minimizes my taxes?"** → `--target ss_age --goal minimize_taxes`
- **"What's the overall best strategy?"** → `--target all --goal maximize_income`

### Key Benefits

1. **Automated optimization**: No manual trial-and-error
2. **Multi-dimensional**: Compare multiple parameters simultaneously
3. **Goal-oriented**: Optimize for what matters to you
4. **Constraint-aware**: Respects realistic bounds on parameters
5. **Fast**: Binary search and grid search algorithms

## Quick Start

```bash
# Find TSP rate to match target income
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target tsp_rate \
  --goal match_income \
  --target-income 120000

# Find optimal retirement date
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target retirement_date \
  --goal maximize_income

# Run all optimizations
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target all \
  --goal maximize_income
```

## Optimization Targets

### TSP Withdrawal Rate (`tsp_rate`)

Finds the optimal TSP withdrawal percentage.

- **Use cases**: Match specific income targets, maximize portfolio longevity
- **Search range**: 2% to 10% (configurable with `--min-rate` and `--max-rate`)
- **Algorithm**: Binary search
- **Speed**: Fast (~10-20 iterations)

**Example:**
```bash
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target tsp_rate \
  --goal match_income \
  --target-income 150000 \
  --min-rate 0.03 \
  --max-rate 0.06
```

### Retirement Date (`retirement_date`)

Finds the optimal retirement date.

- **Use cases**: Maximize income, balance work vs. retirement timing
- **Search range**: -24 to +36 months from base scenario (configurable)
- **Algorithm**: Grid search (monthly granularity)
- **Speed**: Moderate (~60 evaluations for 5-year range)

**Example:**
```bash
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target retirement_date \
  --goal maximize_income
```

### Social Security Age (`ss_age`)

Finds the optimal Social Security claiming age.

- **Use cases**: Maximize lifetime benefits, minimize taxes
- **Search range**: 62 to 70 (configurable with `--min-ss-age` and `--max-ss-age`)
- **Algorithm**: Grid search (yearly granularity)
- **Speed**: Fast (~9 evaluations for full range)

**Example:**
```bash
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target ss_age \
  --goal maximize_income \
  --min-ss-age 65 \
  --max-ss-age 70
```

### TSP Balance (`tsp_balance`)

Calculates required TSP balance for specific goals.

- **Status**: Not yet implemented
- **Planned use**: "How much TSP do I need to retire at 60 with $100K/year?"

### All Targets (`all`)

Runs optimization for all targets and compares results.

- **Use cases**: Explore the entire solution space, find trade-offs
- **Output**: Multi-dimensional comparison with recommendations
- **Speed**: Slowest (runs 3 optimizations)

**Example:**
```bash
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target all \
  --goal maximize_income
```

## Optimization Goals

### Match Income (`match_income`)

Find parameters that achieve a specific income target.

- **Required**: `--target-income` flag
- **Best for**: TSP rate optimization
- **Metric**: Minimizes difference between projected and target income
- **Success**: Within $1,000 of target

**Example:**
```bash
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target tsp_rate \
  --goal match_income \
  --target-income 125000
```

### Maximize Income (`maximize_income`)

Find parameters that maximize lifetime net income.

- **Best for**: All targets
- **Metric**: Total net income over projection period
- **Trade-off**: May increase tax burden

**Example:**
```bash
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target retirement_date \
  --goal maximize_income
```

### Maximize Longevity (`maximize_longevity`)

Find parameters that maximize TSP portfolio longevity.

- **Best for**: Risk-averse planning
- **Metric**: Number of years until TSP depletion
- **Trade-off**: May reduce lifetime income

**Example:**
```bash
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target tsp_rate \
  --goal maximize_longevity
```

### Minimize Taxes (`minimize_taxes`)

Find parameters that minimize lifetime tax burden.

- **Best for**: Tax-efficient planning
- **Metric**: Total federal + state + local + FICA taxes
- **Trade-off**: May not maximize income

**Example:**
```bash
./rpgo optimize config.yaml \
  --scenario "Base" \
  --target ss_age \
  --goal minimize_taxes
```

## Command Usage

### Basic Syntax

```bash
./rpgo optimize [config-file] \
  --scenario [scenario-name] \
  --target [target] \
  --goal [goal]
```

### Required Flags

| Flag | Description |
|------|-------------|
| `--scenario` | Name of scenario to optimize (must exist in config) |
| `--target` | What to optimize: `tsp_rate`, `retirement_date`, `ss_age`, or `all` |
| `--goal` | Objective: `match_income`, `maximize_income`, `maximize_longevity`, `minimize_taxes` |

### Optional Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--participant` | Auto-detected | Participant name to optimize |
| `--format` | `table` | Output format: `table` or `json` |
| `--target-income` | None | Target income for `match_income` goal (required for that goal) |
| `--min-rate` | `0.02` | Minimum TSP withdrawal rate (2%) |
| `--max-rate` | `0.10` | Maximum TSP withdrawal rate (10%) |
| `--min-ss-age` | `62` | Minimum Social Security age |
| `--max-ss-age` | `70` | Maximum Social Security age |
| `--debug` | `false` | Enable debug output |

## Constraints

Constraints define the search space for optimization. They ensure realistic and feasible results.

### TSP Rate Constraints

```bash
--min-rate 0.03  # Don't withdraw less than 3%
--max-rate 0.05  # Don't withdraw more than 5%
```

**Common ranges:**
- Conservative: 2-4%
- Moderate: 3-5%
- Aggressive: 4-6%

### Social Security Age Constraints

```bash
--min-ss-age 67  # Don't claim before Full Retirement Age
--max-ss-age 70  # Claim by age 70 (maximum benefit)
```

**Strategic ranges:**
- Early retirement: 62-65
- Balanced: 65-70
- Maximize benefit: 67-70

### Retirement Date Constraints

Currently uses default range: -24 to +36 months from base scenario.

**Future enhancement**: Will support explicit date constraints.

## Examples

### Example 1: Match Current Income

Find the TSP rate needed to match your current $135,000 net income:

```bash
./rpgo optimize my_config.yaml \
  --scenario "Retire 2026" \
  --target tsp_rate \
  --goal match_income \
  --target-income 135000
```

**Output:**
```
BREAK-EVEN OPTIMIZATION RESULTS
================================================================================
Optimization Target: tsp_rate
Optimization Goal:   match_income
Participant:         John Smith
Status:              ✓ Converged
Iterations:          12
Convergence:         Converged to target income within $1000

OPTIMAL PARAMETERS
--------------------------------------------------------------------------------
TSP Withdrawal Rate: 3.47%

PROJECTED RESULTS
--------------------------------------------------------------------------------
First Year Net Income: $135,234.56
Lifetime Income:       $4,123,456.78
TSP Longevity:         30 years
Lifetime Taxes:        $1,234,567.89

TARGET INCOME MATCH
--------------------------------------------------------------------------------
Target Income:    $135,000.00
Achieved Income:  $135,234.56
Difference:       +$234.56
```

### Example 2: Maximize Income by Retirement Timing

Find when to retire to maximize lifetime income:

```bash
./rpgo optimize my_config.yaml \
  --scenario "Retire 2026" \
  --target retirement_date \
  --goal maximize_income
```

**Output:**
```
BREAK-EVEN OPTIMIZATION RESULTS
================================================================================
Optimization Target: retirement_date
Optimization Goal:   maximize_income
Participant:         John Smith
Status:              ✓ Converged
Iterations:          48
Convergence:         Evaluated 48 retirement dates

OPTIMAL PARAMETERS
--------------------------------------------------------------------------------
Retirement Date:     January 1, 2028

PROJECTED RESULTS
--------------------------------------------------------------------------------
First Year Net Income: $142,567.89
Lifetime Income:       $4,567,890.12
TSP Longevity:         30 years
Lifetime Taxes:        $1,345,678.90
```

**Interpretation**: Retiring 2 years later increases lifetime income by $444K.

### Example 3: Minimize Taxes with SS Timing

Find the Social Security claiming age that minimizes taxes:

```bash
./rpgo optimize my_config.yaml \
  --scenario "Retire 2026" \
  --target ss_age \
  --goal minimize_taxes
```

**Output:**
```
BREAK-EVEN OPTIMIZATION RESULTS
================================================================================
Optimization Target: ss_age
Optimization Goal:   minimize_taxes
Participant:         John Smith
Status:              ✓ Converged
Iterations:          9
Convergence:         Evaluated 9 Social Security ages

OPTIMAL PARAMETERS
--------------------------------------------------------------------------------
SS Claiming Age:     70

PROJECTED RESULTS
--------------------------------------------------------------------------------
First Year Net Income: $138,901.23
Lifetime Income:       $4,234,567.89
TSP Longevity:         30 years
Lifetime Taxes:        $1,123,456.78
```

**Interpretation**: Delaying SS to 70 saves $221K in lifetime taxes.

### Example 4: Multi-Dimensional Optimization

Explore all optimization targets to find the best overall strategy:

```bash
./rpgo optimize my_config.yaml \
  --scenario "Retire 2026" \
  --target all \
  --goal maximize_income
```

**Output:**
```
MULTI-DIMENSIONAL OPTIMIZATION RESULTS
================================================================================

SUMMARY OF ALL OPTIMIZATIONS
--------------------------------------------------------------------------------
Optimization         Lifetime Income   TSP Longevity Lifetime Taxes First Year Income
--------------------------------------------------------------------------------
tsp_rate                      $4.12M        28 years       $1.35M         $145.2K
retirement_date               $4.57M        30 years       $1.45M         $142.6K
ss_age                        $4.23M        30 years       $1.23M         $138.9K

BEST SCENARIOS
--------------------------------------------------------------------------------
Best Income:     retirement_date ($4,567,890.12 lifetime)
Best Longevity:  retirement_date (30 years)
Lowest Taxes:    ss_age ($1,123,456.78 lifetime)

RECOMMENDATIONS
--------------------------------------------------------------------------------
• To maximize lifetime income: Optimize retirement_date (retire Jan 2028)
• To maximize TSP longevity (30 years): Optimize retirement_date
• To minimize taxes: Optimize ss_age (saves $1123457)
• ⭐ Optimizing retirement_date provides both high income AND longevity
```

**Interpretation**: Postponing retirement has the biggest impact on lifetime outcomes.

### Example 5: Conservative Strategy

Find a safe TSP withdrawal rate that ensures 30+ year longevity:

```bash
./rpgo optimize my_config.yaml \
  --scenario "Retire 2026" \
  --target tsp_rate \
  --goal maximize_longevity \
  --min-rate 0.02 \
  --max-rate 0.04
```

**Output:**
```
OPTIMAL PARAMETERS
--------------------------------------------------------------------------------
TSP Withdrawal Rate: 2.15%

PROJECTED RESULTS
--------------------------------------------------------------------------------
First Year Net Income: $128,456.78
Lifetime Income:       $3,987,654.32
TSP Longevity:         30 years
Lifetime Taxes:        $1,087,543.21
```

**Interpretation**: A 2.15% withdrawal rate provides maximum portfolio sustainability.

## Understanding Results

### Status Indicators

- **✓ Converged**: Optimization successfully found an optimal solution
- **⚠ Did not converge**: Reached max iterations without convergence (result may still be useful)

### Convergence Information

Explains how the optimizer reached its conclusion:

- **"Converged to target income within $1000"**: Match income goal achieved
- **"Binary search converged"**: TSP rate optimization converged
- **"Evaluated N retirement dates"**: Grid search completed
- **"Max iterations reached"**: Stopped at iteration limit

### Interpreting Trade-offs

Different goals produce different optimal parameters:

| Goal | TSP Rate | Retirement Date | SS Age | Income | Longevity | Taxes |
|------|----------|-----------------|--------|--------|-----------|-------|
| **maximize_income** | Higher | Later | 70 | ↑↑↑ | ↓ | ↑ |
| **maximize_longevity** | Lower | Earlier | 62-67 | ↓ | ↑↑↑ | ↓ |
| **minimize_taxes** | Variable | Earlier | 70 | Variable | ↑ | ↓↓↓ |

**Key insight**: There's no single "best" answer - it depends on your priorities.

## Technical Details

### Solver Algorithms

#### Binary Search (TSP Rate)

- **How it works**: Narrows range by testing midpoint, adjusting bounds based on result
- **Complexity**: O(log n) - very fast
- **Precision**: Converges to within 0.01% (0.0001)
- **Typical iterations**: 10-20

#### Grid Search (Retirement Date, SS Age)

- **How it works**: Evaluates every point in range
- **Complexity**: O(n) - linear in range size
- **Precision**: Exact (monthly for dates, yearly for SS age)
- **Typical iterations**:
  - Retirement date: 48 (4-year range, monthly)
  - SS age: 9 (ages 62-70)

### Transform Integration

The optimizer uses the transform pipeline from Phase 1.1:

```
Base Scenario → Transform → Modified Scenario → Calculate → Evaluate → Update Bounds → Repeat
```

This ensures:
- **Immutability**: Base scenario never modified
- **Composability**: Can combine multiple transforms
- **Validation**: Each transform validated before application

### Performance Characteristics

| Target | Algorithm | Iterations | Time (typical) |
|--------|-----------|------------|----------------|
| `tsp_rate` | Binary search | 10-20 | 5-10 seconds |
| `retirement_date` | Grid search | 48 | 30-60 seconds |
| `ss_age` | Grid search | 9 | 10-15 seconds |
| `all` | All above | 67-77 | 45-85 seconds |

**Factors affecting speed:**
- Projection years in config
- Number of participants
- Complexity of tax calculations
- Historical data loading

### Constraint Validation

All constraints are validated before optimization:

1. **Participant exists** in scenario
2. **Range bounds are valid** (min ≤ max)
3. **Values within legal limits** (e.g., SS age 62-70)
4. **Required parameters present** (e.g., target income for match_income)

Invalid constraints produce clear error messages.

## Common Patterns

### Pattern 1: Income Replacement Analysis

Find multiple ways to replace current income:

```bash
# Option 1: Adjust TSP rate
./rpgo optimize config.yaml --scenario "Base" \
  --target tsp_rate --goal match_income --target-income 140000

# Option 2: Delay retirement
./rpgo optimize config.yaml --scenario "Base" \
  --target retirement_date --goal match_income --target-income 140000

# Option 3: Optimize SS timing
./rpgo optimize config.yaml --scenario "Base" \
  --target ss_age --goal match_income --target-income 140000
```

Compare which approach is most feasible for your situation.

### Pattern 2: Risk vs. Reward

Evaluate conservative vs. aggressive strategies:

```bash
# Conservative: Maximize longevity
./rpgo optimize config.yaml --scenario "Base" \
  --target tsp_rate --goal maximize_longevity

# Aggressive: Maximize income
./rpgo optimize config.yaml --scenario "Base" \
  --target tsp_rate --goal maximize_income
```

### Pattern 3: Tax Optimization

Find the minimum tax strategy across all levers:

```bash
./rpgo optimize config.yaml --scenario "Base" \
  --target all --goal minimize_taxes --format json > tax_optimization.json
```

## Troubleshooting

### "Did not converge"

**Causes:**
- Target income impossible to achieve with constraints
- Search range too narrow
- Scenario configuration issues

**Solutions:**
- Widen constraints (`--min-rate`, `--max-rate`)
- Check scenario for errors
- Try different goal

### "Optimization failed: participant not found"

**Cause:** Participant name doesn't match scenario

**Solution:**
```bash
# List participants in your config, then:
./rpgo optimize config.yaml --scenario "Base" \
  --target tsp_rate --goal maximize_income \
  --participant "Exact Name From Config"
```

### Unexpected Results

**Check:**
1. **Scenario base**: Is the starting scenario realistic?
2. **Constraints**: Are bounds appropriate?
3. **Goal alignment**: Does the goal match your intent?
4. **Time horizon**: Projection years sufficient?

## Next Steps

After optimization:

1. **Review results** across multiple goals
2. **Compare with baseline** using `./rpgo compare`
3. **Adjust config** based on insights
4. **Re-optimize** with refined constraints
5. **Run Monte Carlo** to test robustness

## See Also

- [Compare Command](COMPARE_COMMAND.md) - Compare multiple scenarios
- [Transform Pipeline](../internal/transform/README.md) - How transforms work
- [Break-Even Analysis](../cmd/rpgo/main.go) - Original break-even command
