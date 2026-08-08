package output

import (
	"fmt"
	"io"
)

// ErrorReport describes a failure using the "what / why / next" structure.
type ErrorReport struct {
	// What happened, phrased for the user.
	What string
	// Why it happened (root cause), optional.
	Why string
	// Next is a suggested command or action to resolve it, optional.
	Next string
}

// Fprint writes the error report to w. Color is applied when the colorizer is
// enabled. Only What is required.
func (r ErrorReport) Fprint(w io.Writer, c Colorizer) {
	fmt.Fprintln(w, c.Red("Error: ")+r.What)
	if r.Why != "" {
		fmt.Fprintln(w, "  Why:  "+r.Why)
	}
	if r.Next != "" {
		fmt.Fprintln(w, "  Next: "+c.Cyan(r.Next))
	}
}
