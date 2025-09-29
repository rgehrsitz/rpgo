# CRUSH.md

## Build/Lint/Test Commands

- **Build**: `go build -o fers-calc ./cmd/rpgo`
- **Run**: `./fers-calc --help` or `go run ./cmd/rpgo --help`
- **Run all tests**: `go test ./...`
- **Run tests with verbose output**: `go test -v ./...`
- **Run tests for specific package**: `go test ./internal/calculation`
- **Run single test**: `go test -run ^TestCalculateFERSPension$ ./internal/calculation`
- **Run tests with coverage**: `go test -cover ./...`
- **Format code**: `go fmt ./...`
- **Vet code**: `go vet ./...`
- **Mod tidy**: `go mod tidy`

## Code Style Guidelines (Go)

This financial calculator follows standard Go conventions with domain-specific patterns:

- **Imports**: Group standard library, external packages, then internal packages with blank line separation
- **Financial Types**: Use `decimal.Decimal` for all monetary calculations, never `float64`
- **Struct Tags**: Include both `yaml` and `json` tags: `yaml:"field_name" json:"field_name"`
- **Naming**: 
  - Variables/Functions: `camelCase` for unexported, `PascalCase` for exported
  - Packages: lowercase single word (e.g., `calculation`, `domain`)
  - Test files: `*_test.go` with `TestFunctionName` pattern
- **Error Handling**: Return errors as last value, check immediately, avoid panics
- **Comments**: Godoc comments for exported functions/types, explain "why" not "what"
- **Testing**: Use testify/assert, golden files for complex output validation
- **Types**: Prefer specific types over `interface{}`, use pointer receivers for mutating methods
- **Time**: Use `time.Time` for dates, be explicit about timezone handling