package taskcommon

import (
	"bytes"
	"strconv"
	"time"
)

// FlexInt64 accepts a JSON number, a numeric string, or an RFC3339 timestamp.
//
// Upstream video providers are inconsistent about timestamp fields: CTMOAI
// returns created_at as a quoted string, which a plain int64 field rejects.
// Because the whole response is unmarshalled in one go, that single mismatch
// took down every task status update - the row stayed at queued forever and
// the client only ever saw "unknown".
//
// A value that cannot be understood is reported as 0 rather than an error:
// these fields are informational and must never block a status update.
type FlexInt64 int64

func (f *FlexInt64) UnmarshalJSON(b []byte) error {
	s := string(bytes.Trim(bytes.TrimSpace(b), `"`))
	if s == "" || s == "null" {
		*f = 0
		return nil
	}
	if v, err := strconv.ParseInt(s, 10, 64); err == nil {
		*f = FlexInt64(v)
		return nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		*f = FlexInt64(t.Unix())
		return nil
	}
	*f = 0
	return nil
}

// Int64 returns the underlying value for assignment to a plain int64 field.
func (f FlexInt64) Int64() int64 {
	return int64(f)
}
