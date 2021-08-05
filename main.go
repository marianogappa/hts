package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"

	"github.com/marianogappa/hts/signaltranspiler"
	"github.com/marianogappa/signal-checker/signalchecker"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		log.Fatal("$PORT must be set")
	}

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/transpile", transpileHandler)
	http.HandleFunc("/run", runHandler)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	if err := http.ListenAndServe(fmt.Sprintf(":%v", port), nil); err != nil {
		log.Fatal(err)
	}
}

func transpileHandler(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Input string `json:"input"`
	}
	decoder := json.NewDecoder(r.Body)
	var b body
	err := decoder.Decode(&b)
	if err != nil {
		log.Fatal(err)
	}

	st := signaltranspiler.NewSignalTranspiler()
	output, _ := st.Transpile(b.Input)
	fmt.Printf("%+v\n", output)
	bs, err := json.Marshal(output)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintln(w, string(bs))
}

func runHandler(w http.ResponseWriter, r *http.Request) {
	type body struct {
		Input string `json:"input"`
	}
	decoder := json.NewDecoder(r.Body)
	var b body
	err := decoder.Decode(&b)
	if err != nil {
		log.Fatal(err)
	}

	st := signaltranspiler.NewSignalTranspiler()
	output, _ := st.Transpile(b.Input)

	signalOutput, err := signalchecker.CheckSignal(output.SignalInput)
	if err != nil {
		log.Println(err)
	}
	output.SignalOutput = signalOutput

	bs, err := json.Marshal(output)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Fprintln(w, string(bs))
}

//go:embed index.html
var templString string

func rootHandler(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("html-tmpl").Parse(string(templString)))
	err := t.Execute(w, nil)
	if err != nil {
		log.Fatal(err)
	}
}
