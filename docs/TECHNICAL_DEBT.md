# Technical Debt & Known Issues Tracking

This document tracks technical debt, known limitations, and issues that need to be addressed in future development cycles.

## Monte Carlo Integration - Technical Debt

### High Priority Issues

#### 1. TSP Longevity Integration
**Issue**: Market variability affects TSP balance depletion timing, but current implementation doesn't properly integrate TSP return variability into the calculation engine's TSP balance calculations.

**Impact**: TSP longevity shows 30 years for all percentiles instead of varying based on market conditions.

**Root Cause**: The Monte Carlo engine generates variable TSP returns but doesn't properly integrate them into the calculation engine's TSP growth calculations throughout the projection.

**Files Affected**:
- `internal/calculation/fers_montecarlo.go` - Market condition generation
- `internal/calculation/engine.go` - TSP balance calculations
- `internal/calculation/projection.go` - TSP growth logic

**Proposed Solution**:
1. Modify the calculation engine to accept Monte Carlo TSP returns
2. Update TSP balance calculations to use variable returns instead of fixed rates
3. Ensure TSP depletion logic properly reflects market variability

#### 2. TSP Returns Deep Integration
**Issue**: While TSP returns are generated with variability, they need to be properly applied to the calculation engine's TSP growth calculations throughout the projection.

**Impact**: TSP balance projections don't reflect market variability, leading to unrealistic consistency.

**Root Cause**: The `createModifiedConfig` method modifies global assumptions but doesn't integrate TSP returns into the calculation engine's TSP-specific calculations.

**Files Affected**:
- `internal/calculation/fers_montecarlo.go` - `createModifiedConfig` method
- `internal/calculation/engine.go` - TSP calculation methods
- `internal/domain/employee.go` - TSP-related structures

**Proposed Solution**:
1. Extend the domain model to support variable TSP returns
2. Modify calculation engine to use Monte Carlo TSP returns
3. Update TSP balance calculations to reflect market variability

### Medium Priority Issues

#### 3. Monte Carlo Output Formatters
**Issue**: Need HTML and JSON formatters for Monte Carlo results to match other commands.

**Impact**: Limited output options for Monte Carlo results (console only).

**Root Cause**: Monte Carlo results use different data structures than other commands.

**Files Affected**:
- `internal/output/html_formatter.go` - HTML formatter
- `internal/output/json_formatter.go` - JSON formatter
- `cmd/rpgo/main.go` - Output format handling

**Proposed Solution**:
1. Create HTML formatter for `FERSMonteCarloResult`
2. Create JSON formatter for `FERSMonteCarloResult`
3. Update CLI command to support HTML/JSON output

#### 4. TSP Allocation Integration
**Issue**: Monte Carlo should respect and vary TSP allocations (C/S/I/F/G fund percentages) based on market conditions.

**Impact**: TSP allocations are fixed rather than reflecting market-driven allocation changes.

**Root Cause**: Current implementation uses fixed TSP allocations from configuration.

**Files Affected**:
- `internal/calculation/fers_montecarlo.go` - TSP allocation handling
- `internal/domain/employee.go` - TSP allocation structures

**Proposed Solution**:
1. Add TSP allocation variability to market conditions
2. Modify calculation engine to use variable allocations
3. Update TSP balance calculations to reflect allocation changes

## Phase 3.3 Sensitivity Analysis - Deferred Issues

### Critical Issues (Deferred)

#### 1. Parameter Changes Not Applied
**Issue**: Parameter changes not being applied correctly to calculations in sensitivity analysis.

**Impact**: Sensitivity analysis shows incorrect results.

**Root Cause**: Parameter modification logic has bugs in the sensitivity analysis implementation.

**Files Affected**:
- `cmd/rpgo/sensitivity.go` - Parameter modification logic
- `internal/calculation/sensitivity.go` - Sensitivity analysis engine

**Status**: DEFERRED - Needs debugging and fixing before production use.

#### 2. Sensitivity Metrics Showing $0.00
**Issue**: Sensitivity metrics showing $0.00 for all cases.

**Impact**: Sensitivity analysis produces meaningless results.

**Root Cause**: Calculation engine not properly receiving modified parameters.

**Files Affected**:
- `internal/calculation/sensitivity.go` - Sensitivity calculation logic
- `cmd/rpgo/sensitivity.go` - Parameter passing logic

**Status**: DEFERRED - Needs debugging and fixing before production use.

## General Technical Debt

### Low Priority Issues

#### 1. State/Local Tax Implementation
**Issue**: Only Pennsylvania state tax implemented.

**Impact**: Limited geographic coverage for state tax calculations.

**Files Affected**:
- `internal/calculation/taxes.go` - State tax calculations
- `internal/domain/employee.go` - Location structures

#### 2. External Pensions Integration
**Issue**: Schema exists but not fully integrated.

**Impact**: Limited support for external pension calculations.

**Files Affected**:
- `internal/domain/employee.go` - External pension structures
- `internal/calculation/pension.go` - Pension calculation logic

#### 3. RMD Enhancement
**Issue**: Basic RMD logic present but could be enhanced.

**Impact**: RMD calculations may not be comprehensive.

**Files Affected**:
- `internal/calculation/rmd.go` - RMD calculation logic

## Tracking Status

| Issue | Priority | Status | Assigned | Target Date |
|-------|----------|--------|----------|-------------|
| TSP Longevity Integration | High | Open | - | - |
| TSP Returns Deep Integration | High | Open | - | - |
| Monte Carlo Output Formatters | Medium | Open | - | - |
| TSP Allocation Integration | Medium | Open | - | - |
| Sensitivity Analysis Parameter Bugs | High | Deferred | - | - |
| Sensitivity Analysis $0.00 Bug | High | Deferred | - | - |
| State/Local Tax Expansion | Low | Open | - | - |
| External Pensions Integration | Low | Open | - | - |
| RMD Enhancement | Low | Open | - | - |

## Notes

- **Monte Carlo Integration**: Core functionality is complete and working, but TSP variability integration needs improvement
- **Sensitivity Analysis**: Deferred due to critical bugs that need debugging
- **General Issues**: Lower priority items that can be addressed in future releases

## Next Steps

1. **Immediate**: Address TSP longevity integration in Monte Carlo
2. **Short-term**: Implement Monte Carlo output formatters
3. **Medium-term**: Debug and fix sensitivity analysis issues
4. **Long-term**: Address general technical debt items
