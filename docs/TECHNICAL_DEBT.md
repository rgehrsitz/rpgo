# Technical Debt

This document tracks known technical debt items in the RPGO codebase.

## High Priority

### Non-Deterministic Map Iteration

**Issue**: Go's map iteration order is non-deterministic, which can cause small differences in calculation results between runs due to the order in which participants are processed.

**Impact**: 
- Test flakiness (mitigated with tolerance-based assertions)
- Potential for small calculation differences ($300-2500 out of $2M+ lifetime income, ~0.02-0.12% variance)
- First year calculations are generally consistent, but multi-year projections can vary slightly

**Root Cause**:
- `AnnualCashFlow` uses `map[string]decimal.Decimal` for participant-specific data
- Even with sorted iteration in aggregation functions, the underlying calculation may be affected by processing order
- The issue manifests around year 12 of projections, possibly related to RMD calculations or other age-based logic

**Mitigation**:
- Added `SortedMapKeys()` helper function for deterministic map iteration
- Sorted participant names before iteration in critical calculation paths
- Implemented tolerance-based assertions in tests ($2500 tolerance, ~0.12% of typical $2M lifetime income)

**Proposed Solution**:
1. Replace `map[string]T` with deterministic data structures throughout the codebase:
   - Option A: Use slices with binary search (requires participant indexing)
   - Option B: Use ordered map implementation (e.g., `github.com/wk8/go-ordered-map`)
   - Option C: Continue using maps but ensure all iterations are sorted (current approach)

2. Implement comprehensive determinism tests that verify bit-exact reproducibility

3. Consider adding a configuration option to use a fixed seed for any randomness

**Tracking**: 
- Tests affected: `test/integration/basic_integration_test.go::TestDataConsistency`
- Related files:
  - `internal/domain/projection.go` (AnnualCashFlow struct and methods)
  - `internal/calculation/projection.go` (main projection loop)
  - `internal/calculation/taxes.go` (tax aggregation)
  - `internal/config/input.go` (configuration normalization)

**Effort Estimate**: Medium (2-3 days for comprehensive fix)

**Priority**: Medium (calculations are still accurate within acceptable tolerance, but perfect determinism would be ideal for testing and reproducibility)

## Medium Priority

### Sensitivity Analysis Bugs

**Issue**: The inflation sensitivity analysis feature has fundamental bugs and has been deferred.

**Status**: Feature cancelled in Phase 3.3

**Impact**: Users cannot perform automated parameter sweep testing for robustness analysis

**Proposed Solution**: 
- Rewrite the sensitivity analysis engine with proper Monte Carlo integration
- Use the existing Monte Carlo framework for probabilistic analysis instead of deterministic sweeps

**Effort Estimate**: High (4-5 days)

**Priority**: Low (can be replaced by Monte Carlo analysis)

## Low Priority

### Test Coverage Gaps

**Issue**: Several packages have lower than desired test coverage

**Current Coverage**:
- CLI: 9.6%
- Config: 37.6%
- Domain: 36.0%
- Transform: 45.3%
- Output: 44.9%
- Calculation: 48.1%
- Compare: 59.5%

**Target Coverage**: 70%+ for all packages

**Proposed Solution**: 
- Add unit tests for edge cases
- Add integration tests for complex scenarios
- Add property-based tests for financial calculations

**Effort Estimate**: High (ongoing)

**Priority**: Low (existing coverage catches most bugs)

## Notes

- Items should be tracked in GitHub Issues when the repository is made public
- This document should be updated as technical debt is addressed or new debt is identified
- Each item should have a clear path forward, even if the solution is "accept as-is"
