# Integration Tests

This directory contains comprehensive integration tests for the RPGO retirement planning calculator. These tests are designed to catch issues where unit tests pass but features don't work in the overall system.

## Test Structure

### Core Integration Tests

- **`basic_integration_test.go`** - Tests core end-to-end functionality including:
  - Configuration loading and validation
  - Calculation engine execution
  - Output generation in all formats (console, JSON, CSV, HTML)
  - Error handling
  - Performance requirements
  - Data consistency across operations

- **`integration_suite_test.go`** - Test suite runner and smoke tests

### Advanced Integration Tests (Work in Progress)

The following test files contain comprehensive integration tests for advanced features, but may need API updates to work with the current codebase:

- **`cli_commands_test.go`** - Tests all CLI commands (calculate, compare, optimize, break-even, historical)
- **`config_validation_test.go`** - Tests configuration loading and validation across different formats
- **`transform_pipeline_test.go`** - Tests transform pipeline and template system
- **`monte_carlo_integration_test.go`** - Tests Monte Carlo simulations with historical data
- **`irmaa_integration_test.go`** - Tests IRMAA calculations and alerts
- **`withdrawal_sequencing_test.go`** - Tests withdrawal sequencing strategies
- **`roth_conversion_test.go`** - Tests Roth conversion planning features
- **`tui_integration_test.go`** - Tests TUI functionality and user interactions

## Running Integration Tests

### Using Make

```bash
# Run all integration tests
make test-integration

# Run smoke tests only (quick validation)
make test-integration-smoke

# Run performance benchmarks
make test-integration-benchmarks

# Run regression tests
make test-integration-regression

# Run all tests (unit + integration smoke)
make test-all
```

### Using Go Test Directly

```bash
# Run all integration tests
go test ./test/integration/... -v -timeout=10m

# Run specific test
go test ./test/integration/... -run="TestBasicIntegration" -v

# Run with coverage
go test ./test/integration/... -cover -v
```

## Test Data

Integration tests use configuration files from `../testdata/`:

- **`generic_example_config.yaml`** - Generic test configuration with two participants
- **`example_config.yaml`** - Example configuration with Robert and Dawn

## Test Categories

### 1. Basic Integration Tests
- Configuration loading and validation
- Calculation engine execution
- Output format generation
- Error handling

### 2. CLI Command Tests
- All major CLI commands
- Command-line argument parsing
- Output format validation
- Error conditions

### 3. Feature-Specific Tests
- Transform pipeline and templates
- Monte Carlo simulations
- IRMAA calculations
- Withdrawal sequencing
- Roth conversion planning
- TUI functionality

### 4. Performance Tests
- Calculation performance benchmarks
- Output generation performance
- Memory usage validation

### 5. Regression Tests
- Data consistency across runs
- Output format consistency
- Configuration serialization/deserialization

## Test Environment

Integration tests set up a controlled test environment:

- Set `RPGO_TEST_MODE=true` environment variable
- Reduce log noise with `RPGO_LOG_LEVEL=error`
- Create temporary directories for test artifacts
- Clean up resources after tests complete

## Best Practices

1. **Test Real Workflows** - Integration tests should exercise complete user workflows, not just individual components.

2. **Use Real Data** - Tests use actual configuration files and data to catch real-world issues.

3. **Test Error Conditions** - Include tests for missing files, invalid configurations, and other error scenarios.

4. **Performance Validation** - Ensure operations complete within reasonable time limits.

5. **Data Consistency** - Verify that repeated operations produce identical results.

6. **Cross-Component Testing** - Test interactions between different system components.

## Troubleshooting

### Common Issues

1. **Missing Test Data** - Ensure test configuration files exist in `../testdata/`
2. **Timeout Issues** - Increase timeout for complex calculations
3. **API Mismatches** - Some advanced tests may need updates when APIs change
4. **Environment Issues** - Check that required data directories exist

### Debugging

Enable verbose output to see detailed test execution:

```bash
go test ./test/integration/... -v -timeout=30m
```

For specific test debugging:

```bash
go test ./test/integration/... -run="TestBasicIntegration" -v -timeout=5m
```

## Future Enhancements

1. **API Alignment** - Update advanced integration tests to match current APIs
2. **Visual Regression Testing** - Add tests for HTML output consistency
3. **Load Testing** - Add tests for large-scale scenarios
4. **Cross-Platform Testing** - Ensure tests work on different operating systems
5. **CI/CD Integration** - Add automated integration test runs in CI pipeline



