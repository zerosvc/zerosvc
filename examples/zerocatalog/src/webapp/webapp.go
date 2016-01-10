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
	goji.Get("/:file", cfg.Serve)
	goji.Get("/", cfg.Serve)
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
	// do not use it in anything close to production, proper version should load templates first and then just match path
	tpl, err := template.ParseGlob("template/*")
	if err != nil {
		log.Error("error parsing template:", err)
		return
	}
	tplName := c.URLParams["file"]
	if tplName == "" {
		tplName = "index"
	}

	if tpl.Lookup(tplName) == nil  {
		cfg.staticHandler.ServeHTTP(w,r)
		return
	}
	tpl.ExecuteTemplate(w, tplName, cfg.catalogState)
}
