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
	defgate  *DefaultGateway
	address  string
	user     string
	password string
	port     int
	session  *sh.Session
}

func getRandomPort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func New(_address string, _user string, _password string) (*Environment, error) {
	_port, err := getRandomPort()
	if err != nil {
		return nil, fmt.Errorf("error to get random port for socks5 proxy: %w", err)
	}
	e := &Environment{
		address:  _address,
		user:     _user,
		port:     _port,
		password: _password,
		session:  sh.NewSession(),
	}
	if err := e.SetDefaultGateway(); err != nil {
		return nil, err
	}
	if err := e.SetRoutes(); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Environment) SetDefaultGateway() error {
	out, err := e.session.Command("ip", "ro", "sh").Output()
	if err != nil {
		return err
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

func (e *Environment) Sudo(command string, args ...string) ([]byte, error) {
	log.Println(command, args)
	args = append([]string{"-S", command}, args...)
	return e.session.Command("echo", e.password).Command("sudo", args).CombinedOutput()
}

func (e *Environment) SetRoutes() error {
	// ip ro add 88.151.117.196 (адресс ssh сервера) via 192.168.0.1 (default gateway) dev wlp4s0
	// ip ro add 10.0.0.0/8 via 192.168.0.1 dev wlp4s0
	// ip ro add 172.16.0.0/12 via 192.168.0.1 dev wlp4s0
	// ip ro add 192.168.0.0/16 via 192.168.0.1 dev wlp4s0
	if _, err := e.Sudo("ip", "ro", "add", e.address, "via", e.defgate.address, "dev", e.defgate.device); err != nil {
		return fmt.Errorf("error to set route to ssh server: %w", err)
	}
	for _, local := range LocalNets {
		if _, err := e.Sudo("ip", "ro", "add", local, "via", e.defgate.address, "dev", e.defgate.address, "dev", e.defgate.device); err != nil {
			return fmt.Errorf("error to set route to local networks via default gateway: %w", err)
		}
	}
	return nil
}

func (e *Environment) Shutdown() error {
	if _, err := e.Sudo("ip", "ro", "del", e.address); err != nil {
		return fmt.Errorf("error to delete route to ssh server: %w", err)
	}
	for _, local := range LocalNets {
		if _, err := e.Sudo("ip", "ro", "del", local); err != nil {
			return fmt.Errorf("error to delete route to local networks via default gateway: %w", err)
		}
	}
	return nil
}

func main() {
	address := flag.String("address", "", "Ssh remote server address")
	user := flag.String("user", "", "Ssh remote user")
	password := flag.String("password", "", "Sudo password")
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
		*password,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer env.Shutdown()

	func(s chan os.Signal) {
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
		<-s
	}(make(chan os.Signal, 1))
}
