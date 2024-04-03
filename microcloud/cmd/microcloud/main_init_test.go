package main

import (
	"testing"

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

func TestValidateSystems(t *testing.T) {
	handler, err := service.NewHandler("testSystem", "192.168.1.1/16", "/tmp/microcloud_test_handler", true, true)
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
			"ipv4.gateway":    "190.168.4.1/24",
			"ipv4.ovn.ranges": "190.168.4.50-190.168.4.60",
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

	invalidSystems := map[string]InitSystem{
		//"gatewayNotCIDR": newSystemWithUplinkNetwork(map[string]string{
		//	"ipv4.gateway": "192.168.1.1",
		//}),
		"backwardsRange": newSystemWithUplinkNetwork(map[string]string{
			"ipv4.gateway":    "10.42.0.1/16",
			"ipv4.ovn.ranges": "10.42.5.255-10.42.1.1",
		}),
		"rangesOutsideGateway": newSystemWithUplinkNetwork(map[string]string{
			"ipv4.gateway":    "10.1.1.0/24",
			"ipv4.ovn.ranges": "10.2.2.50-10.2.2.100",
		}),
		"uplinkInsideManagementNet": newSystemWithUplinkNetwork(map[string]string{
			"ipv4.gateway":    "192.168.1.1/24",
			"ipv4.ovn.ranges": "192.168.1.50-192.168.1.200",
		}),
		"conflictLocal": newSystemWithUplinkNetwork(map[string]string{
			"ipv4.gateway": "192.168.1.1/16",
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
