package output

import (
	"bytes"
	_ "embed"
	"html/template"

	"github.com/rpgo/retirement-calculator/internal/domain"
)

// HTMLFormatter produces an HTML report (current implementation ports legacy static HTML).
type HTMLFormatter struct{}

func (h HTMLFormatter) Name() string { return "html" }

//go:embed templates/report.html.tmpl
var htmlTemplateSource string

var htmlTemplate = template.Must(template.New("report").Funcs(template.FuncMap{
	"curr": FormatCurrency,
	"pct":  FormatPercentage,
}).Parse(htmlTemplateSource))

func (h HTMLFormatter) Format(results *domain.ScenarioComparison) ([]byte, error) {
	var buf bytes.Buffer
	rec := AnalyzeScenarios(results)
	data := struct {
		*domain.ScenarioComparison
		Recommendation Recommendation
		Assumptions    []string
	}{results, rec, DefaultAssumptions}
	if err := htmlTemplate.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
