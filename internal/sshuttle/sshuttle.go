package sshuttle

import (
	"fmt"
	"net"

	"github.com/codeskyblue/go-sh"
	"github.com/xjasonlyu/tun2socks/v2/engine"
)

var (
	LocalNets = []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}
)

type DefaultGateway struct {
	device  string
	address string
}

type Environment struct {
	defgate   *DefaultGateway
	address   string
	user      string
	port      string
	proxyport string
	session   *sh.Session
	key       *engine.Key
}

func getRandomPort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func New(_address string, _user string, _port int) (*Environment, error) {
	_proxyport, err := getRandomPort()
	if err != nil {
		return nil, fmt.Errorf("error to get random port for socks5 proxy: %w", err)
	}
	e := &Environment{
		address:   _address,
		user:      _user,
		port:      fmt.Sprint(_port),
		proxyport: fmt.Sprint(_proxyport),
		session:   sh.NewSession(),
		key: &engine.Key{
			Device:   "tun://gatewaytun",
			LogLevel: "silent",
			Proxy:    fmt.Sprintf("socks5://127.0.0.1:%d", _proxyport),
		},
	}
	if err := e.SetDefaultGateway(); err != nil {
		return nil, err
	}
	return e, nil
}

func CommandError(message string, out []byte, err error) error {
	return fmt.Errorf(message, string(out), err)
}

func (e *Environment) ConnectSshProxy() error {
	// ssh -f host01.d6f.ru -p 31659 -D 1337 -N
	if out, err := e.session.Command("ssh", "-p", e.port, fmt.Sprintf("%s@%s", e.user, e.address), "-D", e.proxyport, "-N").Output(); err != nil {
		return fmt.Errorf("error to connect throw ssh -p %s %s@%s -D %s -N %s: %w", e.port, e.user, e.address, e.proxyport, out, err)
	}
	return nil
}

func (e *Environment) RunTunToSocks() {
	// tun2socks -device tun://gatewaytun -proxy socks5://127.0.0.1:1337
	engine.Insert(e.key)
	engine.Start()
}
