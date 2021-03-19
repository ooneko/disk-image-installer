package configdrive

import (
	"fmt"

	"diskimage-installer/pkg/config"
	"diskimage-installer/pkg/hardware"

	"github.com/pkg/errors"
)

func getNetworkMetaData(networkInterfaces []hardware.NetworkInterface, config config.Node) (NetworkMetaData, error) {
	if config.Network.Bond.IsEmpty() {
		data, err := processNoneBondNetwork(networkInterfaces, config)
		if err != nil {
			return NetworkMetaData{}, errors.Wrap(err, "ProcessNoneBondNetwork:")
		}
		return data, nil
	}
	data, err := processBondNetwork(networkInterfaces, config)
	if err != nil {
		return NetworkMetaData{}, errors.Wrap(err, "ProcessBondNetwork:")
	}
	return data, nil
}

func processBondNetwork(networkInterfaces []hardware.NetworkInterface, nodeConfig config.Node) (NetworkMetaData, error) {
	networkdata := NetworkMetaData{}

	links := []Link{}
	bondLinks := []string{}
	// add physical link
	if nodeConfig.Network.Bond.BondAll {
		for _, n := range networkInterfaces {
			if n.HasCarrier {
				links = append(links, Link{
					ID:         n.MACAddress,
					Type:       LinkTypePhy,
					MacAddress: n.MACAddress,
					MTU:        nodeConfig.Network.MTU,
					BondMaster: "bond0", // cloud-init names bond interface with "bond%d"
				})
				bondLinks = append(bondLinks, n.MACAddress)
			}
		}
	} else {
		interfaceMap := getNetworkInterfaceMap(networkInterfaces)
		for _, l := range nodeConfig.Network.Bond.Links {
			if i, ok := interfaceMap[l]; ok {
				if !i.HasCarrier {
					return networkdata, fmt.Errorf("interface %s has no carrier", i.Name)
				}
				links = append(links, Link{
					ID:         i.MACAddress,
					Type:       LinkTypePhy,
					MacAddress: i.MACAddress,
					MTU:        nodeConfig.Network.MTU,
					BondMaster: "bond0", // cloud-init names bond interface with "bond%d"
				})
				bondLinks = append(bondLinks, i.MACAddress)
			} else {
				return networkdata, fmt.Errorf("host has no interface %s", l)
			}
		}
	}
	// add bond link
	links = append(links, Link{
		ID:             "bond0",
		Type:           LinkTypeBond,
		BondMode:       nodeConfig.Network.Bond.Mode,
		BondHashPolicy: nodeConfig.Network.Bond.HashPolicy,
		Bondmiimon:     nodeConfig.Network.Bond.Miimon,
		BondLinks:      bondLinks,
	})
	networkdata.Links = links

	networks := []Network{
		{
			ID:        fmt.Sprintf("ipv4-%s", "bond0"),
			Link:      "bond0",
			Type:      NetworkTypeIPv4,
			IPAddress: nodeConfig.Network.IPv4Address,
			Netmask:   nodeConfig.Network.NetMask,
			Routes: []Route{
				{
					Network: "0.0.0.0",
					Netmask: "0.0.0.0",
					Gateway: nodeConfig.Network.Gateway,
				},
			},
		},
	}
	networkdata.Networks = networks

	services := []Service{}
	for _, dns := range nodeConfig.Network.DNS {
		services = append(services, Service{
			Type:    ServiceTypeDNS,
			Address: dns,
		})
	}
	networkdata.Services = services

	return networkdata, nil
}

func processNoneBondNetwork(networkInterfaces []hardware.NetworkInterface, config config.Node) (NetworkMetaData, error) {
	bootInterface := &hardware.NetworkInterface{}
	for _, n := range networkInterfaces {
		// FIXME(huabinhong) 没有DHCP的情况下，无法得知哪块网卡是应该配置的
		// 这里选择第一块插了网线的网卡作为boot interface
		if n.HasCarrier {
			bootInterface = &n
		}
	}
	if bootInterface == nil {
		return NetworkMetaData{}, fmt.Errorf("there is no interface has carrier")
	}

	networkdata := NetworkMetaData{}
	networks := []Network{
		{
			ID:        fmt.Sprintf("ipv4-%s", bootInterface.MACAddress),
			Link:      bootInterface.MACAddress,
			Type:      NetworkTypeIPv4,
			IPAddress: config.Network.IPv4Address,
			Netmask:   config.Network.NetMask,
			Routes: []Route{
				{
					Network: "0.0.0.0",
					Netmask: "0.0.0.0",
					Gateway: config.Network.Gateway,
				},
			},
		},
	}
	networkdata.Networks = networks

	services := []Service{}
	for _, dns := range config.Network.DNS {
		services = append(services, Service{
			Type:    ServiceTypeDNS,
			Address: dns,
		})
	}
	networkdata.Services = services

	links := []Link{
		{
			ID:         bootInterface.MACAddress,
			Type:       LinkTypePhy,
			MacAddress: bootInterface.MACAddress,
			MTU:        config.Network.MTU,
		}}
	networkdata.Links = links

	return networkdata, nil
}

func getNetworkInterfaceMap(networkInterfaces []hardware.NetworkInterface) map[string]hardware.NetworkInterface {
	interfaceMap := make(map[string]hardware.NetworkInterface)
	for _, n := range networkInterfaces {
		interfaceMap[n.MACAddress] = n
	}
	return interfaceMap
}
