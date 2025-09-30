# CLI Reference Guide

## Overview

The FERS Retirement Calculator CLI (`rpgo`) provides comprehensive retirement planning tools for federal employees, including deterministic calculations, Monte Carlo risk analysis, and historical data management.

## Main Commands

### `calculate [input-file]` — Run retirement scenarios

Calculate retirement scenarios using a YAML configuration file.

**Flags:**

- `--format, -f`: Output format (default: "console")
- `--verbose, -v`: Enable verbose output
- `--debug`: Enable debug output for detailed calculations
- `--regulatory-config`: Path to regulatory config file (default: regulatory.yaml if it exists)

**Supported output formats:**

- `console` (default): Verbose console summary
- `console-lite`: Compact console summary
- `csv`: Summary CSV format
- `detailed-csv`: Full year-by-year CSV
- `html`: Interactive HTML report with charts
- `json`: Structured JSON output
- `all`: Writes multiple outputs with timestamps

**Examples:**

```bash
# Basic calculation with console output
./rpgo calculate config.yaml

# Generate HTML report
./rpgo calculate config.yaml --format html > report.html

# Verbose console output
./rpgo calculate config.yaml --verbose

# Debug mode for troubleshooting
./rpgo calculate config.yaml --debug
```

### `validate [input-file]` — Validate configuration file

Validate a YAML configuration file for syntax and structural correctness.

**Example:**

```bash
./rpgo validate config.yaml
```

### `break-even [input-file]` — Calculate break-even analysis

Calculate the TSP withdrawal rate needed to match current net income in retirement.

**Flags:**

- `--debug`: Enable debug output for detailed calculations

**Example:**

```bash
./rpgo break-even config.yaml
```

### `historical` — Manage and analyze historical financial data

Subcommands for loading, analyzing, and querying historical TSP, inflation, and COLA data.

#### `historical load [data-path]`

Load historical data from the specified directory path.

**Example:**

```bash
./rpgo historical load ./data
```

#### `historical stats [data-path]`

Display statistical summaries of historical data including TSP fund returns, inflation, and COLA rates.

**Example:**

```bash
./rpgo historical stats ./data
```

#### `historical query [data-path] [year] [fund-type]`

Query specific historical data for a given year and fund type.

**Fund types:** C, S, I, F, G, inflation, cola

**Example:**

```bash
# Query C Fund returns for 2020
./rpgo historical query ./data 2020 C

# Query inflation rate for 2020
./rpgo historical query ./data 2020 inflation

# Query COLA rate for 2020
./rpgo historical query ./data 2020 cola
```

#### `historical monte-carlo [data-path]` — Run Monte Carlo simulations

Run Monte Carlo simulations to analyze retirement portfolio sustainability using historical or statistical market data.

**Flags:**

- `--simulations, -s`: Number of simulations to run (default: 1000)
- `--years, -y`: Number of years to project (default: 25)
- `--historical, -d`: Use historical data (default: true)
- `--balance, -b`: Initial portfolio balance (default: 1000000)
- `--withdrawal, -w`: Annual withdrawal amount, or percentage as decimal for fixed_percentage strategy (e.g., 0.04 for 4%) (default: 40000)
- `--strategy, -t`: Withdrawal strategy (default: "fixed_amount")
- `--regulatory-config`: Path to regulatory config file (default: regulatory.yaml if it exists)

**Withdrawal strategies:**

- `fixed_amount`: Constant dollar amount each year
- `fixed_percentage`: Percentage of current portfolio balance (use decimal for withdrawal, e.g., 0.04 for 4%)
- `inflation_adjusted`: Dollar amount adjusted for inflation annually
- `guardrails`: Dynamically adjusts withdrawals based on portfolio performance

**Examples:**

**Baseline (4% Rule) Analysis:**

```bash
./rpgo historical monte-carlo ./data \
  --simulations 1000 \
  --balance 1000000 \
  --withdrawal 40000
```

**Guardrails Strategy for 30 Years:**

```bash
./rpgo historical monte-carlo ./data \
  --simulations 1000 \
  --balance 500000 \
  --withdrawal 25000 \
  --strategy guardrails \
  --years 30
```

**Statistical Sampling Mode:**

```bash
./rpgo historical monte-carlo ./data \
  --simulations 500 \
  --historical false \
  --balance 750000 \
  --withdrawal 35000 \
  --strategy inflation_adjusted
```

**Fixed Percentage Strategy (5% of balance):**

```bash
./rpgo historical monte-carlo ./data \
  --simulations 1000 \
  --balance 1000000 \
  --withdrawal 0.05 \
  --strategy fixed_percentage \
  --years 30
```

### `version` — Show version information

Display version, commit, and build information.

**Example:**

```bash
./rpgo version
```

## Common Use Cases

### 1. Initial Setup

```bash
# Validate configuration
./rpgo validate config.yaml

# Load historical data
./rpgo historical load ./data
```

### 2. Basic Retirement Analysis

```bash
# Run deterministic calculations
./rpgo calculate config.yaml

# Generate HTML report (save to file)
./rpgo calculate config.yaml --format html > report.html

# Run break-even analysis
./rpgo break-even config.yaml
```

### 3. Monte Carlo Risk Analysis

```bash
# Conservative 4% rule test (simple portfolio)
./rpgo historical monte-carlo ./data \
  --simulations 1000 \
  --balance 1000000 \
  --withdrawal 40000

# High-precision analysis with more simulations
./rpgo historical monte-carlo ./data \
  --simulations 5000 \
  --balance 1000000 \
  --withdrawal 40000

# Aggressive withdrawal with guardrails strategy
./rpgo historical monte-carlo ./data \
  --simulations 1000 \
  --balance 500000 \
  --withdrawal 30000 \
  --strategy guardrails \
  --years 30

# Inflation-adjusted withdrawal strategy
./rpgo historical monte-carlo ./data \
  --simulations 500 \
  --balance 750000 \
  --withdrawal 35000 \
  --strategy inflation_adjusted
```

### 4. Data Analysis

```bash
# View historical statistics
./rpgo historical stats ./data

# Query specific data points
./rpgo historical query ./data 2020 C
./rpgo historical query ./data 2020 inflation
./rpgo historical query ./data 2020 cola
```

## HTML Reports

The HTML output format generates interactive reports with visualizations to help understand retirement projections:

### Features

- **Scenario Summary Table**: Key metrics for all scenarios (first year income, 5/10-year projections, success rates, TSP longevity)
- **Calendar Year Comparisons**: Absolute year comparisons (2030, 2035, 2040) for apples-to-apples analysis
- **Pre-retirement Baseline**: Shows what current income would be in future years with COLA adjustments
- **Interactive Charts**:
  - **TSP Balance Over Time**: Line chart showing TSP balance projections for each scenario
  - **Net Income Comparison**: Line chart comparing net income trajectories between scenarios  
  - **Income Sources Breakdown**: Stacked bar chart showing composition of income (salary, pension, TSP, Social Security) in first retirement year

### Usage

```bash
# Generate and save HTML report
./rpgo calculate config.yaml --format html > retirement_analysis.html

# Open in browser (macOS)
open retirement_analysis.html

# Open in browser (Linux)
xdg-open retirement_analysis.html
```

### Technical Details

- Uses Chart.js for interactive visualizations
- Responsive design that works on desktop and mobile
- No external dependencies required when viewing (CDN-based)
- Charts are fully interactive with hover tooltips and zoom capabilities

## Output Interpretation

## Output formats and aliases

Supported formats (use with `--format`):

- `console` (verbose console summary)
- `console-lite` (compact console summary)
- `csv` (summary CSV)
- `detailed-csv` (full year-by-year CSV)
- `html` (interactive HTML report with charts)
- `json` (structured JSON)
- `all` (writes multiple outputs: verbose text + detailed CSV)

Aliases (mapped to canonical names):

- `console-verbose` → `console`
- `verbose` → `console`
- `csv-detailed` → `detailed-csv`
- `csv-summary` → `csv`
- `html-report` → `html`
- `json-pretty` → `json`

Notes:

- All output formats write to stdout by default. Redirect to save files (e.g., `> report.html`, `> report.json`, `> report.csv`).
- The `all` format creates timestamped files directly rather than writing to stdout.
- If an unknown format is provided, the error will list supported formats and aliases.

### Monte Carlo Results

**Success Rate**: Percentage of simulations where portfolio lasts the full projection period

- 🟢 **LOW RISK**: 95%+ success rate
- 🟡 **MODERATE RISK**: 85-95% success rate
- 🟠 **HIGH RISK**: 75-85% success rate
- 🔴 **VERY HIGH RISK**: <75% success rate

**Percentile Ranges**: Distribution of final portfolio values

- **P10**: 10% of simulations end with this balance or less
- **P25**: 25% of simulations end with this balance or less
- **P50**: Median ending balance
- **P75**: 75% of simulations end with this balance or less
- **P90**: 90% of simulations end with this balance or less

### Recommendations

**For Low Success Rates (<85%)**:

- Consider reducing withdrawal amount
- Increase allocation to bonds (F/G funds)
- Consider working longer or saving more

**For High Success Rates (>95%)**:

- Current plan appears sustainable
- Consider increasing withdrawal or taking more risk

**General Guidelines**:

- Start with 4% withdrawal rate
- Use guardrails strategy for flexibility
- Maintain 20-40% bond allocation
- Monitor and adjust annually

## Performance Tips

### Simulation Count

- **Quick Testing**: 100-500 simulations
- **Final Analysis**: 1000-5000 simulations
- **Research**: 10000+ simulations

### Data Sources

- **Historical Mode**: More realistic, slower execution
- **Statistical Mode**: Faster execution, less realistic sequences

### Memory Usage

- Scales with simulation count
- 1000 simulations: ~10MB memory
- 10000 simulations: ~100MB memory

## Error Handling

### Common Issues

**Historical Data Not Found**:

```bash
Error: Data path './data' does not exist
```

*Solution*: Ensure the data directory exists and contains the required CSV files.

**Invalid Configuration**:

```bash
Error: Configuration file is invalid
```

*Solution*: Use `./rpgo validate config.yaml` to check for errors.

**Simulation Errors**:

```bash
Error: Failed to run Monte Carlo simulation
```

*Solution*: Check that historical data is loaded and parameters are reasonable.

## Getting Help

```bash
# General help
./rpgo --help

# Command-specific help
./rpgo calculate --help
./rpgo historical --help
./rpgo historical monte-carlo --help

# Validate configuration
./rpgo validate config.yaml
```

This CLI reference provides comprehensive guidance for using all features of the FERS Retirement Calculator, from basic deterministic calculations to advanced Monte Carlo risk analysis.
