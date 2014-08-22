package main

import "fmt"
import "io/ioutil"
import "strings"
import "strconv"

type load struct {
	one float32
	five float32
	fifteen float32
}

func checkLoad(warnLevel, critLevel string) (status checkStatus, output string) {
	warnLoad := toLoad(warnLevel)
	critLoad := toLoad(critLevel)

	data,_ := ioutil.ReadFile("/proc/loadavg")
	loadavgData := string(data)
	rawData := strings.Join(strings.Fields(loadavgData)[0:3], ",")
	load := toLoad(rawData)

	isCrit := threshold(load, critLoad)
	isWarn := threshold(load, warnLoad)

	switch {
	case isCrit:
		status = StatusCrit
	case isWarn:
		status = StatusWarn
	default:
		status = StatusOk
	}

	output = fmt.Sprintf("CheckLoad %v: %0.2f, %0.2f, %0.2f", status.string(), load.one, load.five, load.fifteen)

	return status, output
}

func toLoad(data string) *load {
	arr := strings.Split(data, ",")
	return &load{
		one: toFloat(arr[0]), 
		five: toFloat(arr[1]),
		fifteen: toFloat(arr[2]),
	}
}

func toFloat(str string) float32 {
	f,_ := strconv.ParseFloat(str, 32)
	return float32(f)
}

func threshold(actualLoad, compareLoad *load) bool {
	oneOverThreshold := actualLoad.one >= compareLoad.one
	fiveOverThreshold := actualLoad.five >= compareLoad.five
	fifteenOverThreshold := actualLoad.fifteen >= compareLoad.fifteen
	return  oneOverThreshold || fiveOverThreshold || fifteenOverThreshold
}