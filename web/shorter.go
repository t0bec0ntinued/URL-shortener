package main

import (
	"database/sql"
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

	router := mux.NewRouter() // инициализация роутера
	staticFileDirectory := http.Dir("../ui/static/")
	staticFileHandler := http.StripPrefix("/static/", http.FileServer(staticFileDirectory))
	router.PathPrefix("/static/").Handler(staticFileHandler).Methods("GET")
	router.HandleFunc("/", home)       // отслеживаем переход по localhost:8080/
	router.HandleFunc("/{key}", short) //  отслеживаем переход по localhost:8080/{key}

	log.Fatal(http.ListenAndServe(":8080", router))

}

const (
	letters = "0123456789aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ" //Алфавит символов для сокращённой ссылки
	connStr = "user=test_user password=1234 dbname=test_db sslmode=disable"    //Данные для подключения к БД
)

var m = make(map[string]string)

type Result struct {
	Input  string // Введённая ссылка
	Output string // Сокращённая ссылка
}

// Забираем ссылку из поля ввода и сохраняем в БД вместе с сокращённой
func home(w http.ResponseWriter, r *http.Request) {
	templ, _ := template.ParseFiles("../ui/html/index.html")
	result := Result{}

	if r.Method == "POST" {
		if !isValid(r.FormValue("s")) { // Проверяем ссылку на корректность
			result.Input = "" //Если некорректна, оставляем пустой
		} else {

			result.Input = r.FormValue("s") //Помещаем в структуру экземпляр исходной ссылки
			result.Output = shorting()      //И результат её сокращения
			s := Result{

				Output: result.Output,
			}
			s.createQR() //Создаём QR-код

			if os.Args[len(os.Args)-1] == "-d" {

				db, err := sql.Open("postgres", connStr) //Подключаемся к БД
				if err != nil {
					panic(err)
				}
				defer db.Close()                                                                           //Закрываем соединение
				db.Exec("insert into testtb (input, output) values ($1, $2)", result.Input, result.Output) //Помещаем в БД исходную и сокращённую ссылки

			} else {

				m[result.Output] = result.Input // Ставим в соответствие ключ (сокращённую сслыку) и значение (исходную)

			}

		}

	}
	templ.Execute(w, result)
}

// Функция генерации QR-кода
func (s Result) createQR() {
	qrCode := qrcode.WriteFile("http://localhost:8080/"+s.Output, qrcode.Medium, 256, "../ui/static/img/qr.jpg")

	if qrCode != nil {
		fmt.Printf("Sorry couldn't create qrcode:,%v", qrCode)

	}
}

// Функция обработки перехода по QR-коду и сокращённой ссылке
func short(w http.ResponseWriter, r *http.Request) {
	var link string

	vars := mux.Vars(r)
	if os.Args[len(os.Args)-1] == "-d" {

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			panic(err)
		}
		defer db.Close()
		rows := db.QueryRow("select input from testtb where output=$1 limit 1", vars["key"]) //Вытаскиваем исходную ссылку, соответствующую ключу
		rows.Scan(&link)

	} else {
		link = m[vars["key"]] // То же самое, но не из БД, а из карты соответствий
	}

	fmt.Fprintf(w, "<script>location='%s';</script>", link) //Вставляем исходную ссылку в строку поиска
}

var mass []string

// Функция сокращения ссылок
func shorting() string {
	if os.Args[len(os.Args)-1] == "-d" { //При использовании флага -d
		var maxShort string

		var maxLen int
		db, err := sql.Open("postgres", connStr) // Открываем соединение с БД

		if err != nil {

			panic(err)

		}

		defer db.Close() // После завершения закрываем соединение с БД

		rowsMax := db.QueryRow("SELECT length(output) FROM public.testtb order by length(output) desc limit 1") // Берём из базы данных максимальную длину output
		rowsMax.Scan(&maxLen)                                                                                   // Записываем в maxLen

		rows := db.QueryRow("select output from testtb where length(output) = $1 order by output desc limit 1", maxLen) // Берём из базы данных output с максимальным лингвистическим значением
		rows.Scan(&maxShort)                                                                                            // Записываем в maxShort

		for i := len(maxShort) - 1; i >= 0; i-- {
			if maxShort[i] != 'Z' {
				return maxShort[:i] + string(letters[strings.Index(letters, string(maxShort[i]))+1]) + maxShort[i+1:]
			}
		}
		return string(letters[0]) + strings.Repeat(string(letters[0]), maxLen)
	} else { //Без флага
		maxLen := 0

		keys := make([]string, 0, len(m))
		for key := range m {
			if len(key) > maxLen {
				maxLen = len(key)
			}
			keys = append(keys, key)
		}
		maxShort := maxShort(keys)

		for i := len(maxShort) - 1; i >= 0; i-- {
			if maxShort[i] != 'Z' {
				return maxShort[:i] + string(letters[strings.Index(letters, string(maxShort[i]))+1]) + maxShort[i+1:]
			}
		}
		return string(letters[0]) + strings.Repeat(string(letters[0]), maxLen)
	}
}

// Функция определяет лингвистически максимальное значение сокращённой ссылки при хранении локально
func maxShort(s []string) string {
	maxShort := "0"
	for _, val := range s {
		if len(val) > len(maxShort) {
			maxShort = val
			continue
		} else if len(val) < len(maxShort) {
			continue
		} else {
			for i, v := range val {
				if strings.Index(letters, string(v)) > strings.Index(letters, string(maxShort[i])) {
					maxShort = val
				}
			}
		}
	}
	return maxShort
}

func isValid(token string) bool { //Проверяем введённую ссылку на валидность
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
