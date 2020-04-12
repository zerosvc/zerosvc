package main

import (
	//	"fmt"
	"github.com/urfave/cli"
	"github.com/zerosvc/go-zerosvc"
	"github.com/zerosvc/zerosvc/examples/zerocatalog/catalog"
	"github.com/zerosvc/zerosvc/examples/zerocatalog/webapp"
	"log"
	"os"
)

var version string

func main() {
	app := cli.NewApp()
	app.Name = "zerocatalog"
	app.Description = "example catalog browser"
	app.Version = version
	app.HideHelp = true
	app.Flags = []cli.Flag{
		cli.BoolFlag{Name: "help, h", Usage: "show help"},
		cli.StringFlag{
			Name:  "mqtt-url",
			Usage: "URL for the MQ server. Use tls:// to enable encryption (default: tcp://mqtt:mqtt@127.0.0.1:1883)",
			Value: "tcp://127.0.0.1:1883",
		},
	}
	app.Action = func(c *cli.Context) error {
		var catalogState catalog.State
		nodename := zerosvc.GetFQDN() + "@catalog"
		catalogState.Node = zerosvc.NewNode(nodename)
		tr := zerosvc.NewTransport(
			zerosvc.TransportMQTT,
			c.GlobalString("mqtt-url"),
			zerosvc.TransportMQTTConfig{},
		)
		connErr := tr.Connect()
		if connErr != nil {
			log.Panicf("Can't connect to MQTT: %s", connErr)
		}
		catalogState.Node.SetTransport(tr)
		go func() {
			evCh, err := catalogState.Node.GetEventsCh("discovery/#")
			if err != nil {
				log.Printf("Can't get events: %s", err)
				return
			}

			for ev := range evCh {
				log.Printf("got %+v", ev)
			}
		}()
		webapp.Run(&catalogState)
		return nil
	}
	app.Run(os.Args)
}
