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
	"os/exec"
	"strconv"
	"strings"
	"text/template"

	_"github.com/go-sql-driver/mysql"
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
	QuestionNumber int `db:"questionNumber"`
}

type Answer struct {
	Id      string `db:"id"`
	Login      string `db:"login"`
	Answer   string `db:"answer"`
	QuestionNumber      string `db:"questionNumber"`
	State      string `db:"state"`
}

type Path struct {
	FirstPart string
	SecondPart      string
}

var (
	configFile = flag.String("Config", "conf.json", "Where to read the Config from")
	store = sessions.NewCookieStore([]byte(config.SessionSalt))
)

var config struct {
	MysqlLogin    string `json:"mysqlLogin"`
	MysqlPassword string `json:"mysqlPassword"`
	MysqlHost     string `json:"mysqlHost"`
	MysqlVmDb       string `json:"mysqlVmDb"`
	MysqlAnswersDb       string `json:"mysqlAnswersDb"`
	PathToConfVm  string `json:"pathToConfVm"`
	SessionSalt   string `json:"sessionSalt"`
	PathToAnswers   string `json:"pathToAnswers"`
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

func (s *server) getAnswersFromDb() []Answer {
	Answers := make([]Answer, 0)
	if err := s.Db.Select(&Answers, "SELECT * FROM `answers` "); err != nil {
		log.Println(err)
		return Answers
	}
	return Answers
}

func (s *server) getAnswerFromDbByLogin( login string, questionNumber int ) Answer {
	answer := Answer{}
	if err := s.Db.Get(&answer, "SELECT answer FROM `answers` WHERE questionNumber=? AND login=? ",questionNumber, login); err != nil {
		log.Println(err)
		return answer
	}
	return answer
}

func (s *server) getUserFromDbById(id int) Users {
	users := Users{}
	var login string
	if err := s.Db.Get(&login, "SELECT login FROM `suggestions` WHERE id=?", id); err != nil {
		log.Println(err)
	}
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
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/login/", 302)
	}else {
		user := s.getUserFromDbByLogin(session.Values["login"].(string))
		if user.Permission != "admin" {
			http.Redirect(w, r, "/login/", 302)
			return
		}
	}
	vm := s.getUsersFromDb()
	testTemplate, _ := template.ParseFiles("templates/test.html")
	if err := testTemplate.Execute(w, vm); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) answersHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/login/", 302)
	}else {
		user := s.getUserFromDbByLogin(session.Values["login"].(string))
		if user.Permission != "admin" {
			http.Redirect(w, r, "/login/", 302)
			return
		}
	}
	vm := s.getAnswersFromDb()
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
			session.Values["login"]= login
			session.Save(r,w)
			if userInfo.Permission == "admin"{
				http.Redirect(w,r, "/suggestions/",302)
			}else{
				http.Redirect(w,r, "/user/",302)
			}
		}else{
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
	session.Save(r, w)
	http.Redirect(w, r, "/login/", 302)
}

func (s *server) suggestionsHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
			http.Redirect(w, r, "/login/", 302)
		}else {
		user := s.getUserFromDbByLogin(session.Values["login"].(string))
		if user.Permission != "admin" {
			http.Redirect(w, r, "/login/", 302)
			return
		}
	}
	r.ParseForm()
	post := r.PostForm
	id := strings.Join(post["id_suggestion"], "")
	solution := strings.Join(post["solution"], "")
	id_int, _ := strconv.Atoi(id)
	userOfSuggestion := s.getUserFromDbById(id_int)
	if solution == "1" {
		if _, err := s.Db.Exec("UPDATE `suggestions` SET status = ? WHERE id = ?", solution, id); err != nil {
			log.Println(err)
			return
		}
		s.makeVagrantConf(id_int)
		s.executeTestGenerator(userOfSuggestion.Login)
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
	user := s.getUserFromDbByLogin(session.Values["login"].(string))
	if vmcstatus == "1" {
		//user := Suggestionsstr{"", session.Values["login"].(string), session.Values["state"].(string), "0"}
		if _, err := s.Db.Exec("INSERT INTO `suggestions` (`login`, `state`, `status`) VALUES (?,?,?)", user.Login, user.State, "0"); err != nil {
			log.Println(err)
			return
		}
	} else {
		log.Println("NothingToDo")
	}
	testTemplate, _ := template.ParseFiles("templates/user.html")
	if err := testTemplate.Execute(w, user); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) submitHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/login/", 302)
		return
	}
	user := s.getUserFromDbByLogin(session.Values["login"].(string))
	r.ParseForm()
	post := r.PostForm
	submitStatus := strings.Join(post["submit"], "")
	if submitStatus == "1"{
		answer := strings.Join(post["answer"], "")
		rightAnswer := s.getAnswerFromDbByLogin(session.Values["login"].(string), user.QuestionNumber)
		if answer == rightAnswer.Answer {
			if _, err := s.Db.Exec("UPDATE `answers` SET state = ? WHERE questionNumber = ?", "1", user.QuestionNumber); err != nil {
				log.Println(err)
				return
			}
			if _, err := s.Db.Exec("UPDATE `users` SET questionNumber = ? WHERE login = ?", user.QuestionNumber+1, session.Values["login"].(string)); err != nil {
				log.Println(err)
				return
			}
		}else{
		}
	}
	submitTemplate, _ := template.ParseFiles("templates/submit.html")
	if err := submitTemplate.Execute(w, user); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) makeVagrantConf(id int) {
	usr := s.getUserFromDbById(id)
	generatedvm := Vmvariables{(usr.Password + "\\n" + usr.Password + "\\n"), "fedora-26vm", "testvm" + strconv.Itoa(id), "2048", usr.Login}
	tmpl, _ := template.New("test").ParseFiles("VagrantConfSample.txt")
	var b bytes.Buffer
	tmpl.ExecuteTemplate(&b,"VagrantConfSample.txt", generatedvm)
	var z []byte
	z = b.Bytes()
	if _, err := os.Stat(config.PathToConfVm + string(usr.State) + "/" + usr.Login + "/"); os.IsNotExist(err) {
		os.Mkdir(config.PathToConfVm+string(usr.State)+"/", 0777)
		os.Mkdir(config.PathToConfVm+string(usr.State)+"/"+usr.Login+"/", 0777)
	}
	ioutil.WriteFile(config.PathToConfVm+string(usr.State)+"/"+usr.Login+"/VagrantConf", z, 0777)
	return
}

func (s *server) makeTestingScripts(login string, file string,script string) string{
	usr := s.getUserFromDbByLogin(login)
	generatedAnswers := Path{config.PathToAnswers+usr.State+"/"+usr.Login+"/", file}
	tmpl, _ := template.New("test2").ParseFiles("scripts/"+script)
	var q bytes.Buffer
	tmpl.ExecuteTemplate(&q, script, generatedAnswers)
	var w []byte
	w = q.Bytes()
	if _, err := os.Stat(config.PathToAnswers + string(usr.State) + "/" + usr.Login + "/"); os.IsNotExist(err) {
		os.Mkdir(config.PathToAnswers+string(usr.State)+"/", 0777)
		os.Mkdir(config.PathToAnswers+string(usr.State)+"/"+usr.Login+"/", 0777)
	}
	ioutil.WriteFile(config.PathToAnswers+string(usr.State)+"/"+usr.Login+"/"+script, w, 0777)
	return config.PathToAnswers+string(usr.State)+"/"+usr.Login+"/"
}

func (s *server) executeBash(path string, file string) {
	cmd := exec.Command("bash", path+file)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return
}
func (s *server) executeRm(path string, file string) {
	cmd := exec.Command("rm", path+file)
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
	return
}

func (s *server) openFile(path string, filename string, login string, questionNumber string) {
	file, err := os.Open(path+filename)
	if err != nil {
		// handle the error here
		return
	}
	defer file.Close()
	stat,_ := file.Stat()
	bs := make([]byte, stat.Size()-1)
	file.Read(bs)
	user := s.getUserFromDbByLogin(login)
	if _, err := s.Db.Exec("INSERT INTO `answers` (`id`, `login`, `answer`, `state`, `questionNumber`) VALUES (NULL, ?, ?, '0',?)", user.Login, string(bs), questionNumber); err != nil {
		log.Println(err)
		return
	}
}


func (s *server) executeTestGenerator(login string) {
	path := s.makeTestingScripts(login,"temp.txt", "md5.sh")
	s.executeBash(path, "md5.sh")
	s.openFile(path, "temp.txt", login, "0")
	s.executeRm(path, "temp.txt")
	s.makeTestingScripts(login,"key.txt", "iso.sh")
	s.executeBash(path, "iso.sh")
	s.openFile(path, "key.txt", login, "1")
	s.executeRm(path, "key.txt")
	s.executeRm(path, "temp.file")
	s.executeRm(path, "key.tar.gz")
	s.makeTestingScripts(login,"text.txt", "find.sh")
	s.executeBash(path, "find.sh")
	if _, err := s.Db.Exec("INSERT INTO `answers` (`id`, `login`, `answer`, `state`, `questionNumber`) VALUES (NULL, ?, ?, '0',?)", login, "AVWEF-EFfQE-wD3FF-asFew-WEFWQ", "2"); err != nil {
		log.Println(err)
		return
	}
	//s.openFile(path, "key.txt", login, "1")
}

func main() {
	loadConfig(*configFile)
	s := server{
		Db: sqlx.MustConnect("mysql", config.MysqlLogin+":"+config.MysqlPassword+"@tcp("+config.MysqlHost+")/"+config.MysqlVmDb+"?charset=utf8"),
	}
	defer s.Db.Close()
	http.HandleFunc("/users/", s.usersHandler)
	http.HandleFunc("/suggestions/", s.suggestionsHandler)
	http.HandleFunc("/user/", s.userHandler)
	http.HandleFunc("/login/", s.loginHandler)
	http.HandleFunc("/logout/", s.logoutHandler)
	http.HandleFunc("/answers/", s.answersHandler)
	http.HandleFunc("/submit/", s.submitHandler)
	http.ListenAndServe(":4006", nil)
}
