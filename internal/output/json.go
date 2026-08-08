// Package output renders command results as tables, JSON, colored text and
// structured error messages. JSON rendering is pure: it never contains logs,
// color escapes or human prose.
package output

import (
	"encoding/json"
	"io"
)

// JSON encodes v as indented JSON to w with a trailing newline. It is pure and
// deterministic for stable, machine-readable output.
func JSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}
