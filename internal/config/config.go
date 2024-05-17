package config

import (
	"flag"
	"fmt"
)

type Config struct {
	Address *string
	User    *string
	Port    *int
	Metric  *int
}

func New() (*Config, error) {
	_config := &Config{}
	_config.Address = flag.String("address", "", "ssh remote server address")
	_config.User = flag.String("user", "", "ssh remote user")
	_config.Port = flag.Int("port", 22, "ssh remote port")
	_config.Metric = flag.Int("metric", 5, "metric value for default gateway")
	flag.Parse()
	if *_config.Address == "" {
		return nil, fmt.Errorf("you must set remote ssh server address")
	}
	if *_config.User == "" {
		return nil, fmt.Errorf("you must set remote ssh user")
	}
	return _config, nil
}
