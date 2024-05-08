package sshuttle

import (
	"fmt"

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
	// ip link set gatewaytun up
	// ip route add default dev gatewaytun metric 50
	// if out, err := e.session.Command("ip", "link", "set", "gatewaytun", "up").Output(); err != nil {
	// return CommandError("error to up gatewaytun %s: %w", out, err)
	// }
	// if out, err := e.session.Command("ip", "route", "add", "default", "dev", "gatewaytun", "metric", "50").Output(); err != nil {
	// 	return CommandError("error to set default gateway to gatewaytun with metric 50 %s: %w", out, err)
	// }
	return nil
}
