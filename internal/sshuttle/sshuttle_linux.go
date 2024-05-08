package sshuttle

import (
	"fmt"
	"strings"
)

func (e *Environment) SetDefaultGateway() error {
	out, err := e.session.Command("ip", "ro", "sh").Output()
	if err != nil {
		return CommandError("error to get default gateway %s: %w", out, err)
	}
	s := strings.Fields(strings.TrimSpace(string(out)))
	if len(s) < 6 {
		return fmt.Errorf("error to get default gateway")
	}
	e.defgate = &DefaultGateway{
		device:  s[4],
		address: s[2],
	}
	return nil
}

func (e *Environment) SetRoutes() error {
	// ip ro add 88.151.117.196 via 192.168.0.1  dev wlp4s0
	// ip ro add 10.0.0.0/8 via 192.168.0.1 dev wlp4s0
	// ip ro add 172.16.0.0/12 via 192.168.0.1 dev wlp4s0
	// ip ro add 192.168.0.0/16 via 192.168.0.1 dev wlp4s0
	if out, err := e.session.Command("ip", "ro", "add", e.address, "via", e.defgate.address, "dev", e.defgate.device).Output(); err != nil {
		return CommandError("error to set route to ssh server %s: %w", out, err)
	}
	for _, local := range LocalNets {
		if out, err := e.session.Command("ip", "ro", "add", local, "via", e.defgate.address, "dev", e.defgate.address, "dev", e.defgate.device).Output(); err != nil {
			return CommandError("error to set route to local networks via default gateway %s: %w", out, err)
		}
	}
	return nil
}

func (e *Environment) Shutdown() error {
	var Err error
	if out, err := e.session.Command("ip", "ro", "del", e.address).Output(); err != nil {
		Err = CommandError("error to delete route to ssh server %s: %w", out, err)
	}
	for _, local := range LocalNets {
		if out, err := e.session.Command("ip", "ro", "del", local).Output(); err != nil {
			Err = CommandError("error to delete route to local networks via default gateway %s: %w", out, err)
		}
	}
	return Err
}

func (e *Environment) SetProxyToTun() error {
	// ip link set gatewaytun up
	// ip route add default dev gatewaytun metric 50
	if out, err := e.session.Command("ip", "link", "set", "gatewaytun", "up").Output(); err != nil {
		return CommandError("error to up gatewaytun %s: %w", out, err)
	}
	if out, err := e.session.Command("ip", "route", "add", "default", "dev", "gatewaytun", "metric", "50").Output(); err != nil {
		return CommandError("error to set default gateway to gatewaytun with metric 50 %s: %w", out, err)
	}
	return nil
}
