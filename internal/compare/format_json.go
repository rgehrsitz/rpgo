package compare

import (
	"encoding/json"
)

// JSONFormatter formats comparison results as JSON
type JSONFormatter struct {
	Pretty bool // If true, format with indentation
}

// Format generates JSON output for comparison results
func (jf *JSONFormatter) Format(compSet *ComparisonSet) (string, error) {
	var data []byte
	var err error

	if jf.Pretty {
		data, err = json.MarshalIndent(compSet, "", "  ")
	} else {
		data, err = json.Marshal(compSet)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}
