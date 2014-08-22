package main

import "fmt"
import "os"
import "strconv"
import "github.com/docopt/docopt-go"

const(
  appName = "acaleph-monitor"
  version = "1.0"
  StatusOk checkStatus = 0
  StatusWarn checkStatus = 1
  StatusCrit checkStatus = -1

  usage = `Acaleph Monitoring.

Usage:
  acaleph-stat [ceph]
  acaleph-stat [cpu|mem|load|disk|ntp] --warn <warn> --crit <crit>
  acaleph-stat proc <process_name>
  acaleph-stat --help
  acaleph-stat --version

Options:
  cpu               Show the cpu status.
  mem               Show the memory status.
  load              Show the load status.
  disk              Show the disk status. This includes OSDs and RBDs.
  ntp               Show the ntp status.
  --help            Show this screen.
  --version         Show version.
  --warn <warn>     value to be considered as warning state
  --crit <crit>     value to be considered fatal
`
)


type checkStatus int

func (cs checkStatus) string() string {
  switch cs {
  case StatusOk:
    return "OK"
  case StatusWarn:
    return "WARNING"
  case StatusCrit:
    return "CRITICAL"
  default:
    return "UNKNOWN"
  }
}

func (cs checkStatus) int() int {
  return int(cs)
}

func main() {
  var status checkStatus
  var output string

	args,_ := docopt.Parse(usage, nil, true, appName+" "+version, false)
	if args["cpu"].(bool) {
    warnLevel,_ := strconv.ParseFloat(args["--warn"].(string), 64)
    critLevel,_ := strconv.ParseFloat(args["--crit"].(string), 64)
		status, output = checkCpu(warnLevel, critLevel)
	}

  if args["mem"].(bool) {
    warnLevel,_ := strconv.Atoi(args["--warn"].(string))
    critLevel,_ := strconv.Atoi(args["--crit"].(string))
    status, output = checkMem(warnLevel, critLevel)
  }

  if args["load"].(bool) {
    warnLevel := args["--warn"].(string)
    critLevel := args["--crit"].(string)
    status, output = checkLoad(warnLevel, critLevel)
  }

  if args["disk"].(bool) {
    warnLevel,_ := strconv.Atoi(args["--warn"].(string))
    critLevel,_ := strconv.Atoi(args["--crit"].(string))
    status, output = checkDisk(warnLevel, critLevel)
  }

  if args["ntp"].(bool) {
    warnLevel,_ := strconv.ParseFloat(args["--warn"].(string), 64)
    critLevel,_ := strconv.ParseFloat(args["--crit"].(string), 64)
    status, output = checkNtp(warnLevel, critLevel)
  }

  fmt.Println(output)
  os.Exit(status.int())
}
