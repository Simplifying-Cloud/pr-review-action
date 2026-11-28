package review

import "testing"

func TestGate(t *testing.T) {
	cases := []struct {
		verdict   string
		truncated bool
		fail      bool
	}{
		{"approve", false, false},
		{"comment", false, false},
		{"request_changes", false, true},
		{"approve", true, true},
		{"comment", true, true},
	}
	for _, c := range cases {
		err := Gate(c.verdict, c.truncated)
		if (err != nil) != c.fail {
			t.Errorf("Gate(%q, %v) = %v, want fail=%v", c.verdict, c.truncated, err, c.fail)
		}
	}
}

func TestParseOutputUnparseableFailsClosed(t *testing.T) {
	out, err := ParseOutput("not json at all")
	if err != nil {
		t.Fatal(err)
	}
	if out.Verdict != "request_changes" {
		t.Errorf("verdict = %q, want request_changes", out.Verdict)
	}
}
