package pose

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// Review round 9, N2: however many processes the shim ended, its report fits
// the dispatcher's read bound and still counts them all.
func TestTheShimReportIsBounded(t *testing.T) {
	r := reviewShimReport{Error: strings.Repeat("e", 100000)}
	for i := 0; i < 50000; i++ {
		r.Ended = append(r.Ended, strconv.Itoa(4000000+i))
	}
	raw := boundedReviewShimReport(r)
	if len(raw) >= 64<<10 {
		t.Fatalf("the report is %d bytes, beyond the dispatcher's 64 KiB read", len(raw))
	}
	var got reviewShimReport
	if err := json.Unmarshal(raw, &got); err != nil || got.EndedTotal != 50000 || len(got.Ended) != reviewShimReportMaxPids {
		t.Fatalf("the bounded report lost its count: %d of %d, %v", len(got.Ended), got.EndedTotal, err)
	}
}
