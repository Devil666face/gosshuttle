package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "embed"

	"gosshuttle/internal/config"
	"gosshuttle/internal/sshuttle"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

//go:embed wintun.dll
var WintunDLL []byte

const Wintun = "wintun.dll"

func main() {
	_config, err := config.New()
	if err != nil {
		log.Fatal(err)
	}
	env, err := sshuttle.New(
		*_config.Address,
		*_config.User,
		*_config.Port,
		*_config.Metric,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := os.Remove(Wintun); err != nil {
			log.Println(err)
		}
	}()
	if err := os.WriteFile(Wintun, WintunDLL, 0777); err != nil {
		log.Printf("error to write wintun.dll: %v\n", err)
		return
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
