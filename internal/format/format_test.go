package format

import "testing"

func TestSize(t *testing.T) {
	cases := map[int64]string{
		0:               "0 B",
		512:             "512 B",
		1536:            "1,5 KB",
		GB:              "1,0 GB",
		952*GB + 800*MB: "952,8 GB",
		-2 * MB:         "-2,0 MB",
	}
	for in, want := range cases {
		if got := Size(in); got != want {
			t.Errorf("Size(%d) = %q, quiero %q", in, got, want)
		}
	}
}

func TestPercent(t *testing.T) {
	cases := []struct {
		a, b int64
		want string
	}{{50, 100, "50%"}, {1, 1000, "<1%"}, {5, 0, "-"}, {2, 3, "67%"}}
	for _, c := range cases {
		if got := Percent(c.a, c.b); got != c.want {
			t.Errorf("Percent(%d,%d) = %q, quiero %q", c.a, c.b, got, c.want)
		}
	}
}

func TestBar(t *testing.T) {
	if got := Bar(1, 2, 4); got != "██░░" {
		t.Errorf("Bar = %q", got)
	}
	if got := Bar(5, 2, 3); got != "███" {
		t.Errorf("Bar desbordada = %q", got)
	}
	if got := Bar(1, 0, 3); got != "" {
		t.Errorf("Bar con total 0 = %q", got)
	}
}

func TestThousands(t *testing.T) {
	cases := map[int64]string{0: "0", 999: "999", 1000: "1.000", 727466: "727.466", -1234567: "-1.234.567"}
	for in, want := range cases {
		if got := Thousands(in); got != want {
			t.Errorf("Thousands(%d) = %q, quiero %q", in, got, want)
		}
	}
}

func TestParseSize(t *testing.T) {
	cases := map[string]int64{"1GB": GB, "500MB": 500 * MB, "1,5gb": GB + GB/2, " 2 TB ": 2 * TB, "1024": 1024, "10kb": 10 * KB}
	for in, want := range cases {
		got, err := ParseSize(in)
		if err != nil || got != want {
			t.Errorf("ParseSize(%q) = %d, %v; quiero %d", in, got, err, want)
		}
	}
	if _, err := ParseSize("mucho"); err == nil {
		t.Error("ParseSize(\"mucho\") debería fallar")
	}
}
