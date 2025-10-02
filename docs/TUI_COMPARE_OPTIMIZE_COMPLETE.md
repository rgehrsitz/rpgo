# TUI Compare and Optimize Scenes - Implementation Complete

**Date:** October 2025
**Status:** ✅ Complete

## Overview

The Compare and Optimize scenes provide powerful interactive tools for scenario analysis and break-even optimization within the TUI.

## Compare Scene

### Features Implemented

- **Multi-select Interface**: Checkbox-based selection of scenarios
- **Navigation**: Arrow keys (↑/↓) to move between scenarios
- **Selection**: Space or 'x' to toggle scenario selection
- **Validation**: Requires minimum 2 scenarios before allowing comparison
- **Parallel Calculation**: All selected scenarios calculated simultaneously
- **Side-by-side Table**: Clean comparison matrix showing key metrics
- **Best Value Highlighting**: Star (★) indicator for best values
- **Column Alignment**: ANSI-aware padding for proper formatting
- **Clear Function**: 'c' key to clear selections and start new comparison

### Metrics Displayed

1. **First Year Income** - Net income in first full retirement year
2. **TSP Longevity** - How many years TSP funds will last
3. **Final TSP Balance** - Remaining balance when depleted
4. **Lifetime Income** - Total income over projection period

### User Flow

1. Press `c` to navigate to Compare scene
2. Use ↑/↓ to navigate scenario list
3. Press Space/x to select/deselect scenarios (minimum 2)
4. Press Enter to run comparison
5. View side-by-side results with best values highlighted
6. Press `c` to clear and start new comparison
7. Press ESC to return to previous scene

### Technical Implementation

**Files:**
- `internal/tui/scenes/compare.go` - Complete scene implementation (430 lines)
- `internal/tui/model.go` - Added `calculateMultipleScenariosCmd` function
- `internal/tui/update.go` - Message handlers for comparison workflow

**Key Components:**
- `CompareModel` struct with selection tracking
- `getSelectedScenarios()` - Deterministic ordering using index iteration
- `renderComparisonTable()` - ANSI-aware table rendering
- `padRight()` helper - Uses `lipgloss.Width()` for correct padding

### Issues Resolved

#### Issue 1: Random Scenario Ordering
**Problem:** Map iteration caused inconsistent ordering between selection and results lookup.
**Solution:** Changed iteration from `for idx, isSelected := range m.selectedScenarios` to `for idx := 0; idx < len(m.scenarios); idx++` to ensure deterministic ordering.

#### Issue 2: Column Alignment
**Problem:** Lipgloss ANSI escape codes broke column alignment, causing values to overlap.
**Solution:** Updated `padRight()` to use `lipgloss.Width(s)` instead of `len(s)` to correctly calculate visible width.

## Optimize Scene

### Features Implemented

- **Break-even Solver**: Calculates optimal TSP withdrawal rate
- **Three-stage Workflow**:
  1. Select scenario to optimize
  2. Enter target annual net income
  3. View detailed results
- **Interactive Input**: Text input field with validation
- **Calculation Integration**: Uses existing `CalculateBreakEvenTSPWithdrawalRate`
- **Detailed Results**: Shows optimal rate, projected income, and breakdown
- **Visual Feedback**: Color-coded difference from target
- **Restart Capability**: 'n' key to run new optimization

### Results Display

1. **Optimal TSP Withdrawal Rate** - Calculated percentage
2. **Target vs Actual Income** - Comparison with difference highlighted
3. **Income Breakdown**:
   - Total Gross Income
   - Federal Tax
   - State Tax (if applicable)
   - Healthcare Costs (FEHB + Medicare)
   - Net Income

### User Flow

1. Press `o` to navigate to Optimize scene
2. Use ↑/↓ to select scenario
3. Press Enter to proceed to input
4. Type target annual net income (e.g., "150000")
5. Press Enter to run optimization
6. View results with optimal withdrawal rate
7. Press `n` for new optimization or ESC to go back

### Technical Implementation

**Files:**
- `internal/tui/scenes/optimize.go` - Complete scene implementation (440 lines)
- `internal/tui/model.go` - Added `optimizeBreakEvenCmd` function
- `internal/tui/update.go` - Message handlers for optimization workflow
- `internal/tui/tuimsg/messages.go` - Updated `OptimizationStartedMsg` structure

**Key Components:**
- `OptimizeModel` struct with three-mode state machine
- `textinput.Model` from Bubbles for target income entry
- `OptimizeResult` struct for result data
- Integration with `calculation.CalculateBreakEvenTSPWithdrawalRate`

**Dependencies Added:**
- `github.com/charmbracelet/bubbles/textinput` - Interactive text input
- `github.com/atotto/clipboard` - Required by textinput (transitive)

### Mode State Machine

```
ModeSelectScenario → ModeSetTarget → ModeShowResults
       ↑                                    |
       |                                    |
       +-------- (n key for new) -----------+
```

## Integration Points

Both scenes integrate with:
- **Configuration System**: Load scenarios from YAML
- **Calculation Engine**: Run full projections
- **Break-even Solver**: Optimize TSP withdrawal rates
- **Message System**: Async calculations with loading states
- **Resize System**: Respond to terminal size changes

## Testing

### Manual Testing Completed

- ✅ Scenario selection and deselection
- ✅ Comparison with 2+ scenarios
- ✅ Column alignment with various value widths
- ✅ Best value highlighting
- ✅ Clear and restart functionality
- ✅ Break-even optimization with various targets
- ✅ Text input validation
- ✅ Results display with income breakdown
- ✅ Terminal resize responsiveness

### Test Configurations

Used `test_config_generic.yaml` with multiple scenarios:
- "Both Retire in 2025"
- "Rob Retires at 62 - Feb 2027"

## Code Statistics

**Total Lines Added:**
- Compare Scene: ~430 lines
- Optimize Scene: ~440 lines
- Model/Update Integration: ~100 lines
- **Total: ~970 lines**

## Future Enhancements

### Potential Compare Scene Additions
- Export comparison to CSV/JSON
- Diff view showing parameter differences
- More metrics (IRMAA risk, tax burden, etc.)
- Chart visualization of differences

### Potential Optimize Scene Additions
- Multi-dimensional optimization (date + rate + SS age)
- Constraint specification (e.g., minimum TSP longevity)
- Sensitivity analysis around optimal value
- Save optimization results

## Lessons Learned

1. **ANSI Code Handling**: Always use `lipgloss.Width()` instead of `len()` when calculating string widths for alignment
2. **Map Iteration Order**: Go maps have random iteration order - use indexed loops for deterministic ordering
3. **Debug Output**: Strategic debug info placement helped quickly identify the root cause
4. **State Machines**: Multi-mode UIs benefit from explicit mode tracking
5. **Type Conversions**: Be careful with interface{} type assertions in message passing

## References

- [Bubble Tea Documentation](https://github.com/charmbracelet/bubbletea)
- [Bubbles Components](https://github.com/charmbracelet/bubbles)
- [Lipgloss Styling](https://github.com/charmbracelet/lipgloss)
- Implementation Plan: `docs/IMPLEMENTATION_PLAN.md`
- Phase 1.4 Status: `docs/PHASE_1.4_FOUNDATION_COMPLETE.md`
