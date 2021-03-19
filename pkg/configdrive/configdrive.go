package configdrive

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"diskimage-installer/pkg/config"
	"diskimage-installer/pkg/hardware"
)

var metaDataVersions = []string{
	"2012-08-10",
	"2015-10-15",
	"latest",
}

type MetaData struct {
	AvailabilityZone string            `json:"availability_zone"`
	Files            []string          `json:"files,omitempty"`
	Hostname         string            `json:"hostname"`
	Name             string            `json:"name"`
	Meta             map[string]string `json:"meta,omitempty"`
	PublickKeys      map[string]string `json:"public_keys"`
	UUID             string            `json:"uuid"`
}

type NetworkMetaData struct {
	Links    []Link    `json:"links"`
	Networks []Network `json:"networks"`
	Services []Service `json:"services"`
}

type LinkType string

const (
	LinkTypeVlan LinkType = "vlan"
	LinkTypePhy  LinkType = "phy"
	LinkTypeBond LinkType = "bond"
)

type Link struct {
	ID             string   `json:"id,omitempty"`
	Type           LinkType `json:"type"`
	VlanID         int      `json:"vlan_id,omitempty"`
	VlanLink       string   `json:"vlan_link,omitempty"`
	VlanMac        string   `json:"vlan_mac_address,omitempty"`
	BondMaster     string   `json:"bond-master,omitempty"`
	BondLinks      []string `json:"bond_links,omitempty"`
	BondMode       string   `json:"bond_mode,omitempty"`
	BondHashPolicy string   `json:"bond_xmit_hash_policy,omitempty"`
	Bondmiimon     int      `json:"bond_miimon,omitempty"`
	MacAddress     string   `json:"ethernet_mac_address,omitempty"`
	MTU            string   `json:"mtu,omitempty"`
}

type NetworkType string

const (
	NetworkTypeIPv4     NetworkType = "static"
	NetworkTypeIPv4DHCP NetworkType = "dhcp4"
)

type Network struct {
	ID        string      `json:"id"`
	Link      string      `json:"link"`
	Type      NetworkType `json:"type"`
	IPAddress string      `json:"ip_address"`
	Netmask   string      `json:"netmask"`
	DNS       []string    `json:"dns_nameservers,omitempty"`
	Routes    []Route     `json:"routes"`
}

type Route struct {
	Network string `json:"network"`
	Netmask string `json:"netmask"`
	Gateway string `json:"gateway"`
}

type ServiceType string

const ServiceTypeDNS ServiceType = "dns"

type Service struct {
	Type    ServiceType `json:"type"`
	Address string      `json:"address"`
}

// Generate generate config drive and return the path of it.
func Generate(networkInterfaces []hardware.NetworkInterface, nodeconfig config.Node, logger *zap.Logger) (string, error) {
	metadata := MetaData{
		Hostname: nodeconfig.Name,
		Name:     nodeconfig.Name,
		UUID:     uuid.NewString(),
	}
	networkData, err := getNetworkMetaData(networkInterfaces, nodeconfig)
	if err != nil {
		return "", err
	}
	metadataByte, err := json.MarshalIndent(metadata, "", "\t")
	if err != nil {
		return "", err
	}
	networkDataByte, err := json.MarshalIndent(networkData, "", "\t")
	if err != nil {
		return "", err
	}

	dir, err := ioutil.TempDir("/tmp", "configdriver-")
	if err != nil {
		return "", err
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			logger.Sugar().Warnf("remove %s failed", dir)
		}
	}()

	contentDir := filepath.Join(dir, "openstack", "content")
	if err := os.MkdirAll(contentDir, 0755); err != nil {
		return "", err
	}

	for _, version := range metaDataVersions {
		versiondir := filepath.Join(dir, "openstack", version)
		if err := os.MkdirAll(versiondir, 0755); err != nil {
			return "", err
		}

		metadataFile := path.Join(versiondir, "meta_data.json")
		if err := ioutil.WriteFile(metadataFile, metadataByte, 0644); err != nil {
			return "", err
		}

		networkDataFile := path.Join(versiondir, "network_data.json")
		if err := ioutil.WriteFile(networkDataFile, networkDataByte, 0644); err != nil {
			return "", err
		}
	}

	configdriverISO := path.Join("/tmp", fmt.Sprintf("%s%s.iso", "configdrive-", metadata.UUID))
	args := []string{
		"-R",
		"-V",
		"config-2",
		"-o",
		configdriverISO,
		dir,
	}
	cmd := exec.Command("mkisofs", args...)
	_, err = cmd.Output()
	if err != nil {
		return "", err
	}
	logger.Sugar().Infof("iso is writen to %s\n", configdriverISO)
	return configdriverISO, nil
}
