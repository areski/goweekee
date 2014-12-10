package main

import (
	// "errors"
	// "strings"
	// "io"
	"flag"
	"fmt"
	"github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/justinas/alice"
	"gopkg.in/yaml.v2"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"regexp"
	"runtime/debug"
	"time"
)

var (
	addr       = flag.Bool("addr", false, "find open address and print to final-port.txt")
	configfile = flag.String("configfile", "config.yaml", "path and filename of the config file")
)

// Hold the structure for the wiki configuration
type Config struct {
	// First letter of variables need to be capital letter
	Template_directory string
	Data_directory     string
}

var config Config

// Default Template and Data directory
var TEMPLATE_DIR = "./templates/"
var DATA_DIR = "./data/"

// Hold the structure for the page configuration
type Page struct {
	Title string
	Body  []byte
}

var templates *template.Template

var validPath = regexp.MustCompile("^/(edit|save|view|list)/([a-zA-Z0-9]*)$")

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

func editHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	title := vars["title"]
	// gettitle := context.Get(r, "title")
	// title := gettitle.(string)
	p, err := loadPage(title)
	if title == "" {
		http.Redirect(w, r, "/list/", http.StatusFound)
		return
	}
	if err != nil {
		p = &Page{Title: title}
	}
	renderTemplate(w, "edit", p)
}

func viewHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	title := vars["title"]
	// gettitle := context.Get(r, "title")
	// title := gettitle.(string)
	if title == "" {
		http.Redirect(w, r, "/list/", http.StatusFound)
		return
	}
	p, err := loadPage(title)
	if err != nil {
		http.Redirect(w, r, "/edit/"+title, http.StatusFound)
		return
	}
	renderTemplate(w, "view", p)
}

func saveHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	title := vars["title"]
	// gettitle := context.Get(r, "title")
	// title := gettitle.(string)
	body := r.FormValue("body")
	p := &Page{Title: title, Body: []byte(body)}
	err := p.save()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/view/"+title, http.StatusFound)
}

func listHandler(w http.ResponseWriter, r *http.Request) {
	datafiles, err := ioutil.ReadDir(DATA_DIR)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for _, f := range datafiles {
		fmt.Println(f.Name())
	}
	err = templates.ExecuteTemplate(w, "list.html", datafiles)
	fmt.Println(err)

	// if err != nil {
	// 	http.Error(w, err.Error(), http.StatusInternalServerError)
	// }
	fmt.Fprintf(w, "You are on the about page.")
}

func parseTitleHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		m := validPath.FindStringSubmatch(r.URL.Path)
		fmt.Println(r.URL.Path)
		if m == nil {
			http.NotFound(w, r)
			return
		}
		log.Printf("[parseTitleHandler] %v\n", m[2])
		context.Set(r, "title", m[2])
		// next.ServeHTTP()(w, r)
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

func loggingHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		t1 := time.Now()
		next.ServeHTTP(w, r)
		t2 := time.Now()
		log.Printf("[%s] %q %v\n", r.Method, r.URL.String(), t2.Sub(t1))
	}
	return http.HandlerFunc(fn)
}

func aboutHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "You are on the about page.")
}

func recoverHandler(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("[!PANIC!] %+v", err)
				log.Printf("%s: %s", err, debug.Stack()) // line 20
				http.Error(w, http.StatusText(500), 500)
			}
		}()
		next.ServeHTTP(w, r)
	}
	return http.HandlerFunc(fn)
}

// func authHandler(next http.Handler) http.Handler {
// 	fn := func(w http.ResponseWriter, r *http.Request) {
// 		authToken := r.Header().Get("Authorization")
// 		user, err := getUser(authToken)

// 		if err != nil {
// 			http.Error(w.http.StatusText(401), 401)
// 			return
// 		}
// 		context.Set(r, "user", user)
// 		next.ServeHTTP()(w, r)
// 	}
// 	return http.HandleFunc(fn)
// }

// func adminHandler(w http.ResponseWriter, r *http.Requests) {
// 	user := context.Get(r, "user")
// 	json.NewEncoder(w).Encode(user)
// }

func main() {
	// Parse CLI
	flag.Parse()

	// implement request router and dispatcher.
	rtr := mux.NewRouter()
	commonHandlers := alice.New(context.ClearHandler, loggingHandler)
	rtr.Handle("/", commonHandlers.ThenFunc(listHandler)).Methods("GET")
	rtr.Handle("/about", commonHandlers.ThenFunc(aboutHandler)).Methods("GET")

	rtr.Handle("/view/{title}", commonHandlers.Append(parseTitleHandler).ThenFunc(viewHandler))
	rtr.Handle("/edit/{title}", commonHandlers.Append(parseTitleHandler).ThenFunc(editHandler))
	rtr.Handle("/save/{title}", commonHandlers.Append(parseTitleHandler).ThenFunc(saveHandler))
	http.Handle("/", rtr)

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
		// Change global Tempalte & Data vars
		TEMPLATE_DIR = config.Template_directory
		DATA_DIR = config.Data_directory

		// templates = template.Must(template.ParseFiles(TEMPLATE_DIR+"edit.html", TEMPLATE_DIR+"view.html", TEMPLATE_DIR+"list.html"))
		templates = template.Must(template.ParseGlob(TEMPLATE_DIR + "*.html"))
	} else {
		templates = template.Must(template.ParseGlob(TEMPLATE_DIR + "*.html"))
	}

	// set command line "addr" parameter
	if *addr {
		// if addr is set we will find open address and print to final-port.txt
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

	log.Println("Listening...")
	http.ListenAndServe(":8080", nil)
}
