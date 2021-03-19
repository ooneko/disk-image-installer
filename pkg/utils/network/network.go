package network

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
)

func GetBIOSDevName(adapter string) (string, error) {
	out := bytes.Buffer{}
	cmd := exec.Command("biosdevname", "-i", adapter)
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			code := ws.ExitStatus()
			switch code {
			case 127:
				return "", fmt.Errorf("executable 'biosdevname' not found")
			case 2:
				return "", fmt.Errorf("system BIOS does not provide naming information")
			case 4:
				return "", fmt.Errorf("the system is a virtual machine")
			}
		}
	}
	return out.String(), nil
}

func HasCarrier(adapter string) (bool, error) {
	data, err := getInterfaceInfo(adapter, "carrier")
	if err != nil {
		return false, err
	}
	hascarrier, err := strconv.Atoi(data)
	if err != nil {
		return false, err
	}
	if hascarrier == 1 {
		log.Printf("adapter %s has carrier", adapter)
		return true, nil
	}
	return false, nil
}

func GetMacAddr(adapter string) (string, error) {
	data, err := getInterfaceInfo(adapter, "address")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func GetSpeed(adapter string) (int, error) {
	data, err := getInterfaceInfo(adapter, "speed")
	if err != nil {
		return -1, err
	}
	speed, err := strconv.Atoi(string(data))
	if err != nil {
		return -1, fmt.Errorf("strconv.Atoi %s: %v", string(data), err)
	}
	return speed, nil
}

func getInterfaceInfo(adapter, attr string) (string, error) {
	f := fmt.Sprintf("/sys/class/net/%s/%s", adapter, attr)
	data, err := ioutil.ReadFile(f)
	if err != nil {
		return "", fmt.Errorf("read %s: %v", f, err)
	}
	return strings.TrimSuffix(string(data), "\n"), nil
}

func IsDevice(adaper string) bool {
	if _, err := os.Stat(fmt.Sprintf("/sys/class/net/%s/device/", adaper)); os.IsNotExist(err) {
		return false
	}
	return true
}
