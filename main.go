package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/codeskyblue/go-sh"
	_ "github.com/xjasonlyu/tun2socks/v2/dns"
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

func CommandError(message string, out []byte, err error) error {
	return fmt.Errorf(message, string(out), err)
}

func (e *Environment) SetRoutes() error {
	// ip ro add 88.151.117.196 (адресс ssh сервера) via 192.168.0.1 (default gateway) dev wlp4s0
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

func main() {
	address := flag.String("address", "", "Ssh remote server address")
	user := flag.String("user", "", "Ssh remote user")
	port := flag.Int("port", 22, "Ssh remote port")
	flag.Parse()
	if *address == "" {
		log.Fatalln("you must set remote ssh server address")
	}
	if *user == "" {
		log.Fatalln("you must set remote ssh user")
	}
	env, err := New(
		*address,
		*user,
		*port,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer env.Shutdown()

	if err := env.SetRoutes(); err != nil {
		log.Println(err)
		return
	}

	go func() {
		if err := env.ConnectSshProxy(); err != nil {
			log.Println(err)
			return
		}
	}()

	env.RunTunToSocks()
	defer engine.Stop()

	if err := env.SetProxyToTun(); err != nil {
		log.Println(err)
		return
	}

	func(s chan os.Signal) {
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
		<-s
		return
	}(make(chan os.Signal, 1))
}
