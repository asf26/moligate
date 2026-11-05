package taskcommon

import (
	"encoding/json"
	"testing"
)

func TestFlexInt64AcceptsNumberStringAndRFC3339(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int64
	}{
		{"json number", `{"v": 1793854733}`, 1793854733},
		{"numeric string", `{"v": "1793854733"}`, 1793854733},
		{"rfc3339", `{"v": "2026-11-05T04:38:53Z"}`, 1793853533},
		{"empty string", `{"v": ""}`, 0},
		{"null", `{"v": null}`, 0},
		{"absent", `{}`, 0},
		{"garbage", `{"v": "not-a-time"}`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got struct {
				V FlexInt64 `json:"v"`
			}
			if err := json.Unmarshal([]byte(tc.body), &got); err != nil {
				t.Fatalf("unmarshal %s: %v", tc.body, err)
			}
			if got.V.Int64() != tc.want {
				t.Fatalf("%s: got %d, want %d", tc.body, got.V.Int64(), tc.want)
			}
		})
	}
}

// A bad timestamp must never take down the surrounding decode: the task
// status update reads the same struct and only cares about status/progress.
func TestFlexInt64DoesNotBreakSiblingFields(t *testing.T) {
	var got struct {
		Status  string    `json:"status"`
		Created FlexInt64 `json:"created_at"`
	}
	body := `{"status":"completed","created_at":"2026-11-05T04:38:53Z"}`
	if err := json.Unmarshal([]byte(body), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Status != "completed" {
		t.Fatalf("status = %q, want completed", got.Status)
	}
	if got.Created.Int64() == 0 {
		t.Fatal("created_at not parsed")
	}
}
