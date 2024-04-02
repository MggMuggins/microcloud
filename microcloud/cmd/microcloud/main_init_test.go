package main

import (
	"net"
	"testing"

	"github.com/canonical/lxd/shared"
	lxdAPI "github.com/canonical/lxd/shared/api"
	"github.com/canonical/microcloud/microcloud/mdns"
	"github.com/canonical/microcloud/microcloud/service"
)

func newSystemWithNetworks(networks []lxdAPI.NetworksPost) InitSystem {
	return InitSystem{
		ServerInfo: mdns.ServerInfo{
			Address: "127.0.0.1",
		},
		Networks: networks,
	}
}

func newSystemWithUplinkNetwork(config map[string]string) InitSystem {
	return newSystemWithNetworks([]lxdAPI.NetworksPost{{
		Name: "UPLINK",
		Type: "physical",
		NetworkPut: lxdAPI.NetworkPut{
			Config: config,
		},
	}})
}

func newSystemWithOvnNetwork(uplinkConfig, ovnConfig map[string]string) InitSystem {
	return newSystemWithNetworks([]lxdAPI.NetworksPost{
		{
			Name: "UPLINK",
			Type: "physical",
			NetworkPut: lxdAPI.NetworkPut{
				Config: uplinkConfig,
			},
		},
		{
			Name: "default",
			Type: "ovn",
			NetworkPut: lxdAPI.NetworkPut{
				Config: ovnConfig,
			},
		},
	})
}

func TestValidateSystems(t *testing.T) {
	handler, err := service.NewHandler("testSystem", "192.168.1.1/24", "/tmp/microcloud_test_handler", true, true)
	if err != nil {
		t.Fatalf("Failed to create test service handler: %s", err)
	}

	validSystems := map[string]InitSystem{
		"plainGateway": newSystemWithUplinkNetwork(map[string]string{
			"ipv4.gateway": "10.234.0.1/16",
		}),
		"16Net": newSystemWithUplinkNetwork(map[string]string{
			"ipv4.gateway":    "10.42.0.1/16",
			"ipv4.ovn.ranges": "10.42.1.1-10.42.5.255",
		}),
		"24Net": newSystemWithUplinkNetwork(map[string]string{
			"ipv4.gateway":    "192.168.4.1/24",
			"ipv4.ovn.ranges": "192.168.4.50-192.168.4.60",
		}),
	}

	for systemName, system := range validSystems {
		systems := map[string]InitSystem{
			// The handler must have the same name as one of the systems in order
			// to perform validation (we must be "bootstrapping a cluster")
			"testSystem": newSystemWithNetworks([]lxdAPI.NetworksPost{}),
			systemName:   system,
		}
		err = validateSystems(handler, systems)
		if err != nil {
			t.Fatalf("Valid system %q failed validate: %s", systemName, err)
		}
	}

	uplinkConfig := map[string]string{
		"ipv4.gateway":    "10.28.0.1/16",
		"ipv4.ovn.ranges": "10.28.10.1-10.28.20.1",
		"ipv6.gateway":    "f6ef:9374:923a:632::1/64",
	}

	invalidSystems := map[string]InitSystem{
		//"gatewayNotCIDR": newSystemWithUplinkNetwork(map[string]string{
		//	"ipv4.gateway": "192.168.1.1",
		//}),
		"backwardsRange": newSystemWithUplinkNetwork(map[string]string{
			"ipv4.gateway":    "10.42.0.1/16",
			"ipv4.ovn.ranges": "10.42.5.255-10.42.1.1",
		}),
		"localhostGateway": newSystemWithUplinkNetwork(map[string]string{
			"ipv4.gateway": "192.168.1.1/24",
		}),
		"conflict4Net": newSystemWithOvnNetwork(uplinkConfig, map[string]string{
			"ipv4.address": "10.28.62.1/24",
		}),
		"conflict6Net": newSystemWithOvnNetwork(uplinkConfig, map[string]string{
			"ipv6.address": "f6ef:9374:923a:632:82::7",
		}),
	}

	for systemName, system := range invalidSystems {
		systems := map[string]InitSystem{
			"testSystem": newSystemWithNetworks([]lxdAPI.NetworksPost{}),
			systemName:   system,
		}
		err = validateSystems(handler, systems)
		if err == nil {
			t.Fatalf("Invalid system %q passed validation", systemName)
		}
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
