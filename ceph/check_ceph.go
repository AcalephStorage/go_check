package main

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"time"

	"os/exec"

	"github.com/newrelic/go_nagios"
	"gopkg.in/alecthomas/kingpin.v1"
)

var (
	keyring     = kingpin.Flag("keyring", "Path to cephx authentication keyring file").String()
	monitor     = kingpin.Flag("monitor", "Optional monitor IP").String()
	cluster     = kingpin.Flag("cluster", "Optional cluster name").String()
	timeout     = kingpin.Flag("timeout", "Timeout").Default("10").Int()
	ignoreFlags = kingpin.Flag("ignore-flags", "Optional ceph warning flags to ignore").String()
	detailed    = kingpin.Flag("detailed", "Show ceph health detail on warns/errors (verbose!)").Bool()
	osdTree     = kingpin.Flag("osd-tree", "Show OSD tree on warns/errors (verbose!)").Bool()
)

func main() {
	kingpin.Version("1.0.0")
	kingpin.Parse()
	checkCeph()
}

func checkCeph() {
	output := runCmd("ceph health")
	if !strings.HasPrefix(output, "HEALTH_OK") {
		output = filterIgnoredWarnings(output)
	}

	if strings.HasPrefix(output, "HEALTH_OK") {
		nagios.Ok(output)
	}

	if *detailed {
		output = runCmd("ceph health detailed")
	}

	if *osdTree {
		output += runCmd("ceph osd tree")
	}

	if strings.HasPrefix(output, "HEALTH_WARN") {
		nagios.Warning(output)
	} else {
		nagios.Critical(errors.New(output))
	}
}

func runCmd(cmd string) (result string) {
	if *cluster != "" {
		cmd += " --cluster=" + *cluster
	}
	if *keyring != "" {
		cmd += " -k " + *keyring
	}
	if *monitor != "" {
		cmd += " -m " + *monitor
	}

	command := strings.Split(cmd, " ")
	cephCmd := exec.Command(command[0], command[1:]...)
	var out bytes.Buffer
	cephCmd.Stdout = &out
	cephCmd.Stderr = &out

	if err := cephCmd.Start(); err != nil {
		nagios.Unknown(err.Error())
	}

	done := make(chan error)
	go func() {
		done <- cephCmd.Wait()
	}()

	select {
	case <-time.After(time.Duration(*timeout) * time.Second):
		nagios.Critical(errors.New("Execution timed out"))
	case err := <-done:
		if err != nil {
			nagios.Unknown(err.Error())
		}
	}
	return out.String()
}

func filterIgnoredWarnings(result string) string {
	// remove HEALTH_WARN
	r := strings.Replace(result, "HEALTH_WARN", "", -1)

	// remove flag set
	rxp, err := regexp.Compile("\\ ?flag\\(s\\) set")
	if err != nil {
		nagios.Unknown(err.Error())
	}
	r = rxp.ReplaceAllString(r, "")

	// remove new lines
	r = strings.Replace(r, "\n", "", -1)

	for _, flag := range strings.Split(*ignoreFlags, ",") {
		exp := ",?" + flag + ",?"
		rxp, err := regexp.Compile(exp)
		if err != nil {
			nagios.Unknown(err.Error())
		}
		r = rxp.ReplaceAllString(r, "")
	}

	if len(r) == 0 {
		return strings.Replace(result, "HEALTH_WARN", "HEALTH_OK", -1)
	} else {
		return result
	}

}
