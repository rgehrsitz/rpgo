package transform

import (
	"fmt"

	"github.com/rgehrsitz/rpgo/internal/domain"
)

// ScenarioTransform defines the interface for all scenario transformations.
// Transforms are composable operations that modify scenarios in predictable ways,
// enabling features like scenario comparison, break-even analysis, and interactive UX.
type ScenarioTransform interface {
	// Apply transforms a base scenario and returns a new modified scenario.
	// Returns an error if the transformation cannot be applied (e.g., invalid parameters).
	Apply(base *domain.GenericScenario) (*domain.GenericScenario, error)

	// Name returns a short identifier for this transform (e.g., "postpone_retirement").
	Name() string

	// Description returns a human-readable description of what this transform does.
	Description() string

	// Validate checks if the transform parameters are valid without applying it.
	// Returns an error if parameters are invalid (e.g., negative months, participant doesn't exist).
	Validate(base *domain.GenericScenario) error
}

// ApplyTransforms applies a sequence of transforms to a base scenario.
// Transforms are applied in order, with each transform receiving the output of the previous one.
// Returns an error if any transform fails to apply.
func ApplyTransforms(base *domain.GenericScenario, transforms []ScenarioTransform) (*domain.GenericScenario, error) {
	if base == nil {
		return nil, fmt.Errorf("base scenario cannot be nil")
	}

	if len(transforms) == 0 {
		// No transforms to apply, return a deep copy of the base
		return base.DeepCopy(), nil
	}

	// Start with the base scenario
	current := base

	// Apply each transform in sequence
	for i, transform := range transforms {
		if transform == nil {
			return nil, fmt.Errorf("transform at index %d is nil", i)
		}

		// Validate before applying
		if err := transform.Validate(current); err != nil {
			return nil, fmt.Errorf("transform %s validation failed: %w", transform.Name(), err)
		}

		// Apply the transform
		next, err := transform.Apply(current)
		if err != nil {
			return nil, fmt.Errorf("transform %s failed: %w", transform.Name(), err)
		}

		current = next
	}

	return current, nil
}

// TransformError represents an error that occurred during transformation.
type TransformError struct {
	TransformName string
	Operation     string
	Reason        string
	Err           error
}

func (e *TransformError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("transform %s (%s): %s: %v", e.TransformName, e.Operation, e.Reason, e.Err)
	}
	return fmt.Sprintf("transform %s (%s): %s", e.TransformName, e.Operation, e.Reason)
}

func (e *TransformError) Unwrap() error {
	return e.Err
}

// NewTransformError creates a new TransformError.
func NewTransformError(transformName, operation, reason string, err error) error {
	return &TransformError{
		TransformName: transformName,
		Operation:     operation,
		Reason:        reason,
		Err:           err,
	}
}
