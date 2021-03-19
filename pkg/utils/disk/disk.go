package disk

import (
	"bytes"
	"fmt"
	"math/rand"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"diskimage-installer/pkg/utils"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

const (
	MaxConfigDriveSizeMB int = 64
	MaxMBRDiskSizeMB     int = 2097152
)

type PartitionType string

var GPT PartitionType = "GPT"
var MBR PartitionType = "MBR"
var NoPartition PartitionType = "NoPartition"
var Unknown PartitionType = "Unknown"
var PartProbeAttemps int = 5

func GetPartitionTableType(device string) (PartitionType, error) {
	if err := partProbe(device); err != nil {
		return Unknown, errors.Wrap(err, "getPartitionTableType:")
	}

	var out bytes.Buffer
	cmd := exec.Command("blkid", device, "--probe")
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return Unknown, err
	}

	data := strings.Split(out.String(), ": ")
	attr := data[len(data)-1]
	tag := parseDeviceAttr(attr)
	pttype, ok := tag["PTTYPE"]
	if !ok {
		return NoPartition, nil
	}
	if pttype == "gpt" {
		return GPT, nil
	}
	return MBR, nil
}

func parseDeviceAttr(attr string) map[string]string {
	result := map[string]string{}
	for _, a := range strings.Split(attr, " ") {
		kv := strings.Split(a, "=")
		result[kv[0]] = kv[1]
	}
	return result
}

func partProbe(device string) error {
	fn := func() error {
		if out, err := utils.RunCommand("partprobe", device); err != nil {
			return errors.Wrapf(err, "partprobe %s: %v", device, out)
		}
		return nil
	}
	return retry(PartProbeAttemps, 1*time.Second, fn)
}

func fixGPTStructs(device string) error {
	out, err := verifyDevice(device)
	if err != nil {
		return err
	}
	search_str := "it doesn't reside\nat the end of the disk"
	if !strings.Contains(out, search_str) {
		return nil
	}
	// move backup GTP structures to the end of the disk.
	cmd := exec.Command("sgdisk", "-e", device)
	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "sgdisk -e:")
	}
	return nil
}

func FixGTPPartition(device string) error {
	pttype, err := GetPartitionTableType(device)
	if err != nil {
		return errors.Wrap(err, "getPartitionTableType")
	}
	if pttype == GPT {
		return fixGPTStructs(device)
	}
	return nil
}

func UdevSettle() error {
	cmd := exec.Command("udevadm", "settle")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("udevadm settle: %v", err)
	}
	return nil
}

func DD(src, dest string) error {
	cmd := fmt.Sprintf("dd if=%s of=%s bs=%s oflag=sync", src, dest, "1M")
	zap.L().Sugar().Debug(cmd)
	command := strings.Split(cmd, " ")
	if out, err := utils.RunCommand(command[0], command[1:]...); err != nil {
		return fmt.Errorf("RunCommand: %s: %v", out, err)
	}
	return nil
}

func CreateGPTPartion(device, option string) error {
	var out bytes.Buffer
	cmd := exec.Command("sgdisk", "-n", option, device)
	cmd.Stderr = &out
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "sgdisk -n %s %s: %v", option, device, out.String())
	}
	return nil
}

func CreateMBRPartitionForConfigDrive(device string) error {
	startlimit := fmt.Sprintf("-%dMiB", MaxConfigDriveSizeMB)
	endlimit := "-0"
	toolarge, err := isDiskLargeThanMAX(device)
	if err != nil {
		return errors.Wrap(err, "isDiskLargeThanMAX:")
	}
	if toolarge {
		startlimit = strconv.Itoa(MaxMBRDiskSizeMB - MaxConfigDriveSizeMB - 1)
		endlimit = strconv.Itoa(MaxMBRDiskSizeMB - 1)
	}
	out, err := utils.RunCommand("parted", "-a", "optimal", "-s", "--", device,
		"mkpart", "primary", "fat32", startlimit, endlimit)
	if err != nil {
		return errors.Wrapf(err, "parted: %v", out)
	}
	if err := RescanDevice(device); err != nil {
		return errors.Wrapf(err, "rescanDevice(%s)", device)
	}
	return nil
}

func isDiskLargeThanMAX(device string) (bool, error) {
	out, err := utils.RunCommand("blockdev", "--getsize64", device)
	if err != nil {
		return false, errors.Wrap(err, out)
	}
	sizebytes := strings.TrimSpace(out)
	b, err := strconv.Atoi(sizebytes)
	if err != nil {
		return false, err
	}
	mb := b / 1024 / 1024
	if mb > MaxMBRDiskSizeMB {
		return true, nil
	}
	return false, nil
}

func RescanDevice(device string) error {
	if _, err := utils.RunCommand("sync"); err != nil {
		return err
	}
	if err := UdevSettle(); err != nil {
		return err
	}
	if err := partProbe(device); err != nil {
		return err
	}
	out, err := verifyDevice(device)
	if err != nil {
		return errors.Wrap(err, out)
	}
	return nil
}

func verifyDevice(device string) (string, error) {
	out, err := utils.RunCommand("sgdisk", "-v", device)
	if err != nil {
		return out, err
	}
	return out, err
}

func retry(attemps int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if attemps--; attemps > 0 {
			jitter := time.Duration(rand.Int63n(int64(sleep)))
			sleep = sleep + jitter/2
			time.Sleep(sleep)
			return retry(attemps, 2*sleep, f)
		}
		return err
	}
	return nil
}

func GetDiskUUID(device string) (string, error) {
	out, err := utils.RunCommand("hexdump", "-s", "440", "-n", "4", "-e", "\"0x%08x\"", device)
	if err != nil {
		return "", fmt.Errorf("hexdump: %v", err)
	}
	return out, nil
}
