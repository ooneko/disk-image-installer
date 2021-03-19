package installer

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"diskimage-installer/pkg/config"
	"diskimage-installer/pkg/configdrive"
	"diskimage-installer/pkg/hardware"
	"diskimage-installer/pkg/shell"
	"diskimage-installer/pkg/utils/disk"
	diskutils "diskimage-installer/pkg/utils/disk"
)

type ImgaeInstaller struct {
	config.Node
	hardwareManager *hardware.HardWareManager
	logger          *zap.Logger
}

func NewInstaller(node config.Node, logger *zap.Logger) *ImgaeInstaller {
	return &ImgaeInstaller{
		Node:            node,
		logger:          logger,
		hardwareManager: hardware.NewHardWareManager(logger),
	}
}

func (i *ImgaeInstaller) Write(rootDevice string) error {
	// Checksum of image
	_ = shell.WriteImage
	f, err := ioutil.TempFile(os.TempDir(), "diskimage-installer-")
	if err != nil {
		return fmt.Errorf("create tempfile: %v", err)
	}
	defer os.Remove(f.Name())
	if err := ioutil.WriteFile(f.Name(), shell.WriteImage, 0700); err != nil {
		return fmt.Errorf("write script to file %s: %v", f.Name(), err)
	}
	var out bytes.Buffer
	cmd := exec.Command("/bin/bash", f.Name(), i.ImageInfo.Image, rootDevice)
	cmd.Stdout = &out
	cmd.Stderr = &out
	i.logger.Sugar().Infof("write image %s to device %s", i.ImageInfo.Image, rootDevice)
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "write image: %s", out.String())
	}
	i.logger.Sugar().Infof("Write image successed\n %s", out.String())
	return nil
}

func (i *ImgaeInstaller) InstallOS() error {
	configdrivefile, err := i.genConfigDrive()
	if err != nil {
		return errors.Wrap(err, "ImgaeInstaller.genConfigDrive:")
	}

	if err := i.configRaidController(); err != nil {
		return err
	}

	rootDevice, err := i.getInstallDevice()
	if err != nil {
		return fmt.Errorf("getInstallDevice: %v", err)
	}
	if err := i.Write(rootDevice.Name); err != nil {
		return fmt.Errorf("install os: %v", err)
	}
	// Write config drive
	if err := i.CreateConfigDrivePartition(configdrivefile, rootDevice); err != nil {
		return errors.Wrap(err, "ImgaeInstaller.WriteConfigDrive:")
	}
	// Config boot
	uuid, err := disk.GetDiskUUID(rootDevice.Name)
	if err != nil {
		return errors.Wrapf(err, "disk.GetDiskUUID(%s)", rootDevice.Name)
	}
	i.logger.Sugar().Infof("root uuid: %s", uuid)
	return nil
}

func (i ImgaeInstaller) configRaidController() error {
	if i.RaidConfig == nil {
		return nil
	}
	return nil
}

func (i *ImgaeInstaller) genConfigDrive() (string, error) {
	if i.Network.IsEmpty() {
		return "", nil
	}
	networkinterfaces, err := i.hardwareManager.ListNetworkInterface()
	if err != nil {
		return "", errors.Wrap(err, "hardwareManager.ListNetworkInterface")
	}
	configdrivePath, err := configdrive.Generate(networkinterfaces, i.Node, i.logger)
	if err != nil {
		return "", errors.Wrap(err, "configdrive.Generate:")
	}

	return configdrivePath, nil
}

func (i *ImgaeInstaller) getInstallDevice() (BlockDevice, error) {
	devices, err := listAllBlockDevice()
	if err != nil {
		i.logger.Sugar().Error("listAllBlockDevice: %v", err)
		return BlockDevice{}, err
	}

	for _, d := range devices {
		if i.matchRootDevice(d, i.RootDevice) {
			i.logger.Sugar().Infof("found root disk is %s", d.Name)
			return d, nil
		}

	}
	return BlockDevice{}, fmt.Errorf("install Device not found")
}

func (i *ImgaeInstaller) matchRootDevice(d BlockDevice, root map[string]string) bool {
	for k, v := range root {
		if k == "name" && v == d.Name {
			return true
		}
		if k == "hctl" && v == d.Hctl {
			return true
		}
		if k == "uuid" && v == d.UUID {
			return true
		}
		if k == "size" && v == d.Size {
			return true
		}
	}
	return false
}

func (i *ImgaeInstaller) CreateConfigDrivePartition(configdrivefile string, device BlockDevice) error {
	if err := diskutils.RescanDevice(device.Name); err != nil {
		return errors.Wrap(err, "diskutils.RescanDevice")
	}
	info, err := os.Stat(configdrivefile)
	if err != nil {
		return err
	}
	partitions, err := listPartitions(device)
	if err != nil {
		return errors.Wrapf(err, "listPartitions(%s)", device.Name)
	}
	curPartitions := convertListPartitionToMap(partitions)
	// convert size to MiB
	configDriveSize := info.Size() / 1024 / 1024
	if int(configDriveSize) > diskutils.MaxConfigDriveSizeMB {
		return fmt.Errorf("config drive oversize: %s", configdrivefile)
	}
	i.logger.Sugar().Infof("Adding config drive partition to device %s", device.Name)
	pttype, err := diskutils.GetPartitionTableType(device.Name)
	if err != nil {
		return errors.Wrap(err, "diskutils.GetPartitionTableType:")
	}
	if pttype == diskutils.GPT {
		if err := diskutils.FixGTPPartition(device.Name); err != nil {
			return errors.Wrap(err, "diskutils.FixGTPPartition:")
		}
		option := fmt.Sprintf("0:-%dMB:0", diskutils.MaxConfigDriveSizeMB)
		i.logger.Sugar().Info("Creating GPT Partition")
		if err := diskutils.CreateGPTPartion(device.Name, option); err != nil {
			return errors.Wrap(err, "diskutils.CreateGPTPartion:")
		}
	} else {
		i.logger.Sugar().Info("Creating MBR Partition")
		if err := diskutils.CreateMBRPartitionForConfigDrive(device.Name); err != nil {
			i.logger.Sugar().Errorf("CreateMBRPartitionForConfigDrive: %v", err)
			return err
		}
	}

	partitions, err = listPartitions(device)
	if err != nil {
		return errors.Wrapf(err, "listPartitions(%s)", device.Name)
	}
	newPartitions := convertListPartitionToMap(partitions)

	configDrivePartition := Partition{}
	for k, p := range newPartitions {
		if _, ok := curPartitions[k]; ok {
			continue
		}
		configDrivePartition = p
	}
	i.logger.Sugar().Infof("writing configdrive to partition %s", configDrivePartition.Name)
	if err := diskutils.DD(configdrivefile, configDrivePartition.Name); err != nil {
		return errors.Wrap(err, "diskutils.DD")
	}
	return nil
}
