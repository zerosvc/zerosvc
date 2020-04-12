package main

import (
	"encoding/base64"
	"fmt"
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli"
	"github.com/zerosvc/go-zerosvc"
	"log"
	"math/rand"
	"os"
	"time"
)

var version string
var exit = make(chan bool, 1)

type TsEvent struct {
	Message   string `json:"msg"`
	Signature string `json:"sig"`
}

func main() {
	app := cli.NewApp()
	app.Name = "timebot"
	app.Description = "example zerosvc service"
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
		if c.Bool("help") {
			cli.ShowAppHelp(c)
			os.Exit(1)
		}
		url := c.GlobalString("mqtt-url")
		log.Printf("connecting to %s", url)
		nodename := zerosvc.GetFQDN() + "@time"
		node := zerosvc.NewNode(nodename, uuid.NewV4().String())
		// create MQTT transport
		tr := zerosvc.NewTransport(
			zerosvc.TransportMQTT,
			url,
			zerosvc.TransportMQTTConfig{
				// cleanup heartbeats after disconnect - MQTTv3 doesn't have TTL on messages
				LastWillTopic:  node.HeartbeatPath(),
				LastWillRetain: true,
			},
		)
		node.Signer, _ = zerosvc.NewSignerEd25519()
		node.Services["time"] = zerosvc.Service{
			Path:        "time",
			Description: "time bot",
		}
		err := tr.Connect()
		if err != nil {
			log.Panicf("can't connect: %s", err)
		}
		node.SetTransport(tr)
		log.Printf("running heartbeat")
		go node.Heartbeater()
		log.Printf("subscribing to %s", node.Services["time"].Path+"/#")
		for {
			ch, err := node.GetEventsCh(node.Services["time"].Path + "/#")
			if err != nil {
				log.Fatalf("error opening command channel: %s", err)
			}
			log.Print("Waiting for events")
			for ev := range ch {
				log.Printf("got request from node[%s] on path %s", ev.NodeName(), ev.RoutingKey)
				if len(ev.ReplyTo) == 0 {
					log.Print("err: no replyTo, dunno where to send event")
				} else {
					re := node.NewEvent()
					var ts TsEvent
					t := time.Now()
					ts.Signature = base64.StdEncoding.EncodeToString(node.Signer.Sign(ev.Body))
					yearFraction := int(float64(t.YearDay()) + (float64(t.Hour()/24) + float64(t.Minute()/60/24)))
					year := t.Year() - (int(t.Year()/1000) * 1000)
					millenium := int(t.Year()/1000 + 1)
					switch rand.Intn(3) {
					case 0:
						ts.Message = t.Format(time.RFC3339Nano)
					case 1:
						ts.Message = fmt.Sprintf("0 %3d %3d.M%d", yearFraction, year, millenium)
					case 2:
						ts.Message = fmt.Sprintf("%d02%d.%d", millenium, year, (yearFraction / 10))
					}
					err := re.Marshal(&ts)
					if err != nil {
						log.Printf("err marshalling: %s", err)
					}
					err = ev.Reply(re)
					if err != nil {
						log.Printf("err sending: %s", err)
					}

				}
			}
			log.Printf("channel closed, usually means upstream connection closed")
			time.Sleep(time.Second)
		}
		return nil
	}
	app.Run(os.Args)
}
