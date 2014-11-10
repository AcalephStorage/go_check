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

var (
	warnLevel = kingpin.Flag("warn-level", "warn level").Default("10").Float()
	critLevel = kingpin.Flag("crit-level", "crit level").Default("100").Float()
)

func main() {
	kingpin.Version("1.0.0")
	kingpin.Parse()
	checkNtp(*warnLevel, *critLevel)
}

func checkNtp(warnLevel, critLevel float64) {
	cmd := exec.Command("ntpq", "-c", "rv 0 offset")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		nagios.Unknown(err.Error())
	}
	result := strings.TrimSpace(out.String())
	offset, err := strconv.ParseFloat(strings.Split(result, "=")[1], 64)
	if err != nil {
		nagios.Unknown(err.Error())
	}

	status := &nagios.NagiosStatus{}
	switch {
	case offset >= critLevel || offset <= -critLevel:
		status.Value = nagios.NAGIOS_CRITICAL
	case offset >= warnLevel || offset <= -warnLevel:
		status.Value = nagios.NAGIOS_WARNING
	default:
		status.Value = nagios.NAGIOS_OK
	}

	status.Message = fmt.Sprintf("CheckNTP: Offset: %0.2f", offset)
	nagios.ExitWithStatus(status)
}
