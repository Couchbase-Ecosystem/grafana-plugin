package plugin

import (
	"strings"
	"testing"
	"time"

	"github.com/grafana/grafana-plugin-sdk-go/backend"
)

func TestExpandTimeMacros(t *testing.T) {
	tr := backend.TimeRange{
		From: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		To:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
	}

	tests := []struct {
		name      string
		query     string
		wantQuery string
		wantField string
		wantErr   string
	}{
		{
			name:      "str_time_range",
			query:     "SELECT ts, v FROM b WHERE str_time_range(ts)",
			wantQuery: "SELECT * FROM (SELECT ts, v FROM b WHERE STR_TO_MILLIS(ts) > STR_TO_MILLIS('2024-01-01T00:00:00Z') AND STR_TO_MILLIS(ts) <= STR_TO_MILLIS('2024-01-02T00:00:00Z')) AS data ORDER by str_to_millis(data.ts) ASC",
			wantField: "ts",
		},
		{
			name:      "time_range",
			query:     "SELECT ts, v FROM b WHERE time_range(ts)",
			wantQuery: "SELECT * FROM (SELECT ts, v FROM b WHERE TO_NUMBER(ts) > STR_TO_MILLIS('2024-01-01T00:00:00Z') AND TO_NUMBER(ts) <= STR_TO_MILLIS('2024-01-02T00:00:00Z')) AS data ORDER by TO_NUMBER(data.ts) ASC",
			wantField: "ts",
		},
		{
			name:      "case insensitive with whitespace",
			query:     "SELECT ts FROM b WHERE TIME_RANGE (ts)",
			wantQuery: "SELECT * FROM (SELECT ts FROM b WHERE TO_NUMBER(ts) > STR_TO_MILLIS('2024-01-01T00:00:00Z') AND TO_NUMBER(ts) <= STR_TO_MILLIS('2024-01-02T00:00:00Z')) AS data ORDER by TO_NUMBER(data.ts) ASC",
			wantField: "ts",
		},
		{
			name:    "no macro",
			query:   "SELECT ts FROM b",
			wantErr: "Failed to detect time field",
		},
		{
			name:    "two str_time_range calls",
			query:   "SELECT * FROM b WHERE str_time_range(a) AND str_time_range(b)",
			wantErr: "Only one call to STR_TIME_RANGE",
		},
		{
			name:    "two time_range calls",
			query:   "SELECT * FROM b WHERE time_range(a) AND time_range(b)",
			wantErr: "Only one call to TIME_RANGE",
		},
		{
			name:    "mixed macros",
			query:   "SELECT * FROM b WHERE str_time_range(a) AND time_range(b)",
			wantErr: "Only one call to TIME_RANGE",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotQuery, gotField, err := expandTimeMacros(tt.query, tr)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotQuery != tt.wantQuery {
				t.Errorf("query =\n  %s\nwant\n  %s", gotQuery, tt.wantQuery)
			}
			if gotField != tt.wantField {
				t.Errorf("field = %q, want %q", gotField, tt.wantField)
			}
		})
	}
}

// query() must surface macro errors before it touches the cluster, so a
// zero-value datasource (no connection) is enough to exercise this path.
func TestQueryReportsMacroErrorWithoutCluster(t *testing.T) {
	ds := &CouchbaseDatasource{}
	resp := ds.query(nil, &QueryRequest{Query: "SELECT ts FROM b"})
	if resp.Error == nil || !strings.Contains(resp.Error.Error(), "Failed to detect time field") {
		t.Fatalf("resp.Error = %v, want time-field detection error", resp.Error)
	}
}
