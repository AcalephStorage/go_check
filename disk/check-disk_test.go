package main

import (
	"testing"

	"github.com/AcalephStorage/go_check/Godeps/_workspace/src/github.com/newrelic/go_nagios"
)

const sampleData = `Filesystem     Type 1024-blocks     Used Available Capacity Mounted on
/dev/xvda1     ext4    20511356  3854292  15717948      20% /
/dev/xvdb      ext3    38565344 15262992  21336684      42% /mnt
/dev/xvdd1     xfs     41819168 15450200  26368968      37% /var/lib/ceph/osd/ceph-0
/dev/xvdf1     xfs     41819168 12089480  29729688      29% /var/lib/ceph/osd/ceph-2
/dev/xvde1     xfs     41819168 14816796  27002372      36% /var/lib/ceph/osd/ceph-1
/dev/rbd1      xfs      1011008   122792    888216      13% /opt/acaleph-internal/mysql
/dev/rbd2      xfs      1011008    44928    966080       5% /opt/acaleph-internal/redis
/dev/rbd3      xfs     30704992    35712  30669280       1% /opt/acaleph-internal/graphite
/dev/rbd4      xfs     30704992  3746240  26958752      13% /opt/acaleph-internal/logstash
/dev/rbd5      xfs     30704992  2785432  27919560      10% /opt/acaleph-internal/influxdb`

func TestParseOutput(t *testing.T) {
	expectedDevice := &diskResult{
		filesystem: "/dev/xvda1",
		deviceType: "ext4",
		blocks:     20511356,
		available:  15717948,
		used:       3854292,
		capacity:   20,
		mounted:    "/",
	}

	devices, outputText := parseResult(sampleData)

	if len(devices) != 10 || outputText == "" {
		t.Errorf("expected 10 devices, %d parsed.", len(devices))
	}
	sample := devices[0]
	if sample.filesystem != expectedDevice.filesystem {
		t.Errorf("expected file system is %s, filesystem is %s.", expectedDevice.filesystem, sample.filesystem)
	}
	if sample.deviceType != expectedDevice.deviceType {
		t.Errorf("expected device type is %s, device type is %s.", expectedDevice.deviceType, sample.deviceType)
	}
	if sample.blocks != expectedDevice.blocks {
		t.Errorf("expected blocks are %d, device type are %d.", expectedDevice.blocks, sample.blocks)
	}
	if sample.used != expectedDevice.used {
		t.Errorf("expected used is %d, used is %d.", expectedDevice.used, sample.used)
	}
	if sample.available != expectedDevice.available {
		t.Errorf("expected available is %d, device type is %d.", expectedDevice.available, sample.available)
	}
	if sample.capacity != expectedDevice.capacity {
		t.Errorf("expected capacity type is %d, capacity is %d.", expectedDevice.capacity, sample.capacity)
	}
}

func TestSummarizePass(t *testing.T) {
	devices := []*diskResult{
		&diskResult{
			capacity: 20,
		},
		&diskResult{
			capacity: 20,
		},
		&diskResult{
			capacity: 20,
		},
	}
	crit, warn, probs := summarize(devices, 90, 80)
	if crit > 0 || warn > 0 || probs != "" {
		t.Error("check should have passed but some haved failed:", devices)
	}
}

func TestSummarizeWarn(t *testing.T) {
	devices := []*diskResult{
		&diskResult{
			capacity:   85,
			usage:      system,
			filesystem: "/dev/sdx",
		},
		&diskResult{
			capacity: 20,
		},
		&diskResult{
			capacity: 20,
		},
	}
	crit, warn, probs := summarize(devices, 90, 80)
	if crit > 0 || warn != 1 || probs == "" {
		t.Error("Should only have 1 warning and 2 passing", devices)
	}
}

func TestSummarizeFail(t *testing.T) {
	devices := []*diskResult{
		&diskResult{
			capacity:   85,
			usage:      system,
			filesystem: "/dev/sdx",
		},
		&diskResult{
			capacity:   95,
			usage:      system,
			filesystem: "/dev/sdx",
		},
		&diskResult{
			capacity: 20,
		},
	}
	crit, warn, probs := summarize(devices, 90, 80)
	if crit != 1 || warn != 1 || probs == "" {
		t.Error("Should have 1 warning, 1 critical and 1 passing", devices)
	}
}

func TestDoCheckPass(t *testing.T) {
	status := doCheck(0, 0, "", "")
	if status.Value != nagios.NAGIOS_OK {
		t.Error("status should be OK")
	}
}

func TestDoCheckWarn(t *testing.T) {
	status := doCheck(0, 4, "", "")
	if status.Value != nagios.NAGIOS_WARNING {
		t.Error("status should be WARNING")
	}
}

func TestDoCheckFail(t *testing.T) {
	status := doCheck(1, 2, "", "")
	if status.Value != nagios.NAGIOS_CRITICAL {
		t.Error("status should be CRITICAL")
	}
}
