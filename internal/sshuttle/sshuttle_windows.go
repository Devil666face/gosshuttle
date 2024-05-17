package sshuttle

import (
	"fmt"
	"strings"

	"github.com/jackpal/gateway"
)

func (e *Environment) SetDefaultGateway() error {
	gateway, err := gateway.DiscoverGateway()
	if err != nil {
		return fmt.Errorf("error to get default gateway")
	}
	e.defgate = &DefaultGateway{
		device:  "",
		address: gateway.String(),
	}
	return nil
}

func (e *Environment) SetRoutes() error {
	// route add 172.252.212.8/32 20.20.20.21
	if out, err := e.session.Command("route", "add", fmt.Sprintf("%s/32", e.address), e.defgate.address).Output(); err != nil {
		return CommandError("error to set route to ssh server %s: %w", out, err)
	}
	for _, local := range LocalNets {
		if out, err := e.session.Command("route", "add", local, e.defgate.address).Output(); err != nil {
			return CommandError("error to set route to local networks via default gateway %s: %w", out, err)
		}
	}
	return nil
}

func (e *Environment) Shutdown() error {
	var Err error
	// route add 172.252.212.8/32 20.20.20.21
	if out, err := e.session.Command("route", "delete", fmt.Sprintf("%s/32", e.address)).Output(); err != nil {
		Err = CommandError("error to delete route to ssh server %s: %w", out, err)
	}
	for _, local := range LocalNets {
		if out, err := e.session.Command("route", "delete", local).Output(); err != nil {
			Err = CommandError("error to delete route to local networks via default gateway %s: %w", out, err)
		}
	}
	return Err
}

func (e *Environment) SetProxyToTun() error {
	// netsh interface ipv4 show interface
	// netsh interface ipv4 show interfaces interface=Ethernet level=normal
	// route add 0.0.0.0 mask 0.0.0.0 192.168.1.1 metric 10 if <Interface1_index>
	out, err := e.session.Command("netsh", "interface", "ipv4", "show", "interfaces", "interface=gatewaytun", "level=normal").Output()
	if err != nil {
		return CommandError("error to get gatewaytun interface id %s: %w", out, err)
	}

	cut := strings.Fields(strings.TrimSpace(string(out)))
	if len(cut) < 11 {
		return CommandError("error to get gatewaytun interface id %s: %w", out, err)
	}
	idx := cut[10]

	if out, err := e.session.Command("route", "add", "0.0.0.0", "mask", "0.0.0.0", e.defgate.address, "metric", e.metric, "if", idx).Output(); err != nil {
		return CommandError("error to set default gateway to gatewaytun %s: %w", out, err)
	}
	return nil
}
