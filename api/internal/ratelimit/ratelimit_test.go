package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

// fakeRow returns a preset count (or error) from Scan.
type fakeRow struct {
	count int
	err   error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) > 0 {
		if p, ok := dest[0].(*int); ok {
			*p = r.count
		}
	}
	return nil
}

// fakeQuerier returns rows from a script, one per call, and records args.
type fakeQuerier struct {
	rows     []fakeRow
	i        int
	lastArgs []any
}

func (q *fakeQuerier) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	q.lastArgs = args
	if q.i >= len(q.rows) {
		// default: keep returning the last
		return q.rows[len(q.rows)-1]
	}
	r := q.rows[q.i]
	q.i++
	return r
}

func fixedNow() time.Time { return time.Date(2026, 6, 1, 12, 30, 0, 0, time.UTC) }

func TestLimiter(t *testing.T) {
	tests := []struct {
		name       string
		max        int
		rows       []fakeRow
		wantOK     bool
		wantRetry  bool // expect retryAfter > 0
	}{
		{
			name:   "under cap allowed",
			max:    5,
			rows:   []fakeRow{{count: 1}},
			wantOK: true,
		},
		{
			name:   "at cap still allowed (count == max)",
			max:    5,
			rows:   []fakeRow{{count: 5}},
			wantOK: true,
		},
		{
			name:      "over cap blocked with retry-after",
			max:       5,
			rows:      []fakeRow{{count: 6}},
			wantOK:    false,
			wantRetry: true,
		},
		{
			name:   "fail-open on db error",
			max:    5,
			rows:   []fakeRow{{err: errors.New("db down")}},
			wantOK: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := &fakeQuerier{rows: tt.rows}
			lim := NewWithQuerier(q, tt.max, time.Hour, "test", fixedNow)
			ok, retry := lim.Allow(context.Background(), "iphash")

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v", ok, tt.wantOK)
			}
			if tt.wantRetry && retry <= 0 {
				t.Errorf("retryAfter = %v, want > 0", retry)
			}
			if !tt.wantRetry && retry != 0 {
				t.Errorf("retryAfter = %v, want 0", retry)
			}
		})
	}
}

// The window is truncated to the configured duration and passed to the query.
func TestLimiter_WindowKeying(t *testing.T) {
	q := &fakeQuerier{rows: []fakeRow{{count: 1}}}
	lim := NewWithQuerier(q, 10, time.Hour, "myslug", fixedNow)
	lim.Allow(context.Background(), "iphash")

	if len(q.lastArgs) != 3 {
		t.Fatalf("expected 3 args (ip, window, slug), got %d", len(q.lastArgs))
	}
	ws, ok := q.lastArgs[1].(time.Time)
	if !ok {
		t.Fatalf("window arg not a time.Time: %T", q.lastArgs[1])
	}
	want := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC) // truncated to the hour
	if !ws.Equal(want) {
		t.Errorf("window_start = %v, want %v", ws, want)
	}
	if q.lastArgs[2] != "myslug" {
		t.Errorf("slug arg = %v, want myslug", q.lastArgs[2])
	}
}

// Successive calls increment (the fake scripts increasing counts; the limiter
// just reports the returned count against the cap).
func TestLimiter_Increment(t *testing.T) {
	q := &fakeQuerier{rows: []fakeRow{{count: 1}, {count: 2}, {count: 3}}}
	lim := NewWithQuerier(q, 2, time.Hour, "test", fixedNow)

	if ok, _ := lim.Allow(context.Background(), "ip"); !ok {
		t.Error("call 1 should be allowed (count 1)")
	}
	if ok, _ := lim.Allow(context.Background(), "ip"); !ok {
		t.Error("call 2 should be allowed (count 2 == max)")
	}
	if ok, _ := lim.Allow(context.Background(), "ip"); ok {
		t.Error("call 3 should be blocked (count 3 > max)")
	}
}
