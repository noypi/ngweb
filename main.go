package main

import (
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"sort"
)

type TomlConfig struct {
	ServerName string
	Port       string
	Route      []*Route
	Tls        bool
	Certfile   string
	Keyfile    string
}

type Route struct {
	Path      string
	Pattern   string
	Priority  int
	Filealias bool
	regex     *regexp.Regexp
}

type RegexHandler struct {
	routes map[string]*Route
	priseq []*Route
}

func (a RegexHandler) Len() int           { return len(a.priseq) }
func (a RegexHandler) Swap(i, j int)      { a.priseq[i], a.priseq[j] = a.priseq[j], a.priseq[i] }
func (a RegexHandler) Less(i, j int) bool { return a.priseq[i].Priority > a.priseq[j].Priority }

var (
	g_configfile string
	g_help       bool
	g_genconfig  bool
	g_tomlconfig *TomlConfig = new(TomlConfig)
)

func init() {
	flag.StringVar(&g_configfile, "config", "./config.toml", "loads the configuration file.")
	flag.BoolVar(&g_help, "help", false, "display this help.")
	flag.BoolVar(&g_genconfig, "genconfig", false, "generate a sample config.")
	flag.Parse()

	g_tomlconfig.ServerName = "ng"
	g_tomlconfig.Port = "80"
	g_tomlconfig.Tls = false
}

func defaultRoutes() (routes []*Route) {
	return append(routes, &Route{
		Path:     "./pub",
		Pattern:  "/",
		Priority: 0,
	})
}

func main() {
	if g_help {
		fmt.Println("\nngweb [options]")
		flag.PrintDefaults()
		fmt.Println("\n@ngtutorial.com\n")
		return
	} else if g_genconfig {
		fmt.Printf("%s", g_sampleconfig)
		return
	}

	//-- parse config file
	if _, err := toml.DecodeFile(g_configfile, &g_tomlconfig); err != nil {
		log.Fatal("ERR:decoding toml=", err)
		return
	}

	server := fmt.Sprintf("%s:%s", g_tomlconfig.ServerName, g_tomlconfig.Port)

	log.Printf("http://%s/\n", server)
	log.Println("displaying routes...")

	if 0 >= len(g_tomlconfig.Route) {
		g_tomlconfig.Route = defaultRoutes()
	}

	handler := new(RegexHandler)
	handler.routes = map[string]*Route{}
	for _, route := range g_tomlconfig.Route {
		log.Printf("pattern=%s; path=%s; priority=%d\n", route.Pattern, route.Path, route.Priority)
		if _, ok := handler.routes[route.Pattern]; ok {
			log.Fatal("ERR: duplicate pattern=", route.Pattern, ", path=", route.Path)
		}
		regex, err := regexp.Compile("^" + route.Pattern)
		if nil != err {
			log.Fatal("ERR: regex=", err)
		}
		route.regex = regex
		handler.routes[route.Pattern] = route
		handler.priseq = append(handler.priseq, route)
	}

	sort.Sort(handler)

	var err error

	if g_tomlconfig.Tls {
		err = http.ListenAndServeTLS(server, g_tomlconfig.Certfile, g_tomlconfig.Keyfile, handler)
	} else {
		err = http.ListenAndServe(server, handler)
	}

	if nil != err {
		log.Fatal("ERR: server=", err)
	}
}

func (this *RegexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	for _, route := range this.priseq {
		if route.regex.MatchString(r.URL.Path) {
			serveStaticFile(w, r, route)
			return
		}
	}
}

func serveStaticFile(w http.ResponseWriter, r *http.Request, route *Route) {
	var fpath string
	if route.Filealias {
		fpath = route.Path
	} else {
		fpath = filepath.Clean(route.Path + "/" + r.URL.Path)
	}
	log.Println("serving fpath=", fpath)
	http.ServeFile(w, r, fpath)
}

var g_sampleconfig = `
# add ng to "/etc/hosts"
# ... example:
# ...
# 127.0.1.1	ng
# 127.0.1.1	ngtutorial.com
# ...
# for windows:
# %system32%/driver/etc/hosts
servername = "ng"
port = "80"

#
# enable HTTPS, set tls = true
# tls = false
#
# specify the certificate file for HTTPS connections.
# certfile = "./priv/cert.pem"
#
# specify the key file for HTTPS connections.
# keyfile = "./priv/key.pem"
#

[[route]]
pattern = "/(api|guide|misc|tutorial|error)"
path =  "/d/dev/js/angular-1.3.0/docs/index.html"
filealias = true
priority = 20

[[route]]
pattern = "/angular.*js"
path =  "/d/dev/js/angular-1.3.0/"
priority = 10

[[route]]
pattern = "/" 
path =  "/d/dev/js/angular-1.3.0/docs"
priority = 0


# 
# ngweb -genconfig > config.toml
#
`
