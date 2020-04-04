package cmd

import (
	"fmt"
	"testing"
	"time"
)

func mustParseTime(t *testing.T, s string) (c time.Time) {
	var err error
	if c, err = time.Parse(time.RFC3339, s); err != nil {
		t.Fatal(err)
	}
	return
}

func Test_fromRawRangeToTime(t *testing.T) {
	input := []struct {
		now     time.Time
		name    string
		fromRaw string
		toRaw   string
		from    time.Time
		to      time.Time
		err     error
	}{
		{
			now:     mustParseTime(t, "2002-10-02T15:00:00Z"),
			to:      mustParseTime(t, "2002-10-02T13:00:00Z"),
			from:    mustParseTime(t, "2002-10-02T10:00:00Z"),
			fromRaw: "-5h",
			toRaw:   "-2h",
			err:     nil,
		},
		{
			now:     mustParseTime(t, "2002-10-02T15:00:00Z"),
			toRaw:   "-5h",
			fromRaw: "-2h",
			err:     ErrFromAheadOfTo,
		},
		{
			now:     mustParseTime(t, "2002-10-02T15:00:00Z"),
			toRaw:   "2002-10-01T03:00:00Z",
			fromRaw: "2002-10-02T02:00:00Z",
			err:     ErrFromAheadOfTo,
		},
		{
			now:     mustParseTime(t, "2002-10-02T15:00:00Z"),
			to:      mustParseTime(t, "2002-10-02T03:00:00Z"),
			from:    mustParseTime(t, "2002-10-02T02:00:00Z"),
			toRaw:   "2002-10-02T03:00:00Z",
			fromRaw: "2002-10-02T02:00:00Z",
			err:     nil,
		},
	}

	for ii, v := range input {
		t.Run(fmt.Sprintf("[%d] FROM %s, TO: %s", ii, v.from.Format(time.RFC3339), v.to.Format(time.RFC3339)), func(t *testing.T) {
			from, to, err := fromRawRangeToTime(v.now, v.fromRaw, v.toRaw)
			if err != v.err {
				t.Fatalf("Expected err: %s got %s", v.err, err)
			}

			if from.String() != v.from.String() {
				t.Errorf("Expected FROM %s got %s", v.from.Format(time.RFC3339), from.Format(time.RFC3339))
			}

			if to.String() != v.to.String() {
				t.Errorf("Expected TO: %s got %s", v.to.Format(time.RFC3339), to.Format(time.RFC3339))
			}
		})
	}
}

func Test_fromTimeToString(t *testing.T) {
	input := []struct {
		now     time.Time
		name    string
		fromRaw string
		toRaw   string
		from    time.Time
		to      time.Time
		err     error
	}{
		{
			now:     mustParseTime(t, "2002-10-02T15:00:00Z"),
			to:      mustParseTime(t, "2002-10-02T13:00:00Z"),
			from:    mustParseTime(t, "2002-10-02T10:00:00Z"),
			fromRaw: "-5h",
			toRaw:   "-2h",
			err:     nil,
		},
		{
			now:     mustParseTime(t, "2002-10-02T15:00:00Z"),
			to:      mustParseTime(t, "2002-10-02T03:00:00Z"),
			from:    mustParseTime(t, "2002-10-02T02:00:00Z"),
			toRaw:   "2002-10-02T03:00:00Z",
			fromRaw: "2002-10-02T02:00:00Z",
			err:     nil,
		},
		{
			now:     mustParseTime(t, "2020-03-01T17:00:00Z"),
			from:    mustParseTime(t, "2020-02-10T15:00:00Z"),
			to:      mustParseTime(t, "2020-03-01T15:00:00Z"),
			fromRaw: "2020-02-10T15:00:00Z",
			toRaw:   "-2h",
			err:     nil,
		},
	}

	for ii, v := range input {
		t.Run(fmt.Sprintf("[%d] FROM %s, TO: %s", ii, v.from.Format(time.RFC3339), v.to.Format(time.RFC3339)), func(t *testing.T) {
			from, err := fromStringToTime(v.now, v.fromRaw)
			if err != nil {
			}
			if from.String() != v.from.String() {
				t.Errorf("Expected FROM %s got %s", v.from.Format(time.RFC3339), from.Format(time.RFC3339))
			}

			to, err := fromStringToTime(v.now, v.toRaw)
			if to.String() != v.to.String() {
				t.Errorf("Expected TO: %s got %s", v.to.Format(time.RFC3339), to.Format(time.RFC3339))
			}
		})
	}
}
