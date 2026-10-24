package format

import (
	"testing"
	"time"
)

func TestBytes(t *testing.T) {
	cases := map[uint64]string{0: "0 B", 1023: "1023 B", 1024: "1.0 KB", 1536: "1.5 KB", 16 << 30: "16.0 GB", 2 << 40: "2.0 TB"}
	for in, want := range cases {
		if got := Bytes(in); got != want {
			t.Errorf("Bytes(%d) = %s, want %s", in, got, want)
		}
	}
}

func TestUptime(t *testing.T) {
	cases := map[time.Duration]string{
		0:                                 "-",
		42 * time.Minute:                  "42m",
		3*time.Hour + 5*time.Minute:       "3h 05m",
		(12*24+3)*time.Hour + time.Minute: "12d 3h 01m",
	}
	for in, want := range cases {
		if got := Uptime(in); got != want {
			t.Errorf("Uptime(%v) = %s, want %s", in, got, want)
		}
	}
}

func TestLatency(t *testing.T) {
	if Latency(0) != "" || Latency(300*time.Microsecond) != "<1 ms" || Latency(3500*time.Microsecond) != "3.5 ms" || Latency(42*time.Millisecond) != "42 ms" {
		t.Error("unexpected latency formatting")
	}
}

func TestJoinAndOrDash(t *testing.T) {
	if Join([]string{"", " "}, ",") != "-" || Join([]string{"a", "", "b"}, ", ") != "a, b" || OrDash("") != "-" {
		t.Error("unexpected join")
	}
}
