package main

import (
	"github.com/docker/machine/libmachine/drivers/plugin"
	"github.com/vleedev/docker-machine-driver-lightsail/lightsail"
)

func main() {
	plugin.RegisterDriver(lightsail.NewDriver("", ""))
}
