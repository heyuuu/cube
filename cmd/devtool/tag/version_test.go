package tag

import "testing"

func TestParseVersionTag(t *testing.T) {
	tests := []struct {
		in   string
		want VersionTag
		ok   bool
	}{
		{"v0.1.0", VersionTag{0, 1, 0}, true},
		{"v1.2.3", VersionTag{1, 2, 3}, true},
		{"0.1.0", VersionTag{}, false},
		{"v1.2", VersionTag{}, false},
		{"v1.2.3.4", VersionTag{}, false},
		{"v1.2.x", VersionTag{}, false},
		{"v1.2.-3", VersionTag{}, false},
	}
	for _, tt := range tests {
		got, ok := ParseVersionTag(tt.in)
		if ok != tt.ok || got != tt.want {
			t.Errorf("ParseVersionTag(%q) = %v, %v; want %v, %v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}

func TestVersionTagCompare(t *testing.T) {
	tests := []struct {
		a, b VersionTag
		want int
	}{
		{VersionTag{0, 1, 0}, VersionTag{0, 1, 0}, 0},
		{VersionTag{0, 1, 0}, VersionTag{0, 2, 0}, -1},
		{VersionTag{0, 2, 0}, VersionTag{0, 1, 0}, 1},
		{VersionTag{1, 0, 0}, VersionTag{0, 9, 9}, 1},
		{VersionTag{1, 2, 3}, VersionTag{1, 2, 4}, -1},
	}
	for _, tt := range tests {
		if got := tt.a.Compare(tt.b); got != tt.want {
			t.Errorf("%v.Compare(%v) = %d; want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
