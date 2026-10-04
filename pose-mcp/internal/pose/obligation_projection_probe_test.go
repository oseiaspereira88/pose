package pose

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestProbeProjectionOnRepo(t *testing.T) {
	root := os.Getenv("POSE_PROBE_ROOT")
	if root == "" {
		t.Skip()
	}
	started := time.Now()
	r, err := Store{Root: root}.ProjectObligations(ObligationQuery{})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println("elapsed", time.Since(started), "obligations", len(r.Obligations), "counts", r.Counts, "complete", r.Complete)
	fmt.Println("durations", r.Durations)
	for _, c := range r.Coverage {
		fmt.Println("coverage", c.Producer, c.State, c.Detail)
	}
	fmt.Println("limitations", r.Limitations)
	n := 0
	for _, o := range r.Obligations {
		if o.Category != ObligationResidualDebt && n < 25 {
			fmt.Println(o.Category, o.ReasonCode, o.Effects, o.Message)
			n++
		}
	}
}
