package main

import "bytes"
import "fmt"
import "strings"
import "strconv"
import "os/exec"


func checkNtp(warnLevel, critLevel float64) (status checkStatus, output string) {
	cmd := exec.Command("ntpq", "-c", "rv 0 offset")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
	result := strings.TrimSpace(out.String())
	offset,_ := strconv.ParseFloat(strings.Split(result, "=")[1], 64)

	switch {
	case offset >= critLevel || offset <= -critLevel:
		status = StatusCrit
	case offset >= warnLevel || offset <= -warnLevel:
		status = StatusWarn
	default:
		status = StatusOk
	}

	output = fmt.Sprintf("CheckNTP %v: Offset: %0.2f", status.string(), offset)
	return status, output
}