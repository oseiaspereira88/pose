package cli

import (
	"runtime"
	"testing"
)

func TestCheckWorkerCountDefaultsToCoresAndClamps(t *testing.T) {
	t.Setenv("POSE_CHECK_WORKERS", "")
	cores := runtime.NumCPU()
	if got := checkWorkerCount(cores * 4); got != cores {
		t.Errorf("default = %d, want the core count %d", got, cores)
	}
	// Never more workers than items, and never fewer than one.
	if got := checkWorkerCount(3); got != min(3, cores) {
		t.Errorf("with 3 items = %d, want %d", got, min(3, cores))
	}
	if got := checkWorkerCount(1); got != 1 {
		t.Errorf("with 1 item = %d, want 1", got)
	}
}

func TestCheckWorkerCountOverride(t *testing.T) {
	for _, tc := range []struct {
		value string
		items int
		want  int
	}{
		{"1", 100, 1},
		{"4", 100, 4},
		{"64", 100, 64},
		{"64", 5, 5}, // still clamped by the item count
	} {
		t.Run(tc.value+"/"+itoaForTest(tc.items), func(t *testing.T) {
			t.Setenv("POSE_CHECK_WORKERS", tc.value)
			if got := checkWorkerCount(tc.items); got != tc.want {
				t.Errorf("POSE_CHECK_WORKERS=%s with %d items = %d, want %d", tc.value, tc.items, got, tc.want)
			}
		})
	}
}

// A gate is the wrong place to fail over an environment variable, so a value that
// makes no sense is ignored and the default stands.
func TestCheckWorkerCountIgnoresUnusableValues(t *testing.T) {
	cores := runtime.NumCPU()
	for _, value := range []string{"0", "-4", "many", "8.5", " ", "1e3"} {
		t.Setenv("POSE_CHECK_WORKERS", value)
		if got := checkWorkerCount(cores * 4); got != cores {
			t.Errorf("POSE_CHECK_WORKERS=%q = %d, want the default %d", value, got, cores)
		}
	}
	// A value with surrounding whitespace is still a number.
	t.Setenv("POSE_CHECK_WORKERS", "  6 ")
	if got := checkWorkerCount(100); got != 6 {
		t.Errorf("padded value = %d, want 6", got)
	}
}

func itoaForTest(n int) string {
	if n == 0 {
		return "0"
	}
	digits := ""
	for n > 0 {
		digits = string(rune('0'+n%10)) + digits
		n /= 10
	}
	return digits
}
