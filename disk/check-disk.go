package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"os/exec"

	"github.com/AcalephStorage/go_check/Godeps/_workspace/src/github.com/newrelic/go_nagios"
	"github.com/AcalephStorage/go_check/Godeps/_workspace/src/gopkg.in/alecthomas/kingpin.v1"
)

const (
	system = iota
	osd
	rbd
)

type useType int

func (dt useType) string() string {
	switch dt {
	case system:
		return "SYSTEM"
	case osd:
		return "OSD"
	case rbd:
		return "RBD"
	default:
		return ""
	}
}

type diskResult struct {
	filesystem string
	deviceType string
	blocks     int64
	used       int64
	available  int64
	capacity   int
	mounted    string
	usage      useType
}

var (
	warnLevel = kingpin.Flag("warn-level", "warn level").Default("85").Int()
	critLevel = kingpin.Flag("crit-level", "crit level").Default("95").Int()
)

func main() {
	kingpin.Version("1.0.0")
	kingpin.Parse()
	checkDisk(*warnLevel, *critLevel)
}

func checkDisk(warnLevel, critLevel int) {
	cmd := exec.Command("df", "-PT", "-x", "tmpfs", "-x", "devtmpfs")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		nagios.Unknown(err.Error())
	}
	result := out.String()

	var buf bytes.Buffer
	lines := strings.Split(result, "\n")
	devices := make([]*diskResult, len(lines)-1)
	for i, line := range lines {
		if i == 0 {
			fmt.Fprintln(&buf, "  ", line)
			continue
		}
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		device := &diskResult{
			filesystem: fields[0],
			deviceType: fields[1],
			blocks:     toInt64(fields[2]),
			used:       toInt64(fields[3]),
			available:  toInt64(fields[4]),
			capacity:   toIntFromPercent(fields[5]),
			mounted:    fields[6],
		}
		fillUseType(device)
		devices[i-1] = device
		fmt.Fprintln(&buf, "  ", line)
	}

	var problemMessages bytes.Buffer
	critCount := 0
	warnCount := 0
	for _, device := range devices {
		if device == nil {
			continue
		}
		switch {
		case device.capacity >= critLevel:
			fmt.Fprintln(&problemMessages, format(device))
			critCount++
		case device.capacity >= warnLevel:
			fmt.Fprintln(&problemMessages, format(device))
			warnCount++
		}
	}

	status := &nagios.NagiosStatus{}
	switch {
	case critCount > 0:
		status.Value = nagios.NAGIOS_CRITICAL
	case warnCount > 0:
		status.Value = nagios.NAGIOS_WARNING
	default:
		status.Value = nagios.NAGIOS_OK
	}

	var messages bytes.Buffer

	fmt.Fprintf(&messages, "CheckDisk.\n")
	fmt.Fprintf(&messages, "%v", buf.String())
	if status.Value != nagios.NAGIOS_OK {
		fmt.Fprintf(&messages, "\n\nAlerts:\n")
		fmt.Fprintln(&messages, problemMessages.String())
	}
	status.Message = messages.String()
	nagios.ExitWithStatus(status)
}

func toInt64(str string) int64 {
	result, _ := strconv.ParseInt(str, 10, 64)
	return result
}

func toIntFromPercent(str string) int {
	result, _ := strconv.Atoi(str[:len(str)-1])
	return result
}

func fillUseType(device *diskResult) {
	switch {
	case strings.HasPrefix(device.mounted, "/var/lib/ceph/osd/"):
		device.usage = osd
	case strings.HasPrefix(device.filesystem, "/dev/rbd"):
		device.usage = rbd
	default:
		device.usage = system
	}
}

func format(device *diskResult) string {
	return fmt.Sprintf("  %v device %v is at %v%% capacity.", device.usage.string(), device.filesystem, device.capacity)
}
