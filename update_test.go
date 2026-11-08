package main

import (
    "testing"
    "strings"
)

func TestCompareVersion(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"v1.2.1", "1.2.0", true}, {"v1.2.0", "1.2.0", false}, {"1.2.0", "1.2.1", false}, {"v2.0.0", "1.99.0", true}, {"bad", "1.0.0", false},
	}
	for _, c := range cases {
		if got := newerVersion(c.a, c.b); got != c.want {
			t.Fatalf("newerVersion(%q,%q)=%v want %v", c.a, c.b, got, c.want)
		}
	}
}
func TestChecksumFor(t *testing.T) {
	text := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef  ORCA_Nova_Portable_Windows.zip\n"
	sum, err := checksumFor(text, "ORCA_Nova_Portable_Windows.zip")
	if err != nil || len(sum) != 64 {
		t.Fatalf("sum=%q err=%v", sum, err)
	}
	if _, err := checksumFor(text, "invalid.zip"); err == nil {
		t.Fatal("should reject missing checksum")
	}
}

func TestChecksumEntries(t *testing.T) {
    const target = "ORCA_Nova_Portable_Windows.zip"
    good := strings.Repeat("a", 64) + "  " + target + "\n"
    sum, err := checksumFor(good, target)
    if err != nil || sum != strings.Repeat("a", 64) { t.Fatalf("checksum=%q err=%v",sum,err) }
    if _, err := checksumFor("incorrect  "+target, target); err == nil { t.Fatal("invalid digest accepted") }
    if _, err := checksumFor(good, "other.zip"); err == nil { t.Fatal("wrong asset accepted") }
}
