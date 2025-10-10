package output

import (
	"fmt"
	"os"
	"testing"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestFormatterFunc_Format(t *testing.T) {
	called := false
	var receivedResults *domain.ScenarioComparison

	formatter := FormatterFunc{
		ID: "test-formatter",
		F: func(results *domain.ScenarioComparison) ([]byte, error) {
			called = true
			receivedResults = results
			return []byte("test output"), nil
		},
	}

	testResults := buildTestComparison()
	output, err := formatter.Format(testResults)

	assert.NoError(t, err, "Should not error")
	assert.True(t, called, "Should call the function")
	assert.Equal(t, testResults, receivedResults, "Should pass the results")
	assert.Equal(t, []byte("test output"), output, "Should return the function output")
}

func TestFormatterFunc_Name(t *testing.T) {
	formatter := FormatterFunc{
		ID: "test-formatter",
		F: func(results *domain.ScenarioComparison) ([]byte, error) {
			return []byte("test"), nil
		},
	}

	assert.Equal(t, "test-formatter", formatter.Name(), "Should return the ID")
}

func TestWriteFormatted(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	originalDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalDir)

	formatter := FormatterFunc{
		ID: "test-formatter",
		F: func(results *domain.ScenarioComparison) ([]byte, error) {
			return []byte("test output content"), nil
		},
	}

	testResults := buildTestComparison()
	filename, err := WriteFormatted(formatter, testResults, "txt")

	assert.NoError(t, err, "Should not error")
	assert.Contains(t, filename, "retirement_report_", "Should have correct prefix")
	assert.Contains(t, filename, ".txt", "Should have correct extension")

	// Check that the file was created and has the right content
	content, err := os.ReadFile(filename)
	assert.NoError(t, err, "Should be able to read the file")
	assert.Equal(t, "test output content", string(content), "Should have correct content")
}

func TestWriteFormatted_FormatterError(t *testing.T) {
	formatter := FormatterFunc{
		ID: "error-formatter",
		F: func(results *domain.ScenarioComparison) ([]byte, error) {
			return nil, fmt.Errorf("formatter error")
		},
	}

	testResults := buildTestComparison()
	filename, err := WriteFormatted(formatter, testResults, "txt")

	assert.Error(t, err, "Should error when formatter fails")
	assert.Empty(t, filename, "Should return empty filename on error")
	assert.Contains(t, err.Error(), "formatter error", "Should propagate formatter error")
}

func TestConsoleFormatter_Name(t *testing.T) {
	formatter := ConsoleFormatter{}
	assert.Equal(t, "console-lite", formatter.Name(), "Should return correct name")
}

func TestConsoleFormatter_Format_EmptyScenarios(t *testing.T) {
	formatter := ConsoleFormatter{}

	results := &domain.ScenarioComparison{
		BaselineNetIncome: decimal.NewFromInt(100000),
		Scenarios:         []domain.ScenarioSummary{},
	}

	output, err := formatter.Format(results)

	assert.NoError(t, err, "Should not error")
	assert.NotEmpty(t, output, "Should return output")

	content := string(output)
	assert.Contains(t, content, "RETIREMENT SCENARIO SUMMARY", "Should have header")
	assert.Contains(t, content, "Current Net Income: $100000.00", "Should show baseline income")
}

func TestConsoleFormatter_Format_WithRecommendation(t *testing.T) {
	formatter := ConsoleFormatter{}

	results := buildTestComparison()
	output, err := formatter.Format(results)

	assert.NoError(t, err, "Should not error")
	assert.NotEmpty(t, output, "Should return output")

	content := string(output)
	assert.Contains(t, content, "RETIREMENT SCENARIO SUMMARY", "Should have header")
	assert.Contains(t, content, "Recommended: B", "Should have recommendation")
	assert.Contains(t, content, "Δ $5000.00", "Should show income change")
}

func TestConsoleVerboseFormatter_Name(t *testing.T) {
	formatter := ConsoleVerboseFormatter{}
	assert.Equal(t, "console", formatter.Name(), "Should return correct name")
}

func TestConsoleVerboseFormatter_Format_EmptyScenarios(t *testing.T) {
	formatter := ConsoleVerboseFormatter{}

	results := &domain.ScenarioComparison{
		BaselineNetIncome: decimal.NewFromInt(100000),
		Scenarios:         []domain.ScenarioSummary{},
	}

	output, err := formatter.Format(results)

	assert.NoError(t, err, "Should not error")
	assert.NotEmpty(t, output, "Should return output")

	content := string(output)
	assert.Contains(t, content, "DETAILED FERS RETIREMENT INCOME ANALYSIS", "Should have verbose header")
	assert.Contains(t, content, "Combined Net Income:  $100000.00", "Should show baseline income")
}

func TestCSVSummarizer_Name(t *testing.T) {
	formatter := CSVSummarizer{}
	assert.Equal(t, "csv", formatter.Name(), "Should return correct name")
}

func TestCSVSummarizer_Format(t *testing.T) {
	formatter := CSVSummarizer{}

	results := buildTestComparison()
	output, err := formatter.Format(results)

	assert.NoError(t, err, "Should not error")
	assert.NotEmpty(t, output, "Should return output")

	content := string(output)
	assert.Contains(t, content, "Scenario", "Should have CSV header")
	assert.Contains(t, content, "A,", "Should have scenario A")
	assert.Contains(t, content, "B,", "Should have scenario B")
}

func TestJSONFormatter_Name(t *testing.T) {
	formatter := JSONFormatter{}
	assert.Equal(t, "json", formatter.Name(), "Should return correct name")
}

func TestJSONFormatter_Format(t *testing.T) {
	formatter := JSONFormatter{}

	results := buildTestComparison()
	output, err := formatter.Format(results)

	assert.NoError(t, err, "Should not error")
	assert.NotEmpty(t, output, "Should return output")

	content := string(output)
	assert.Contains(t, content, "\"baselineNetIncome\"", "Should have JSON structure")
	assert.Contains(t, content, "\"scenarios\"", "Should have scenarios array")
	assert.Contains(t, content, "\"A\"", "Should have scenario A")
	assert.Contains(t, content, "\"B\"", "Should have scenario B")
}

func TestHTMLFormatter_Name(t *testing.T) {
	formatter := HTMLFormatter{}
	assert.Equal(t, "html", formatter.Name(), "Should return correct name")
}

func TestHTMLFormatter_Format(t *testing.T) {
	formatter := HTMLFormatter{}

	results := buildTestComparison()
	output, err := formatter.Format(results)

	assert.NoError(t, err, "Should not error")
	assert.NotEmpty(t, output, "Should return output")

	content := string(output)
	assert.Contains(t, content, "<!DOCTYPE html>", "Should have HTML structure")
	assert.Contains(t, content, "<title>", "Should have title")
	assert.Contains(t, content, "FERS Retirement Planning Analysis", "Should have main heading")
}

func TestAvailableFormatterNames(t *testing.T) {
	names := AvailableFormatterNames()

	assert.NotEmpty(t, names, "Should return formatter names")

	// Check that expected formatters are present
	formatterNames := make(map[string]bool)
	for _, name := range names {
		formatterNames[name] = true
	}

	assert.True(t, formatterNames["console-lite"], "Should include console-lite")
	assert.True(t, formatterNames["console"], "Should include console")
	assert.True(t, formatterNames["csv"], "Should include csv")
	assert.True(t, formatterNames["detailed-csv"], "Should include detailed-csv")
	assert.True(t, formatterNames["json"], "Should include json")
	assert.True(t, formatterNames["html"], "Should include html")
}

func TestAvailableFormatAliases(t *testing.T) {
	aliases := AvailableFormatAliases()

	assert.NotEmpty(t, aliases, "Should return format aliases")

	// Check that expected aliases are present
	aliasMap := make(map[string]bool)
	for _, alias := range aliases {
		aliasMap[alias] = true
	}

	assert.True(t, aliasMap["verbose"], "Should include verbose alias")
	assert.True(t, aliasMap["console-verbose"], "Should include console-verbose alias")
}

func TestGetFormatterByName_ExistingFormatter(t *testing.T) {
	formatter := GetFormatterByName("console-lite")

	assert.NotNil(t, formatter, "Should return formatter")
	assert.Equal(t, "console-lite", formatter.Name(), "Should return correct formatter")
}

func TestGetFormatterByName_NonExistentFormatter(t *testing.T) {
	formatter := GetFormatterByName("non-existent")

	assert.Nil(t, formatter, "Should return nil formatter for non-existent name")
}
