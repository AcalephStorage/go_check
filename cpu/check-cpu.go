package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/newrelic/go_nagios"
	"gopkg.in/alecthomas/kingpin.v1"
)

var (
	warnLevel = kingpin.Flag("warn-level", "warn level").Default("0.75").Float()
	critLevel = kingpin.Flag("crit-level", "critical level").Default("0.85").Float()
)

func main() {
	kingpin.Version("1.0.0")
	kingpin.Parse()
	checkCpu(*warnLevel, *critLevel)
}

func checkCpu(warnLevel, critLevel float64) {
	before := readCpuStat()
	if before == nil {
		nagios.Unknown("Unable to check CPU status")
	}
	time.Sleep(1 * time.Second)
	after := readCpuStat()
	total, _, each := compute(before, after)

	status := &nagios.NagiosStatus{}
	switch {
	case total >= critLevel:
		status.Value = nagios.NAGIOS_CRITICAL
	case total >= warnLevel:
		status.Value = nagios.NAGIOS_WARNING
	default:
		status.Value = nagios.NAGIOS_OK
	}

	status.Message = fmt.Sprintf("total=%0.2f user=%0.2f nice=%0.2f system=%0.2f idle=%0.2f iowait=%0.2f irq=%0.2f softirq=%0.2f steal=%0.2f guest=%0.2f guest_nice=%0.2f", total, each[0], each[1], each[2], each[3], each[4], each[5], each[6], each[7], each[8], each[9])
	nagios.ExitWithStatus(status)
}

func compute(before []int64, after []int64) (total, free float64, each []float64) {
	diff := make([]int64, len(after))
	totalDiff := int64(0)
	for i := range after {
		a := after[i]
		b := before[i]
		diff[i] = a - b
		totalDiff += diff[i]
	}
	each = make([]float64, len(after))
	for i, d := range diff {
		each[i] = 100 * (float64(d) / float64(totalDiff))
	}

	free = each[3]
	total = 100.0 - free
	return total, free, each
}

// [user, nice, system, idle, iowait, irq, softirq, steal, guest, guest_nice]

func readCpuStat() []int64 {
	file, _ := os.Open("/proc/stat")
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		stat, _ := regexp.MatchString("^cpu ", line)
		if stat {
			arr := strings.Fields(line)
			return toIntArray(arr[1:])
		}
	}
	return nil
}

func toIntArray(arr []string) []int64 {
	intArr := make([]int64, len(arr))
	for i, str := range arr {
		val, _ := strconv.ParseInt(str, 10, 64)
		intArr[i] = val
	}
	return intArr
}
