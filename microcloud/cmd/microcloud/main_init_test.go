package main

import (
	"testing"

	lxdAPI "github.com/canonical/lxd/shared/api"
	"github.com/canonical/microcloud/microcloud/mdns"
	"github.com/canonical/microcloud/microcloud/service"
)

func newSystemWithNetworks(systemName string, networks []lxdAPI.NetworksPost) InitSystem {
	return InitSystem{
		ServerInfo: mdns.ServerInfo{
			Name:    systemName,
			Address: "192.168.1.28",
		},
		Networks: networks,
	}
}

func newSystemWithUplinkNetConfig(systemName string, config map[string]string) InitSystem {
	return newSystemWithNetworks(systemName, []lxdAPI.NetworksPost{{
		Name: "UPLINK",
		Type: "physical",
		NetworkPut: lxdAPI.NetworkPut{
			Config: config,
		},
	}})
}

func newTestHandler(addr string, t *testing.T) *service.Handler {
	handler, err := service.NewHandler("testSystem", addr, "/tmp/microcloud_test_hander", true, true)
	if err != nil {
		t.Fatalf("Failed to create test service handler: %s", err)
	}
	return handler
}

func newTestSystemsMap(system InitSystem) map[string]InitSystem {
	return map[string]InitSystem{
		// The handler must have the same name as one of the systems in order
		// for validateSystems to perform validation
		// (we must be "bootstrapping a cluster")
		"testSystem":           newSystemWithNetworks("testSystem", []lxdAPI.NetworksPost{}),
		system.ServerInfo.Name: system,
	}

}

func ensureValidateSystemsPasses(handler *service.Handler, testSystems []InitSystem, t *testing.T) {
	for _, system := range testSystems {
		systems := newTestSystemsMap(system)

		err := validateSystems(handler, systems)
		if err != nil {
			t.Fatalf("Valid system %q failed validate: %s", system.ServerInfo.Name, err)
		}
	}
}

func ensureValidateSystemsFails(handler *service.Handler, testSystems []InitSystem, t *testing.T) {
	for _, system := range testSystems {
		systems := newTestSystemsMap(system)

		err := validateSystems(handler, systems)
		if err == nil {
			t.Fatalf("Invalid system %q passed validation", system.ServerInfo.Name)
		}
	}
}

func TestValidateSystemsIP6(t *testing.T) {
	handler := newTestHandler("fc00:feed:beef::bed1", t)

	validSystems := []InitSystem{
		newSystemWithUplinkNetConfig("64Net", map[string]string{
			"ipv6.gateway": "fc00:bad:feed::1/64",
		}),
	}

	ensureValidateSystemsPasses(handler, validSystems, t)

	invalidSystems := []InitSystem{
		newSystemWithUplinkNetConfig("uplinkInsideManagement6Net", map[string]string{
			"ipv6.gateway": "fc00:feed:beef::1/64",
		}),
	}

	ensureValidateSystemsFails(handler, invalidSystems, t)
}

func TestValidateSystemsIP4(t *testing.T) {
	handler := newTestHandler("192.168.1.27", t)

	validSystems := []InitSystem{
		newSystemWithUplinkNetConfig("plainGateway", map[string]string{
			"ipv4.gateway": "10.234.0.1/16",
		}),
		newSystemWithUplinkNetConfig("16Net", map[string]string{
			"ipv4.gateway":    "10.42.0.1/16",
			"ipv4.ovn.ranges": "10.42.1.1-10.42.5.255",
		}),
		newSystemWithUplinkNetConfig("24Net", map[string]string{
			"ipv4.gateway":    "190.168.4.1/24",
			"ipv4.ovn.ranges": "190.168.4.50-190.168.4.60",
		}),
	}

	ensureValidateSystemsPasses(handler, validSystems, t)

	invalidSystems := []InitSystem{
		//"gatewayNotCIDR": newSystemWithUplinkNetwork(map[string]string{
		//	"ipv4.gateway": "192.168.1.1",
		//}),
		newSystemWithUplinkNetConfig("backwardsRange", map[string]string{
			"ipv4.gateway":    "10.42.0.1/16",
			"ipv4.ovn.ranges": "10.42.5.255-10.42.1.1",
		}),
		newSystemWithUplinkNetConfig("rangesOutsideGateway", map[string]string{
			"ipv4.gateway":    "10.1.1.0/24",
			"ipv4.ovn.ranges": "10.2.2.50-10.2.2.100",
		}),
		newSystemWithUplinkNetConfig("uplinkInsideManagementNet", map[string]string{
			"ipv4.gateway":    "192.168.1.1/24",
			"ipv4.ovn.ranges": "192.168.1.50-192.168.1.200",
		}),
		newSystemWithUplinkNetConfig("uplinkInsideManagementNetNoRange", map[string]string{
			"ipv4.gateway": "192.168.1.1/16",
		}),
	}

	ensureValidateSystemsFails(handler, invalidSystems, t)
}
