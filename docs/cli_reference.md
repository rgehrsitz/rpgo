# CLI Reference Guide

## Overview

The FERS Retirement Calculator provides a comprehensive command-line interface for retirement planning analysis, including deterministic calculations and probabilistic Monte Carlo simulations.

## Main Commands

### `calculate` - Run Retirement Scenarios
```bash
./fers-calc calculate [input-file] [flags]
```

**Description**: Calculate retirement scenarios using deterministic projections.

**Flags**:
- `--format, -f`: Output format (console, html, json, csv) [default: console]
- `--monte-carlo, -m`: Run Monte Carlo simulation [default: false]
- `--simulations, -s`: Number of Monte Carlo simulations [default: 1000]
- `--verbose, -v`: Verbose output [default: false]

**Examples**:
```bash
# Basic calculation
./fers-calc calculate config.yaml

# Generate HTML report
./fers-calc calculate config.yaml --format html

# Run with Monte Carlo
./fers-calc calculate config.yaml --monte-carlo --simulations 500
```

### `example` - Generate Example Configuration
```bash
./fers-calc example [output-file]
```

**Description**: Generate an example configuration file to get started.

**Example**:
```bash
./fers-calc example my_config.yaml
```

### `validate` - Validate Configuration
```bash
./fers-calc validate [input-file]
```

**Description**: Validate a configuration file without running calculations.

**Example**:
```bash
./fers-calc validate config.yaml
```

### `break-even` - Break-Even Analysis
```bash
./fers-calc break-even [input-file]
```

**Description**: Calculate break-even TSP withdrawal rates to match current net income.

**Example**:
```bash
./fers-calc break-even config.yaml
```

### `monte-carlo` - FERS Monte Carlo Simulations
```bash
./fers-calc monte-carlo [config-file] [data-path] [flags]
```

**Description**: Run comprehensive FERS Monte Carlo simulations that model all retirement components (pension, SS, TSP, taxes, FEHB) with variable market conditions.

**Flags**:
- `--simulations, -s`: Number of simulations to run [default: 1000]
- `--historical, -d`: Use historical data (false for statistical) [default: true]
- `--seed, -r`: Random seed (0 for auto-generated) [default: 0]

**Examples**:
```bash
# Basic FERS Monte Carlo simulation
./fers-calc monte-carlo config.yaml ./data

# High-precision simulation with 5000 runs
./fers-calc monte-carlo config.yaml ./data --simulations 5000

# Statistical mode (not historical data)
./fers-calc monte-carlo config.yaml ./data --historical false

# Reproducible results with fixed seed
./fers-calc monte-carlo config.yaml ./data --seed 12345
```

## Historical Data Commands

### `historical load` - Load Historical Data
```bash
./fers-calc historical load [data-path]
```

**Description**: Load and validate historical financial data.

**Example**:
```bash
./fers-calc historical load ./data
```

### `historical stats` - Display Statistics
```bash
./fers-calc historical stats [data-path]
```

**Description**: Display statistical summaries of historical data.

**Example**:
```bash
./fers-calc historical stats ./data
```

### `historical query` - Query Specific Data
```bash
./fers-calc historical query [data-path] [year] [fund-type]
```

**Description**: Query specific historical data for a given year and fund type.

**Fund Types**: C, S, I, F, G, inflation, cola

**Examples**:
```bash
# Query TSP C Fund return for 2020
./fers-calc historical query ./data 2020 C

# Query inflation rate for 2020
./fers-calc historical query ./data 2020 inflation

# Query COLA rate for 2020
./fers-calc historical query ./data 2020 cola
```

## Monte Carlo Simulation Commands

### Command Types Comparison

**Configuration-Based Commands** (use YAML config files):
- `calculate` - Uses `config.yaml` for all parameters
- `break-even` - Uses `config.yaml` for employee data

**Flag-Based Commands** (use command-line flags):
- `historical monte-carlo` - Uses flags for all parameters, `[data-path]` for historical data location
- `monte-carlo` - Uses config file + flags, comprehensive FERS analysis

### `historical monte-carlo` - Run Simple Portfolio Monte Carlo Simulations
```bash
./fers-calc historical monte-carlo [data-path] [flags]
```

**Description**: Run Monte Carlo simulations to analyze simple portfolio withdrawal sustainability.

**Note**: This command uses command-line flags to specify all parameters directly. The `[data-path]` parameter specifies the directory containing historical data files (e.g., `./data`). This is a simplified portfolio-only analysis.

### `monte-carlo` - Run Comprehensive FERS Monte Carlo Simulations
```bash
./fers-calc monte-carlo [config-file] [data-path] [flags]
```

**Description**: Run comprehensive FERS Monte Carlo simulations that model all retirement components (pension, SS, TSP, taxes, FEHB) with variable market conditions.

**Note**: This command uses a configuration file (like `calculate`) to specify all FERS retirement details, plus command-line flags for Monte Carlo parameters. This provides a complete FERS retirement risk analysis.

**Flags**:
- `--simulations, -s`: Number of simulations to run [default: 1000]
- `--years, -y`: Number of years to project [default: 25]
- `--historical, -d`: Use historical data (false for statistical) [default: true]
- `--balance, -b`: Initial portfolio balance [default: 1000000]
- `--withdrawal, -w`: Annual withdrawal amount [default: 40000]
- `--strategy, -t`: Withdrawal strategy [default: fixed_amount]

**Withdrawal Strategies**:
- `fixed_amount`: Same dollar amount each year
- `fixed_percentage`: Percentage of current portfolio balance
- `inflation_adjusted`: Initial amount adjusted for inflation
- `guardrails`: Dynamic adjustment based on portfolio performance

**Examples**:

**Basic 4% Rule Analysis**:
```bash
# Note: ./data is the historical data directory, not a config file
./fers-calc historical monte-carlo ./data \
  --simulations 1000 \
  --balance 1000000 \
  --withdrawal 40000
```

**Conservative Analysis with Guardrails**:
```bash
# ./data contains historical TSP returns, inflation, and COLA data
./fers-calc historical monte-carlo ./data \
  --simulations 1000 \
  --balance 500000 \
  --withdrawal 25000 \
  --strategy guardrails \
  --years 30
```

**Statistical Mode (Not Historical)**:
```bash
# Uses statistical distributions instead of historical data
./fers-calc historical monte-carlo ./data \
  --simulations 500 \
  --historical false \
  --balance 750000 \
  --withdrawal 35000 \
  --strategy inflation_adjusted
```

**Quick Test Run**:
```bash
# Quick test with fewer simulations for faster results
./fers-calc historical monte-carlo ./data \
  --simulations 100 \
  --years 20 \
  --balance 500000 \
  --withdrawal 25000
```

## Common Use Cases

### 1. Initial Setup
```bash
# Generate example configuration
./fers-calc example config.yaml

# Validate configuration
./fers-calc validate config.yaml

# Load historical data
./fers-calc historical load ./data
```

### 2. Basic Retirement Analysis
```bash
# Run deterministic calculations
./fers-calc calculate config.yaml

# Generate HTML report
./fers-calc calculate config.yaml --format html

# Run break-even analysis
./fers-calc break-even config.yaml
```

### 3. Monte Carlo Risk Analysis
```bash
# Conservative 4% rule test (simple portfolio)
./fers-calc historical monte-carlo ./data \
  --simulations 1000 \
  --balance 1000000 \
  --withdrawal 40000

# Comprehensive FERS Monte Carlo analysis
./fers-calc monte-carlo config.yaml ./data \
  --simulations 1000

# High-precision FERS analysis
./fers-calc monte-carlo config.yaml ./data \
  --simulations 5000 \
  --seed 12345

# Aggressive 6% rule with guardrails (simple portfolio)
./fers-calc historical monte-carlo ./data \
  --simulations 1000 \
  --balance 500000 \
  --withdrawal 30000 \
  --strategy guardrails \
  --years 30

# Inflation-adjusted strategy (simple portfolio)
./fers-calc historical monte-carlo ./data \
  --simulations 500 \
  --balance 750000 \
  --withdrawal 35000 \
  --strategy inflation_adjusted
```

### 4. Data Analysis
```bash
# View historical statistics
./fers-calc historical stats ./data

# Query specific data points
./fers-calc historical query ./data 2020 C
./fers-calc historical query ./data 2020 inflation
./fers-calc historical query ./data 2020 cola
```

## Output Interpretation

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
*Solution*: Use `./fers-calc validate config.yaml` to check for errors.

**Simulation Errors**:
```bash
Error: Failed to run Monte Carlo simulation
```
*Solution*: Check that historical data is loaded and parameters are reasonable.

## Getting Help

```bash
# General help
./fers-calc --help

# Command-specific help
./fers-calc calculate --help
./fers-calc historical --help
./fers-calc historical monte-carlo --help

# Validate configuration
./fers-calc validate config.yaml
```

This CLI reference provides comprehensive guidance for using all features of the FERS Retirement Calculator, from basic deterministic calculations to advanced Monte Carlo risk analysis. 