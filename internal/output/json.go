package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/tidylogic/db-diff/internal/diff"
)

// WriteJSON marshals the DiffResult to pretty-printed JSON and writes to w.
func WriteJSON(w io.Writer, result *diff.DiffResult) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		return fmt.Errorf("json output: %w", err)
	}
	return nil
}
