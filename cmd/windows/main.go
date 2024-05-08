package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	_ "embed"

	"gosshuttle/internal/sshuttle"

	"github.com/xjasonlyu/tun2socks/v2/engine"
)

//go:embed wintun.dll
var WintunDLL []byte

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
	env, err := sshuttle.New(
		*address,
		*user,
		*port,
	)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := os.Remove("wintun.dll"); err != nil {
			log.Println(err)
		}
	}()
	if err := os.WriteFile("wintun.dll", WintunDLL, 0777); err != nil {
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