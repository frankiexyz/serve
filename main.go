// Serves the current working directory over HTTP (static file server).  Has a directory listing and all that stuff.
package main

import (
	"crypto/md5"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/fatih/color"
	"github.com/goji/httpauth"
)

var path string
var user string
var pass string
var listenAddress string
var spin *spinner.Spinner

func upload(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	if r.Method == "GET" {
		crutime := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(crutime, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))

		t, _ := template.ParseFiles("upload.gtpl")
		t.Execute(w, token)
	} else {
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()
		fmt.Fprintf(w, "%v", handler.Header)
		f, err := os.OpenFile("./"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer f.Close()
		io.Copy(f, file)
	}
}

func init() {

	// Use the working directory as the default location to serve
	wd, err := os.Getwd()
	if err != nil {
		fmt.Println("Could not determine current working directory.", err)
		os.Exit(1)
	}

	// input flags
	flag.StringVar(&listenAddress, "l", ":8000", "The address for the server to listen on. Examples: :80, 127.0.0.1:8000")
	flag.StringVar(&path, "p", wd, "The path for the server to serve.")
	flag.StringVar(&user, "user", "", "Simple Auth username.")
	flag.StringVar(&pass, "pass", "", "Simple Auth password.")
	flag.Parse()
}

func main() {

	// watch for kill signals and exit nicely
	go watchForKill()

	// setup a spinner
	spin = spinner.New(spinner.CharSets[14], time.Millisecond*50)
	spin.Color("green")
	spin.FinalMSG = "" // causes it to erase the current message when stopped

	// formulate the proper clickable listen address for output
	var listenAddressClickable string
	if len(strings.Split(listenAddress, ".")) < 4 {
		listenAddressClickable = "http://0.0.0.0" + listenAddress
	} else {
		listenAddressClickable = "http://" + listenAddress
	}

	// configure the spinner output and start it up
	var spinnerMessage string
	spinnerMessage = color.WhiteString(" %s", "Server running at ")
	spinnerMessage = spinnerMessage + color.YellowString("%s ", path)
	spinnerMessage = spinnerMessage + color.WhiteString("%s ", "on")
	spinnerMessage = spinnerMessage + color.GreenString("%s", listenAddressClickable)
	spin.Suffix = spinnerMessage
	spin.Start()

	// initialze a file server handler
	if len(user) > 0 && len(pass) > 0 {
		http.Handle("/", httpauth.SimpleBasicAuth(user, pass)((http.FileServer(http.Dir(path)))))
		http.HandleFunc("/upload", upload)
	} else {
		http.Handle("/", http.FileServer(http.Dir(path)))
	}
	err := http.ListenAndServe(listenAddress, nil)
	spin.Stop()
	if err != nil {
		fmt.Println("Server exited with error: ", err)
		os.Exit(254)
	}
	os.Exit(0)
}

// watchForKill watches for kill and interrupt signals
func watchForKill() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)
	<-c
	spin.Stop()
	os.Exit(0)
}
