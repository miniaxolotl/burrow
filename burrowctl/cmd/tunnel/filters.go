package tunnel

import (
	"errors"
	"time"

	"burrow/burrowctl/internal/inspection"
	"github.com/spf13/cobra"
)

func addFilters(cmd *cobra.Command) {
	f := cmd.Flags()
	f.StringArray("method", nil, "Exact HTTP method (repeat for OR)")
	f.StringArray("status", nil, "Final code, 2xx–5xx or unknown (repeat for OR)")
	f.String("path", "", "Escaped path-segment prefix (no query)")
	f.String("search", "", "Literal displayed-metadata substring (max 256 bytes)")
	f.String("route", "", "Local matched route only (unsupported for server logs)")
	f.String("min-duration", "", "Inclusive minimum observed duration")
	f.String("sort", "", "Finite ordering: duration or status (unknown last)")
	f.Bool("descending", false, "Descending finite ordering; requires --sort")
}

func readFilters(cmd *cobra.Command, source string) (inspection.Query, error) {
	f := cmd.Flags()
	q := inspection.Query{}
	q.Methods, _ = f.GetStringArray("method")
	q.Statuses, _ = f.GetStringArray("status")
	q.Path, _ = f.GetString("path")
	q.Search, _ = f.GetString("search")
	q.Route, _ = f.GetString("route")
	q.Sort, _ = f.GetString("sort")
	q.Descending, _ = f.GetBool("descending")
	if f.Changed("min-duration") {
		raw, _ := f.GetString("min-duration")
		d, err := time.ParseDuration(raw)
		if err != nil {
			return q, errors.New("minimum duration must be a Go duration")
		}
		q.Minimum = &d
	}
	if follow, _ := f.GetBool("follow"); follow && (q.Sort != "" || q.Descending) {
		return q, errors.New("follow supports append order only; remove --sort/--descending")
	}
	return q, q.Validate(source)
}
