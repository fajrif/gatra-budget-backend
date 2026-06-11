package models

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Date represents a date without time component
type Date struct {
	time.Time
}

const DateFormat = "2006-01-02"

// UnmarshalJSON implements json.Unmarshaler interface
func (d *Date) UnmarshalJSON(b []byte) error {
	s := string(b)
	// Remove quotes
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}

	if s == "null" || s == "" {
		d.Time = time.Time{}
		return nil
	}

	t, err := time.Parse(DateFormat, s)
	if err != nil {
		return err
	}
	d.Time = t
	return nil
}

// MarshalJSON implements json.Marshaler interface
func (d Date) MarshalJSON() ([]byte, error) {
	if d.Time.IsZero() {
		return []byte("null"), nil
	}
	return []byte(fmt.Sprintf(`"%s"`, d.Time.Format(DateFormat))), nil
}

// Scan implements sql.Scanner interface
func (d *Date) Scan(value interface{}) error {
	if value == nil {
		d.Time = time.Time{}
		return nil
	}

	switch v := value.(type) {
	case time.Time:
		d.Time = v
		return nil
	case string:
		t, err := time.Parse(DateFormat, v)
		if err != nil {
			return err
		}
		d.Time = t
		return nil
	default:
		return fmt.Errorf("cannot scan type %T into Date", value)
	}
}

// Value implements driver.Valuer interface
func (d Date) Value() (driver.Value, error) {
	if d.Time.IsZero() {
		return nil, nil
	}
	// Store as date only in database
	return d.Time.Format(DateFormat), nil
}

// String returns the date in YYYY-MM-DD format
func (d Date) String() string {
	if d.Time.IsZero() {
		return ""
	}
	return d.Time.Format(DateFormat)
}
