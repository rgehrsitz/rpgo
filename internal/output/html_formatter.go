package output

import (
	"bytes"
	_ "embed"
	"html/template"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// HTMLFormatter produces an HTML report (current implementation ports legacy static HTML).
type HTMLFormatter struct{}

func (h HTMLFormatter) Name() string { return "html" }

//go:embed templates/report.html.tmpl
var htmlTemplateSource string

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"curr":   FormatCurrency,
	"pct":    FormatPercentage,
	"minus1": func(i int) int { return i - 1 },
	"add":    func(a, b decimal.Decimal) decimal.Decimal { return a.Add(b) },
	"addInt": func(a, b int) int { return a + b },
	"slice": func(items []domain.ScenarioSummary, start int) []domain.ScenarioSummary {
		if start >= len(items) {
			return []domain.ScenarioSummary{}
		}
		return items[start:]
	},
	"hasWithdrawalSequencing": func(scenario domain.ScenarioSummary) bool {
		if scenario.Projection == nil {
			return false
		}
		for _, year := range scenario.Projection {
			if year.WithdrawalTaxable.GreaterThan(decimal.Zero) ||
				year.WithdrawalTraditional.GreaterThan(decimal.Zero) ||
				year.WithdrawalRoth.GreaterThan(decimal.Zero) {
				return true
			}
		}
		return false
	},
	"getWithdrawalYears": func(scenario domain.ScenarioSummary) []domain.AnnualCashFlow {
		var withdrawalYears []domain.AnnualCashFlow
		if scenario.Projection == nil {
			return withdrawalYears
		}
		for _, year := range scenario.Projection {
			if year.WithdrawalTaxable.GreaterThan(decimal.Zero) ||
				year.WithdrawalTraditional.GreaterThan(decimal.Zero) ||
				year.WithdrawalRoth.GreaterThan(decimal.Zero) {
				withdrawalYears = append(withdrawalYears, year)
			}
		}
		return withdrawalYears
	},
	"analyzeWithdrawalStrategy": func(years []domain.AnnualCashFlow) string {
		if len(years) == 0 {
			return "No data"
		}

		taxableTotal := decimal.Zero
		traditionalTotal := decimal.Zero
		rothTotal := decimal.Zero

		for _, year := range years {
			taxableTotal = taxableTotal.Add(year.WithdrawalTaxable)
			traditionalTotal = traditionalTotal.Add(year.WithdrawalTraditional)
			rothTotal = rothTotal.Add(year.WithdrawalRoth)
		}

		if taxableTotal.GreaterThan(decimal.Zero) && traditionalTotal.IsZero() && rothTotal.IsZero() {
			return "Taxable-first (Standard)"
		} else if rothTotal.GreaterThan(decimal.Zero) && traditionalTotal.IsZero() && taxableTotal.IsZero() {
			return "Roth-first (Tax Efficient)"
		} else if traditionalTotal.GreaterThan(decimal.Zero) && rothTotal.IsZero() && taxableTotal.IsZero() {
			return "Traditional-first"
		} else {
			return "Mixed withdrawal sources"
		}
	},
}).Parse(htmlTemplateSource))

func (h HTMLFormatter) Format(results *domain.ScenarioComparison) ([]byte, error) {
	var buf bytes.Buffer
	rec := AnalyzeScenarios(results)

	// Use assumptions from results if available, otherwise fall back to defaults
	assumptions := results.Assumptions
	if len(assumptions) == 0 {
		assumptions = DefaultAssumptions
	}

	data := struct {
		*domain.ScenarioComparison
		Recommendation Recommendation
		Assumptions    []string
	}{results, rec, assumptions}
	if err := htmlTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
