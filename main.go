package main

import (
	"bytes"
	_"database/sql"
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"text/template"

	_"github.com/go-sql-driver/mysql"
	_"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
	"github.com/gorilla/sessions"
)

type server struct {
	Db *sqlx.DB
}

type Vmvariables struct {
	Password string `db:"password"`
	Box      string `db:"box"`
	Hostname string `db:"hostname"`
	Memory   string `db:"memory"`
	Login    string `db:"login"`
}

type Suggestionsstr struct {
	Id     string `db:"id"`
	Login  string `db:"login"`
	State  string `db:"state"`
	Status string `db:"status"`
}

type Users struct {
	Login      string `db:"login"`
	Password   string `db:"password"`
	Permission string `db:"permission"`
	FirstName  string `db:"firstName"`
	LastName   string `db:"lastName"`
	State      string `db:"state"`
}

var (
	configFile = flag.String("Config", "conf.json", "Where to read the Config from")
	store = sessions.NewCookieStore([]byte(config.SessionSalt))
)

var config struct {
	MysqlLogin    string `json:"mysqlLogin"`
	MysqlPassword string `json:"mysqlPassword"`
	MysqlHost     string `json:"mysqlHost"`
	MysqlDb       string `json:"mysqlDb"`
	PathToConfVm  string `json:"pathToConfVm"`
	SessionSalt   string `json:"sessionSalt"`
}

func loadConfig(path string) error {
	jsonData, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, &config)
}

func (s *server) getUsersFromDb() []Users {
	users := make([]Users, 0)
	if err := s.Db.Select(&users, "SELECT * FROM `users` "); err != nil {
		log.Println(err)
		return users
	}
	return users
}

func (s *server) getUserFromDbById(id int) Users {
	users := Users{}
	var login string
	if err := s.Db.Get(&login, "SELECT login FROM `suggestions` WHERE id=?", id); err != nil {
		log.Println(err)
	}
	log.Println(login)
	if err := s.Db.Get(&users, "SELECT * FROM `users` WHERE login=?", login); err != nil {
		log.Println(err)
		return users
	}
	return users
}

func (s *server) getUserFromDbByLogin(login string) Users {
	users := Users{}
	if err := s.Db.Get(&users, "SELECT * FROM `users` WHERE login=?", login); err != nil {
		log.Println(err)
		return users
	}
	return users
}

func (s *server) usersHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil || session.Values["permission"] != "admin" {
		http.Redirect(w, r, "/login/", 302)
		return
	}
	vm := s.getUsersFromDb()
	testTemplate, _ := template.ParseFiles("templates/test.html")
	if err := testTemplate.Execute(w, vm); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) loginHandler(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	post := r.PostForm
	session, _ := store.Get(r, "loginData")
	login := strings.Join(post["login"], "")
	password := strings.Join(post["password"], "")
	state := strings.Join(post["log in"], "")
	userInfo := s.getUserFromDbByLogin(login)
	if state == "1"{
		if password == userInfo.Password {
			log.Println("zbs")
			session.Values["login"]= login
			session.Values["permission"] = userInfo.Permission
			session.Values["state"] = userInfo.State
			if userInfo.Permission == "admin"{
				session.Save(r,w)
				http.Redirect(w,r, "/suggestions/",302)
			}else{
				session.Save(r,w)
				http.Redirect(w,r, "/user/",302)
			}
		}else{
			log.Println("ne zbs")
		}
	}else{
		log.Println("no entering was found")
	}

	testTemplate, _ := template.ParseFiles("templates/login.html")
	if err := testTemplate.Execute(w, ""); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Loaded %s page from %s", r.URL.Path, r.Header.Get("X-Real-IP"))
	session, _ := store.Get(r, "loginData")
	session.Values["login"] = nil
	session.Values["permission"] = nil
	session.Save(r, w)
	http.Redirect(w, r, "/login/", 302)
}

func (s *server) suggestionsHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil || session.Values["permission"] != "admin" {
		http.Redirect(w, r, "/login/", 302)
		return
	}
	r.ParseForm()
	post := r.PostForm
	id := strings.Join(post["id_suggestion"], "")
	solution := strings.Join(post["solution"], "")
	if solution == "1" {
		if _, err := s.Db.Exec("UPDATE `suggestions` SET status = ? WHERE id = ?", solution, id); err != nil {
			log.Println(err)
			return
		}
		id_int, _ := strconv.Atoi(id)
		log.Println(id_int)
		s.makeVagrantConf(id_int)
		log.Println("Job's done!")
	} else if solution == "2" {
		if _, err := s.Db.Exec("UPDATE `suggestions` SET status = ? WHERE id = ?", solution, id); err != nil {
			log.Println(err)
			return
		}
	}
	suggdb := make([]Suggestionsstr, 0)
	if err := s.Db.Select(&suggdb, "SELECT * FROM `suggestions` "); err != nil {
		log.Println(err)
		return
	}
	testTemplate, _ := template.ParseFiles("templates/suggestions.html")
	if err := testTemplate.Execute(w, suggdb); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) userHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/login/", 302)
		return
	}
	r.ParseForm()
	post := r.PostForm
	vmcstatus := strings.Join(post["createvm"], "")
	if vmcstatus == "1" {
		user := Suggestionsstr{"", session.Values["login"].(string), session.Values["state"].(string), "0"}
		if _, err := s.Db.Exec("INSERT INTO `suggestions` (`login`, `state`, `status`) VALUES (?,?,?)", user.Login, user.State, user.Status); err != nil {
			log.Println(err)
			return
		}
	} else {
		log.Println("NothingToDo")
	}
	testTemplate, _ := template.ParseFiles("templates/user.html")
	if err := testTemplate.Execute(w, ""); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) makeVagrantConf(id int) {
	usr := s.getUserFromDbById(id)
	generatedvm := Vmvariables{(usr.Password + "\\n" + usr.Password + "\\n"), "fedora-26vm", "testvm" + strconv.Itoa(id), "2048", usr.Login}
	tmpl, _ := template.New("test").ParseFiles("VagrantConfSample.txt")
	var b bytes.Buffer
	tmpl.ExecuteTemplate(&b, "VagrantConfSample.txt", generatedvm)
	var z []byte
	z = b.Bytes()
	if _, err := os.Stat(config.PathToConfVm + string(usr.State) + "/" + usr.Login + "/"); os.IsNotExist(err) {
		os.Mkdir(config.PathToConfVm+string(usr.State)+"/", 0777)
		os.Mkdir(config.PathToConfVm+string(usr.State)+"/"+usr.Login+"/", 0777)
	}
	log.Println(config.PathToConfVm + string(usr.State) + "/")
	ioutil.WriteFile(config.PathToConfVm+string(usr.State)+"/"+usr.Login+"/VagrantConf", z, 0777)
	return
}

func main() {
	loadConfig(*configFile)
	s := server{
		Db: sqlx.MustConnect("mysql", config.MysqlLogin+":"+config.MysqlPassword+"@tcp("+config.MysqlHost+")/"+config.MysqlDb+"?charset=utf8"),
	}

	defer s.Db.Close()
	http.HandleFunc("/users/", s.usersHandler)
	http.HandleFunc("/suggestions/", s.suggestionsHandler)
	http.HandleFunc("/user/", s.userHandler)
	http.HandleFunc("/login/", s.loginHandler)
	http.HandleFunc("/logout/", s.logoutHandler)
	http.ListenAndServe(":4006", nil)
}
