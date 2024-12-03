package retry

import (
	"testing"
	"time"
)

func TestIterTiming(t *testing.T) {
	testIter := Strategy{
		Delay:       0.1e9,
		MaxDuration: 0.25e9,
		Regular:     true,
	}
	want := []time.Duration{0, 0.1e9, 0.2e9, 0.2e9}
	got := make([]time.Duration, 0, len(want))
	t0 := time.Now()
	i := testIter.Start()

	for {
		got = append(got, time.Now().Sub(t0))
		if !i.Next(nil) {
			break
		}
	}
	got = append(got, time.Now().Sub(t0))

	if i.WasStopped() {
		t.Error("unexpected stop")
	}

	if len(got) != len(want) {
		t.Errorf("got %d attempts, want %d", len(got), len(want))
	}

	const margin = 0.01e9
	for i, g := range got {
		lo := want[i] - margin
		hi := want[i] + margin
		if g < lo || g > hi {
			t.Errorf("attempt %d want %g got %g", i, want[i].Seconds(), g.Seconds())
		}
	}
}

func TestParseStrategy(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		want        Strategy
		wantErr     bool
		errContains string
	}{
		// Basic cases
		{
			name:  "basic exponential",
			input: "delay=100ms maxdelay=1s factor=2.0",
			want: Strategy{
				Delay:    100 * time.Millisecond,
				MaxDelay: time.Second,
				Factor:   2.0,
			},
		},
		{
			name:  "default factor",
			input: "delay=100ms maxdelay=1s",
			want: Strategy{
				Delay:    100 * time.Millisecond,
				MaxDelay: time.Second,
			},
		},

		// Duration formats
		{
			name:  "microsecond delay",
			input: "delay=100us maxdelay=1ms factor=1.5",
			want: Strategy{
				Delay:    100 * time.Microsecond,
				MaxDelay: time.Millisecond,
				Factor:   1.5,
			},
		},
		{
			name:  "minute delay",
			input: "delay=1m maxdelay=1h factor=2.0",
			want: Strategy{
				Delay:    time.Minute,
				MaxDelay: time.Hour,
				Factor:   2.0,
			},
		},

		// Max attempts
		{
			name:  "with max count",
			input: "delay=100ms maxdelay=1s factor=2.0 maxcount=5",
			want: Strategy{
				Delay:    100 * time.Millisecond,
				MaxDelay: time.Second,
				Factor:   2.0,
				MaxCount: 5,
			},
		},

		// Max duration
		{
			name:  "with max duration",
			input: "delay=100ms maxdelay=1s factor=2.0 maxduration=5s",
			want: Strategy{
				Delay:       100 * time.Millisecond,
				MaxDelay:    time.Second,
				Factor:      2.0,
				MaxDuration: 5 * time.Second,
			},
		},

		// Regular mode
		{
			name:  "regular mode",
			input: "delay=100ms maxdelay=1s regular=true",
			want: Strategy{
				Delay:    100 * time.Millisecond,
				MaxDelay: time.Second,
				Regular:  true,
			},
		},

		// Complex combinations
		{
			name:  "all parameters",
			input: "delay=100ms maxdelay=1s factor=2.0 maxcount=5 maxduration=5s regular=true",
			want: Strategy{
				Delay:       100 * time.Millisecond,
				MaxDelay:    time.Second,
				Factor:      2.0,
				MaxCount:    5,
				MaxDuration: 5 * time.Second,
				Regular:     true,
			},
		},

		// Error cases
		{
			name:        "invalid duration",
			input:       "delay=invalid",
			wantErr:     true,
			errContains: "invalid duration",
		},
		{
			name:        "invalid factor",
			input:       "delay=100ms factor=invalid",
			wantErr:     true,
			errContains: "factor",
		},
		{
			name:        "invalid maxcount",
			input:       "delay=100ms maxcount=invalid",
			wantErr:     true,
			errContains: "maxcount",
		},
		{
			name:        "missing delay",
			input:       "maxdelay=1s",
			wantErr:     true,
			errContains: "delay",
		},
		{
			name:        "invalid regular",
			input:       "delay=100ms regular=notbool",
			wantErr:     true,
			errContains: "regular",
		},

		// Edge cases
		{
			name:  "zero maxcount",
			input: "delay=100ms maxcount=0",
			want: Strategy{
				Delay:    100 * time.Millisecond,
				MaxCount: 0,
			},
		},
		{
			name:  "very small delay",
			input: "delay=1ns maxdelay=1ms",
			want: Strategy{
				Delay:    time.Nanosecond,
				MaxDelay: time.Millisecond,
			},
		},
		{
			name:  "very large delay",
			input: "delay=1h maxdelay=24h",
			want: Strategy{
				Delay:    time.Hour,
				MaxDelay: 24 * time.Hour,
			},
		},
		{
			name:  "negative delay",
			input: "delay=-100ms",
			want: Strategy{
				Delay: -100 * time.Millisecond,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseStrategy(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			compareStrategy(t, got, &tt.want)
		})
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && s != substr
}

func compareStrategy(t *testing.T, got, want *Strategy) {
	t.Helper()
	if got.Delay != want.Delay {
		t.Errorf("Delay: got %v, want %v", got.Delay, want.Delay)
	}
	if got.MaxDelay != want.MaxDelay {
		t.Errorf("MaxDelay: got %v, want %v", got.MaxDelay, want.MaxDelay)
	}
	if got.Factor != want.Factor {
		t.Errorf("Factor: got %v, want %v", got.Factor, want.Factor)
	}
	if got.MaxCount != want.MaxCount {
		t.Errorf("MaxCount: got %v, want %v", got.MaxCount, want.MaxCount)
	}
	if got.MaxDuration != want.MaxDuration {
		t.Errorf("MaxDuration: got %v, want %v", got.MaxDuration, want.MaxDuration)
	}
	if got.Regular != want.Regular {
		t.Errorf("Regular: got %v, want %v", got.Regular, want.Regular)
	}
}

func BenchmarkReuseIter(b *testing.B) {
	strategy := Strategy{
		Delay:    1,
		MaxCount: 1,
	}
	b.ReportAllocs()
	var i Iter
	for j := 0; j < b.N; j++ {
		for i.Reset(&strategy, nil); i.Next(nil); {
		}
	}
}
