package exec

import (
	"context"
	"testing"
	"time"
)

func TestRunInputFeedsStdinWithoutShell(t *testing.T) {
	res, err := RunInput(context.Background(), time.Second, "restore-data", "tee")
	if err != nil {
		t.Fatal(err)
	}
	if res.Stdout != "restore-data" {
		t.Fatalf("stdout=%q", res.Stdout)
	}
}
