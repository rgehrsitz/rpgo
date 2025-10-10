# RPGO Quick Reference Guide

**Version:** Phase 4.1 Complete  
**Date:** December 2024

## 🚀 **Available Commands**

### **Core Commands**
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

# Run Monte Carlo analysis
./rpgo fers-monte-carlo config.yaml --scenario "Scenario 1" --simulations 1000 --format console
```

### **Interactive TUI**
```bash
# Launch interactive terminal interface
./rpgo-tui config.yaml
```

## 📊 **Output Formats**

- `console` - Detailed console output (default)
- `console-lite` - Concise summary
- `html` - HTML report with charts
- `json` - JSON output for programmatic use
- `csv` - CSV export for spreadsheet analysis

## 🔧 **Configuration Examples**

### **Basic Household**
```yaml
household:
  participants:
    - name: "Alice Johnson"
      birth_date: "1965-06-15T00:00:00Z"
      is_federal: true
      hire_date: "1990-09-01T00:00:00Z"
      current_salary: 120000
      high_3_salary: 115000
      tsp_balance_traditional: 500000
      tsp_balance_roth: 100000
      tsp_contribution_percent: 0.10
      ss_benefit_fra: 2800
      fehb_premium_per_pay_period: 200
      is_primary_fehb_holder: true
      survivor_benefit_election_percent: 0.50
  filing_status: "single"
```

### **Healthcare Configuration**
```yaml
participants:
  - name: "Alice Johnson"
    healthcare:
      pre_medicare_coverage: "fehb"
      medicare_part_b: true
      medicare_part_d: true
      medicare_part_d_plan: "standard"
      medigap_plan: "G"
      drop_fehb_at_65: true
```

### **Roth Conversion Scenario**
```yaml
scenarios:
  - name: "With Roth Conversions"
    participant_scenarios:
      "Alice Johnson":
        roth_conversions:
          conversions:
            - year: 2028
              amount: 50000
              source: "traditional_tsp"
            - year: 2029
              amount: 50000
              source: "traditional_tsp"
```

### **Survivor Analysis Scenario**
```yaml
scenarios:
  - name: "Survivor Analysis"
    mortality:
      participants:
        "Alice Johnson":
          death_age: 69
      assumptions:
        survivor_spending_factor: 0.75
        tsp_spousal_transfer: "merge"
        filing_status_switch: "next_year"
```

## 🎯 **Key Features**

### **IRMAA Analysis**
- Medicare premium surcharge calculations
- Breach detection and warnings
- Recommendations for optimization

### **Tax-Smart Withdrawal Sequencing**
- Optimization of withdrawal order
- Tax efficiency strategies
- Bracket-fill optimization

### **Roth Conversion Planner**
- Multi-year optimization
- Bracket-fill strategy
- Multiple optimization objectives

### **Healthcare Cost Modeling**
- Pre-65 coverage options (FEHB, COBRA, Marketplace)
- Medicare Part B/D with IRMAA
- Medigap integration
- Age-based transitions

### **Survivor Viability Analysis**
- Pre-death vs post-death comparison
- Income replacement analysis
- Life insurance needs calculation
- Viability scoring and recommendations

## 📈 **Sample Outputs**

### **IRMAA Risk Analysis**
```
IRMAA RISK ANALYSIS (Medicare Premium Surcharges):
---------------------------------------------------
⚠️  IRMAA BREACHES DETECTED
  Years with breaches:    22
  First breach year:      8
  Total IRMAA cost:       $70023.60

High Risk Years:
Year    MAGI          Status      Tier      Annual Cost
-----------------------------------------------------------------
8       $111403.36    ✗ Breach    Tier1     $838.80
15      $131958.64    ✗ Breach    Tier2     $2935.20
24      $164259.64    ✗ Breach    Tier3     $6289.20
```

### **Survivor Viability Analysis**
```
SURVIVOR VIABILITY ANALYSIS
================================================================================
Scenario: Survivor Analysis - Alice Dies at 69
Deceased: Alice Johnson (age 69 in 2034)
Survivor: Bob Johnson (age 71 at time of death)

VIABILITY ASSESSMENT:
  Survivor Income vs. Target (75% of couple):
    Target:    $126608.15
    Actual:    $110439.80
    Shortfall: $16168.36 (12.8%)  🔴 CRITICAL

LIFE INSURANCE NEEDS:
  To bridge $16168.36/year gap for 20 years:
    Present Value (4% discount): $228522.61
    Recommended coverage: $274227.13
```

## 🔍 **Troubleshooting**

### **Common Issues**
1. **Date Format**: Use `YYYY-MM-DDTHH:MM:SSZ` format
2. **Participant Names**: Must match exactly between household and scenarios
3. **Filing Status**: Use `"single"` or `"married_filing_jointly"`
4. **Healthcare Config**: Optional but recommended for accurate modeling

### **Debug Mode**
```bash
# Enable debug output
./rpgo calculate config.yaml --debug
```

## 📚 **Documentation**

- **Implementation Plan**: `docs/IMPLEMENTATION_PLAN.md`
- **Project Status**: `docs/PROJECT_STATUS.md`
- **CLI Reference**: `docs/cli_reference.md`
- **Examples**: `docs/examples/`

## 🎯 **Next Features**

- **Part-Time Work Modeling**: Phased retirement scenarios
- **Inflation Sensitivity Analysis**: Parameter sweep testing
- **Enhanced HTML Reports**: Interactive charts and drill-downs
- **Monte Carlo Integration**: Statistical modeling with market variability

---

**For more information, see the full documentation in the `docs/` directory.**
