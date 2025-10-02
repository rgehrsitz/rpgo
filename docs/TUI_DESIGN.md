# FERS Retirement Planner TUI - Design Document

## Vision

Create an **amazing, flexible, and useful** Terminal User Interface (TUI) for retirement planning that makes complex financial analysis feel intuitive and interactive. Inspired by Charm's Crush AI Agent, our TUI will provide a modern, elegant terminal experience that rivals web applications.

## Design Principles

### 1. **Immediate Feedback**

- Real-time recalculation as parameters change
- Live preview of scenario impacts
- Instant validation and error messages

### 2. **Progressive Disclosure**

- Start simple, reveal complexity on demand
- Contextual help always available
- Multi-level detail views (summary → details → deep dive)

### 3. **Elegant & Functional**

- Beautiful styling with Lipgloss
- Smooth animations and transitions
- Keyboard-first design with mouse support
- Responsive layout adapting to terminal size

### 4. **Power User Friendly**

- Keyboard shortcuts for everything
- Command palette for quick actions
- Configurable keybindings
- Scriptable/automatable

## Core Architecture

### Tech Stack

- **Bubble Tea**: Main TUI framework (Elm Architecture)
- **Bubbles**: Pre-built UI components (lists, tables, inputs, spinners)
- **Lipgloss**: Styling and layout
- **Charm Log**: Debugging and logging

### Model-Update-View Pattern

```go
type Model struct {
    // Application state
    config       *domain.Configuration
    currentScene Scene
    scenarios    []domain.GenericScenario
    activeIndex  int

    // UI components
    scenarioList    list.Model
    parameterPanel  ParameterPanel
    resultsView     ResultsView
    chartView       ChartView
    helpView        help.Model

    // State
    loading         bool
    calculating     bool
    error           error
    width, height   int
}

type Scene int
const (
    SceneHome Scene = iota
    SceneScenarios
    SceneParameters
    SceneCompare
    SceneOptimize
    SceneResults
    SceneHelp
)
```go

## UI Layout

### Main Screen Layout

```text
┌─ FERS Retirement Planner ─────────────────────────────────────────────────────┐
│ Home | Scenarios | Parameters | Compare | Optimize | Help              [?][X] │
├───────────────────────────────────────────────────────────────────────────────┤
│                                                                               │
│  ┌─ Current Scenario ──────────────┐  ┌─ Key Metrics ────────────────────┐  │
│  │ Early Retirement 2026           │  │ First Year Income:  $142,567     │  │
│  │ John & Jane Smith               │  │ Lifetime Income:    $4.23M       │  │
│  │ Retire: Jan 2026                │  │ TSP Longevity:      28 years     │  │
│  └─────────────────────────────────┘  │ Lifetime Taxes:     $1.35M       │  │
│                                        └──────────────────────────────────┘  │
│  ┌─ Quick Adjustments ─────────────────────────────────────────────────────┐  │
│  │                                                                          │  │
│  │  TSP Rate:       [=====●====] 3.5%   ← → or type to adjust            │  │
│  │  SS Age (John):  [========●=] 67     ← → or type to adjust            │  │
│  │  Retire Date:    Jan 2026            ⏎ to change                       │  │
│  │                                                                          │  │
│  │  [Apply Changes] [Reset] [Save As...]                                  │  │
│  └──────────────────────────────────────────────────────────────────────────┘  │
│                                                                               │
│  ┌─ Income Projection ──────────────────────────────────────────────────────┐  │
│  │  $200K ┤                                                                 │  │
│  │        ┤     ╭─────────────────────────────                            │  │
│  │  $150K ┤    ╱                                                           │  │
│  │        ┤   ╱                                                            │  │
│  │  $100K ┤  ╱                                                             │  │
│  │        ┼──────────────────────────────────────────────────────────>    │  │
│  │         2026  2030        2040        2050                              │  │
│  └──────────────────────────────────────────────────────────────────────────┘  │
│                                                                               │
├───────────────────────────────────────────────────────────────────────────────┤
│ ↑/↓: Navigate | ←/→: Adjust | ⏎: Edit | Tab: Next Panel | ?: Help | Q: Quit │
└───────────────────────────────────────────────────────────────────────────────┘
```text

## Scenes/Views

### 1. Home Scene

**Purpose**: Dashboard overview of current scenario

**Components**:

- Scenario summary card
- Key metrics display (styled with Lipgloss boxes)
- Quick adjustment sliders (using custom Bubble component)
- Mini chart preview
- Action buttons

**Interactions**:

- Arrow keys to adjust sliders
- Enter to drill into specific parameter
- Tab to cycle through adjustable fields
- Number keys for quick scenario selection

### 2. Scenarios Scene

**Purpose**: Browse, select, and manage scenarios

**Layout**:

```text
┌─ Scenarios ─────────────────────────────────────────────────────────┐
│                                                                     │
│  ┌─ Scenario List ──────┐  ┌─ Preview ─────────────────────────┐  │
│  │                       │  │ Early Retirement 2026             │  │
│  │ ● Base Scenario       │  │                                   │  │
│  │   Early Retire 2026   │  │ Participants: 2                   │  │
│  │   Delayed Retire 2028 │  │ Retirement: Jan 2026              │  │
│  │   Conservative        │  │                                   │  │
│  │   Aggressive          │  │ First Year:  $142,567             │  │
│  │                       │  │ Lifetime:    $4.23M               │  │
│  │ [New] [Duplicate]     │  │ TSP Life:    28 years             │  │
│  │ [Delete] [Import]     │  │                                   │  │
│  └───────────────────────┘  └───────────────────────────────────┘  │
│                                                                     │
│  Quick Actions:                                                    │
│  [C]ompare Selected  [O]ptimize  [E]dit  [R]ename  [D]uplicate    │
└─────────────────────────────────────────────────────────────────────┘
```text

**Components**:

- List component (from Bubbles) for scenarios
- Preview panel with scenario details
- Action buttons
- Search/filter input

**Interactions**:

- j/k or ↑/↓ to navigate list
- Enter to select and edit
- c to compare
- o to optimize
- / to search
- n to create new

### 3. Parameters Scene

**Purpose**: Deep-dive parameter editing for selected scenario

**Layout**:

```text
┌─ Parameters: Early Retirement 2026 ─────────────────────────────────┐
│                                                                     │
│  Participant: [John Smith ▼]                                       │
│                                                                     │
│  ┌─ Retirement ──────────────────────────────────────────────────┐  │
│  │ Date:          [Jan 01, 2026        ] 📅                      │  │
│  │ Age at Retire: 61 years, 3 months                             │  │
│  │ Years Service: 39 years                                       │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ Social Security ─────────────────────────────────────────────┐  │
│  │ Start Age:     [====●=========] 67  (62-70)                   │  │
│  │ Monthly Benefit: $3,200 (at FRA)                              │  │
│  │ At 67:         $3,200/mo                                      │  │
│  │ If delay to 70: $3,968/mo (+24%)                             │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ TSP Withdrawals ─────────────────────────────────────────────┐  │
│  │ Strategy:      [Fixed Percentage ▼]                           │  │
│  │ Rate:          [===●==========] 3.5%  (2%-10%)               │  │
│  │ Annual Amount: $29,750                                        │  │
│  │ Monthly:       $2,479                                         │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  [Apply] [Reset] [Preview Impact]                                  │
└─────────────────────────────────────────────────────────────────────┘
```text

**Components**:

- Dropdown for participant selection (custom Bubble)
- Date picker (custom component)
- Sliders with live feedback
- Input fields with validation
- Dropdown for strategy selection

**Interactions**:

- Tab to move between fields
- Arrow keys for sliders
- Type to edit values directly
- Real-time validation
- Live preview of impacts

### 4. Compare Scene

**Purpose**: Side-by-side comparison of scenarios

**Layout**:
 
```text
┌─ Compare Scenarios ─────────────────────────────────────────────────┐
│                                                                     │
│  Select scenarios to compare:                                      │
│  ☑ Base Scenario          ☑ Early Retire 2026    ☐ Conservative   │
│                                                                     │
│  ┌───────────────────────────────────────────────────────────────┐  │
│  │             Base         Early 2026     Difference            │  │
│  ├───────────────────────────────────────────────────────────────┤  │
│  │ 1st Year    $135,000     $142,567       +$7,567  (+5.6%)     │  │
│  │ Lifetime    $3.98M       $4.23M         +$250K   (+6.3%)     │  │
│  │ TSP Life    30 years     28 years       -2 years             │  │
│  │ Taxes       $1.21M       $1.35M         +$140K               │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ Lifetime Income Comparison ──────────────────────────────────┐  │
│  │  $200K ┤     ╭───── Early 2026                                │  │
│  │        ┤    ╱                                                 │  │
│  │  $150K ┤   ╱  ╭──── Base                                      │  │
│  │        ┤  ╱  ╱                                                │  │
│  │  $100K ┤ ╱  ╱                                                 │  │
│  │        ┼────────────────────────────────────────────────>     │  │
│  │         2026    2030      2040      2050                      │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  💡 Early retirement increases income but reduces TSP longevity    │
│                                                                     │
│  [Export CSV] [Save Report] [Add Template]                         │
└─────────────────────────────────────────────────────────────────────┘
```text

**Components**:

- Checkbox list for scenario selection
- Table component (from Bubbles) for metrics
- Chart comparison view
- Insight/recommendation panel

**Interactions**:

- Space to toggle scenario selection
- Tab through comparison table
- Export to CSV/JSON
- Apply template variations

### 5. Optimize Scene

**Purpose**: Run break-even solver with interactive configuration

**Layout**:

```text
┌─ Optimize Parameters ───────────────────────────────────────────────┐
│                                                                     │
│  Scenario: [Early Retirement 2026 ▼]                               │
│  Participant: [John Smith ▼]                                       │
│                                                                     │
│  ┌─ Optimization Target ────────────────────────────────────────┐  │
│  │  ○ TSP Withdrawal Rate                                        │  │
│  │  ○ Retirement Date                                            │  │
│  │  ○ Social Security Age                                        │  │
│  │  ● All (Multi-dimensional)                                    │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ Optimization Goal ──────────────────────────────────────────┐  │
│  │  ○ Match Income Target:  [$120,000        ]                  │  │
│  │  ● Maximize Lifetime Income                                  │  │
│  │  ○ Maximize TSP Longevity                                    │  │
│  │  ○ Minimize Taxes                                            │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ Constraints ────────────────────────────────────────────────┐  │
│  │  TSP Rate:     2.0% to 10.0%    [Customize]                  │  │
│  │  SS Age:       62 to 70         [Customize]                  │  │
│  │  Retire Date:  -24mo to +36mo   [Customize]                  │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  [Run Optimization]  [Save Configuration]                          │
│                                                                     │
│  Status: ⏳ Running optimization... (12/50 iterations)             │
│          Testing TSP rate 4.5%...                                  │
│          [████████░░░░░░░░░░░░] 24%                               │
└─────────────────────────────────────────────────────────────────────┘
```text

**During optimization**:

```text
┌─ Optimization Results ──────────────────────────────────────────────┐
│                                                                     │
│  ✓ Optimization Complete (47 iterations, 42 seconds)               │
│                                                                     │
│  ┌─ Best Results ──────────────────────────────────────────────┐   │
│  │                                                              │   │
│  │  Best Income:     Optimize retirement_date                  │   │
│  │    → Retire Jan 2028 (24 months later)                     │   │
│  │    → Lifetime income: $4.57M (+$590K)                      │   │
│  │                                                              │   │
│  │  Best Longevity:  Optimize tsp_rate                         │   │
│  │    → TSP rate: 2.8%                                         │   │
│  │    → TSP lasts: 30+ years                                   │   │
│  │                                                              │   │
│  │  Lowest Taxes:    Optimize ss_age                           │   │
│  │    → Claim at 70                                            │   │
│  │    → Save $221K in taxes                                    │   │
│  └──────────────────────────────────────────────────────────────┘   │
│                                                                     │
│  💡 Recommendation: Postponing retirement has biggest impact        │
│                                                                     │
│  [Apply to Scenario] [Compare All] [Export Results]                │
└─────────────────────────────────────────────────────────────────────┘
```text

**Components**:

- Radio buttons for target selection
- Radio buttons for goal selection
- Input fields with validation
- Progress bar during optimization (from Bubbles)
- Spinner component (from Bubbles)
- Results table
- Recommendation panel

**Interactions**:

- j/k to navigate options
- Space to select radio
- Enter to start optimization
- Real-time progress updates
- Apply results directly to scenario

### 6. Results/Charts Scene

**Purpose**: Detailed visualization and analysis

**Layout**:

```text
┌─ Results: Early Retirement 2026 ────────────────────────────────────┐
│                                                                     │
│  View: [Summary] [Income] [TSP Balance] [Taxes] [Cash Flow]        │
│                                                                     │
│  ┌─ Annual Income Projection ───────────────────────────────────┐  │
│  │                                                               │  │
│  │  $200K ┤                                                      │  │
│  │        ┤    ╭─────────────────────────────                   │  │
│  │  $150K ┤   ╱  Net Income                                     │  │
│  │        ┤  ╱                                                   │  │
│  │  $100K ┤ ╱    ╰───── Taxes                                   │  │
│  │        ┼─────────────────────────────────────────────>        │  │
│  │         2026  2030    2040    2050                            │  │
│  │                                                               │  │
│  │  Legend: ─ Net Income  ─ Gross Income  ─ Taxes              │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  ┌─ Key Metrics ────────────────────────────────────────────────┐  │
│  │  First Year:    $142,567    TSP Longevity:  28 years        │  │
│  │  Year 5:        $165,234    Final TSP:      $892,456        │  │
│  │  Year 10:       $178,901    Avg Tax Rate:   28.3%           │  │
│  │  Lifetime:      $4.23M      Total Taxes:    $1.35M          │  │
│  └───────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  [Export Data] [Print] [Share] [Year Details]                      │
└─────────────────────────────────────────────────────────────────────┘
```text

**Components**:

- Tab bar for view selection
- ASCII chart rendering (using custom chart library or manual drawing)
- Table for detailed year-by-year data
- Sparklines for mini trends
- Color-coded metrics

**Interactions**:

- Tab to switch views
- Arrow keys to scrub through years
- Enter to see detailed breakdown
- Export to CSV/JSON

### 7. Help Scene

**Purpose**: Interactive help and keyboard shortcuts

**Layout**:

```text
┌─ Help ──────────────────────────────────────────────────────────────┐
│                                                                     │
│  ┌─ Keyboard Shortcuts ─────────────────────────────────────────┐  │
│  │  Global:                                                      │  │
│  │    ?     Show/hide help                                       │  │
│  │    q     Quit application                                     │  │
│  │    Tab   Next panel                                           │  │
│  │    Esc   Go back / Cancel                                     │  │
│  │                                                                │  │
│  │  Navigation:                                                  │  │
│  │    ↑/k   Move up                                              │  │
│  │    ↓/j   Move down                                            │  │
│  │    ←/h   Previous / Decrease                                  │  │
│  │    →/l   Next / Increase                                      │  │
│  │                                                                │  │
│  │  Scenes:                                                      │  │
│  │    1     Home                                                 │  │
│  │    2     Scenarios                                            │  │
│  │    3     Parameters                                           │  │
│  │    4     Compare                                              │  │
│  │    5     Optimize                                             │  │
│  │    6     Results                                              │  │
│  │                                                                │  │
│  │  Actions:                                                     │  │
│  │    Enter Select / Edit / Apply                                │  │
│  │    Space Toggle selection                                     │  │
│  │    c     Compare scenarios                                    │  │
│  │    o     Optimize parameters                                  │  │
│  │    s     Save changes                                         │  │
│  │    r     Reset to defaults                                    │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                     │
│  [Close Help]                                                       │
└─────────────────────────────────────────────────────────────────────┘
```text

**Components**:

- Help component from Bubbles
- Categorized keyboard shortcuts
- Searchable help content

## Component Design

### Custom Components to Build

#### 1. ParameterSlider

```go
type ParameterSlider struct {
    Label       string
    Value       decimal.Decimal
    Min, Max    decimal.Decimal
    Step        decimal.Decimal
    Unit        string  // "%", "$", "years", etc.
    Focused     bool
    Width       int
    OnChange    func(decimal.Decimal)
}
```go

**Features**:

- Visual slider bar with filled/unfilled sections
- Current value display
- Arrow key adjustment
- Direct input mode (type value)
- Percentage/value formatting
- Live validation

#### 2. MetricCard

```go
type MetricCard struct {
    Title       string
    Value       string
    Trend       string  // "↑", "↓", "→"
    TrendValue  string
    Color       lipgloss.Color
    Width       int
}
```

**Features**:

- Styled box with Lipgloss
- Large value display
- Trend indicator
- Color coding (green=good, red=bad, yellow=neutral)
- Sparkline mini-chart

#### 3. ASCIIChart

```go
type ASCIIChart struct {
    Data        []ChartSeries
    Width       int
    Height      int
    XLabel      string
    YLabel      string
    ShowLegend  bool
}

type ChartSeries struct {
    Name   string
    Data   []decimal.Decimal
    Color  lipgloss.Color
    Style  LineStyle  // solid, dashed, dotted
}
```

**Features**:

- Multi-line charts
- Auto-scaling Y-axis
- Axis labels
- Legend
- Color-coded series
- Grid lines (optional)

#### 4. ScenarioCard

```go
type ScenarioCard struct {
    Scenario    *domain.GenericScenario
    Summary     *domain.ScenarioSummary
    Selected    bool
    Focused     bool
    Width       int
}
```

**Features**:

- Compact scenario overview
- Key metrics preview
- Selection indicator
- Focus highlight

#### 5. ProgressPanel

```go
type ProgressPanel struct {
    Title       string
    Message     string
    Progress    float64  // 0.0 to 1.0
    Spinner     spinner.Model
    Width       int
}
```

**Features**:

- Progress bar
- Status message
- Spinner for indeterminate operations
- Cancellation support

## State Management

### Application State

```go
type AppState struct {
    // Core data
    ConfigPath   string
    Config       *domain.Configuration
    Scenarios    []domain.GenericScenario

    // Calculated results (cached)
    Summaries    map[string]*domain.ScenarioSummary

    // UI state
    CurrentScene Scene
    PreviousScene Scene
    ActiveScenario int

    // Calculation state
    CalcEngine   *calculation.CalculationEngine
    Calculating  map[string]bool  // scenario name -> is calculating

    // Compare state
    CompareSelection []int
    CompareResults   *compare.ComparisonSet

    // Optimize state
    OptimizeTarget   breakeven.OptimizationTarget
    OptimizeGoal     breakeven.OptimizationGoal
    OptimizeProgress float64
    OptimizeResult   *breakeven.OptimizationResult
}
```

### Message Types

```go
// Navigation messages
type SceneChangeMsg Scene
type GoBackMsg struct{}

// Calculation messages
type StartCalculationMsg string  // scenario name
type CalculationCompleteMsg struct {
    ScenarioName string
    Summary      *domain.ScenarioSummary
    Error        error
}

// Parameter update messages
type ParameterChangedMsg struct {
    Scenario    string
    Participant string
    Parameter   string
    Value       interface{}
}

// Optimization messages
type StartOptimizationMsg breakeven.OptimizationRequest
type OptimizationProgressMsg float64
type OptimizationCompleteMsg *breakeven.OptimizationResult

// Comparison messages
type StartCompareMsg []string
type CompareCompleteMsg *compare.ComparisonSet

// UI messages
type WindowSizeMsg struct{ Width, Height int }
type ErrorMsg error
```

## Styling Guide

### Color Palette (using Lipgloss)

```go
var (
    // Brand colors
    ColorPrimary   = lipgloss.Color("#7D56F4")  // Purple
    ColorSecondary = lipgloss.Color("#00D9FF")  // Cyan
    ColorAccent    = lipgloss.Color("#FF6B9D")  // Pink

    // Semantic colors
    ColorSuccess   = lipgloss.Color("#00C853")  // Green
    ColorWarning   = lipgloss.Color("#FFD600")  // Yellow
    ColorDanger    = lipgloss.Color("#FF1744")  // Red
    ColorInfo      = lipgloss.Color("#2979FF")  // Blue

    // UI colors
    ColorText      = lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#FAFAFA"}
    ColorTextDim   = lipgloss.AdaptiveColor{Light: "#666666", Dark: "#999999"}
    ColorBorder    = lipgloss.AdaptiveColor{Light: "#DDDDDD", Dark: "#444444"}
    ColorBackground = lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#1A1A1A"}
    ColorHighlight = lipgloss.AdaptiveColor{Light: "#F0F0F0", Dark: "#2A2A2A"}
)
```

### Style Definitions

```go
var (
    // Title bar
    StyleTitleBar = lipgloss.NewStyle().
        Background(ColorPrimary).
        Foreground(lipgloss.Color("#FFFFFF")).
        Bold(true).
        Padding(0, 1)

    // Panels/Cards
    StylePanel = lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(ColorBorder).
        Padding(1, 2)

    StylePanelFocused = StylePanel.Copy().
        BorderForeground(ColorPrimary).
        BorderStyle(lipgloss.ThickBorder())

    // Metrics
    StyleMetricValue = lipgloss.NewStyle().
        Bold(true).
        Foreground(ColorPrimary).
        MarginRight(1)

    StyleMetricPositive = StyleMetricValue.Copy().
        Foreground(ColorSuccess)

    StyleMetricNegative = StyleMetricValue.Copy().
        Foreground(ColorDanger)

    // Status bar
    StyleStatusBar = lipgloss.NewStyle().
        Foreground(ColorTextDim).
        Background(ColorHighlight).
        Padding(0, 1)

    // Help text
    StyleHelpKey = lipgloss.NewStyle().
        Foreground(ColorSecondary).
        Bold(true)

    StyleHelpText = lipgloss.NewStyle().
        Foreground(ColorTextDim)
)
```

## Performance Optimizations

### 1. Async Calculations

```go
func (m Model) calculateScenarioAsync(scenarioName string) tea.Cmd {
    return func() tea.Msg {
        // Run in goroutine, return message when done
        summary, err := m.calcEngine.RunScenario(...)
        return CalculationCompleteMsg{scenarioName, summary, err}
    }
}
```

### 2. Debounced Updates

For slider adjustments, debounce recalculations:

```go
type DebouncedParameter struct {
    value    decimal.Decimal
    timer    *time.Timer
    duration time.Duration
}
```

### 3. Result Caching

Cache scenario summaries to avoid recalculation:

```go
type ResultCache struct {
    summaries map[string]*CachedResult
    mu        sync.RWMutex
}

type CachedResult struct {
    summary    *domain.ScenarioSummary
    calculated time.Time
    hash       string  // hash of scenario parameters
}
```

### 4. Viewport Rendering

Only render visible portions of large lists/tables:

```go
type VirtualList struct {
    items       []Item
    viewport    viewport.Model
    visibleStart int
    visibleEnd   int
}
```

## Testing Strategy

### 1. Unit Tests

- Test each custom component independently
- Mock tea.Model interface
- Test Update() logic with various messages
- Test View() rendering

### 2. Integration Tests with teatest

```go
func TestScenarioSelection(t *testing.T) {
    tm := teatest.NewTestModel(t, initialModel)

    // Simulate key presses
    tm.Send(tea.KeyMsg{Type: tea.KeyDown})
    tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

    // Assert state changes
    finalModel := tm.FinalModel().(Model)
    assert.Equal(t, 1, finalModel.ActiveScenario)
}
```

### 3. Visual Regression Testing with VHS

Record terminal sessions for:

- Onboarding flow
- Parameter adjustment
- Optimization workflow
- Comparison flow

## Development Workflow

### 1. Live Reload Setup

```bash
# Watch for changes and rebuild
watchexec -e go -r -- go run ./cmd/rpgo-tui
```

### 2. Debug Logging

```go
import "github.com/charmbracelet/log"

func init() {
    f, _ := tea.LogToFile("debug.log", "debug")
    defer f.Close()
}

// In Update() or View()
log.Debug("Received message", "type", msg)
```

### 3. Message Dumping

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Dump all messages for debugging
    spew.Fdump(debugFile, msg)

    // Normal update logic
    ...
}
```

## Progressive Implementation Plan

### Phase 1: Foundation (Week 1)

1. Set up Bubble Tea project structure
2. Implement basic Model-Update-View
3. Create Home scene with static data
4. Add scene navigation
5. Implement basic styling

### Phase 2: Core Components (Week 1-2)

1. Build ParameterSlider component
2. Build MetricCard component
3. Build ScenarioCard component
4. Implement Scenarios scene
5. Add config loading

### Phase 3: Interactivity (Week 2)

1. Make parameters adjustable
2. Add async calculation
3. Implement real-time preview
4. Add debouncing
5. Implement result caching

### Phase 4: Advanced Features (Week 2-3)

1. Build Compare scene
2. Build Optimize scene with progress
3. Implement ASCIIChart component
4. Build Results/Charts scene
5. Add data export

### Phase 5: Polish (Week 3)

1. Implement Help scene
2. Add keyboard shortcuts
3. Improve responsive layout
4. Add animations/transitions
5. Performance tuning

### Phase 6: Testing & Docs (Week 3)

1. Write unit tests
2. Add integration tests
3. Create VHS demos
4. Write user documentation
5. Add inline help

## Future Enhancements

### Phase 2+ Features

- **Themes**: Multiple color schemes (dark, light, high-contrast)
- **Plugins**: Extensible component system
- **Templates**: Save and share parameter presets
- **Export**: Generate PDF reports
- **Collaboration**: Share scenarios via URL/file
- **AI Assistant**: Natural language parameter adjustment
- **What-if Analysis**: Quick scenario variations
- **Goal Seeking**: "I need $150K/year, what should I do?"
- **Risk Analysis**: Monte Carlo integration in TUI
- **Alerts**: Notify on important thresholds (IRMAA, RMD)

## Success Metrics

### User Experience

- ✅ Can adjust parameters and see results in <1 second
- ✅ Can complete comparison workflow in <30 seconds
- ✅ Can run optimization without reading docs
- ✅ Terminal remains responsive even during calculations
- ✅ Works in terminals of varying sizes (80x24 minimum)

### Technical

- ✅ No blocking operations in Update() or View()
- ✅ <100ms render time for any view
- ✅ Memory usage <50MB for typical configs
- ✅ All tests passing
- ✅ Works on Linux, macOS, Windows

### Delight Factors

- ✅ Smooth animations and transitions
- ✅ Helpful inline tips and recommendations
- ✅ Beautiful, polished visual design
- ✅ Keyboard shortcuts for power users
- ✅ "Wow" factor when first launched

## Conclusion

This TUI will transform the FERS retirement planner from a powerful but complex CLI tool into an **intuitive, beautiful, and productive** terminal application. By leveraging the Charm ecosystem (Bubble Tea, Bubbles, Lipgloss), we can create an experience that rivals modern web applications while maintaining the speed and efficiency of the terminal.

The key is to start simple, iterate quickly, and progressively add features while maintaining the core principles of immediate feedback, progressive disclosure, and elegant functionality.
