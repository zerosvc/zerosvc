package main

import (
	uuid "github.com/satori/go.uuid"
	"github.com/urfave/cli"
	"github.com/zerosvc/go-zerosvc"
	"log"
	"os"
	"time"
)

var version string
var exit = make(chan bool, 1)

type TsEvent struct {
	Message   string `json:"msg"`
	Signature string `json:"msg"`
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
		nodename := zerosvc.GetFQDN() + "@time-client"
		node := zerosvc.NewNode(nodename, uuid.NewV4().String())
		// create MQTT transport
		tr := zerosvc.NewTransport(
			zerosvc.TransportMQTT,
			url,
			zerosvc.TransportMQTTConfig{},
		)
		node.Signer, _ = zerosvc.NewSignerEd25519()
		node.Services["time"] = zerosvc.Service{
			Path:        "time",
			Description: "time bot",
		}
		err := tr.Connect()
		if err != nil {
			log.Panicf("can't connect: %s", err)
		} else {
			log.Print("connected")
		}
		node.SetTransport(tr)
		ev := node.NewEvent()
		ev.Body = []byte("byte")
		log.Print("derp")
		replyPath, replyCh, err := node.GetReplyChan()
		log.Print("derp")
		if err != nil {
			log.Fatalf("err on reply ch: %s", err)
		}
		log.Print("derp")
		log.Printf("reply ch: %s", replyPath)
		ev.ReplyTo = replyPath

		err = node.SendEvent("time/"+node.Name, ev)
		if err != nil {
			log.Printf("err: %s", err)
			os.Exit(1)
		}
		log.Print("sent request, waiting for reply")
		select {
		case resp := <-replyCh:
			log.Printf("got event resp: %s", string(resp.Body))
		case <-time.After(time.Second * 4):
			log.Print("no response, make sure timebot-server is running")
		}

		return nil
	}
	app.Run(os.Args)
}
