package policy

import (
	"errors"
	"strings"
	"testing"
)

func validDoc() Document {
	return Document{
		AppLimits: []AppLimit{
			{BundleID: "com.example.app", DailyLimitSeconds: 3600},
		},
		DowntimeWindows: []DowntimeWindow{
			{Start: "21:00", End: "07:00", Days: []string{"MONDAY", "TUESDAY"}},
		},
		BlockList: []string{"com.example.bad"},
	}
}

func TestValidate_HappyPath(t *testing.T) {
	d := validDoc()
	if err := d.Validate(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestValidate_EmptyDocumentIsValid(t *testing.T) {
	d := EmptyDocument()
	if err := d.Validate(); err != nil {
		t.Errorf("expected EmptyDocument to validate, got %v", err)
	}
}

func TestValidate_AppLimitFailures(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*Document)
		fragment string
	}{
		{
			name: "empty bundle id",
			mutate: func(d *Document) {
				d.AppLimits[0].BundleID = ""
			},
			fragment: "bundle_id is empty",
		},
		{
			name: "bundle id too long",
			mutate: func(d *Document) {
				d.AppLimits[0].BundleID = strings.Repeat("a", 257)
			},
			fragment: "bundle_id too long",
		},
		{
			name: "daily limit zero",
			mutate: func(d *Document) {
				d.AppLimits[0].DailyLimitSeconds = 0
			},
			fragment: "daily_limit_seconds must be > 0",
		},
		{
			name: "daily limit negative",
			mutate: func(d *Document) {
				d.AppLimits[0].DailyLimitSeconds = -1
			},
			fragment: "daily_limit_seconds must be > 0",
		},
		{
			name: "daily limit exceeds 24h",
			mutate: func(d *Document) {
				d.AppLimits[0].DailyLimitSeconds = 25 * 60 * 60
			},
			fragment: "exceeds 24h",
		},
		{
			name: "too many app limits",
			mutate: func(d *Document) {
				d.AppLimits = make([]AppLimit, maxAppLimits+1)
				for i := range d.AppLimits {
					d.AppLimits[i] = AppLimit{BundleID: "com.example.app", DailyLimitSeconds: 60}
				}
			},
			fragment: "app_limits exceeds",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := validDoc()
			tc.mutate(&d)
			err := d.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrInvalidDocument) {
				t.Errorf("not ErrInvalidDocument: %v", err)
			}
			if !strings.Contains(err.Error(), tc.fragment) {
				t.Errorf("missing %q in %q", tc.fragment, err.Error())
			}
		})
	}
}

func TestValidate_DowntimeWindowFailures(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*Document)
		fragment string
	}{
		{
			name: "bad start format",
			mutate: func(d *Document) {
				d.DowntimeWindows[0].Start = "9:00"
			},
			fragment: "start",
		},
		{
			name: "bad end format",
			mutate: func(d *Document) {
				d.DowntimeWindows[0].End = "25:00"
			},
			fragment: "end",
		},
		{
			name: "zero-length window",
			mutate: func(d *Document) {
				d.DowntimeWindows[0].Start = "08:00"
				d.DowntimeWindows[0].End = "08:00"
			},
			fragment: "zero-length window",
		},
		{
			name: "empty days",
			mutate: func(d *Document) {
				d.DowntimeWindows[0].Days = []string{}
			},
			fragment: "days is empty",
		},
		{
			name: "invalid weekday",
			mutate: func(d *Document) {
				d.DowntimeWindows[0].Days = []string{"FUNDAY"}
			},
			fragment: `"FUNDAY"`,
		},
		{
			name: "lowercase weekday rejected",
			mutate: func(d *Document) {
				d.DowntimeWindows[0].Days = []string{"monday"}
			},
			fragment: `"monday"`,
		},
		{
			name: "duplicate days",
			mutate: func(d *Document) {
				d.DowntimeWindows[0].Days = []string{"MONDAY", "MONDAY"}
			},
			fragment: "duplicate",
		},
		{
			name: "too many windows",
			mutate: func(d *Document) {
				d.DowntimeWindows = make([]DowntimeWindow, maxDowntimeWindows+1)
				for i := range d.DowntimeWindows {
					d.DowntimeWindows[i] = DowntimeWindow{
						Start: "21:00",
						End:   "22:00",
						Days:  []string{"MONDAY"},
					}
				}
			},
			fragment: "downtime_windows exceeds",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := validDoc()
			tc.mutate(&d)
			err := d.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrInvalidDocument) {
				t.Errorf("not ErrInvalidDocument: %v", err)
			}
			if !strings.Contains(err.Error(), tc.fragment) {
				t.Errorf("missing %q in %q", tc.fragment, err.Error())
			}
		})
	}
}

func TestValidate_BlockListFailures(t *testing.T) {
	cases := []struct {
		name     string
		mutate   func(*Document)
		fragment string
	}{
		{
			name: "empty entry",
			mutate: func(d *Document) {
				d.BlockList = []string{""}
			},
			fragment: "is empty",
		},
		{
			name: "entry too long",
			mutate: func(d *Document) {
				d.BlockList = []string{strings.Repeat("a", 257)}
			},
			fragment: "too long",
		},
		{
			name: "too many entries",
			mutate: func(d *Document) {
				d.BlockList = make([]string, maxBlockListEntries+1)
				for i := range d.BlockList {
					d.BlockList[i] = "com.example.bad"
				}
			},
			fragment: "block_list exceeds",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := validDoc()
			tc.mutate(&d)
			err := d.Validate()
			if err == nil {
				t.Fatal("expected error")
			}
			if !errors.Is(err, ErrInvalidDocument) {
				t.Errorf("not ErrInvalidDocument: %v", err)
			}
			if !strings.Contains(err.Error(), tc.fragment) {
				t.Errorf("missing %q in %q", tc.fragment, err.Error())
			}
		})
	}
}

func TestValidate_EdgeTimes(t *testing.T) {
	// 00:00 and 23:59 are valid bounds.
	d := validDoc()
	d.DowntimeWindows[0].Start = "00:00"
	d.DowntimeWindows[0].End = "23:59"
	if err := d.Validate(); err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}
