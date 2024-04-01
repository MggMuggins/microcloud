package main

import (
	"net"
	"testing"

	"github.com/canonical/lxd/shared"
	lxdAPI "github.com/canonical/lxd/shared/api"
	"github.com/canonical/microcloud/microcloud/mdns"
	"github.com/canonical/microcloud/microcloud/service"
)

func networkSystem(network lxdAPI.NetworksPost) InitSystem {
	return InitSystem{
		ServerInfo: mdns.ServerInfo{
			Version:    "",
			Name:       "",
			Address:    "",
			Interface:  "",
			Services:   nil,
			AuthSecret: "",
		},
		AvailableDisks:     nil,
		MicroCephDisks:     nil,
		TargetNetworks:     nil,
		TargetStoragePools: nil,
		Networks: []lxdAPI.NetworksPost{
			network,
		},
		StoragePools:   nil,
		StorageVolumes: nil,
		JoinConfig:     nil,
	}
}

func uplinkNetworkSystem(config map[string]string) InitSystem {
	return networkSystem(lxdAPI.NetworksPost{
		Name: "UPLINK",
		Type: "physical",
		NetworkPut: lxdAPI.NetworkPut{
			Config:      config,
			Description: "",
		},
	})
}

func TestValidateSystems(t *testing.T) {
	handler, err := service.NewHandler("test_handler", "localhost", "/tmp/microcloud_test_handler", true, true)
	if err != nil {
		t.Fatalf("Failed to create test service handler: %s", err)
	}

	validSystems := map[string]InitSystem{
		"0": uplinkNetworkSystem(map[string]string{
			"ipv4.gateway": "10.234.0.1/16",
		}),
	}

	if err := validateSystems(handler, validSystems); err != nil {
		t.Fatalf("Valid systems failed validate: %s", err)
	}
}

func TestSubnetContainsRange(t *testing.T) {
	validSubnetRanges := map[string]string{
		"10.214.0.0/16":  "10.214.0.55-10.214.0.255",
		"192.168.2.0/23": "192.168.2.20-192.168.3.127",
	}

	for cidr, rng := range validSubnetRanges {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			t.Fatalf("Failed to parse cidr %q", cidr)
		}

		ipRange, err := shared.ParseIPRange(rng)
		if err != nil {
			t.Fatalf("Failed to parse IPRange %q", rng)
		}

		if !subnetContainsRange(subnet, ipRange) {
			t.Fatalf("Range %q fell outside prefix %q", ipRange, subnet)
		}
	}

	invalidSubnetRanges := map[string]string{
		"10.214.0.0/16":  "10.214.0.2-10.215.0.255",
		"192.168.2.0/23": "192.168.2.49-192.168.4.130",
	}

	for cidr, rng := range invalidSubnetRanges {
		_, subnet, err := net.ParseCIDR(cidr)
		if err != nil {
			t.Fatalf("Failed to parse cidr %q", cidr)
		}

		ipRange, err := shared.ParseIPRange(rng)
		if err != nil {
			t.Fatalf("Failed to parse IPRange %q", rng)
		}

		if subnetContainsRange(subnet, ipRange) {
			t.Fatalf("Range %q fell inside prefix %q", ipRange, subnet)
		}
	}
}
