package main

import (
	"database/sql"
	//"errors"
	"fmt"
	_ "github.com/lib/pq"
	//"github.com/ttacon/libphonenumber"
	_ "go/parser"
	"html/template"
	"log"
	//"net"
	"net/http"
	"os"
	//"strconv"
)

var (
	IPaddrWeb, addrWeb, webPort string
	errserv                     int
)
var partials = []string{
	"./static/base.html",
	"./static/personal_new.html",
	"./static/personal_show.html",
	"./static/personals_index.html",
	"./static/css/footer.partial.tmpl.html",
	"./static/css/header.partial.tmpl.html",
	"./static/css/sidebar.partial.tmpl.html",
}
var admin struct { // администратор
	User     string
	Email    string
	Passw    string
	ErrEmail string // ошибка ввода почты
	Passpass string
	Ready    string // 1 - идентификация прошла
	Errors   string // "1" - ошибка при вводе полей
	Empty    string // "1" - остались пустые поля
}

type person struct { // данные по сотруднику при вводе и отображении в personal.HTML
	Title   string
	Kadr    string
	Address string
	Ready   string // "1" - ввод корректен
	Errors  string // "1" - ошибка при вводе полей
	Empty   string // "1" - остались пустые поля
}

//var personalhtml person // переменная по сотруднику при вводе и отображении в personal.HTML

type frombase struct { // строка  при чтении/записи из/в базы personaldb
	id      int
	title   string
	kadr    string
	address string
}

var (
	personals struct {
		Ready       string
		Persontable []person //person // таблица по сотрудниам  в personals_index.html
	}
)

func CheckError(err error) {
	if err != nil {
		panic(err)
	}
}

const (
	answerServer     = "Hello, I am a server."
	readyServer      = "I'm ready!"
	defaultNet       = "tcp"
	defaultIp        = "192.168.1.101"
	defaultLocalhost = "localhost"
	defaultPort      = "8181"
)
const (
	defaultUser  = "yp"
	defaultEmail = "yp@yp.com"
	defaultPassw = "123"
)

// проверка на ввод  'Y = 1
func yesNo() int {
	var yesNo string
	len := 4
	data := make([]byte, len)
	n, err := os.Stdin.Read(data)
	yesNo = string(data[0 : n-1])
	if err == nil && (yesNo == "Y" || yesNo == "y" || yesNo == "Н" || yesNo == "н") {
		return 1
	} else {
		return 0
	}
}

func server(addrWeb string, db *sql.DB) {
	http.HandleFunc("/", indexHandler)
	http.Handle("/personals_index", http.HandlerFunc(personalsIndexHandler(db)))
	http.Handle("/personal_new", http.HandlerFunc(personalNewhandler(db)))
	http.Handle("/personal_show", http.HandlerFunc(personalShowhandler(db)))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
	fmt.Println("Топай на web страницу--->" + addrWeb + "!") // отладочная печать
	err := http.ListenAndServe(addrWeb, nil)
	if err != nil {
		errserv = 1
	} else {
		errserv = 0
	}
	return
}

// первая страница проверка доступа
func indexHandler(w http.ResponseWriter, req *http.Request) {
	files := append(partials, "./static/index.html")

	t, err := template.ParseFiles(files...) // Parse template file.
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Index Internal Server ParseFiles Error", http.StatusInternalServerError)
		return
	}
	admin.User = defaultUser
	admin.Email = defaultEmail
	admin.Passw = defaultPassw
	admin.Ready = "1"
	err = t.ExecuteTemplate(w, "base", admin)
	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Execute Error", http.StatusInternalServerError)
		return
	}
}

// просмотр таблицы из personaldb
func personalsIndexHandler(db *sql.DB) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {

		files := append(partials, "./static/personals_index.html")
		t, err := template.ParseFiles(files...) // Parse template file.
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal PersonalsIndex ParseFiles Error", http.StatusInternalServerError)
			return
		}
		del := req.URL.Query().Get("del")
		title := req.URL.Query().Get("title")
		if del == "del" {
			_, err = db.Exec("DELETE FROM personals WHERE title = $1", title)
			if err != nil { // удаление старой записи
				panic(err)
			}
		}
		personals.Persontable = nil

		rows, err1 := db.Query(`SELECT "title","kadr", "address" FROM "personals"`)
		if err1 != nil {
			fmt.Println(" table Personals ошибка чтения ")
			panic(err1)
		}
		defer rows.Close()

		for rows.Next() {
			var p frombase
			err = rows.Scan( // пересылка  данных строки базы personals в personrow
				//&p.id,
				&p.title,
				&p.kadr,
				&p.address,
			)
			if err != nil {
				//fmt.Println("indexPersonals ошибка распаковки строки ")
				panic(err)
			}
			var personalhtml person
			personalhtml.Title = p.title
			personalhtml.Kadr = p.kadr
			personalhtml.Address = p.address
			personalhtml.Ready = "1"
			personalhtml.Errors = "0"
			personalhtml.Empty = "0"
			personals.Persontable = append( // добавление строки в таблицу Personalstab для personals_index.html
				personals.Persontable,
				personalhtml,
			)
		}
		//fmt.Println(personals.Persontable)
		personals.Ready = "1"
		err = t.ExecuteTemplate(w, "base", personals)
		if err != nil {
			log.Println(err.Error())
			fmt.Println("indexPersonals ошибка записи  ")
			http.Error(w, "Internal Server Execute Error indexPersonals", http.StatusInternalServerError)
			return
		}
	}
}

// просмотр записи из personaldb
func personalShowhandler(db *sql.DB) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {

		files := append(partials, "./static/personal_show.html")
		t, err := template.ParseFiles(files...) // Parse template file.
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server ParseFiles Error", http.StatusInternalServerError)
			return
		}
		title := req.URL.Query().Get("title")
		row := db.QueryRow("SELECT * FROM personals WHERE title=$1", title)
		var personalhtml person
		personalhtml.Ready = "1"  // 1 - ввод успешный
		personalhtml.Errors = "0" // 1 - ошибки при вводе
		personalhtml.Empty = "0"  // 1 - есть пустые поля

		// чтение строки из таблицы

		var p frombase
		err = row.Scan( // пересылка  данных строки базы personals в personrow
			&p.id,
			&p.title,
			&p.kadr,
			&p.address,
		)
		if err != nil {
			fmt.Println("indexShow ошибка распаковки строки ")
			panic(err)
			return
		}
		personalhtml.Title = p.title
		personalhtml.Kadr = p.kadr
		personalhtml.Address = p.address
		err = t.ExecuteTemplate(w, "base", personalhtml)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Execute Error", http.StatusInternalServerError)
			return
		}
	}
}

// новая запись формы personal в базу personaldb
func personalNewhandler(db *sql.DB) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {

		files := append(partials, "./static/personal_new.html")
		t, err := template.ParseFiles(files...) // Parse template file.

		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server ParseFiles Error personalNew", http.StatusInternalServerError)
			return
		}
		var personalhtml person
		personalhtml.Ready = "0"
		if req.Method == "POST" {
			req.ParseForm()
			personalhtml.Ready = "0"  // 1 - ввод успешный
			personalhtml.Errors = "0" // 1 - ошибки при вводе
			personalhtml.Empty = "0"  // 1 - есть пустые поля
			personalhtml.Title = req.Form["title"][0]
			personalhtml.Kadr = req.Form["kadr"][0]
			personalhtml.Address = req.Form["address"][0]

			if personalhtml.Title == "" || personalhtml.Kadr == "" || personalhtml.Address == "" {
				personalhtml.Empty = "1"
				personalhtml.Errors = "1"
			}
			if personalhtml.Errors == "0" {
				var personals frombase
				personalhtml.Ready = "1"
				personals.title = personalhtml.Title
				personals.kadr = personalhtml.Kadr
				personals.address = personalhtml.Address

				sqlStatement := `INSERT INTO personals (title, kadr, address) VALUES ($1,$2,$3)`
				_, err = db.Exec(sqlStatement,
					personals.title,
					personals.kadr,
					personals.address)
				if err != nil {
					fmt.Println("Ошибка записи новой строки в personals")
					fmt.Println(err)
				}
				//fmt.Println(res)
			}
		}
		err = t.ExecuteTemplate(w, "base", personalhtml)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Execute Error", http.StatusInternalServerError)
			return
		}
	}
}

func main() {
	var yes int

	IPaddrWeb = ""
	komand := 1
	fmt.Println("------------------------------------")
	fmt.Println("|          WEB server              |")
	fmt.Println("|    отвечаем на любые запросы!    |")
	fmt.Println("|                                  |")
	fmt.Println("|   (c) jiliaevyp@gmail.com        |")
	fmt.Println("------------------------------------")

	//// Создаем соединение с базой данных
	connStr := "user=yp password=12345 dbname=postgres sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("ошибка подключения к базе <jiliaevdb>")
		panic(err)
	} else {
		fmt.Println("Соединение с базой <jiliaevdb> установлено ")
	}
	defer db.Close()

	for komand == 1 {
		addrWeb = defaultLocalhost + ":" + defaultPort
		fmt.Println("Сервер:  ", addrWeb, "\n")
		fmt.Println("Загрузите web страницу")
		fmt.Println("-------------------------------------------------")
		fmt.Println("Адрес сервера:         ", addrWeb)
		fmt.Println("-------------------------------------------------")
		fmt.Print("Запускаю сервер? (Y)   ")
		fmt.Println("Отменить?  (Enter)")
		yes = yesNo() //yesNo()
		if yes == 1 {
			go server(addrWeb, db)
			if errserv != 0 {
				fmt.Print("*** Ошибка при загрузке сервера ***", "\n", "\n")
			} else {
				fmt.Println("---------------------------")
				fmt.Println(answerServer, "   ", addrWeb)
				fmt.Println(readyServer)
				fmt.Print("---------------------------", "\n")
			}
		} else {
			fmt.Print("\n", "Запуск отменен", "\n", "\n")
		}
		fmt.Print("Перезапустить? (Y)   ")
		fmt.Println("Закончить?  (Enter)")
		komand = yesNo()
	}
	fmt.Println("Рад был для Вас сделать что-то полезное !")
	fmt.Print("Обращайтесь в любое время без колебаний!", "\n", "\n")
}
