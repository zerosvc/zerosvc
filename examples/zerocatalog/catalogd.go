package main

import (
	//	"fmt"
	"catalog"
	"github.com/op/go-logging"
	"github.com/zerosvc/go-zerosvc"
	"os"
	"strings"
	"webapp"
)

var version string
var log = logging.MustGetLogger("main")
var stdout_log_format = logging.MustStringFormatter("%{color:bold}%{time:2006-01-02T15:04:05.9999Z-07:00}%{color:reset}%{color} [%{level:.1s}] %{color:reset}%{shortpkg}[%{longfunc}] %{message}")
var static_dir = "public"

type config struct {
	StaticDir string
	AmqpAddr  string
}

var cfg = config{
	StaticDir: "public",
	AmqpAddr:  "amqp://guest:guest@localhost:5672",
}

func main() {
	stderrBackend := logging.NewLogBackend(os.Stderr, "", 0)
	stderrFormatter := logging.NewBackendFormatter(stderrBackend, stdout_log_format)
	logging.SetBackend(stderrFormatter)
	logging.SetFormatter(stdout_log_format)

	log.Info("Starting app")
	log.Debug("version: %s", version)
	if !strings.ContainsRune(version, '-') {
		log.Warning("once you tag your commit with name your version number will be prettier")
	}
	log.Error("now add some code!")
	hostname, err := os.Hostname()
	if err != nil {
		log.Error("hostname resolution failed:", err)
	}
	var catalogState catalog.State
	catalogState.Node = zerosvc.NewNode(hostname + "@catalog")

	webapp.Run(&catalogState)
}
