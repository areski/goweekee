package main

import (
	// "errors"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	// "strings"
)

var (
	addr       = flag.Bool("addr", false, "find open address and print to final-port.txt")
	configfile = flag.String("configfile", "config.yaml", "path and filename of the config file")
)

type Config struct {
	// First letter of variables need to be capital letter
	Template_directory string
	Data_directory     string
}

var config Config

// config.Data_directory
var TMPL_DIR = "./templates/"
var DATA_DIR = "./data/"

type Page struct {
	Title string
	Body  []byte
}

var templates *template.Template

var validPath = regexp.MustCompile("^/(edit|save|view|list)/([a-zA-Z0-9]+)$")

func (p *Page) save() error {
	filename := DATA_DIR + p.Title + ".txt"
	return ioutil.WriteFile(filename, p.Body, 0600)
}

// func getTitle(w http.ResponseWriter, r *http.Request) (string, error) {
// 	m := validPath.FindStringSubmatch(r.URL.Path)
// 	if m == nil {
// 		http.NotFound(w, r)
// 		return "", errors.New("Invalid Page Title")
// 	}
// 	return m[2], nil
// }

func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
	err := templates.ExecuteTemplate(w, tmpl+".html", p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// func renderTemplate(w http.ResponseWriter, tmpl string, p *Page) {
// 	t, err := template.ParseFiles(tmpl + ".html")
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	err = t.Execute(w, p)
// 	if err != nil {
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 	}
// }

func loadPage(title string) (*Page, error) {
	filename := DATA_DIR + title + ".txt"
	body, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return &Page{Title: title, Body: body}, nil
}

func editHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func viewHandler(w http.ResponseWriter, r *http.Request, title string) {
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request, title string) {
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func listHandler(w http.ResponseWriter, r *http.Request, title string) {
	datafiles, err := ioutil.ReadDir(DATA_DIR)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// layoutData := struct {
	// 	datafiles []string
	// }
	for _, f := range datafiles {
		fmt.Println(f.Name())

	}

	// renderTemplate(w, "list", datafiles)
	err = templates.ExecuteTemplate(w, "list.html", datafiles)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// Similar to Decorator in Python
func makeHandler(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		fn(w, r, m[2])
	}
}

func main() {
	flag.Parse()
	// prepare handler
	http.HandleFunc("/list/", makeHandler(listHandler))
	http.HandleFunc("/view/", makeHandler(viewHandler))
	http.HandleFunc("/edit/", makeHandler(editHandler))
	http.HandleFunc("/save/", makeHandler(saveHandler))

	config = Config{}

	// Load configfile and configure template
	if len(*configfile) > 0 {
		fmt.Println("config file => " + *configfile)
		source, err := ioutil.ReadFile(*configfile)
		fmt.Println(string(source))
		if err != nil {
			panic(err)
		}
		// decode the yaml source
		err = yaml.Unmarshal(source, &config)
		if err != nil {
			panic(err)
		}
		TMPL_DIR = config.Template_directory
		DATA_DIR = config.Data_directory
	}

	templates = template.Must(template.ParseFiles(TMPL_DIR+"edit.html", TMPL_DIR+"view.html", TMPL_DIR+"list.html"))

	if *addr {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
		err = ioutil.WriteFile("final-port.txt", []byte(l.Addr().String()), 0644)
		if err != nil {
			log.Fatal(err)
		}
		s := &http.Server{}
		s.Serve(l)
		return
	}
	http.ListenAndServe(":8080", nil)
}
