package hardware

import (
	"net"
	"os"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	netutil "diskimage-installer/pkg/utils/network"
)

type NetworkInterface struct {
	Name        string
	BIOSDevName string
	MACAddress  string
	HasCarrier  bool
	Speed       int
}

type HardWareManager struct {
	logger *zap.Logger
}

func NewHardWareManager(logger *zap.Logger) *HardWareManager {
	return &HardWareManager{
		logger: logger,
	}
}

func (m *HardWareManager) GetBootMode() string {
	if _, err := os.Stat("/sys/fireware/efi"); os.IsNotExist(err) {
		return "bios"
	}
	return "efi"
}

func (m *HardWareManager) ListNetworkInterface() ([]NetworkInterface, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, errors.Wrap(err, "net.Interfaces():")
	}

	devices := []NetworkInterface{}
	for _, i := range interfaces {
		if !netutil.IsDevice(i.Name) {
			continue
		}
		n, err := m.GetNetworkInterfaceInfo(i.Name)
		if err != nil {
			return nil, err
		}
		devices = append(devices, n)
	}
	return devices, nil

}

func (m *HardWareManager) GetNetworkInterfaceInfo(adapter string) (NetworkInterface, error) {
	macaddr, err := netutil.GetMacAddr(adapter)
	if err != nil {
		return NetworkInterface{}, errors.Wrap(err, "netutil.GetMacAddr:")
	}
	biosdevname, err := netutil.GetBIOSDevName(adapter)
	if err != nil {
		m.logger.Sugar().Warnf("netutil.GetBIOSDevName: %v", err)
	}
	speed, err := netutil.GetSpeed(adapter)
	if err != nil {
		return NetworkInterface{}, errors.Wrap(err, "netutil.GetSpeed")
	}
	hascarrier, err := netutil.HasCarrier(adapter)
	if err != nil {
		return NetworkInterface{}, errors.Wrap(err, "netutil.HasCarrier")
	}
	n := NetworkInterface{
		Name:        adapter,
		BIOSDevName: biosdevname,
		MACAddress:  macaddr,
		HasCarrier:  hascarrier,
		Speed:       speed,
	}
	return n, nil
}
