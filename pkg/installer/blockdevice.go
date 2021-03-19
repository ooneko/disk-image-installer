package installer

import (
	"encoding/json"
	"fmt"
	"strings"

	"diskimage-installer/pkg/utils"
	diskutils "diskimage-installer/pkg/utils/disk"

	"github.com/pkg/errors"
)

type BlockDevice struct {
	Name          string `json:"name"`
	Kname         string `json:"kname"`
	Model         string `json:"model"`
	Size          string `json:"size"`
	UUID          string `json:"uuid"`
	Rotational    string `json:"rota"`
	Type          string `json:"type"`
	Hctl          string `json:"hctl"`
	Serial        string `json:"serial"`
	WWN           string `json:"wwn"`
	Vendor        string `json:"vendor"`
	InterfaceType string `json:"tran"`

	DiskType string
}

type Partition struct {
	Name   string
	Size   string
	FsType string
}

func listAllBlockDevice() ([]BlockDevice, error) {
	if err := diskutils.UdevSettle(); err != nil {
		return nil, fmt.Errorf("udevSettle: %v", err)
	}
	out, err := utils.RunCommand("lsblk", "-O", "-J")
	if err != nil {
		return nil, errors.Wrapf(err, "lsblk: %v", out)
	}
	data := map[string][]BlockDevice{}
	if err := json.Unmarshal([]byte(out), &data); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}

	result := []BlockDevice{}
	for _, d := range data["blockdevices"] {
		if d.Type != "disk" {
			continue
		}
		// Sometimes model has whitespace, it should be trimed
		d.Model = strings.TrimSpace(d.Model)
		d.Name = "/dev/" + d.Name

		if d.Rotational == "0" {
			d.DiskType = "ssd"
		} else {
			d.DiskType = "hdd"
		}
		result = append(result, d)
	}
	return result, nil
}

func listPartitions(device BlockDevice) ([]Partition, error) {
	out, err := utils.RunCommand("lsblk", "-O", "-J")
	if err != nil {
		return nil, errors.Wrapf(err, "lsblk: %v", out)
	}

	blockdevices := map[string][]interface{}{}
	if err := json.Unmarshal([]byte(out), &blockdevices); err != nil {
		return nil, fmt.Errorf("json unmarshal: %v", err)
	}
	for _, d := range blockdevices["blockdevices"] {
		attr := d.(map[string]interface{})
		if attr["kname"] != device.Kname {
			continue
		}
		children, ok := attr["children"]
		if !ok {
			return nil, nil
		}
		result := []Partition{}
		for _, item := range children.([]interface{}) {
			kv := item.(map[string]interface{})
			p := Partition{}
			name := kv["name"].(string)

			p.Name = "/dev/" + name
			p.Size = kv["size"].(string)
			p.FsType = kv["fstype"].(string)
			result = append(result, p)
		}
		return result, nil
	}
	return nil, nil
}

func convertListPartitionToMap(partitions []Partition) map[string]Partition {
	result := make(map[string]Partition)
	for _, p := range partitions {
		result[p.Name] = p
	}
	return result
}
