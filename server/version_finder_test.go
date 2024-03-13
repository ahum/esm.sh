package server

import (
	"fmt"
	"testing"
)

func TestSimple(t *testing.T) {

	arr := []string{"1.4.5", "1.0.0", "1.2.3", "0.1.0"}
	v, err := FindBestVersion("1.x", arr)
	t.Log(v)
	if v != "1.4.5" {
		t.Fatalf("invalid version %s, expected 1.4.5", v)
	}
	if err != nil {
		t.Fatal(err)
	}

}

var versionTests = []struct {
	target    string
	available []string
	expected  string
}{
	{"1.x", []string{"1.0.0"}, "1.0.0"},
	{"1.x", []string{"1.3.4", "1.0.0", "1.2.1"}, "1.3.4"},
	{"~1.2.4", []string{"1.2.5", "1.3.0", "1.2.4"}, "1.2.5"},
	// {"%-a", "[%-a]"},
	// {"%+a", "[%+a]"},
	// {"%#a", "[%#a]"},
	// {"% a", "[% a]"},
	// {"%0a", "[%0a]"},
	// {"%1.2a", "[%1.2a]"},
	// {"%-1.2a", "[%-1.2a]"},
	// {"%+1.2a", "[%+1.2a]"},
	// {"%-+1.2a", "[%+-1.2a]"},
	// {"%-+1.2abc", "[%+-1.2a]bc"},
	// {"%-1.2abc", "[%-1.2a]bc"},
}

func TestFindVersion(t *testing.T) {
	for _, tt := range versionTests {
		label := fmt.Sprintf("%s %s=%s", tt.target, tt.available, tt.expected)
		t.Run(label, func(t *testing.T) {
			v, err := FindBestVersion(tt.target, tt.available)
			if err != nil {
				t.Errorf("got %s", err)
			}
			if v != tt.expected {
				t.Errorf("got %q, want %q", v, tt.expected)
			}
		})
	}
}
