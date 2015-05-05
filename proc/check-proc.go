package main

import (
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"io/ioutil"
	"os/exec"

	"github.com/AcalephStorage/go_check/Godeps/_workspace/src/github.com/newrelic/go_nagios"
	"github.com/AcalephStorage/go_check/Godeps/_workspace/src/gopkg.in/alecthomas/kingpin.v1"
)

type procMap map[string]string
type procMaps []procMap
type rejectFunc func(p procMap) bool

var (
	warnOver    = kingpin.Flag("warn-over", "Trigger a warning if over a number").Int()
	critOver    = kingpin.Flag("crit-over", "Trigger critical if over a number").Int()
	warnUnder   = kingpin.Flag("warn-under", "Trigger a warning if under a number").Default("1").Int()
	critUnder   = kingpin.Flag("crit-under", "Trigger critical if under a number").Default("1").Int()
	metric      = kingpin.Flag("metric", "Trigger critical if there are metric procs").String()
	matchSelf   = kingpin.Flag("match-self", "Match itself").Bool()
	matchParent = kingpin.Flag("match-parent", "Match parent").Bool()
	pattern     = kingpin.Flag("pattern", "Match a command against this pattern").String()
	filePid     = kingpin.Flag("file-pid", "Check against a specific PID").String()
	vsz         = kingpin.Flag("virtual-memory-size", "Trigger on a Virtual Memory size is bigger than this").Int64()
	rss         = kingpin.Flag("resident-set-size", "Trigger on a Resident Set size is bigger than this").Int64()
	pcpu        = kingpin.Flag("proportional-set-size", "Trigger on a Proportional Set Size is bigger than this").Float()
	threadCount = kingpin.Flag("thread-count", "Trigger on a Thread Count is bigger than this").Int()
	state       = kingpin.Flag("state", "Trigger on a specific state, example: Z for zombie").String()
	user        = kingpin.Flag("user", "Trigger on a specific user").String()
	esecOver    = kingpin.Flag("esec-over", "Match processes that older that this, in SECONDS").Int()
	esecUnder   = kingpin.Flag("esec-under", "Match process that are younger than this, in SECONDS").Int()
	cpuOver     = kingpin.Flag("cpu-over", "Match processes cpu time that is older than this, in SECONDS").Int()
	cpuUnder    = kingpin.Flag("cpu-under", "Match processes cpu time that is younger than this, in SECONDS").Int()
)

func main() {
	kingpin.Version("1.0.0")
	kingpin.Parse()

	procs := getProcs()
	procs.filterPid(*filePid)
	procs.filterSelf(*matchSelf)
	procs.filterPattern(*pattern)
	procs.filterVsz(*vsz)
	procs.filterRss(*rss)
	procs.filterPcpu(*pcpu)
	procs.filterThreadCount(*threadCount)
	procs.filterEsecUnder(*esecUnder)
	procs.filterEsecOver(*esecOver)
	procs.filterCpuUnder(*cpuUnder)
	procs.filterCpuOver(*cpuOver)
	procs.filterState(*state)
	procs.filterUser(*user)

	count, message := procs.summary()

	status := &nagios.NagiosStatus{}
	switch {
	case *critUnder != 0 && count < *critUnder:
		status.Value = nagios.NAGIOS_CRITICAL
	case *critOver != 0 && count > *critOver:
		status.Value = nagios.NAGIOS_CRITICAL
	case *warnUnder != 0 && count < *warnUnder:
		status.Value = nagios.NAGIOS_WARNING
	case *warnOver != 0 && count > *warnOver:
		status.Value = nagios.NAGIOS_WARNING
	default:
		status.Value = nagios.NAGIOS_OK
	}
	status.Message = message
	nagios.ExitWithStatus(status)
}

func getProcs() procMaps {
	lines := readLines("ps axwwo user,pid,vsz,rss,pcpu,nlwp,state,etime,time,command")
	procs := make(procMaps, len(lines))
	for i, line := range lines {
		proc := toMap(line, "user", "pid", "vsz", "rss", "pcpu", "nlwp", "state", "etime", "time", "command")
		procs[i] = proc
	}
	return procs
}

func readLines(command string) []string {
	cmdArr := strings.Split(command, " ")
	cmd := exec.Command(cmdArr[0], cmdArr[1:]...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		nagios.Unknown(err.Error())
	}
	lines := out.String()

	lineArr := strings.Split(lines, "\n")
	if len(lineArr) > 1 {
		return lineArr[1 : len(lineArr)-1]
	} else {
		return []string{}
	}
}

func toMap(line string, keys ...string) procMap {
	ps := make(procMap)

	rxp := regexp.MustCompile("\\ +")
	lineArr := rxp.Split(line, 10)

	for i := 0; i < len(lineArr); i++ {
		key := keys[i]
		val := lineArr[i]
		ps[key] = val
	}

	return ps
}

func readPid(filePid string) int64 {
	if filePid == "" {
		return 0
	}

	if _, err := os.Stat(filePid); err != nil {
		nagios.Unknown("could not read pid file " + filePid)
	}

	dat, err := ioutil.ReadFile(filePid)
	if err != nil {
		nagios.Unknown("could not read pid file " + filePid)
	}

	pidStr := strings.TrimSpace(string(dat))
	pid, err := strconv.ParseInt(pidStr, 10, 64)
	if err != nil {
		nagios.Unknown("could not read pid file " + filePid)
	}

	return pid
}

func timeToSec(etime string) int {
	rxp, err := regexp.Compile("((\\d+)-)?((\\d\\d):)?(\\d\\d):(\\d\\d)")
	if err != nil {
		nagios.Unknown(err.Error())
	}
	matches := rxp.FindStringSubmatch(etime)
	multipier := []int{0, 0, 86400, 0, 3600, 60, 1}
	seconds := 0
	for i, match := range matches {
		if val, err := strconv.Atoi(match); err == nil {
			seconds += val * multipier[i]
		}
	}
	return seconds
}

func (pms *procMaps) reject(rf rejectFunc) {
	newPms := make(procMaps, 0)
	for _, pm := range *pms {
		if !rf(pm) {
			newPms = append(newPms, pm)
		}
	}
	*pms = newPms
}

func (pms *procMaps) filterPid(filePid string) {
	if filePid := readPid(filePid); filePid != 0 {
		pms.reject(func(p procMap) bool {
			if pid, err := strconv.ParseInt(p["pid"], 10, 64); err != nil {
				nagios.Unknown(err.Error())
			} else {
				return pid == filePid
			}
			return false
		})
	}
}

func (pms *procMaps) filterSelf(matchSelf bool) {
	if !matchSelf {
		pms.reject(func(p procMap) bool {
			procPid, err := strconv.Atoi(p["pid"])
			if err != nil {
				nagios.Unknown(err.Error())
			}
			pid := os.Getpid()
			return procPid == pid
		})
	}
}

func (pms *procMaps) filterPattern(pattern string) {
	if pattern != "" {
		rxp, err := regexp.Compile(pattern)
		if err != nil {
			nagios.Unknown(err.Error())
		}
		pms.reject(func(p procMap) bool {
			return !rxp.MatchString(p["command"])
		})
	}
}

func (pms *procMaps) filterVsz(vsz int64) {
	if vsz > 0 {
		pms.reject(func(p procMap) bool {
			procVsz, err := strconv.ParseInt(p["vsz"], 10, 64)
			if err != nil {
				nagios.Unknown(err.Error())
			}
			return procVsz > vsz
		})
	}
}

func (pms *procMaps) filterRss(rss int64) {
	if rss > 0 {
		pms.reject(func(p procMap) bool {
			procRss, err := strconv.ParseInt(p["rss"], 10, 64)
			if err != nil {
				nagios.Unknown(err.Error())
			}
			return procRss > rss
		})
	}
}

func (pms *procMaps) filterPcpu(pcpu float64) {
	if pcpu > 0 {
		pms.reject(func(p procMap) bool {
			procPcpu, err := strconv.ParseFloat(p["pcpu"], 64)
			if err != nil {
				nagios.Unknown(err.Error())
			}
			return procPcpu > pcpu
		})
	}
}

func (pms *procMaps) filterThreadCount(threadCount int) {
	if threadCount > 0 {
		pms.reject(func(p procMap) bool {
			procTc, err := strconv.Atoi(p["thcount"])
			if err != nil {
				nagios.Unknown(err.Error())
			}
			return procTc > threadCount
		})
	}
}

func (pms *procMaps) filterEsecUnder(esecUnder int) {
	if esecUnder > 0 {
		pms.reject(func(p procMap) bool {
			procEsec := timeToSec(p["etime"])
			return procEsec >= esecUnder
		})
	}
}

func (pms *procMaps) filterEsecOver(esecOver int) {
	if esecOver > 0 {
		pms.reject(func(p procMap) bool {
			procEsec := timeToSec(p["etime"])
			return procEsec <= esecOver
		})
	}
}

func (pms *procMaps) filterCpuUnder(cpuUnder int) {
	if cpuUnder > 0 {
		pms.reject(func(p procMap) bool {
			procCsec := timeToSec(p["time"])
			return procCsec >= cpuUnder
		})
	}
}

func (pms *procMaps) filterCpuOver(cpuOver int) {
	if cpuOver > 0 {
		pms.reject(func(p procMap) bool {
			procCsec := timeToSec(p["time"])
			return procCsec <= cpuOver
		})
	}
}

func (pms *procMaps) filterState(state string) {
	if state != "" {
		pms.reject(func(p procMap) bool {
			return !strings.Contains(state, p["state"])
		})
	}
}

func (pms *procMaps) filterUser(user string) {
	if user != "" {
		pms.reject(func(p procMap) bool {
			return !strings.Contains(user, p["user"])
		})
	}
}

func (pms *procMaps) summary() (count int, msg string) {
	msg = fmt.Sprintf("Found %d matching processes", len(*pms))
	if *pattern != "" {
		msg += fmt.Sprintf("; cmd /%s/", *pattern)
	}
	if *state != "" {
		msg += fmt.Sprintf("; state %s", *state)
	}
	if *user != "" {
		msg += fmt.Sprintf("; user %s", *user)
	}
	if *vsz > 0 {
		msg += fmt.Sprintf("; vsz < %d", *vsz)
	}
	if *rss > 0 {
		msg += fmt.Sprintf("; rss < %d", *rss)
	}
	if *pcpu > 0 {
		msg += fmt.Sprintf("; pcpu < %0.1f", *pcpu)
	}
	if *threadCount > 0 {
		msg += fmt.Sprintf("; thcount < %d", *threadCount)
	}
	if *esecUnder > 0 {
		msg += fmt.Sprintf("; esec < %d", *esecUnder)
	}
	if *esecOver > 0 {
		msg += fmt.Sprintf("; esec > %d", *esecOver)
	}
	if *cpuUnder > 0 {
		msg += fmt.Sprintf("; csec < %d", *cpuUnder)
	}
	if *cpuOver > 0 {
		msg += fmt.Sprintf("; csec > %d", *cpuOver)
	}
	if *filePid != "" {
		msg += fmt.Sprintf("; pid %s", *filePid)
	}

	if *metric != "" {
		for _, p := range *pms {
			if val, err := strconv.Atoi(p[*metric]); err == nil {
				count += val
			}
		}
	} else {
		count = len(*pms)
	}
	return
}
