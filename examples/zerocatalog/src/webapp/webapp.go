package webapp

import (
	"github.com/op/go-logging"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"net/http"
	"fmt"
	"catalog"
	"html/template"
)

type Webapp struct {
	StaticDir string
	TemplateDir string
	catalogState *catalog.State
	staticHandler http.Handler
}

var cfg = Webapp {
	StaticDir: "public",
	TemplateDir: "template",
}
var log = logging.MustGetLogger("main")
func Run(catalogState *catalog.State) {
	cfg.catalogState = catalogState
	cfg.staticHandler = http.FileServer(http.Dir(cfg.StaticDir))
	goji.Get("/status", status)
	goji.Get("/tpl",cfg.Index)
	goji.Get("/index.html", http.FileServer(http.Dir(cfg.StaticDir)))
	goji.Get("/:file", cfg.Serve)
	goji.Get("/*", http.FileServer(http.Dir(cfg.StaticDir)))
	goji.Serve()
}
func status(c web.C, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "OK", c.URLParams["name"])
}
func (cfg *Webapp) Index(c web.C, w http.ResponseWriter, r *http.Request) {
	//	t, err := template.New("webpage").Parse(`<h1>{{.Node.Name}}</h2>`)
	t, err := template.ParseGlob("template/index*")
	if (err != nil) {
		log.Error("tpl parse failed: %s", err)
	}
	t.Execute(w, cfg.catalogState)
}


func (cfg *Webapp) Serve(c web.C, w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseGlob("template/" + c.URLParams["file"] + "*")
	if (err != nil) {
		log.Error("tpl parse failed: %s", err)
		cfg.staticHandler.ServeHTTP(w,r)
		return
	}
	t.Execute(w, cfg.catalogState)

}
