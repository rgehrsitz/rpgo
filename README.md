# FERS Retirement Planning Calculator

A comprehensive retirement planning calculator for Federal Employees Retirement System (FERS) participants, built in Go. This tool provides accurate "apples-to-apples" comparisons between current net income and multiple retirement scenarios, incorporating all recent regulatory changes, tax implications, and federal-specific benefits.

## Features

- **Comprehensive FERS Calculations**: Accurate pension calculations with proper multiplier rules (1.0% standard, 1.1% enhanced for age 62+ with 20+ years)
- **TSP Modeling**: Support for both Traditional and Roth TSP accounts with multiple withdrawal strategies and manual allocations
- **TSP Lifecycle Fund Limitation**: For Monte Carlo simulations, manual TSP allocations are recommended over lifecycle funds for proper market variability
- **Social Security Integration**: Full benefit calculations with 2025 WEP/GPO repeal implementation (2025 wage base: $176,100)
- **Tax Modeling**: Complete federal, state (Pennsylvania), and local tax calculations
- **Multiple Scenarios**: Compare multiple retirement scenarios simultaneously
- **Long-term Projections**: 25+ year projections with COLA adjustments
- **Multiple Output Formats**: Console, HTML, JSON, and CSV reports
- **Monte Carlo Simulations**: Probabilistic analysis using historical market data for portfolio sustainability
- **Historical Data Integration**: Real TSP fund returns, inflation, and COLA data from 1990-2023

## Recent Regulatory Compliance

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
go build -o fers-calc cmd/cli/main.go
```

## Usage

### Quick Start

1. **Generate an example configuration**:
   ```bash
   ./fers-calc example config.yaml
   ```

2. **Run calculations**:
   ```bash
   ./fers-calc calculate config.yaml
   ```

3. **Generate HTML report**:
   ```bash
   ./fers-calc calculate config.yaml --format html > report.html
   ```

4. **Run Monte Carlo simulations**:
   ```bash
   # Simple portfolio Monte Carlo (uses flags)
   ./fers-calc historical monte-carlo ./data --simulations 1000 --balance 1000000 --withdrawal 40000
   
   # Comprehensive FERS Monte Carlo (uses config file)
   ./fers-calc monte-carlo config.yaml ./data --simulations 1000
   ```

   **Important**: For Monte Carlo simulations, ensure your configuration uses manual `tsp_allocation` settings rather than `tsp_lifecycle_fund` settings to get proper market variability in results.

### Command Line Options

To help you understand the core commands, here's a breakdown:

- **`./fers-calc calculate [input-file]`**: Runs a single, deterministic retirement projection based on the fixed assumptions in your input configuration file. This provides a detailed report for one specific set of conditions.

- **`./fers-calc historical monte-carlo [data-path] [flags]`**: Executes simple portfolio Monte Carlo simulations. This command is focused on assessing the sustainability of a specific investment balance with a defined withdrawal strategy, using historical (or statistical) market data. It does not model the full FERS retirement system.

- **`./fers-calc monte-carlo [config-file] [data-path]`**: Runs comprehensive FERS Monte Carlo simulations. This command integrates your full retirement configuration (pension, Social Security, TSP, taxes, FEHB) with historical market data to run thousands of scenarios, providing a probabilistic assessment of your complete retirement plan's success.

```bash
# Basic calculation
./fers-calc calculate [input-file]

# With options
./fers-calc calculate [input-file] --format html --verbose --debug > report.html

# Validate configuration
./fers-calc validate [input-file]

# Generate example
./fers-calc example [output-file]

# Break-even analysis
./fers-calc break-even [input-file]

# Historical data management
./fers-calc historical load ./data
./fers-calc historical stats ./data
./fers-calc historical query ./data 2020 C

# Simple Portfolio Monte Carlo simulations
./fers-calc historical monte-carlo ./data --simulations 1000 --balance 1000000 --withdrawal 40000
./fers-calc historical monte-carlo ./data --strategy guardrails --years 30

# Comprehensive FERS Monte Carlo simulations
./fers-calc monte-carlo config.yaml ./data --simulations 1000
./fers-calc monte-carlo config.yaml ./data --simulations 5000 --seed 12345 --debug
```

#### Logging and Debug Mode

- Use `--debug` on CLI commands (calculate, break-even, monte-carlo) to enable detailed debug logs.
- Debug logs are generated via an internal Logger interface; the CLI wires a simple logger that prints level-prefixed lines (DEBUG/INFO/WARN/ERROR).
- When `--debug` is off, a no-op logger is used to keep output clean.

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

The calculator uses YAML configuration files. Here's an example structure:

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

#### Simple Portfolio Monte Carlo
- **Historical Data**: Real TSP fund returns, inflation, and COLA data (1990-2023)
- **Withdrawal Strategies**: Fixed amount, percentage, inflation-adjusted, guardrails
- **Risk Assessment**: Success rates, percentile analysis, drawdown tracking
- **Asset Allocation**: Customizable TSP fund allocations (C, S, I, F, G funds)
- **Parallel Processing**: Efficient simulation execution for 1000+ scenarios

#### Comprehensive FERS Monte Carlo
- **Full FERS Integration**: Models all retirement components (pension, SS, TSP, taxes, FEHB)
- **Market Variability**: Historical or statistical market condition generation
- **Income Sustainability**: Success rates based on complete retirement income
- **TSP Longevity**: Tracks when TSP balances deplete
- **Tax Implications**: Includes all federal, state, and local taxes
- **Healthcare Costs**: Models FEHB premium increases over time

#### Monte Carlo Examples

**Conservative 4% Rule (25 years)**
```bash
./fers-calc historical monte-carlo ./data \
  --simulations 1000 \
  --balance 1000000 \
  --withdrawal 40000 \
  --strategy fixed_amount
```
*Result: 99% success rate, median ending balance $6.6M*

**Comprehensive FERS Analysis**
```bash
./fers-calc monte-carlo config.yaml ./data \
  --simulations 1000
```
*Result: 100% success rate, median net income $234,681, low risk assessment*

**High-Precision FERS Analysis**
```bash
./fers-calc monte-carlo config.yaml ./data \
  --simulations 5000 \
  --seed 12345
```
*Result: Reproducible results with comprehensive risk analysis*

**Aggressive 6% Rule with Guardrails (Simple Portfolio)**
```bash
./fers-calc historical monte-carlo ./data \
  --simulations 1000 \
  --balance 500000 \
  --withdrawal 30000 \
  --strategy guardrails \
  --years 30
```
*Result: 82% success rate, high risk assessment*

**Inflation-Adjusted Strategy (Simple Portfolio)**
```bash
./fers-calc historical monte-carlo ./data \
  --simulations 500 \
  --balance 750000 \
  --withdrawal 35000 \
  --strategy inflation_adjusted \
  --years 30
```
*Result: 84% success rate, moderate risk assessment*

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

```
rpgo/
├── cmd/cli/                 # Command line interface
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

Run the test suite:

```bash
go test ./...
```

Run specific test packages:

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
