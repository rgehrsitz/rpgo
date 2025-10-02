# IRMAA Alerts - Quick Start Guide

## Fast Proof That IRMAA Detection Works

The IRMAA threshold detection is fully integrated into projections (TUI, console, HTML). You can still run a focused unit test to see tier transitions:

```bash
go test -v ./internal/calculation -run "TestIRMAAThresholdExamples"
```

You'll see output like:

```
MAGI: $199000 (expected ~$199000)
Risk: Warning (expected Warning)  ← Warning detected!
Distance to next threshold: $7000

MAGI: $210700 (expected ~$210700)
Risk: Breach (expected Breach)    ← Breach detected!
Tier: Tier1
Surcharge: $69.90/month            ← Surcharge calculated!
```

## Why You Might See “No IRMAA Concerns”

That usually means your plan's MAGI is below the first threshold (this is good). To confirm the system works, use the provided high‑income test file or temporarily raise taxable withdrawals.

### 2025 IRMAA Thresholds (Married Filing Jointly)

| Threshold | What Triggers It |
|-----------|------------------|
| **$206,000** | First IRMAA tier (+$69.90/month surcharge per person) |
| $258,000 | Second tier (+$174.70/month) |
| $322,000 | Third tier (+$279.50/month) |

### Typical FERS Retirement Income (Example)

Pension $60K + SS taxable portion ~$29.8K + TSP $40K = MAGI ≈ $130K → Safe (far below $206K).

## How to Test IRMAA Alerts

### Option 1: Use Included High-Income Config

File: `test_irmaa_high_income.yaml` (generic schema) — designed to push MAGI into warning/breach territory once both participants retire and Social Security + high variable percentage withdrawals overlap.

Run:

```bash
./rpgo calculate test_irmaa_high_income.yaml -f console | grep -A6 "IRMAA RISK" -i || true
./rpgo calculate test_irmaa_high_income.yaml -f html > report.html
open report.html  # (macOS)
```

### Option 2: Boost Taxable Withdrawals Temporarily

In your scenario (TUI):

1. Open scenario
2. Press `p` (Parameters)
3. Switch withdrawal strategy to variable percentage if needed
4. Set rate to 0.09–0.10
5. Recalculate and view results; if MAGI approaches within $10K → Warning; exceeds threshold → Breach

### Option 3: Single Filer Sensitivity

Lower first threshold ($103K) means moderate pensions + SS can already create warnings/breaches. Use a trimmed config with one participant and raise TSP rate to demonstrate.

## What MAGI Includes (Implemented Simplification)

**Counted in MAGI (increases IRMAA risk):**

- ✓ Salaries and wages
- ✓ FERS pension
- ✓ **Traditional TSP withdrawals** ← This is the big one
- ✓ Taxable Social Security (usually 85%)
- ✓ FERS supplement
- ✓ Interest, dividends, capital gains

**NOT counted in MAGI:**

- ✗ Roth TSP withdrawals (tax-free!)
- ✗ Return of basis from taxable accounts
- ✗ Life insurance proceeds

## IRMAA Mitigation / Optimization Strategies

If you see IRMAA warnings in results:

### 1. Roth Conversions (Before Medicare)

Convert Traditional TSP → Roth TSP **before age 65**:

- Pay tax now at lower rates
- Future Roth withdrawals don't count toward MAGI
- Avoids IRMAA surcharges later

### 2. Time TSP Withdrawals

- Withdraw more in years before Social Security starts
- Withdraw less in high-income years
- Smooth income across retirement

### 3. Delay Social Security

- If TSP withdrawals are high, delay SS to 67 or 70
- Reduces simultaneous income sources
- Gives time for Roth conversions

### 4. Use Roth TSP First

- Withdraw from Roth TSP during Medicare years
- Keep Traditional TSP withdrawals minimal
- Reduces MAGI

## Current Status (Post-Polish)

Fully integrated pipeline:

- Projection engine computes MAGI annually (only once per year record)
- IRMAA risk + surcharge stored on each Medicare-eligible year
- Multi-year analysis attaches `IRMAAAnalysis` to scenario summaries
- Output surfaces: TUI, console (`-f console`), HTML (`-f html`)
- Enhanced recommendation engine provides pattern + severity-aware guidance

## Verifying Output Modes Quickly

```bash
# Console detailed output
./rpgo calculate test_irmaa_high_income.yaml -f console | grep -A4 "IRMAA RISK" -i

# HTML report (open in browser)
./rpgo calculate test_irmaa_high_income.yaml -f html > report.html
open report.html

# TUI (interactive)
./rpgo-tui test_irmaa_high_income.yaml
```

## Deep-Dive Validation (Optional)

```bash
go test -v ./internal/calculation -run TestIRMAAThresholdExamples
go test -v ./internal/calculation -run TestIRMAAIntegration
```

These confirm threshold edges and multi-year breach/warning aggregation logic.

---

Need a breach but still showing Safe? Increase Traditional withdrawal rate OR add a one-time large withdrawal year (e.g., simulate Roth conversion before 65 then large Traditional draw at 67). Small adjustments near the $206K (MFJ) or $103K (Single) lines can flip classifications.
