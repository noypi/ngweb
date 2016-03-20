package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/BurntSushi/toml"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
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
	Findindex bool
	Useindex  string
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

	g_tomlconfig.ServerName = ""
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
		log.Println("ListenAndServeTLS... certfile=", g_tomlconfig.Certfile, "keyfile=", g_tomlconfig.Keyfile)
		err = http.ListenAndServeTLS(server, g_tomlconfig.Certfile, g_tomlconfig.Keyfile, handler)
	} else {
		log.Println("ListenAndServe...")
		err = http.ListenAndServe(server, handler)
	}

	if nil != err {
		log.Fatal("ERR: server=", err)
	}
}

func (this *RegexHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if 0 < len(r.FormValue("clearcache")) {
		g_indexTmpls = map[string]*template.Template{}
	}
	for _, route := range this.priseq {
		if route.regex.MatchString(r.URL.Path) {
			serveStaticFile(w, r, route)
			return
		}
	}
}

var g_indexTmpls = map[string]*template.Template{}

func serveStaticFile(w http.ResponseWriter, r *http.Request, route *Route) {
	var fpath string
	if route.Filealias {
		fpath = route.Path
	} else {
		fpath = filepath.Clean(route.Path + "/" + r.URL.Path)
	}
	bfindindex := route.Findindex && "false" != r.FormValue("findindex")
	if f, _ := os.Stat(fpath); bfindindex && nil != f && f.IsDir() {
		log.Println("trying to use index")
		if _, err := os.Stat(fpath + "/index.html"); nil == err || os.IsExist(err) {
			fpath = filepath.Clean(fpath + "/index.html")
		} else if _, err := os.Stat(route.Useindex); nil == err || os.IsExist(err) {
			t, has := g_indexTmpls[route.Useindex+"!"+fpath]
			if !has {
				t, err = template.New(filepath.Base(route.Useindex)).Funcs(template.FuncMap{
					"getfiles": func(filter string) map[string]interface{} {
						return getfiles(fpath, filter)
					},
				}).ParseFiles(route.Useindex)
				if nil != err {
					log.Println("[ERR]", err)
					http.ServeFile(w, r, fpath)
					return
				} else {
					g_indexTmpls[route.Useindex+"!"+fpath] = t
				}

			}

			data := map[string]interface{}{
				"CurrentPath": fpath,
				"Host":        "http://" + r.Host + r.URL.Path,
			}
			log.Println("executing index...")
			t.Execute(w, data)
			return
		} else {
			log.Println("no index found")
		}
	}

	//log.Println("serving fpath=", fpath)
	http.ServeFile(w, r, fpath)
}

var g_getFilesFilter = map[string]*regexp.Regexp{}

func getfiles(fpath, filter string) (ret map[string]interface{}) {
	if f, _ := os.Stat(fpath); nil == f || !f.IsDir() {
		return
	}

	filesmap := map[string]string{}
	foldersmap := map[string]string{}
	ret = map[string]interface{}{
		"files":   filesmap,
		"folders": foldersmap,
	}

	rfilter, has := g_getFilesFilter[filter]
	if !has {
		rfilter, _ = regexp.Compile(filter)
	}
	currdir := filepath.Dir(fpath + "/")
	log.Println("currdir=", currdir)
	filepath.Walk(fpath, func(path string, info os.FileInfo, err1 error) (err error) {
		if "." == info.Name() || ".." == info.Name() {
			return
		}
		if currdir != filepath.Dir(path) {
			return
		}
		if info.IsDir() {
			if info.Name()[0] == '.' {
				return
			}
			foldersmap[info.Name()] = "folder"
		} else {
			bValid := true
			var f *os.File

			mime := "application/octet-stream"
			if strings.HasSuffix(info.Name(), ".mp4") {
				mime = "video/mp4"
			} else if strings.HasSuffix(info.Name(), ".txt") {
				mime = "text/plain"
			} else if strings.HasSuffix(info.Name(), ".zip") {
				mime = "compress/zip"
			} else {
				f, err = os.Open(path)
				if nil == err {
					mime, _ = MIMETypeFromReader(f)
				}

			}

			if nil != rfilter {
				bValid = rfilter.MatchString(mime)
			}
			if bValid {
				filesmap[info.Name()] = mime
			}
			if nil != f {
				f.Close()
			}

		}

		return

	})
	return
}

func MIMETypeFromReader(r io.Reader) (mime string, reader io.Reader) {
	var buf bytes.Buffer
	io.CopyN(&buf, r, 1024)
	mime = MIMEType(buf.Bytes())
	return mime, io.MultiReader(&buf, r)
}

func MIMEType(hdr []byte) string {
	hlen := len(hdr)
	for _, pte := range prefixTable {
		plen := len(pte.prefix)
		if hlen > plen && bytes.Equal(hdr[:plen], pte.prefix) {
			return pte.mtype
		}
	}
	t := http.DetectContentType(hdr)
	t = strings.Replace(t, "; charset=utf-8", "", 1)
	if t != "application/octet-stream" && t != "text/plain" {
		return t
	}
	return ""
}

type prefixEntry struct {
	prefix []byte
	mtype  string
}

// usable source: http://www.garykessler.net/library/file_sigs.html
// mime types: http://www.iana.org/assignments/media-types/media-types.xhtml
var prefixTable = []prefixEntry{
	{[]byte("GIF87a"), "image/gif"},
	{[]byte("GIF89a"), "image/gif"}, // TODO: Others?
	{[]byte("\xff\xd8\xff\xe2"), "image/jpeg"},
	{[]byte("\xff\xd8\xff\xe1"), "image/jpeg"},
	{[]byte("\xff\xd8\xff\xe0"), "image/jpeg"},
	{[]byte("\xff\xd8\xff\xdb"), "image/jpeg"},
	{[]byte("\x49\x49\x2a\x00\x10\x00\x00\x00\x43\x52\x02"), "image/cr2"},
	{[]byte{137, 'P', 'N', 'G', '\r', '\n', 26, 10}, "image/png"},
	{[]byte{0x49, 0x20, 0x49}, "image/tiff"},
	{[]byte{0x49, 0x49, 0x2A, 0}, "image/tiff"},
	{[]byte{0x4D, 0x4D, 0, 0x2A}, "image/tiff"},
	{[]byte{0x4D, 0x4D, 0, 0x2B}, "image/tiff"},
	{[]byte("8BPS"), "image/vnd.adobe.photoshop"},
	{[]byte("gimp xcf "), "image/xcf"},
	{[]byte("-----BEGIN PGP PUBLIC KEY BLOCK---"), "text/x-openpgp-public-key"},
	{[]byte("fLaC\x00\x00\x00"), "audio/flac"},
	{[]byte{'I', 'D', '3'}, "audio/mpeg"},
	{[]byte{0, 0, 1, 0xB7}, "video/mpeg"},
	{[]byte{0, 0, 0, 0x14, 0x66, 0x74, 0x79, 0x70, 0x71, 0x74, 0x20, 0x20}, "video/quicktime"},
	{[]byte{0, 0x6E, 0x1E, 0xF0}, "application/vnd.ms-powerpoint"},
	{[]byte{0x1A, 0x45, 0xDF, 0xA3}, "video/webm"},
	{[]byte("FLV\x01"), "application/vnd.adobe.flash.video"},
	{[]byte{0x1F, 0x8B, 0x08}, "application/gzip"},
	{[]byte{0x37, 0x7A, 0xBC, 0xAF, 0x27, 0x1C}, "application/x-7z-compressed"},
	{[]byte("BZh"), "application/bzip2"},
	{[]byte{0xFD, 0x37, 0x7A, 0x58, 0x5A, 0}, "application/x-xz"},
	{[]byte{'P', 'K', 3, 4, 0x0A, 0, 2, 0}, "application/epub+zip"},
	{[]byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}, "application/vnd.ms-word"},
	{[]byte{'P', 'K', 3, 4, 0x0A, 0x14, 0, 6, 0}, "application/vnd.openxmlformats-officedocument.custom-properties+xml"},
	{[]byte{'P', 'K', 3, 4}, "application/zip"},
	{[]byte("%PDF"), "application/pdf"},
	{[]byte("{rtf"), "text/rtf1"},
	{[]byte("BEGIN:VCARD\x0D\x0A"), "text/vcard"},
	{[]byte("Return-Path: "), "message/rfc822"},

	// TODO(bradfitz): popular audio & video formats at least
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
findindex = true
useindex = "/d/index.html"
priority = 0


#
# ngweb -genconfig > config.toml
#
`
