package models

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// MonthList is a list of "YYYY-MM" strings stored in a Postgres varchar[]
// column. Elements are validated by the handlers before they reach the
// database, so the array literal never needs quoting.
type MonthList []string

// Value renders the Postgres array literal, e.g. {2026-03,2026-04}.
func (m MonthList) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return "{" + strings.Join(m, ",") + "}", nil
}

// Scan parses a Postgres array literal returned by the driver.
func (m *MonthList) Scan(src any) error {
	var s string
	switch v := src.(type) {
	case nil:
		*m = nil
		return nil
	case string:
		s = v
	case []byte:
		s = string(v)
	default:
		return fmt.Errorf("MonthList: unsupported scan type %T", src)
	}

	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "{")
	s = strings.TrimSuffix(s, "}")
	if s == "" {
		*m = MonthList{}
		return nil
	}
	parts := strings.Split(s, ",")
	out := make(MonthList, len(parts))
	for i, p := range parts {
		out[i] = strings.Trim(strings.TrimSpace(p), `"`)
	}
	*m = out
	return nil
}
