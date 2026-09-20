package cli

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

// Parallelism may change when work happens, never what the gate prints or the order
// it prints it in. This pins both: the findings come out in item order regardless of
// which worker finished first, and the counters match.
func TestFailOrWarnPerItemKeepsItemOrder(t *testing.T) {
	for _, count := range []int{0, 1, 7, 200} {
		t.Run(strconv.Itoa(count), func(t *testing.T) {
			var out, errOut bytes.Buffer
			checker := &nativeChecker{root: t.TempDir(), mode: "tolerant", stdout: &out,
				out: render(&out, &errOut)}
			var started int64
			checker.failOrWarnPerItem(count, func(i int) []string {
				// Reverse-skew the work so a later item finishes first whenever the
				// pool has room: if the replay depended on completion order, this is
				// what would scramble it.
				atomic.AddInt64(&started, 1)
				for spin := 0; spin < (count-i)*2000; spin++ {
					_ = spin
				}
				if i%3 == 2 {
					return nil // an item with nothing to say must not shift the rest
				}
				return []string{fmt.Sprintf("item-%04d", i)}
			})
			if int(started) != count {
				t.Fatalf("ran %d items, want %d", started, count)
			}
			want := []string{}
			for i := 0; i < count; i++ {
				if i%3 != 2 {
					want = append(want, fmt.Sprintf("item-%04d", i))
				}
			}
			got := []string{}
			for _, line := range strings.Split(out.String()+errOut.String(), "\n") {
				if idx := strings.Index(line, "item-"); idx >= 0 {
					got = append(got, strings.TrimSpace(line[idx:]))
				}
			}
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Fatalf("findings out of item order\n got: %v\nwant: %v", got, want)
			}
			if checker.warnings != len(want) || checker.errors != 0 {
				t.Fatalf("counters = %d warnings %d errors, want %d warnings", checker.warnings, checker.errors, len(want))
			}
		})
	}
}

// Under --strict the same messages must escalate, which the replay must not change.
func TestFailOrWarnPerItemRespectsMode(t *testing.T) {
	var out, errOut bytes.Buffer
	checker := &nativeChecker{root: t.TempDir(), mode: "strict", stdout: &out,
		out: render(&out, &errOut)}
	checker.failOrWarnPerItem(20, func(i int) []string { return []string{fmt.Sprintf("m%02d", i)} })
	if checker.errors != 20 || checker.warnings != 0 {
		t.Fatalf("strict mode = %d errors %d warnings, want 20 errors", checker.errors, checker.warnings)
	}
}
