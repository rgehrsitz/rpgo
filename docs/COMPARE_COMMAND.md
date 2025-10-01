# Compare Command Documentation

The `compare` command enables side-by-side comparison of retirement scenarios using built-in strategy templates. This feature helps you evaluate different retirement timing, Social Security claiming, and TSP withdrawal strategies to find the optimal approach for your situation.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Available Templates](#available-templates)
- [Command Usage](#command-usage)
- [Output Formats](#output-formats)
- [Examples](#examples)
- [Understanding Results](#understanding-results)
- [Technical Details](#technical-details)

## Overview

The compare command allows you to:

1. **Apply strategy templates** to a base scenario automatically
2. **Compare multiple strategies** simultaneously
3. **Get recommendations** based on key metrics
4. **Export results** in multiple formats (table, CSV, JSON)

Key metrics compared:

- **Lifetime Income**: Total net income over the projection period
- **TSP Longevity**: How long your TSP balance lasts
- **Final TSP Balance**: Remaining assets at end of projection
- **Tax Impact**: Lifetime tax burden differences

## Quick Start

```bash
# List all available templates
./rpgo compare --list-templates

# Basic comparison
./rpgo compare config.yaml \
  --base "Base Scenario" \
  --with postpone_1yr,delay_ss_70

# Compare popular strategies
./rpgo compare config.yaml \
  --base "Base Scenario" \
  --with conservative,aggressive
```

## Available Templates

### Retirement Timing

| Template | Description |
|----------|-------------|
| `postpone_1yr` | Postpone retirement by 1 year (12 months) |
| `postpone_2yr` | Postpone retirement by 2 years (24 months) |
| `postpone_3yr` | Postpone retirement by 3 years (36 months) |

### Social Security Strategies

| Template | Description |
|----------|-------------|
| `delay_ss_67` | Delay Social Security to age 67 (Full Retirement Age) |
| `delay_ss_70` | Delay Social Security to age 70 (Maximum benefit - 8% increase per year) |

### TSP Withdrawal Strategies

| Template | Description |
|----------|-------------|
| `tsp_need_based` | Switch to need-based withdrawals (withdraw only what's needed) |
| `tsp_fixed_2pct` | Use 2% fixed withdrawal rate |
| `tsp_fixed_3pct` | Use 3% fixed withdrawal rate |
| `tsp_fixed_4pct` | Use 4% fixed withdrawal rate (traditional safe withdrawal rate) |

### Combination Strategies

| Template | Description |
|----------|-------------|
| `conservative` | Postpone 2 years + Delay SS to 70 + 3% TSP withdrawal |
| `aggressive` | Delay SS to 70 + 4% TSP withdrawal |
| `postpone_1yr_delay_ss_70` | Postpone retirement 1 year + Delay SS to 70 |
| `postpone_2yr_delay_ss_70` | Postpone retirement 2 years + Delay SS to 70 |
| `delay_ss_70_tsp_4pct` | Delay SS to 70 + 4% TSP withdrawal |

## Command Usage

### Basic Syntax

```bash
./rpgo compare [config-file] --base [scenario-name] --with [templates]
```

### Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--base` | Yes | Name of the base scenario to compare against |
| `--with` | Yes | Comma-separated list of template names |
| `--format` | No | Output format: `table` (default), `csv`, or `json` |
| `--participant` | No | Participant name for template application (auto-detected if not specified) |
| `--list-templates` | No | Show all available templates and exit |
| `--debug` | No | Enable debug output for detailed calculations |
| `--regulatory-config` | No | Path to regulatory config file (default: regulatory.yaml) |

### Examples

#### Example 1: Compare Retirement Timing

Compare retiring on schedule vs. postponing 1 or 2 years:

```bash
./rpgo compare my_config.yaml \
  --base "Retire 2026" \
  --with postpone_1yr,postpone_2yr
```

**Output:**

```text
RETIREMENT SCENARIO COMPARISON
================================================================================
Base Scenario: Retire 2026
Configuration: my_config.yaml

Scenario                  1st Year Income Lifetime Income   TSP Longevity       Final TSP
--------------------------------------------------------------------------------
Retire 2026 (base)                $141.5K          $3.92M        30 years          $3.73M
--------------------------------------------------------------------------------
Retire 2026_postpone_1yr          $141.5K          $4.08M        30 years          $3.92M
Retire 2026_postpone_2yr          $141.5K          $4.23M        30 years          $4.11M
================================================================================

COMPARISON TO BASE
--------------------------------------------------------------------------------

Retire 2026_postpone_1yr:
  Lifetime Income:  +$153.1K (3.9%)
  Tax Impact:       $125.6K

Retire 2026_postpone_2yr:
  Lifetime Income:  +$306.2K (7.8%)
  Tax Impact:       $251.2K


RECOMMENDATIONS
--------------------------------------------------------------------------------
• Best Income: Retire 2026_postpone_2yr provides $306200 more lifetime income than base scenario
```

#### Example 2: Social Security Optimization

Compare different Social Security claiming ages:

```bash
./rpgo compare my_config.yaml \
  --base "Retire 2026" \
  --with delay_ss_67,delay_ss_70
```

**Why this matters:**

- Claiming at 62: Reduced benefit (70-75% of FRA)
- Claiming at 67 (FRA): Full benefit
- Claiming at 70: Maximum benefit (124% of FRA, 8% increase per year of delay)

#### Example 3: Conservative vs. Aggressive

Compare two popular overall strategies:

```bash
./rpgo compare my_config.yaml \
  --base "Retire 2026" \
  --with conservative,aggressive \
  --format csv > comparison.csv
```

**Conservative Strategy:**

- Postpone retirement 2 years (more savings, higher pension)
- Delay Social Security to 70 (maximum benefit)
- Use 3% TSP withdrawal (lower risk)

**Aggressive Strategy:**

- Retire on schedule
- Delay Social Security to 70 (maximize benefit)
- Use 4% TSP withdrawal (traditional safe rate)

#### Example 4: Export to CSV for Analysis

```bash
./rpgo compare my_config.yaml \
  --base "Retire 2026" \
  --with postpone_1yr,postpone_2yr,delay_ss_70,conservative,aggressive \
  --format csv > retirement_comparison.csv
```

Open in Excel/Google Sheets for custom analysis and charting.

#### Example 5: Export to JSON for Automation

```bash
./rpgo compare my_config.yaml \
  --base "Retire 2026" \
  --with conservative,aggressive \
  --format json > comparison.json
```

Use JSON output for:

- Custom reporting tools
- Dashboard integration
- Automated decision systems
- Data pipelines

## Output Formats

### Table Format (Default)

Human-readable console table with:

- Summary table of all scenarios
- Detailed comparison to base
- Recommendations

Best for: Interactive terminal use, quick reviews

### CSV Format

Comma-separated values with headers:

- One row per scenario
- All metrics in separate columns
- Easy to import into spreadsheets

Best for: Excel analysis, custom charting, data processing

### JSON Format

Structured JSON with complete data:

- Full scenario summaries
- All metrics and comparisons
- Nested projection data

Best for: Programmatic access, automation, custom tools

## Understanding Results

### Key Metrics Explained

#### Lifetime Income

- **Definition**: Total net income over the entire projection period
- **Higher is better**: More money in your pocket over your retirement
- **Factors**: Salary years, pension amounts, Social Security benefits, TSP withdrawals, taxes

#### TSP Longevity

- **Definition**: Number of years until TSP balance is depleted
- **Higher is better**: Longer portfolio sustainability
- **"30 years"**: Balance lasts through entire projection
- **Factors**: Withdrawal rate, TSP returns, balance growth

#### Final TSP Balance

- **Definition**: Remaining TSP assets at end of projection
- **Higher is better**: More legacy wealth, buffer for uncertainty
- **Consider**: Higher balances may indicate under-spending

#### Tax Impact

- **Definition**: Difference in lifetime taxes paid vs. base scenario
- **Lower is better**: Keep more of your money
- **Positive value**: Pay more taxes than base
- **Negative value**: Tax savings vs. base

### Recommendations

The compare command automatically generates recommendations for:

1. **Best Income**: Scenario with highest lifetime income
2. **Best Longevity**: Scenario with longest TSP sustainability
3. **Lowest Taxes**: Scenario with smallest tax burden

**Important**: The "best" scenario depends on your priorities:

- **Income maximization**: Choose highest lifetime income
- **Risk reduction**: Choose longest TSP longevity
- **Tax efficiency**: Choose lowest tax burden
- **Balanced approach**: Look for scenarios strong across multiple metrics

## Technical Details

### How Templates Work

Templates use the transform pipeline architecture to modify scenarios:

1. **Base scenario** is loaded from your config file
2. **Transforms** are applied (e.g., postpone retirement, delay SS)
3. **Modified scenario** is calculated with full projection
4. **Metrics** are computed and compared to base

Templates are **composable**: Each template applies one or more transforms in sequence.

### Transform Immutability

All transforms create **new scenarios** - your base scenario is never modified. This ensures:

- Reproducible comparisons
- No side effects
- Parallel processing capability

### Participant Specification

When applying templates, you must specify which participant to modify:

- **Auto-detection**: Uses first participant if not specified
- **Explicit**: Use `--participant "Name"` for multi-participant households
- **Template scope**: Each template modifies one participant's parameters

### Performance

- Each scenario requires a full projection calculation
- Typical comparison (1 base + 3 alternatives): 2-5 seconds
- Large comparisons (10+ templates): 10-20 seconds
- Historical data mode: Slower due to data loading

## Common Patterns

### Pattern 1: Sensitivity Analysis

Test how sensitive your plan is to one variable:

```bash
# How much does retirement timing matter?
./rpgo compare config.yaml --base "Base" --with postpone_1yr,postpone_2yr,postpone_3yr

# How much does SS timing matter?
./rpgo compare config.yaml --base "Base" --with delay_ss_67,delay_ss_70

# How much does TSP rate matter?
./rpgo compare config.yaml --base "Base" --with tsp_fixed_2pct,tsp_fixed_3pct,tsp_fixed_4pct
```

### Pattern 2: Strategy Screening

Quickly screen popular strategies:

```bash
./rpgo compare config.yaml --base "Base" --with conservative,aggressive
```

Then drill down into the winner with more detailed analysis.

### Pattern 3: Build Custom Strategies

After understanding individual levers, create custom combinations in your config file based on insights from template comparisons.

## Troubleshooting

### "Template not found"

Check available templates:

```bash
./rpgo compare --list-templates
```

Template names are case-sensitive and must match exactly.

### "Base scenario not found"

Verify scenario name in your config matches exactly (including spaces):

```bash
grep "name:" config.yaml
```

### "Participant not found"

Specify participant explicitly:

```bash
./rpgo compare config.yaml --base "Base" --with postpone_1yr --participant "John Smith"
```

### Comparison takes too long

- Reduce projection years in config
- Use fewer templates per comparison
- Disable historical data mode if not needed

## Next Steps

After using the compare command:

1. **Review results** across multiple metrics, not just one
2. **Consider trade-offs** (e.g., higher income but more taxes)
3. **Test assumptions** by modifying your config file
4. **Run break-even analysis** for detailed TSP rate optimization
5. **Document decisions** by exporting CSV/JSON results

See also:

- [Transform Pipeline Documentation](../internal/transform/README.md)
- [Template System](../internal/transform/templates.go)
- [Break-Even Analysis](BREAK_EVEN.md)
