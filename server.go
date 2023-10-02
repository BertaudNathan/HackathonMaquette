package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type PostInfo struct {
	Title       string
	Description string
	Date        string
	Img         string
}

// Compile templates on start of the application

func CreateDB() (db *sql.DB) {
	database, _ := sql.Open("sqlite3", "Base.db")
	statement, _ := database.Prepare("CREATE TABLE IF NOT EXISTS users (username TEXT NO NULL PRIMARY KEY, password TEXT NO NULL)")
	statement.Exec()
	statement, _ = database.Prepare("CREATE TABLE IF NOT EXISTS posts (id INTEGER PRIMARY KEY AUTOINCREMENT,title TEXT NO NULL, description TEXT NO NULL, date TEXT NO NULL,Img TEXT )")
	_, err := statement.Exec()
	if err != nil {
		panic(err.Error())
	}
	_, err = statement.Exec()
	if err != nil {
		panic(err.Error())
	}
	return database
}

func GetPosts(database *sql.DB) []PostInfo {
	title := ""
	description := ""
	date := ""
	img := ""
	id := 0
	result, err := database.Query("SELECT * FROM posts")
	if err != nil {
		panic(err)
	}
	defer result.Close()
	var allpost []PostInfo
	for result.Next() {
		if err := result.Scan(&id, &title, &description, &date, &img); err != nil {
			panic(err.Error())
		}
		allpost = append(allpost, PostInfo{title, description, date, img})
	}
	if err = result.Err(); err != nil {
		panic(err.Error())
	}
	return allpost
}

func HomeFunc(w http.ResponseWriter, r *http.Request) {
	tpl := template.Must(template.New("main").Funcs(template.FuncMap{"mod": func(i, j int) bool { return i%j == 0 }, "inc": func(i int) int {
		return i + 1
	}}).ParseGlob("*.html"))

	database, _ := sql.Open("sqlite3", "Base.db")
	info := GetPosts(database)
	fmt.Println(info)
	_ = tpl.ExecuteTemplate(w, "index.html", info)
	database.Close()
}
func PlanningFunc(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "planning.html")
}

func AboutFunc(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "propos.html")
}
func MemberFunc(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "bureau.html")
}
func ContactFunc(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "contact.html")
}
func FormFunc(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "form.html")
}

func FormVerif(w http.ResponseWriter, r *http.Request) {
	database, _ := sql.Open("sqlite3", "Base.db")
	if err := r.ParseForm(); err != nil {
		http.Redirect(w, r, "/nouveau", http.StatusForbidden)
		return
	}
	title := r.FormValue("title")
	desc := r.FormValue("desc")
	img := r.FormValue("img")
	date := time.Now().Format(time.ANSIC)
	AddPost(database, title, desc, date, img)
	database.Close()
	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

func LogInHandler(w http.ResponseWriter, r *http.Request) {
	t := template.Must(template.ParseFiles("login.html"))
	t.Execute(w, nil)
}

func GetPassword(database *sql.DB, username string) (string, error) {
	password := ""
	err := database.QueryRow("SELECT password FROM users WHERE username='" + username + "'").Scan(&password)
	return password, err
}

func LogInVerifHandler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		fmt.Println("1")
		fmt.Fprintf(w, "ParseForm() err: %v", err)
		return
	}
	name := r.FormValue("name")
	tryedPassword := r.FormValue("password")
	database, _ := sql.Open("sqlite3", "Base.db")
	password, err := GetPassword(database, name)
	if err != nil {
		fmt.Println("2")
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(password), []byte(tryedPassword)) == nil {
		u2, _ := uuid.NewV4()
		cookie := http.Cookie{
			Name:     "ConnexionCookie",
			Value:    u2.String(),
			MaxAge:   3600,
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
		}
		http.SetCookie(w, &cookie)
		database.Exec("")
		http.Redirect(w, r, "/admin", http.StatusTemporaryRedirect)
		fmt.Println("3")
		return
	}
	http.Redirect(w, r, "/login", http.StatusUnauthorized)
	fmt.Println("4")
}

func AddPost(database *sql.DB, title string, desc string, date string, img string) error {
	if !(title == "" || desc == "") {
		statement, err := database.Prepare("INSERT INTO posts (title,description,date,Img) VALUES(?,?,?,?)")
		if err != nil {
			return err
		}
		_, err = statement.Exec(title, desc, date, img)
		if err != nil {
			return err
		}
		return nil
	}
	return nil
}

func AdminHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("test admin")
	c, err := r.Cookie("ConnexionCookie")
	fmt.Println("yo ", c)
	if err != nil {
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
		return
	} else {
		tpl := template.Must(template.New("main").Funcs(template.FuncMap{"mod": func(i, j int) bool { return i%j == 0 }, "inc": func(i int) int {
			return i + 1
		}}).ParseGlob("*.html"))
		database, _ := sql.Open("sqlite3", "Base.db")
		info := GetPosts(database)
		fmt.Println(info)
		_ = tpl.ExecuteTemplate(w, "admin.html", info)
	}
}

func AddUser(database *sql.DB, nom string, password string) error {
	hash, _ := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)

	statement, _ := database.Prepare("INSERT INTO users (username, password) VALUES(?,?)")

	_, err := statement.Exec(nom, string(hash))
	if err != nil {
		return err
	}
	return nil
}

func TestHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "test.html")
}

func main() {
	mux := http.NewServeMux()
	CreateDB()
	cssFileServer := http.FileServer(http.Dir("css"))
	mux.Handle("/css/", http.StripPrefix("/css/", cssFileServer))
	mux.HandleFunc("/", HomeFunc)
	mux.HandleFunc("/A-propos", AboutFunc)
	mux.HandleFunc("/membres", MemberFunc)
	mux.HandleFunc("/nouveau", FormFunc)
	mux.HandleFunc("/planning", PlanningFunc)
	mux.HandleFunc("/contact", ContactFunc)
	mux.HandleFunc("/verif", FormVerif)
	mux.HandleFunc("/admin", AdminHandler)
	mux.HandleFunc("/login", LogInHandler)
	mux.HandleFunc("/logverif", LogInVerifHandler)
	mux.HandleFunc("/test", TestHandler)
	database, _ := sql.Open("sqlite3", "Base.db")
	AddUser(database, "Nathan", "admin")
	database.Close()
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatal(err)
	}
}
