package output

import (
	"bytes"
	_ "embed"
	"html/template"

	"github.com/rgehrsitz/rpgo/internal/domain"
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
	"add":    func(i, j int) int { return i + j },
	"slice": func(items []domain.ScenarioSummary, start int) []domain.ScenarioSummary {
		if start >= len(items) {
			return []domain.ScenarioSummary{}
		}
		return items[start:]
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
