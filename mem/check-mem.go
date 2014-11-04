package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"os/exec"

	"github.com/newrelic/go_nagios"
	"gopkg.in/alecthomas/kingpin.v1"
)

var (
	warnLevel = kingpin.Flag("warn-level", "warn level").Default("30").Int()
	critLevel = kingpin.Flag("crit-level", "crit level").Default("15").Int()
)

func main() {
	kingpin.Version("1.0.0")
	kingpin.Parse()
	checkMem(*warnLevel, *critLevel)
}

func checkMem(warnLevel, critLevel int) {
	cmd := exec.Command("free", "-m")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		nagios.Unknown(err.Error())
	}
	result := out.String()

	var total int
	var available int

	lines := strings.Split(result, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Mem:") {
			total, _ = strconv.Atoi(strings.Fields(line)[1])
		}
		if strings.HasPrefix(line, "-/+ buffers/cache") {
			available, _ = strconv.Atoi(strings.Fields(line)[3])
		}
	}

	availablePercentage := int(float64(available) / float64(total) * float64(100))

	status := &nagios.NagiosStatus{}
	switch {
	case availablePercentage <= critLevel:
		status.Value = nagios.NAGIOS_CRITICAL
	case availablePercentage <= warnLevel:
		status.Value = nagios.NAGIOS_WARNING
	default:
		status.Value = nagios.NAGIOS_OK
	}

	status.Message = fmt.Sprintf("Check Mem: total=%vmB available=%vmB | %v%% Available Memory left.", total, available, availablePercentage)
	nagios.ExitWithStatus(status)
}
