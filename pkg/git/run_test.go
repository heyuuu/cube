package git

import "testing"

// TestFirstLine 取输出首行。
func TestFirstLine(t *testing.T) {
	cases := map[string]string{
		"git@x:a/b\nsecond\n": "git@x:a/b",
		"only\n":              "only",
		"":                    "",
		"  \n":                "",
	}
	for in, want := range cases {
		if got := firstLine(in); got != want {
			t.Errorf("firstLine(%q) = %q，期望 %q", in, got, want)
		}
	}
}
