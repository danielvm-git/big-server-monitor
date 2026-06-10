package processmonitor

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// PortDiscovery abstracts OS-level port and process discovery,
// enabling testability via mocking.
type PortDiscovery interface {
	ListeningPorts() ([]int, error)
	ProcessInfo(port int) (processInfo, error)
}

// LsofPortDiscovery implements PortDiscovery using lsof and OS commands.
type LsofPortDiscovery struct{}

// GetProcessInfo is the public form of ProcessInfo for testing.
func (l *LsofPortDiscovery) GetProcessInfo(port int) (processInfo, error) {
	return l.ProcessInfo(port)
}

// ListeningPorts returns list of TCP ports currently listening.
// Tries lsof first, falls back to netstat on macOS if lsof is unavailable.
func (l *LsofPortDiscovery) ListeningPorts() ([]int, error) {
	ports, err := l.listeningPortsViaLsof()
	if err != nil {
		ports, err2 := l.listeningPortsViaNetstat()
		if err2 != nil {
			return nil, fmt.Errorf("list ports (lsof: %w, netstat: %w)", err, err2)
		}
		return ports, nil
	}
	return ports, nil
}

// listeningPortsViaLsof uses lsof to list TCP listening ports.
func (l *LsofPortDiscovery) listeningPortsViaLsof() ([]int, error) {
	cmd := exec.Command("lsof", "-iTCP", "-sTCP:LISTEN", "-n", "-P")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("lsof: %w", err)
	}
	return parseLsofOutput(output), nil
}

// listeningPortsViaNetstat uses netstat as a fallback on macOS.
func (l *LsofPortDiscovery) listeningPortsViaNetstat() ([]int, error) {
	cmd := exec.Command("netstat", "-an", "-p", "tcp")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("netstat: %w", err)
	}

	ports := make(map[int]bool)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		addr := fields[3]
		port := extractPort(addr)
		if port > 0 {
			ports[port] = true
		}
	}

	result := make([]int, 0, len(ports))
	for port := range ports {
		result = append(result, port)
	}
	return result, nil
}

// ProcessInfo retrieves metadata for a process listening on the given port.
func (l *LsofPortDiscovery) ProcessInfo(port int) (processInfo, error) {
	cmd := exec.Command("lsof", "-iTCP:"+strconv.Itoa(port), "-sTCP:LISTEN", "-n", "-P")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return processInfo{}, fmt.Errorf("lsof port %d: %w", port, err)
	}

	var pid int
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pidStr := fields[1]
		if p, err := strconv.Atoi(pidStr); err == nil && p > 0 {
			pid = p
			break
		}
	}

	if pid == 0 {
		return processInfo{}, fmt.Errorf("no PID found for port %d", port)
	}

	return l.getProcessDetails(pid)
}

func (l *LsofPortDiscovery) getProcessDetails(pid int) (processInfo, error) {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=,etime=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return processInfo{}, fmt.Errorf("ps: %w", err)
	}

	line := strings.TrimSpace(string(output))
	fields := strings.Fields(line)
	if len(fields) < 1 {
		return processInfo{}, fmt.Errorf("no process data for PID %d", pid)
	}

	processName := fields[0]
	binaryPath := l.getBinaryPath(pid)
	workingDir := l.getWorkingDir(pid)
	memoryMB := l.getMemoryMB(pid)
	startTime := l.parseStartTime(pid)

	return processInfo{
		PID:         pid,
		ProcessName: processName,
		BinaryPath:  binaryPath,
		WorkingDir:  workingDir,
		MemoryMB:    memoryMB,
		StartTime:   startTime,
	}, nil
}

func (l *LsofPortDiscovery) getBinaryPath(pid int) string {
	exePath := filepath.Join("/proc", strconv.Itoa(pid), "exe")
	if realPath, err := os.Readlink(exePath); err == nil {
		return realPath
	}

	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid), "-a", "-d", "cwd")
	output, _ := cmd.CombinedOutput()
	if len(output) > 0 {
		return strings.TrimSpace(string(output))
	}

	return ""
}

func (l *LsofPortDiscovery) getWorkingDir(pid int) string {
	cwdPath := filepath.Join("/proc", strconv.Itoa(pid), "cwd")
	if realPath, err := os.Readlink(cwdPath); err == nil {
		return realPath
	}

	cmd := exec.Command("lsof", "-p", strconv.Itoa(pid), "-a", "-d", "cwd")
	output, _ := cmd.CombinedOutput()
	lines := strings.Split(string(output), "\n")
	if len(lines) > 1 {
		fields := strings.Fields(lines[1])
		if len(fields) > 0 {
			return fields[len(fields)-1]
		}
	}

	return ""
}

func (l *LsofPortDiscovery) getMemoryMB(pid int) float64 {
	statusPath := filepath.Join("/proc", strconv.Itoa(pid), "status")
	data, err := os.ReadFile(statusPath)
	if err == nil {
		scanner := bufio.NewScanner(bytes.NewReader(data))
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "VmRSS:") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					kb, _ := strconv.Atoi(fields[1])
					return float64(kb) / 1024
				}
			}
		}
	}

	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "rss=")
	output, _ := cmd.CombinedOutput()
	kb, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return float64(kb) / 1024
}

func (l *LsofPortDiscovery) parseStartTime(pid int) time.Time {
	statPath := filepath.Join("/proc", strconv.Itoa(pid), "stat")
	data, err := os.ReadFile(statPath)
	if err == nil {
		fields := strings.Fields(string(data))
		if len(fields) > 21 {
			ticks, _ := strconv.Atoi(fields[21])
			startTime := time.Now().Add(-time.Duration(ticks) * time.Millisecond)
			return startTime
		}
	}

	return time.Now()
}

// extractPort parses port from address like *:8080 or 127.0.0.1:3000
func extractPort(addr string) int {
	parts := strings.Split(addr, ":")
	if len(parts) < 2 {
		return 0
	}
	port, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0
	}
	return port
}

// parseLsofOutput extracts port numbers from lsof output lines.
func parseLsofOutput(output []byte) []int {
	ports := make(map[int]bool)
	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 9 {
			continue
		}
		// The address field is immediately before the parenthesized state
		// indicator (e.g., (LISTEN)) in the NAME column, or it is the
		// last field when the state indicator is absent.
		idx := len(fields) - 1
		if fields[idx] == "(LISTEN)" && idx > 0 {
			idx--
		}
		addr := fields[idx]
		port := extractPort(addr)
		if port > 0 {
			ports[port] = true
		}
	}

	result := make([]int, 0, len(ports))
	for port := range ports {
		result = append(result, port)
	}
	return result
}
