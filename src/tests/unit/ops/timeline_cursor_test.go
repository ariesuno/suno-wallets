package ops

import (
	opsapp "suno-wallets/src/application/ops"
	"testing"
	"time"
)

func TestTimelineCursor_EncodeDecode(t *testing.T) {
	d := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	cur := opsapp.EncodeCursor(d, "abc")
	d2, id, err := opsapp.DecodeCursor(cur)
	if err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if id != "abc" {
		t.Fatalf("expected id abc, got %s", id)
	}
	if d2.Format("2006-01-02") != "2024-06-01" {
		t.Fatalf("date mismatch")
	}
}
