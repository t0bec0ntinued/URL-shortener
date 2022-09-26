package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/skip2/go-qrcode"
)

func main() {

	router := mux.NewRouter()

	router.HandleFunc("/", home)
	router.HandleFunc("/{key}", short)

	log.Fatal(http.ListenAndServe(":8080", router))

}

var m = make(map[string]string)

type Result struct {
	Input  string
	Output string
}

func home(w http.ResponseWriter, r *http.Request) {
	templ, _ := template.ParseFiles("templates/index.html")
	result := Result{}

	if r.Method == "POST" {
		if !isValid(r.FormValue("s")) {
			result.Input = ""
		} else {

			result.Input = r.FormValue("s")
			result.Output = shorting()
			qrCode := qrcode.WriteFile(result.Output, qrcode.Medium, 1, "qr.png")

			if qrCode != nil {
				fmt.Printf("Sorry couldn't create qrcode:,%v", qrCode)

			}

			if os.Args[len(os.Args)-1] == "-d" {

				db, err := sql.Open("postgres", connStr)
				if err != nil {
					panic(err)
				}
				defer db.Close()
				db.Exec("insert into mytable (input, output) values ($1, $2)", result.Input, result.Output)

			} else {

				m[result.Output] = result.Input

			}

		}

	}
	templ.Execute(w, result)
}

func short(w http.ResponseWriter, r *http.Request) {
	var link string

	vars := mux.Vars(r)
	if os.Args[len(os.Args)-1] == "-d" {

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			panic(err)
		}
		defer db.Close()
		rows := db.QueryRow("select input from mytable where output=$1 limit 1", vars["key"])
		rows.Scan(&link)

	} else {
		link = m[vars["key"]]
	}

	fmt.Fprintf(w, "<script>location='%s';</script>", link)
}

const (
	connStr = "user=postgres password=1234 dbname=db sslmode=disable"
)

func randomBytes(n int) []byte {

	b := make([]byte, n)
	_, err := rand.Read(b)

	if err != nil {
		return nil
	}

	return b
}

func shorting() string {
	b := randomBytes(32)
	k := base64.URLEncoding.EncodeToString(b)

	k = strings.ReplaceAll(k, "+", "")
	k = strings.ReplaceAll(k, "=", "")
	k = strings.ReplaceAll(k, "-", "")
	k = strings.ReplaceAll(k, "_", "")
	k = strings.ReplaceAll(k, "/", "")

	return k[:31]

}

func isValid(token string) bool {
	_, err := url.ParseRequestURI(token)
	if err != nil {
		return false
	}
	u, err := url.Parse(token)
	if err != nil || u.Host == "" {
		return false
	}
	return true
}

