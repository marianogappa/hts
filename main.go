package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/marianogappa/hts/signaltranspiler"
)

func main() {
	port := 8080
	if len(os.Args) >= 2 {
		port, _ = strconv.Atoi(os.Args[1])
	}

	http.HandleFunc("/", rootHandler)
	http.HandleFunc("/transpile", transpileHandler)

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

func rootHandler(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.New("html-tmpl").Parse(templString))
	err := t.Execute(w, nil)
	if err != nil {
		log.Fatal(err)
	}
}

var templString = `
	<!doctype HTML>
	<html>
		<header>
			<style>
				body {
					font-family: Consolas, monospace;
					font-size: 16px;
				}
				textarea{
					height: 250px;
					width: 700px;
					padding: 10px;
				}
				h1 {
					margin-top: 15px;
				}
				#result {
					border: 1px solid black;
					padding: 15px;
					background-color: #000;
				}
				.instruction {
					color: #00FF00;
				}
				.error {
					color: #FF0000;
				}
				.expression {
					color: #00FFFF;
				}
				.punctuation {
					color: #FFFFFF;
				}
				.errorLine {

				}
			</style>
			<script>
				async function transpile() {
					const input = document.querySelector('#input').value
					try {
						const response = await fetch('/transpile', {
						 method: 'POST',
						 headers: {
						   'Content-Type': 'application/json'
						   },
						   body: JSON.stringify({
							 input
							})
						 })
						 const data = await response.json()
						 console.log(data)
						 renderTokenizedInput(data.tokenizedInput)
						 renderErrors(data.errors)
						 renderWarnings(data.warnings)
					} catch(error) {
						  console.log(error)
					}
				}

				function renderTokenizedInput(tokenizedInput) {
					document.querySelector('#result').innerHTML = ''
					tokenizedInput.forEach((tokenizedLine) => {
						const elemLine = document.createElement('div')
						elemLine.classList.add('tokenizedLine')
						tokenizedLine.forEach((tokenizedToken) => {
							const elemToken = document.createElement('span')
							elemToken.classList.add(tokenizedToken.tokenType)
							elemToken.innerHTML = tokenizedToken.input
							elemLine.appendChild(elemToken)
						})
						document.querySelector('#result').appendChild(elemLine)
					})
				}
				function renderErrors(errors) {
					document.querySelector('#errors').innerHTML = ''
					errors.forEach((error) => {
						const errorLine = document.createElement('div')
						errorLine.classList.add('errorLine')
						errorLine.innerHTML = error
						document.querySelector('#errors').appendChild(errorLine)
					})
				}
				function renderWarnings(warnings) {
					document.querySelector('#warnings').innerHTML = ''
					warnings.forEach((error) => {
						const warningLine = document.createElement('div')
						warningLine.classList.add('warningLine')
						warningLine.innerHTML = error
						document.querySelector('#warnings').appendChild(warningLine)
					})
				}
			</script>
		</header>
		<body>
			<h1>Ready to transpile</h1>
			<textarea id="input"></textarea>
			<button onclick="transpile()">Transpile!</button>
			<h1>Transpilation result</h1>
			<div id="result"></div>
			<h1>Errors</h1>
			<div id="errors"></div>
			<h1>Warnings</h1>
			<div id="warnings"></div>
		</body>
	</html>
`
