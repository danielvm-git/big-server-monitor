package processmonitor

import (
	"os/exec"
	"testing"
)

func TestLsofListeningPortsIntegration(t *testing.T) {
	if _, err := exec.LookPath("lsof"); err != nil {
		t.Skip("lsof not available")
	}

	ld := &LsofPortDiscovery{}
	ports, err := ld.ListeningPorts()
	if err != nil {
		t.Fatalf("ListeningPorts failed: %v", err)
	}
	// Should return at least 0 ports (not nil)
	if ports == nil {
		t.Error("expected non-nil ports slice")
	}
}

func TestNetstatListeningPortsIntegration(t *testing.T) {
	if _, err := exec.LookPath("netstat"); err != nil {
		t.Skip("netstat not available")
	}

	ld := &LsofPortDiscovery{}
	ports, err := ld.listeningPortsViaNetstat()
	if err != nil {
		t.Fatalf("listeningPortsViaNetstat failed: %v", err)
	}
	if ports == nil {
		t.Error("expected non-nil ports slice")
	}
}

func TestParseLsofOutput(t *testing.T) {
	tests := []struct {
		name   string
		output []byte
		want   []int
	}{
		{
			name:   "empty output",
			output: []byte(""),
			want:   []int{},
		},
		{
			name: "single port",
			output: []byte("COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"node    12345 user   21u  IPv4 0x1234      0t0  TCP *:8080 (LISTEN)\n"),
			want: []int{8080},
		},
		{
			name: "multiple ports",
			output: []byte("COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"node    12345 user   21u  IPv4 0x1234      0t0  TCP *:3000 (LISTEN)\n" +
				"python  23456 user   10u  IPv4 0x5678      0t0  TCP *:8080 (LISTEN)\n"),
			want: []int{3000, 8080},
		},
		{
			name: "localhost address",
			output: []byte("COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"node    12345 user   21u  IPv4 0x1234      0t0  TCP 127.0.0.1:5432 (LISTEN)\n"),
			want: []int{5432},
		},
		{
			name: "IPv6 address",
			output: []byte("COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"java    34567 user   30u  IPv6 0xabcd      0t0  TCP [::1]:8080 (LISTEN)\n"),
			want: []int{8080},
		},
		{
			name: "duplicate port",
			output: []byte("COMMAND   PID USER   FD   TYPE DEVICE SIZE/OFF NODE NAME\n" +
				"node    12345 user   21u  IPv4 0x1234      0t0  TCP *:3000 (LISTEN)\n" +
				"node    12345 user   22u  IPv6 0x5678      0t0  TCP *:3000 (LISTEN)\n"),
			want: []int{3000},
		},
		{
			name: "short line (ignored)",
			output: []byte("COMMAND   PID\n" +
				"too short\n"),
			want: []int{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseLsofOutput(tt.output)
			if !equalIntSets(got, tt.want) {
				t.Errorf("parseLsofOutput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractPort(t *testing.T) {
	tests := []struct {
		addr string
		want int
	}{
		{"*:8080", 8080},
		{"127.0.0.1:3000", 3000},
		{"[::1]:5432", 5432},
		{"localhost:4000", 4000},
		{"0.0.0.0:9090", 9090},
		{"no-port", 0},
		{"", 0},
		{":8080", 8080},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			got := extractPort(tt.addr)
			if got != tt.want {
				t.Errorf("extractPort(%q) = %d, want %d", tt.addr, got, tt.want)
			}
		})
	}
}

// equalIntSets compares two int slices regardless of order
func equalIntSets(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[int]int, len(a))
	for _, v := range a {
		m[v]++
	}
	for _, v := range b {
		m[v]--
		if m[v] < 0 {
			return false
		}
	}
	return true
}
