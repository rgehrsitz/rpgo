package output

// DefaultAssumptions lists key modeling assumptions rendered in detailed outputs.
// Future: could be loaded from configuration or generated dynamically.
var DefaultAssumptions = []string{
	"General COLA (FERS pension & SS): 2.5% annually",
	"FEHB premium inflation: 4.0% annually",
	"TSP growth pre-retirement: 7.0% annually",
	"TSP growth post-retirement: 5.0% annually",
	"Social Security wage base indexing: ~5% annually (2025 est: $168,600)",
	"Tax brackets: 2025 levels held constant (no inflation indexing)",
}
