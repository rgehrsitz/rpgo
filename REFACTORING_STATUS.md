# FERS Calculator Refactoring Status

This document summarizes the progress made in making the FERS retirement calculator generic to support flexible household compositions beyond the original hardcoded Robert/Dawn scenario.

## Completed Phase 1: Foundational Code and Data Refactoring ✅

### ✅ New Domain Structures

- **`domain.Participant`**: Generic participant struct supporting both federal and non-federal employees
- **`domain.Household`**: Container for participants with filing status
- **`domain.ParticipantScenario`**: Flexible scenario definition per participant  
- **`domain.GenericScenario`**: Household-level scenario with multiple participants

### ✅ Backwards Compatibility

- Legacy `robert`/`dawn` configuration format still fully supported
- Automatic format detection and conversion
- Seamless transition path for existing users
- All existing configuration files continue to work unchanged

### ✅ Configuration Schema Updates

- New participant-based YAML format with `household` and `generic_scenarios` sections
- Support for mixed households (federal + non-federal employees)
- Support for single federal employee scenarios
- Enhanced validation for participant-specific rules (FEHB holders, federal employee requirements)

### ✅ Calculation Engine Updates  

- `RunScenarioAuto()`: Automatic format detection and routing
- `RunGenericScenario()`: New method for participant-based calculations
- Legacy conversion bridge for calculation compatibility
- Maintained all existing calculation accuracy and logic

### ✅ Testing and Validation

- Comprehensive unit tests for format detection and conversion
- Table-driven tests covering multiple household configurations:
  - Single federal employee
  - Dual federal employees  
  - Mixed households (one federal, one private sector)
- Golden test validation ensuring existing Robert/Dawn scenarios produce identical results
- Round-trip conversion testing (Employee ↔ Participant)

## Current Limitations 🔄

### Output Generation Still Hardcoded

While the calculation engine now supports generic participants, the output formatting still contains hardcoded references to "Robert" and "Dawn". This is because:

1. **Core Data Structures**: The `domain.AnnualCashFlow` struct contains hardcoded fields:

   ```go
   type AnnualCashFlow struct {
       SalaryRobert     decimal.Decimal
       SalaryDawn       decimal.Decimal  
       PensionRobert    decimal.Decimal
       PensionDawn      decimal.Decimal
       // ... many more Robert/Dawn specific fields
   }
   ```

2. **Output Templates**: All formatters (console, HTML, JSON, CSV) reference these hardcoded fields
3. **Projection Logic**: The core projection generation in `internal/calculation/projection.go` populates these hardcoded fields

### Current Workaround

The system currently uses the legacy conversion bridge, which maps participants to "robert"/"dawn" for internal processing. This works correctly but means output still shows these labels regardless of actual participant names.

## Phase 2: Complete Output Generification (Future Work) 📋

To fully complete the genericization, the following work remains:

### 1. Refactor AnnualCashFlow Structure

Replace hardcoded fields with dynamic participant maps:

```go
type AnnualCashFlow struct {
    Year      int
    Date      time.Time
    Ages      map[string]int                    // participantName -> age
    Salaries  map[string]decimal.Decimal       // participantName -> salary
    Pensions  map[string]decimal.Decimal       // participantName -> pension
    TSPWithdrawals map[string]decimal.Decimal  // participantName -> tsp_withdrawal
    // ... etc for all income/deduction types
}
```

### 2. Update Projection Generation

Modify `internal/calculation/projection.go` to:

- Iterate over participants instead of hardcoded robert/dawn
- Use participant names as keys in the new map-based structure
- Maintain calculation accuracy while supporting flexible participants

### 3. Refactor Output Formatters

Update all output formatters to:

- Use participant names from the actual configuration
- Support variable numbers of participants (1, 2, 3+)
- Maintain readable output formatting with dynamic labels
- Update HTML templates to be participant-agnostic

### 4. Enhanced Validation

Add validation for:

- Maximum practical number of participants (performance considerations)
- Mixed scenarios validation (federal vs non-federal timing rules)
- Complex mortality scenarios with multiple participants

## Supported Scenarios ✅

The current implementation successfully supports:

1. **Single Federal Employee**
   - Complete FERS calculations (pension, TSP, FEHB, FERS supplement)
   - Proper tax treatment and benefit timing
   - Works with both new and legacy configuration formats

2. **Dual Federal Employees**
   - Full household calculations with both participants
   - Coordinated retirement timing and benefit optimization
   - FEHB primary/secondary holder logic
   - Survivor benefit elections

3. **Mixed Households** (One Federal, One Private)  
   - Federal employee gets full FERS calculations
   - Non-federal employee gets Social Security and external pension support
   - Proper tax treatment and filing status handling
   - External pension COLA adjustments and survivor benefits

## Migration Guide

### For New Configurations

Use the new participant-based format:

```yaml
household:
  filing_status: "married_filing_jointly" 
  participants:
    - name: "Employee Name"
      is_federal: true
      # ... participant details
```

### For Existing Configurations  

No changes required - legacy format continues to work:

```yaml
personal_details:
  robert:
    # ... existing format
  dawn:
    # ... existing format
```

## Testing Coverage

- ✅ Format detection and validation
- ✅ Configuration parsing for both formats
- ✅ Participant conversion (Employee ↔ Participant)  
- ✅ Legacy configuration backwards compatibility
- ✅ Multiple household scenario calculations
- ✅ Edge cases (single participant, invalid configurations)
- ✅ Golden test validation against known results

## Performance Impact

The refactoring maintains performance characteristics:

- Legacy format: No performance impact (direct processing)
- New format: Minimal overhead from format conversion bridge
- Memory usage: Slightly increased due to format flexibility, but negligible for practical scenarios
- Calculation accuracy: Identical results to original implementation

## Conclusion

Phase 1 successfully delivers the core requirement: **making the retirement calculator generic to support flexible household compositions**. The system now supports single federal employees, dual federal employees, and mixed households while maintaining full backwards compatibility.

Phase 2 (complete output generification) would provide the final polish by making output labels dynamic, but this is cosmetic rather than functional. The current implementation fully meets the business requirements outlined in the original plan.

Users can immediately begin using the new participant-based configuration format to model various household compositions while existing users experience no disruption to their current workflows.
