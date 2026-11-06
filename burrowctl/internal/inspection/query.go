package inspection

import (
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"burrow/protocol"
	"github.com/charmbracelet/x/ansi"
)

type Query struct {
	Methods, Statuses         []string
	Path, Search, Route, Sort string
	Minimum                   *time.Duration
	Descending                bool
}

func (q Query) Validate(source string) error {
	if len(q.Methods) > 16 || len(q.Statuses) > 16 || len(q.Path) > 256 || len(q.Search) > 256 || len(q.Route) > 256 {
		return errors.New("filters exceed bounded input limits")
	}
	for _, method := range q.Methods {
		if method == "" || len(method) > 256 {
			return errors.New("method must be an exact HTTP token")
		}
		for _, c := range method {
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || strings.ContainsRune("!#$%&'*+-.^_`|~", c)) {
				return errors.New("method must be an exact HTTP token")
			}
		}
	}
	for _, status := range q.Statuses {
		if status == "unknown" {
			continue
		}
		if len(status) == 3 && status[0] >= '2' && status[0] <= '5' && status[1:] == "xx" {
			continue
		}
		n, err := strconv.Atoi(status)
		if err != nil || n != 101 && (n < 200 || n > 599) {
			return errors.New("status must be 101, 200–599, 2xx–5xx or unknown")
		}
	}
	if q.Path != "" && (!strings.HasPrefix(q.Path, "/") || strings.ContainsAny(q.Path, "?#\r\n\x1b")) {
		return errors.New("path must be an escaped path prefix without query")
	}
	if q.Minimum != nil && *q.Minimum < 0 {
		return errors.New("minimum duration must be nonnegative")
	}
	if q.Sort != "" && q.Sort != "duration" && q.Sort != "status" {
		return errors.New("sort must be duration or status")
	}
	if q.Descending && q.Sort == "" {
		return errors.New("--descending requires --sort")
	}
	if source == "server" && q.Route != "" {
		return errors.New("server logs have no route; use local inspection or requests list --route")
	}
	if source == "capture" && (len(q.Statuses) > 0 || q.Minimum != nil || q.Sort != "") {
		return errors.New("capture summaries have no original response status or duration")
	}
	if source == "capture" && q.Path != "" {
		return errors.New("capture summaries have no request path; use --route")
	}
	return nil
}

func duration(row *protocol.TunnelLog) (time.Duration, bool) {
	if row.DurationNS != nil {
		return time.Duration(*row.DurationNS), *row.DurationNS >= 0
	}
	d, err := time.ParseDuration(row.Duration)
	return d, err == nil && d >= 0
}
func knownStatus(row *protocol.TunnelLog) bool {
	return row.StatusCode == 101 || row.StatusCode >= 200 && row.StatusCode <= 599
}
func Prefix(path, prefix string) bool {
	return prefix == "" || prefix == "/" || path == prefix || strings.HasPrefix(path, strings.TrimSuffix(prefix, "/")+"/")
}

func (q Query) Match(row *protocol.TunnelLog) bool {
	if len(q.Methods) > 0 {
		matched := false
		for _, v := range q.Methods {
			matched = matched || row.Method == v
		}
		if !matched {
			return false
		}
	}
	if len(q.Statuses) > 0 {
		matched := false
		for _, v := range q.Statuses {
			if v == "unknown" {
				matched = matched || !knownStatus(row)
			} else if knownStatus(row) {
				matched = matched || v == strconv.Itoa(row.StatusCode) || len(v) == 3 && v[1:] == "xx" && int(v[0]-'0') == row.StatusCode/100
			}
		}
		if !matched {
			return false
		}
	}
	if q.Minimum != nil {
		d, known := duration(row)
		if !known || d < *q.Minimum {
			return false
		}
	}
	if !Prefix(row.Path, q.Path) || q.Route != "" && row.Route != q.Route {
		return false
	}
	if q.Search != "" && !strings.Contains(strings.ToLower(Safe(row.Method+" "+row.Path+" "+row.Route+" "+row.Reason)), strings.ToLower(q.Search)) {
		return false
	}
	return true
}

func (q Query) Apply(rows []*protocol.TunnelLog) []*protocol.TunnelLog {
	result := make([]*protocol.TunnelLog, 0, len(rows))
	for _, row := range rows {
		if row != nil && q.Match(row) {
			result = append(result, row)
		}
	}
	if q.Sort != "" {
		sort.SliceStable(result, func(i, j int) bool {
			a, b := result[i], result[j]
			av, bv := int64(a.StatusCode), int64(b.StatusCode)
			ak, bk := knownStatus(a), knownStatus(b)
			if q.Sort == "duration" {
				ad, knownA := duration(a)
				bd, knownB := duration(b)
				av, bv = int64(ad), int64(bd)
				ak, bk = knownA, knownB
			}
			if ak != bk {
				return ak
			}
			if !ak || av == bv {
				return a.ID < b.ID
			}
			if q.Descending {
				return av > bv
			}
			return av < bv
		})
	}
	return result
}

func Safe(value string) string {
	return strings.Map(func(c rune) rune {
		if !unicode.IsPrint(c) {
			return -1
		}
		return c
	}, ansi.Strip(value))
}
