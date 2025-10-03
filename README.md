# FERS Retirement Planning Calculator

A comprehensive retirement planning calculator for Federal Employees Retirement System (FERS) participants, built in Go. This tool provides accurate "apples-to-apples" comparisons between current net income and multiple retirement scenarios, incorporating all recent regulatory changes, tax implications, and federal-specific benefits.

**Status:** Phase 3.1 Complete - Advanced Features In Progress (December 2024)  
**Progress:** 75% Complete

## 🚀 **Key Features**

### **Core Retirement Planning**
- **Comprehensive FERS Calculations**: Accurate pension calculations with proper multiplier rules (1.0% standard, 1.1% enhanced for age 62+ with 20+ years)
- **TSP Modeling**: Support for both Traditional and Roth TSP accounts with multiple withdrawal strategies and manual allocations
- **Social Security Integration**: Full benefit calculations with 2025 WEP/GPO repeal implementation (2025 wage base: $176,100)
- **Tax Modeling**: Complete federal, state (Pennsylvania), and local tax calculations
- **Long-term Projections**: 25+ year projections with COLA adjustments

### **Advanced Analysis** ✅ COMPLETE
- **IRMAA Analysis**: Medicare premium surcharge calculations with breach detection and optimization recommendations
- **Tax-Smart Withdrawal Sequencing**: Optimization of withdrawal order for tax efficiency with bracket-fill strategies
- **Roth Conversion Planner**: Multi-year optimization with multiple objectives (minimize taxes, IRMAA, maximize estate)
- **Healthcare Cost Modeling**: Pre-65 coverage, Medicare Part B/D, Medigap with age-based transitions
- **Survivor Viability Analysis**: Financial impact modeling when spouse dies with life insurance needs calculation

### **User Experience**
- **Interactive TUI**: Terminal user interface with real-time parameter adjustment and visualization
- **Multiple Output Formats**: Console, HTML, JSON, and CSV reports
- **Scenario Comparison**: Compare retirement strategies side-by-side using built-in templates
- **Automated Optimization**: Multi-dimensional break-even solver finds optimal retirement parameters

## 🚀 **Quick Start Commands**

```bash
# Run retirement projections
./rpgo calculate config.yaml --format console

# Compare multiple scenarios
./rpgo compare config.yaml --scenarios "Scenario 1,Scenario 2"

# Find optimal retirement parameters
./rpgo optimize config.yaml --objective minimize_lifetime_tax

# Plan Roth conversions
./rpgo plan-roth config.yaml --participant "Alice Johnson" --window 2028-2032

# Analyze survivor viability
./rpgo analyze-survivor config.yaml --deceased "Alice Johnson" --survivor-spending-factor 0.75

# Launch interactive TUI
./rpgo-tui config.yaml
```

## 📚 **Documentation**

- **Quick Reference**: [docs/QUICK_REFERENCE.md](docs/QUICK_REFERENCE.md)
- **Project Status**: [docs/PROJECT_STATUS.md](docs/PROJECT_STATUS.md)
- **Implementation Plan**: [docs/IMPLEMENTATION_PLAN.md](docs/IMPLEMENTATION_PLAN.md)
- **CLI Reference**: [docs/cli_reference.md](docs/cli_reference.md)

- **Social Security Fairness Act (2025)**: WEP/GPO repeal implementation
- **SECURE 2.0 Act**: Updated RMD ages (73 for 1951-1959, 75 for 1960+)
- **2025 Tax Brackets**: Current federal tax calculations
- **Pennsylvania Tax Rules**: State-specific retirement income exemptions

## Installation

### Prerequisites

- Go 1.21 or later
- Git

### Build from Source

```bash
git clone https://github.com/rgehrsitz/rpgo.git
cd rpgo
go mod tidy
go build -o rpgo ./cmd/rpgo
go build -o rpgo-tui ./cmd/rpgo-tui  # Optional: Build TUI interface
```

## Usage

### Quick Start

1. **Copy the sample configuration**

  ```bash
  cp example_config.yaml my_config.yaml
  ```

  Then edit `my_config.yaml` to match your household.

1. **Run calculations**

  ```bash
  ./rpgo calculate my_config.yaml
  ```

1. **Generate an HTML report**

  ```bash
  ./rpgo calculate my_config.yaml --format html > report.html
  ```

1. **Run portfolio Monte Carlo simulations**

  ```bash
  ./rpgo historical monte-carlo ./data --simulations 1000 --balance 1000000 --withdrawal 40000
  ```

### Core CLI commands

- `./rpgo calculate [input-file]` — deterministic retirement projection using a YAML configuration.
- `./rpgo compare [input-file]` — compare retirement strategies using built-in templates (see [Compare Command docs](docs/COMPARE_COMMAND.md)).
- `./rpgo optimize [input-file]` — find optimal retirement parameters using break-even solver (see [Optimize Command docs](docs/OPTIMIZE_COMMAND.md)).
- `./rpgo validate [input-file]` — schema and rules validation without running a projection.
- `./rpgo break-even [input-file]` — computes TSP withdrawal rates needed to match current net income.
- `./rpgo historical load [data-path]` — load and summarize historical datasets.
- `./rpgo historical stats [data-path]` — print descriptive statistics for historical datasets.
- `./rpgo historical query [data-path] [year] [fund]` — fetch a single data point (fund return, inflation, or COLA).
- `./rpgo historical monte-carlo [data-path] [flags]` — run portfolio-only Monte Carlo simulations using flag-driven parameters.

### Interactive TUI (Terminal User Interface)

**Status**: Foundation complete, components in development

```bash
./rpgo-tui my_config.yaml
```

The TUI provides an interactive interface for retirement planning with:

- **Real-time Parameter Adjustment**: Modify retirement dates, TSP rates, and SS claiming ages with immediate recalculation
- **Visual Dashboards**: Overview of key metrics with trend indicators
- **Scenario Browsing**: Navigate and compare multiple scenarios interactively
- **Optimization Interface**: Run break-even solver with live progress updates
- **ASCII Charts**: Visual representation of projections over time
- **Keyboard-First Design**: Efficient navigation without mouse (h=home, s=scenarios, p=parameters, c=compare, o=optimize, r=results, ?=help)

See [TUI Design Documentation](docs/TUI_DESIGN.md) for detailed architecture and features.

#### Compare Command Examples

```bash
# List all available strategy templates
./rpgo compare --list-templates

# Compare retirement timing options
./rpgo compare my_config.yaml --base "Base Scenario" --with postpone_1yr,postpone_2yr

# Compare Social Security claiming strategies
./rpgo compare my_config.yaml --base "Base Scenario" --with delay_ss_67,delay_ss_70

# Compare comprehensive strategies
./rpgo compare my_config.yaml --base "Base Scenario" --with conservative,aggressive

# Export comparison to CSV
./rpgo compare my_config.yaml --base "Base Scenario" --with conservative,aggressive --format csv > comparison.csv
```

See [Compare Command Documentation](docs/COMPARE_COMMAND.md) for detailed usage and examples.

#### Optimize Command Examples

```bash
# Find TSP withdrawal rate to match target income
./rpgo optimize my_config.yaml --scenario "Base Scenario" --target tsp_rate --goal match_income --target-income 120000

# Find optimal retirement date to maximize lifetime income
./rpgo optimize my_config.yaml --scenario "Base Scenario" --target retirement_date --goal maximize_income

# Find optimal Social Security claiming age to minimize taxes
./rpgo optimize my_config.yaml --scenario "Base Scenario" --target ss_age --goal minimize_taxes

# Run all optimizations and compare
./rpgo optimize my_config.yaml --scenario "Base Scenario" --target all --goal maximize_income
```

See [Optimize Command Documentation](docs/OPTIMIZE_COMMAND.md) for detailed usage and examples.

### Logging and Debug Mode

- Use `--debug` on CLI commands (calculate, break-even, monte-carlo) to enable detailed debug logs.
- Debug logs are generated via an internal Logger interface; the CLI wires a simple logger that prints level-prefixed lines (DEBUG/INFO/WARN/ERROR).
- When `--debug` is off, a no-op logger is used to keep output clean.

#### First-year behavior

- TSP withdrawals in the first retirement year are prorated based on how many months a participant is actually retired.
- Employee TSP contributions, FICA, and local Earned Income Tax continue for the portion of the year where wage income is present and stop once the participant is fully retired.
- Social Security benefits begin in the year a participant reaches the configured start age, with partial-year payments when eligibility occurs mid-year.

#### Output formats and aliases

Supported `--format` values:

- `console`, `console-lite`, `csv`, `detailed-csv`, `html`, `json`, `all`

Aliases map to canonical names:

- `console-verbose` → `console`; `verbose` → `console`
- `csv-detailed` → `detailed-csv`; `csv-summary` → `csv`
- `html-report` → `html`; `json-pretty` → `json`

Reports are output to stdout by default. Redirect to files as needed (e.g., `> report.html`).

### Output Formats

- `console`: Formatted text output (default)
- `html`: Interactive HTML report with charts and visualizations
- `json`: Structured JSON data
- `csv`: Comma-separated values for spreadsheet analysis

## Configuration File Format

The calculator supports two configuration formats:

### New Generic Format (Recommended)

The new participant-based format supports flexible household compositions including single federal employees, dual federal employees, or mixed households:

```yaml
household:
  filing_status: "married_filing_jointly"  # or "single"
  participants:
    - name: "John Smith"
      is_federal: true
      birth_date: "1965-03-15T00:00:00Z"
      hire_date: "1987-06-01T00:00:00Z"
      current_salary: 145000
      high_3_salary: 142000
      tsp_balance_traditional: 850000
      tsp_balance_roth: 175000
      tsp_contribution_percent: 0.15
      is_primary_fehb_holder: true
      fehb_premium_per_pay_period: 745
      ss_benefit_fra: 3200
      ss_benefit_62: 2240
      ss_benefit_70: 3968
      survivor_benefit_election_percent: 0.0
      
    - name: "Jane Smith"
      is_federal: false  # Non-federal employee
      birth_date: "1968-09-22T00:00:00Z"
      ss_benefit_fra: 2850
      ss_benefit_62: 1995  
      ss_benefit_70: 3534
      external_pension:
        monthly_benefit: 1500
        start_age: 65
        cola_adjustment: 0.02
        survivor_benefit: 0.5

global_assumptions:
  # ... same as legacy format

scenarios:
  - name: "Early Retirement Scenario"
    participant_scenarios:
      "John Smith":
        participant_name: "John Smith"
        retirement_date: "2026-06-01T00:00:00Z"
        ss_start_age: 62
        tsp_withdrawal_strategy: "4_percent_rule"
      "Jane Smith":
        participant_name: "Jane Smith"
        ss_start_age: 65
```

### Legacy Format (Still Supported)

The legacy format uses fixed "robert" and "dawn" keys for backwards compatibility:

```yaml
personal_details:
  robert:
    name: "Robert"
    birth_date: "1963-06-15"
    hire_date: "1985-03-20"
    current_salary: 95000
    high_3_salary: 93000
    tsp_balance_traditional: 450000
    tsp_balance_roth: 50000
    tsp_contribution_percent: 0.15
    ss_benefit_fra: 2400
    ss_benefit_62: 1680
    ss_benefit_70: 2976
    fehb_premium_monthly: 875
    survivor_benefit_election_percent: 0.0
    
    # TSP allocation (required for Monte Carlo variability)
    tsp_allocation:
      c_fund: "0.60"  # 60% C Fund (Large Cap Stock Index)
      s_fund: "0.20"  # 20% S Fund (Small Cap Stock Index)
      i_fund: "0.10"  # 10% I Fund (International Stock Index)
      f_fund: "0.10"  # 10% F Fund (Fixed Income Index)
      g_fund: "0.00"  # 0% G Fund (Government Securities)

  dawn:
    name: "Dawn"
    birth_date: "1965-08-22"
    hire_date: "1988-07-10"
    current_salary: 87000
    high_3_salary: 85000
    tsp_balance_traditional: 380000
    tsp_balance_roth: 45000
    tsp_contribution_percent: 0.12
    ss_benefit_fra: 2200
    ss_benefit_62: 1540
    ss_benefit_70: 2728
    fehb_premium_monthly: 0.0
    survivor_benefit_election_percent: 0.0
    
    # TSP allocation (required for Monte Carlo variability)
    tsp_allocation:
      c_fund: "0.40"  # 40% C Fund (Large Cap Stock Index)
      s_fund: "0.10"  # 10% S Fund (Small Cap Stock Index)
      i_fund: "0.10"  # 10% I Fund (International Stock Index)
      f_fund: "0.30"  # 30% F Fund (Fixed Income Index)
      g_fund: "0.10"  # 10% G Fund (Government Securities)

global_assumptions:
  inflation_rate: 0.025
  fehb_premium_inflation: 0.065
  tsp_return_pre_retirement: 0.055
  tsp_return_post_retirement: 0.045
  cola_general_rate: 0.025
  projection_years: 25
  current_location:
    state: "Pennsylvania"
    county: "Bucks"
    municipality: "Upper Makefield Township"

scenarios:
  - name: "Early Retirement 2025"
    robert:
      employee_name: "robert"
      retirement_date: "2025-12-31"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"
    dawn:
      employee_name: "dawn"
      retirement_date: "2025-12-31"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"

  - name: "Delayed Retirement 2028"
    robert:
      employee_name: "robert"
      retirement_date: "2028-12-31"
      ss_start_age: 67
      tsp_withdrawal_strategy: "need_based"
      tsp_withdrawal_target_monthly: 3000
    dawn:
      employee_name: "dawn"
      retirement_date: "2028-12-31"
      ss_start_age: 62
      tsp_withdrawal_strategy: "4_percent_rule"
```

## Calculation Details

### FERS Pension

- **Formula**: High-3 Salary × Years of Service × Multiplier
- **Multipliers**:
  - Standard: 1.0% per year of service
  - Enhanced: 1.1% per year if retiring at age 62+ with 20+ years service
- **COLA Rules**:
  - No COLA until age 62
  - CPI ≤ 2%: Full CPI increase
  - CPI 2-3%: Capped at 2%
  - CPI > 3%: CPI minus 1%

### TSP Configuration

#### TSP Allocations vs Lifecycle Funds

For **deterministic calculations**, both approaches work equivalently:

- **Manual TSP Allocation**: Specify exact percentages for each fund (C, S, I, F, G)
- **TSP Lifecycle Fund**: Use predefined lifecycle funds (L2030, L2035, L2040, L Income)

For **Monte Carlo simulations**, use **manual TSP allocations** for proper market variability:

```yaml
# ✅ Recommended for Monte Carlo
tsp_allocation:
  c_fund: "0.60"  # 60% C Fund
  s_fund: "0.20"  # 20% S Fund
  i_fund: "0.10"  # 10% I Fund
  f_fund: "0.10"  # 10% F Fund
  g_fund: "0.00"  # 0% G Fund

# ❌ Avoid for Monte Carlo (produces identical results)
# tsp_lifecycle_fund:
#   fund_name: "L2030"
```

#### TSP Fund Types

- **C Fund**: S&P 500 Index (Large Cap Stock)
- **S Fund**: Small Cap Stock Index (Russell 2000)
- **I Fund**: International Stock Index (MSCI World ex-US)
- **F Fund**: Fixed Income Index (Bloomberg US Aggregate)
- **G Fund**: Government Securities (Guaranteed return)

### TSP Withdrawal Strategies

- **4% Rule**: Initial 4% withdrawal, adjusted for inflation annually
- **Need-Based**: Withdraw based on target monthly income
- **RMD Compliance**: Automatic Required Minimum Distribution calculations
- **Traditional vs Roth**: Optimized withdrawal order (Roth first, then Traditional)

### Monte Carlo Analysis

The current CLI ships with a portfolio-only Monte Carlo simulator under the `historical` command group. It models withdrawal strategies against TSP fund return histories (or statistical distributions) without processing a full FERS configuration.

```bash
./rpgo historical monte-carlo ./data \
  --simulations 1000 \
  --balance 1000000 \
  --withdrawal 40000 \
  --strategy fixed_amount
```

Key options:

- `--balance`, `--withdrawal` – starting balance and annual draw.
- `--strategy` – choose among `fixed_amount`, `fixed_percentage`, `inflation_adjusted`, or `guardrails`.
- `--historical` – toggle between true historical draws (default) and statistical mode.
- `--years` – projection length.
- `--simulations` – number of trials.

> ℹ️ Config-driven “comprehensive FERS” Monte Carlo was removed from the CLI. Use deterministic `calculate` runs to compare scenario files, and rely on the portfolio simulator for withdrawal stress-testing.

### Social Security

- **2025 WEP/GPO Repeal**: No benefit reductions for federal employees
- **Claiming Ages**: 62-70 with proper benefit adjustments
- **Taxation**: Up to 85% taxable based on provisional income

### Tax Calculations

- **Federal**: 2025 tax brackets with standard deductions
- **Pennsylvania**: 3.07% flat rate, retirement income exempt
- **Local**: Earned Income Tax (EIT) only on wages
- **FICA**: Social Security and Medicare taxes on earned income only

## Project Structure

```text
rpgo/
├── cmd/rpgo/               # Command line interface
├── data/                   # Historical financial data
│   ├── tsp-returns/        # TSP fund historical returns
│   ├── inflation/          # CPI-U inflation rates
│   └── cola/               # Social Security COLA rates
├── internal/
│   ├── domain/             # Core domain models
│   ├── calculation/        # Calculation engines
│   ├── config/             # Configuration parsing
│   └── output/             # Report generation
├── pkg/
│   ├── decimal/            # Financial precision utilities
│   └── dateutil/           # Date calculation utilities
├── test/                   # Test files and data
└── docs/                   # Documentation
```

## Testing

Run the full suite:

```bash
go test ./...
```

Run specific packages:

```bash
go test ./internal/calculation
go test ./internal/config
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass
6. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Disclaimer

This calculator is for educational and planning purposes only. It should not be considered as financial advice. Please consult with qualified financial professionals for personalized retirement planning advice. The calculations are based on current regulations and may change over time.

## Support

For issues, questions, or contributions, please use the GitHub issue tracker or contact the maintainers.

## Roadmap

- [x] Monte Carlo simulation for TSP returns
- [x] Historical data integration
- [x] Interactive HTML reports with charts and visualizations
- [ ] TSP lifecycle fund support for Monte Carlo simulations
- [ ] Enhanced withdrawal strategies (floor-ceiling, bond tent)
- [ ] Web interface
- [ ] Additional state tax support
- [ ] Medicare Part B premium calculations
- [ ] Survivor benefit optimization
- [ ] Export to financial planning software
