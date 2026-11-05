package resources

import "testing"

func TestResolveNodePoolOs(t *testing.T) {
	cases := []struct {
		image string
		want  string
	}{
		{"", defaultNodePoolOs},
		{"default", defaultNodePoolOs},
		{"DEFAULT", defaultNodePoolOs},
		{" Default ", defaultNodePoolOs},
		{"tlinux3.1x86_64", "tlinux3.1x86_64"},
	}
	for _, c := range cases {
		if got := resolveNodePoolOs(c.image); got != c.want {
			t.Errorf("resolveNodePoolOs(%q) = %q, want %q", c.image, got, c.want)
		}
	}
}
