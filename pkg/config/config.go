package config

import (
	"reflect"
)

type Node struct {
	Name         string            `json:"name" yaml:"name"`
	IPMI         IPMIInfo          `json:"ipmi" yaml:"ipmi"`
	Network      NetworkInfo       `json:"network" yaml:"network"`
	RootDevice   map[string]string `json:"root_device" yaml:"root_device"`
	SerialNumber string            `json:"sn" yaml:"sn"`
	ImageInfo    *ImageInfo        `json:"image_info" yaml:"image_info"`
	RaidConfig   *RaidConfig       `json:"raid" yaml:"raid"`
}

type DiskType string

var DiskTypeHDD DiskType = "hdd"
var DiskTypeSSD DiskType = "ssd"

type InterfaceType string

var InterfaceSAS InterfaceType = "sas"
var InterfaceSCSI InterfaceType = "scsi"
var InterfaceSATA InterfaceType = "sata"

type RaidLevel string

var RaidlevelJBOD RaidLevel = "JBOD"
var RaidLevel0 RaidLevel = "0"
var RaidLevel1 RaidLevel = "1"
var RaidLevel5 RaidLevel = "5"
var RaidLevel6 RaidLevel = "6"
var RaidLevel10 RaidLevel = "1+0"
var RaidLevel50 RaidLevel = "5+0"
var RaidLevel60 RaidLevel = "6+0"

type RaidConfig struct {
	LogicalDisks []LogicalDisk `json:"logical_disks" yaml:"logical_disks"`
}

type LogicalDisk struct {
	RaidLevel             RaidLevel     `json:"raid_level" yaml:"raid_level"`
	SizeGB                *int          `json:"size_gb" yaml:"size_gb"`
	VolumeName            string        `json:"volume_name" yaml:"volume_name"`
	RootVolume            bool          `json:"is_root_volume" yaml:"is_root_volume"`
	DiskType              DiskType      `json:"disk_type" yaml:"disk_type"`
	InterfaceType         InterfaceType `json:"interface_type" yaml:"interface_type"`
	Controller            string        `json:"controller" yaml:"controller"`
	PhysicalDisks         []string      `json:"physical_disks" yaml:"physical_disks"`
	NumberOfPhysicalDisks int           `json:"number_of_physical_disks" yaml:"number_of_physical_disks"`
}

type IPMIInfo struct {
	Address   string `json:"address" yaml:"address"`
	Port      int    `json:"port" yaml:"port"`
	Username  string `json:"username" yaml:"username"`
	Password  string `json:"password" yaml:"password"`
	Cipher    int32  `json:"cipher" yaml:"cipher"`
	Interface string `json:"interface" yaml:"interface"`
}

type NetworkInfo struct {
	IPv4Address string   `json:"address" yaml:"address"`
	NetMask     string   `json:"netmask" yaml:"netmask"`
	Gateway     string   `json:"gateway" yaml:"gateway"`
	DNS         []string `json:"dns" yaml:"dns"`
	MTU         string   `json:"mtu" yaml:"mtu"`
	Bond        BondInfo `json:"bond" yaml:"bond"`
}

func (n NetworkInfo) IsEmpty() bool {
	return reflect.DeepEqual(n, NetworkInfo{})
}

type BondInfo struct {
	Mode       string   `json:"mode" yaml:"mode"`
	HashPolicy string   `json:"hash_policy" yaml:"hash_policy"`
	Miimon     int      `json:"miimon" yaml:"miimon"`
	Links      []string `json:"links" yaml:"links"`
	BondAll    bool     `json:"bond_all_interface" yaml:"bond_all_interface"`
}

func (b BondInfo) IsEmpty() bool {
	return reflect.DeepEqual(b, BondInfo{})
}

type ImageInfo struct {
	Image       string `json:"image" yaml:"image"`
	ImageURL    string `json:"image_url" yaml:"image_url"`
	DiskFormat  string `json:"disk_format" yaml:"disk_format"`
	MD5Checksum string `json:"md5" yaml:"md5"`
}

type Images map[string]*ImageInfo
