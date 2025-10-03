package transform

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rgehrsitz/rpgo/internal/domain"
	"github.com/shopspring/decimal"
)

// TransformRegistry provides a central registry for all available transforms.
// It enables creation of transforms from string parameters, useful for CLI commands.
type TransformRegistry struct {
	factories map[string]TransformFactory
}

// TransformFactory is a function that creates a transform from parameters.
type TransformFactory func(params map[string]string) (ScenarioTransform, error)

// NewTransformRegistry creates a new registry with all built-in transforms registered.
func NewTransformRegistry() *TransformRegistry {
	registry := &TransformRegistry{
		factories: make(map[string]TransformFactory),
	}

	// Register all built-in transforms
	registry.Register("postpone_retirement", createPostponeRetirement)
	registry.Register("set_retirement_date", createSetRetirementDate)
	registry.Register("delay_ss", createDelaySSClaim)
	registry.Register("modify_tsp_strategy", createModifyTSPStrategy)
	registry.Register("adjust_tsp_rate", createAdjustTSPRate)
	registry.Register("set_tsp_target", createSetTSPTargetIncome)
	registry.Register("set_mortality", createSetMortalityDate)
	registry.Register("set_survivor_spending", createSetSurvivorSpendingFactor)
	registry.Register("set_tsp_transfer", createSetTSPTransferMode)

	// Roth conversion transforms
	registry.Register("enable_roth_conversion", createEnableRothConversion)
	registry.Register("modify_roth_conversion", createModifyRothConversion)
	registry.Register("remove_roth_conversion", createRemoveRothConversion)

	return registry
}

// Register adds a transform factory to the registry.
func (r *TransformRegistry) Register(name string, factory TransformFactory) {
	r.factories[name] = factory
}

// Create creates a transform by name with the given parameters.
func (r *TransformRegistry) Create(name string, params map[string]string) (ScenarioTransform, error) {
	factory, exists := r.factories[name]
	if !exists {
		return nil, fmt.Errorf("unknown transform: %s", name)
	}

	return factory(params)
}

// List returns the names of all registered transforms.
func (r *TransformRegistry) List() []string {
	names := make([]string, 0, len(r.factories))
	for name := range r.factories {
		names = append(names, name)
	}
	return names
}

// ParseTransformSpec parses a transform specification string.
// Format: "transform_name:param1=value1,param2=value2"
// Example: "postpone_retirement:participant=Alice,months=12"
func (r *TransformRegistry) ParseTransformSpec(spec string) (ScenarioTransform, error) {
	parts := strings.SplitN(spec, ":", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid transform spec format, expected 'name:params', got: %s", spec)
	}

	name := strings.TrimSpace(parts[0])
	paramsStr := strings.TrimSpace(parts[1])

	// Parse parameters
	params := make(map[string]string)
	if paramsStr != "" {
		for _, paramPair := range strings.Split(paramsStr, ",") {
			kv := strings.SplitN(paramPair, "=", 2)
			if len(kv) != 2 {
				return nil, fmt.Errorf("invalid parameter format, expected 'key=value', got: %s", paramPair)
			}
			params[strings.TrimSpace(kv[0])] = strings.TrimSpace(kv[1])
		}
	}

	return r.Create(name, params)
}

// Factory functions for each transform

func createPostponeRetirement(params map[string]string) (ScenarioTransform, error) {
	participant, ok := params["participant"]
	if !ok {
		return nil, fmt.Errorf("postpone_retirement requires 'participant' parameter")
	}

	monthsStr, ok := params["months"]
	if !ok {
		return nil, fmt.Errorf("postpone_retirement requires 'months' parameter")
	}

	months, err := strconv.Atoi(monthsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid months value: %w", err)
	}

	return &PostponeRetirement{
		Participant: participant,
		Months:      months,
	}, nil
}

func createSetRetirementDate(params map[string]string) (ScenarioTransform, error) {
	participant, ok := params["participant"]
	if !ok {
		return nil, fmt.Errorf("set_retirement_date requires 'participant' parameter")
	}

	dateStr, ok := params["date"]
	if !ok {
		return nil, fmt.Errorf("set_retirement_date requires 'date' parameter")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	return &SetRetirementDate{
		Participant: participant,
		Date:        date,
	}, nil
}

func createDelaySSClaim(params map[string]string) (ScenarioTransform, error) {
	participant, ok := params["participant"]
	if !ok {
		return nil, fmt.Errorf("delay_ss requires 'participant' parameter")
	}

	ageStr, ok := params["age"]
	if !ok {
		return nil, fmt.Errorf("delay_ss requires 'age' parameter")
	}

	age, err := strconv.Atoi(ageStr)
	if err != nil {
		return nil, fmt.Errorf("invalid age value: %w", err)
	}

	return &DelaySSClaim{
		Participant: participant,
		NewAge:      age,
	}, nil
}

func createModifyTSPStrategy(params map[string]string) (ScenarioTransform, error) {
	participant, ok := params["participant"]
	if !ok {
		return nil, fmt.Errorf("modify_tsp_strategy requires 'participant' parameter")
	}

	strategy, ok := params["strategy"]
	if !ok {
		return nil, fmt.Errorf("modify_tsp_strategy requires 'strategy' parameter")
	}

	preserveRate := false
	if preserveStr, ok := params["preserve_rate"]; ok {
		preserveRate = preserveStr == "true" || preserveStr == "yes" || preserveStr == "1"
	}

	return &ModifyTSPStrategy{
		Participant:  participant,
		NewStrategy:  strategy,
		PreserveRate: preserveRate,
	}, nil
}

func createAdjustTSPRate(params map[string]string) (ScenarioTransform, error) {
	participant, ok := params["participant"]
	if !ok {
		return nil, fmt.Errorf("adjust_tsp_rate requires 'participant' parameter")
	}

	rateStr, ok := params["rate"]
	if !ok {
		return nil, fmt.Errorf("adjust_tsp_rate requires 'rate' parameter")
	}

	rate, err := decimal.NewFromString(rateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid rate value: %w", err)
	}

	return &AdjustTSPRate{
		Participant: participant,
		NewRate:     rate,
	}, nil
}

func createSetTSPTargetIncome(params map[string]string) (ScenarioTransform, error) {
	participant, ok := params["participant"]
	if !ok {
		return nil, fmt.Errorf("set_tsp_target requires 'participant' parameter")
	}

	targetStr, ok := params["target"]
	if !ok {
		return nil, fmt.Errorf("set_tsp_target requires 'target' parameter")
	}

	target, err := decimal.NewFromString(targetStr)
	if err != nil {
		return nil, fmt.Errorf("invalid target value: %w", err)
	}

	return &SetTSPTargetIncome{
		Participant:   participant,
		MonthlyTarget: target,
	}, nil
}

func createSetMortalityDate(params map[string]string) (ScenarioTransform, error) {
	participant, ok := params["participant"]
	if !ok {
		return nil, fmt.Errorf("set_mortality requires 'participant' parameter")
	}

	dateStr, ok := params["date"]
	if !ok {
		return nil, fmt.Errorf("set_mortality requires 'date' parameter")
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return nil, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}

	return &SetMortalityDate{
		Participant: participant,
		DeathDate:   date,
	}, nil
}

func createSetSurvivorSpendingFactor(params map[string]string) (ScenarioTransform, error) {
	factorStr, ok := params["factor"]
	if !ok {
		return nil, fmt.Errorf("set_survivor_spending requires 'factor' parameter")
	}

	factor, err := decimal.NewFromString(factorStr)
	if err != nil {
		return nil, fmt.Errorf("invalid factor value: %w", err)
	}

	return &SetSurvivorSpendingFactor{
		Factor: factor,
	}, nil
}

func createSetTSPTransferMode(params map[string]string) (ScenarioTransform, error) {
	mode, ok := params["mode"]
	if !ok {
		return nil, fmt.Errorf("set_tsp_transfer requires 'mode' parameter")
	}

	return &SetTSPTransferMode{
		Mode: mode,
	}, nil
}

// Roth conversion transform factories

func createEnableRothConversion(params map[string]string) (ScenarioTransform, error) {
	participant, ok := params["participant"]
	if !ok {
		return nil, fmt.Errorf("enable_roth_conversion requires 'participant' parameter")
	}

	// Parse conversions from parameters
	// Format: "year1:amount1,year2:amount2"
	conversionsStr, ok := params["conversions"]
	if !ok {
		return nil, fmt.Errorf("enable_roth_conversion requires 'conversions' parameter")
	}

	var conversions []domain.RothConversion
	if conversionsStr != "" {
		for _, conversionPair := range strings.Split(conversionsStr, ",") {
			parts := strings.SplitN(conversionPair, ":", 2)
			if len(parts) != 2 {
				return nil, fmt.Errorf("invalid conversion format, expected 'year:amount', got: %s", conversionPair)
			}

			year, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, fmt.Errorf("invalid year value: %w", err)
			}

			amount, err := decimal.NewFromString(strings.TrimSpace(parts[1]))
			if err != nil {
				return nil, fmt.Errorf("invalid amount value: %w", err)
			}

			conversions = append(conversions, domain.RothConversion{
				Year:   year,
				Amount: amount,
				Source: "traditional_tsp",
			})
		}
	}

	return &EnableRothConversion{
		Participant: participant,
		Conversions: conversions,
	}, nil
}

func createModifyRothConversion(params map[string]string) (ScenarioTransform, error) {
	participant, ok := params["participant"]
	if !ok {
		return nil, fmt.Errorf("modify_roth_conversion requires 'participant' parameter")
	}

	yearStr, ok := params["year"]
	if !ok {
		return nil, fmt.Errorf("modify_roth_conversion requires 'year' parameter")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return nil, fmt.Errorf("invalid year value: %w", err)
	}

	amountStr, ok := params["amount"]
	if !ok {
		return nil, fmt.Errorf("modify_roth_conversion requires 'amount' parameter")
	}

	amount, err := decimal.NewFromString(amountStr)
	if err != nil {
		return nil, fmt.Errorf("invalid amount value: %w", err)
	}

	return &ModifyRothConversion{
		Participant: participant,
		Year:        year,
		NewAmount:   amount,
	}, nil
}

func createRemoveRothConversion(params map[string]string) (ScenarioTransform, error) {
	participant, ok := params["participant"]
	if !ok {
		return nil, fmt.Errorf("remove_roth_conversion requires 'participant' parameter")
	}

	yearStr, ok := params["year"]
	if !ok {
		return nil, fmt.Errorf("remove_roth_conversion requires 'year' parameter")
	}

	year, err := strconv.Atoi(yearStr)
	if err != nil {
		return nil, fmt.Errorf("invalid year value: %w", err)
	}

	return &RemoveRothConversion{
		Participant: participant,
		Year:        year,
	}, nil
}
