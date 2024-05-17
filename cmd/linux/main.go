package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"gosshuttle/internal/config"
	"gosshuttle/internal/sshuttle"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

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
