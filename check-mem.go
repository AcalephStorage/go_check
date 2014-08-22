package main

import "fmt"
import "bytes"
import "strings"
import "strconv"
import "os/exec"

func checkMem(warnLevel, critLevel int) (status checkStatus, output string) {
	cmd := exec.Command("free", "-m")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
	result := out.String()

	var total int
	var available int

	lines := strings.Split(result, "\n")
	for _,line := range lines {
		if strings.HasPrefix(line, "Mem:") {
			total,_ = strconv.Atoi(strings.Fields(line)[1])
		}
		if strings.HasPrefix(line, "-/+ buffers/cache") {
			available,_ = strconv.Atoi(strings.Fields(line)[3])
		}
	}

	availablePercentage := int(float64(available)/float64(total) * float64(100))
	status = StatusOk
	switch {
	case availablePercentage <= critLevel:
		status = StatusCrit
	case availablePercentage <= warnLevel:
		status = StatusWarn
	}

	output = fmt.Sprintf("Check Mem %v: total=%vmB available=%vmB | %v%% Available Memory left.", status.string(), total, available, availablePercentage)

	return status, output
}