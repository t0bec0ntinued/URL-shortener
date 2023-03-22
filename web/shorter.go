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

	router := mux.NewRouter()
	staticFileDirectory := http.Dir("../ui/static/")
	staticFileHandler := http.StripPrefix("/static/", http.FileServer(staticFileDirectory))
	router.PathPrefix("/static/").Handler(staticFileHandler).Methods("GET")
	router.HandleFunc("/", home)
	router.HandleFunc("/{key}", short)

	log.Fatal(http.ListenAndServe(":8080", router))

}

const (
	letters = "0123456789aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ" //Алфавит символов для сокращённой ссылки
	connStr = "user=postgres password=1234 dbname=db sslmode=disable"          //Данные для подключения к БД
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
				defer db.Close()                                                                            //Закрываем соединение
				db.Exec("insert into mytable (input, output) values ($1, $2)", result.Input, result.Output) //Помещаем в БД исходную и сокращённую ссылки

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
		rows := db.QueryRow("select input from mytable where output=$1 limit 1", vars["key"]) //Вытаскиваем исходную ссылку, соответствующую ключу
		rows.Scan(&link)

	} else {
		link = m[vars["key"]] // То же самое, но не из БД, а из карты соответствий
	}

	fmt.Fprintf(w, "<script>location='%s';</script>", link) //Вставляем исходную ссылку в строку поиска
}

// Функция сокращения ссылок
func shorting() string {

	if os.Args[len(os.Args)-1] == "-d" { //Для запуска с флагом (хранение в БД)
		var strVal string
		var strLen, lessRow int
		db, err := sql.Open("postgres", connStr) //Открываем соединение с базой

		if err != nil {

			panic(err)

		}

		defer db.Close()                                                          //Закрываем
		rowsLen := db.QueryRow("SELECT max(length(output)) FROM public.mytable ") //Выбираем строки с наибольшей длиной
		rowsLen.Scan(&strLen)

		rowsVal := db.QueryRow("select max(output) from public.mytable where length(output) = $1", strLen) //Выбираем наибольшее лингвистическое значение
		rowsVal.Scan(&strVal)                                                                              //с максимальной длиной

		//Проверка на пропуски более коротких строк
		for currLen := 0; currLen < strLen; currLen++ { //Изменяем текущую длину строки от 0 до самой длинной из имеющихся в базе
			rowsLenLess := db.QueryRow("SELECT count(output) where length(output)=$1 FROM public.mytable ", currLen) //Выбираем количество строк с текущей длиной
			rowsLenLess.Scan(&lessRow)
			fmt.Println("count: ", lessRow)
			if lessRow != len(letters)*currLen { //Если существующее количество строк не совпадает с максимальным количеством

				rowsValLess := db.QueryRow("select max(output) from public.mytable where length(output) = $1", currLen) //Записываем в макс. лингвистическое значение
				rowsValLess.Scan(&strVal)                                                                               //макс. значение при текущей длине

			}
		}
		for i := len(strVal) - 1; i >= 0; i-- { //Увеличиваем значение ссылок
			if strVal[i] != 'Z' {
				index := strings.Index(letters, string(strVal[i]))
				strVal = strings.Replace(strVal, string(strVal[i]), "", 1)
				strVal = strVal + string(letters[index+1])
				return strVal
			}
		}
		return string(letters[0]) + strings.Repeat(string(letters[0]), strLen) //Если достигли символа Z, увеличиваем длину строки
		//Нумерацию начинаем с нуля

	} else { //Для запуска без флага (хранение в памяти)
		strLen := 0
		keys := make([]string, 0, len(m))
		for k, _ := range m {
			if len(k) > strLen {
				strLen = len(k)
			}
			keys = append(keys, k)
		}
		strVal := "0"
		for _, val := range keys {
			if len(val) > len(strVal) {
				strVal = val
				continue
			} else if len(val) < len(strVal) {
				continue
			} else {
				for i, v := range val {
					if strings.Index(letters, string(v)) > strings.Index(letters, string(strVal[i])) {
						strVal = val
					}
				}
			}
		}

		for i := len(strVal) - 1; i >= 0; i-- {
			if strVal[i] != 'Z' {
				index := strings.Index(letters, string(strVal[i]))
				strVal = strings.Replace(strVal, string(strVal[i]), "", 1)
				strVal = strVal + string(letters[index+1])
				return strVal
			}
		}
		return string(letters[0]) + strings.Repeat(string(letters[0]), strLen)

	}

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
