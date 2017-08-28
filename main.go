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
	Login          string `db:"login"`
	Password       string `db:"password"`
	Permission     string `db:"permission"`
	FirstName      string `db:"firstName"`
	LastName       string `db:"lastName"`
	State          string `db:"state"`
	QuestionNumber int `db:"questionNumber"`
}

type Answer struct {
	Id             string `db:"id"`
	Login          string `db:"login"`
	Answer         string `db:"answer"`
	QuestionNumber string `db:"questionNumber"`
	State          string `db:"state"`
}

type Path struct {
	FirstPart  string
	SecondPart string
}

var (
	configFile = flag.String("Config", "conf.json", "Where to read the Config from")
	store      = sessions.NewCookieStore([]byte(config.SessionSalt))
)

var config struct {
	MysqlLogin     string `json:"mysqlLogin"`
	MysqlPassword  string `json:"mysqlPassword"`
	MysqlHost      string `json:"mysqlHost"`
	MysqlVmDb      string `json:"mysqlVmDb"`
	MysqlAnswersDb string `json:"mysqlAnswersDb"`
	PathToConfVm   string `json:"pathToConfVm"`
	SessionSalt    string `json:"sessionSalt"`
	PathToAnswers  string `json:"pathToAnswers"`
}

func loadConfig(path string) error {
	jsonData, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonData, &config)
}

func (s *server) getUsersFromDb() ([]Users, error) {
	var users []Users
	err := s.Db.Select(&users, "SELECT * FROM `users` ")
	return users, err
}

func (s *server) getAnswersFromDb() ([]Answer, error) {
	var answers []Answer
	err := s.Db.Select(&answers, "SELECT * FROM `answers` ")
	return answers, err
}

func (s *server) getAnswerFromDbByLogin(login string, questionNumber int) (Answer, error) {
	var answer Answer
	err := s.Db.Get(&answer, "SELECT answer FROM `answers` WHERE questionNumber=? AND login=? ", questionNumber, login)
	return answer, err
}

func (s *server) getUserFromDbById(id int) (Users, error) {
	var users Users
	err := s.Db.Get(&users, "SELECT * FROM `users` WHERE login=(SELECT login FROM `suggestions` WHERE id=?)", id)
	return users, err
}

func (s *server) makeVagrantConf(id int) error {
	usr, err := s.getUserFromDbById(id)
	if err != nil {
		return err
	}
	generatedVm := Vmvariables{usr.Password + "\\n" + usr.Password + "\\n", "fedora-26vm", "testvm" + strconv.Itoa(id), "2048", usr.Login}
	template, err := template.ParseFiles("VagrantConfSample.txt")
	if err != nil {
		return err
	}
	var b bytes.Buffer
	template.ExecuteTemplate(&b, "VagrantConfSample.txt", generatedVm)
	var z []byte
	z = b.Bytes()
	if _, err := os.Stat(config.PathToConfVm + string(usr.State) + "/" + usr.Login + "/"); os.IsNotExist(err) {
		os.Mkdir(config.PathToConfVm+string(usr.State)+"/", 0777)
		os.Mkdir(config.PathToConfVm+string(usr.State)+"/"+usr.Login+"/", 0777)
	}
	ioutil.WriteFile(config.PathToConfVm+string(usr.State)+"/"+usr.Login+"/Vagrantfile", z, 0777)
	s.executeVagrant(config.PathToConfVm + string(usr.State) + "/" + usr.Login + "/")
	return err
}

func (s *server) makeTestingScripts(login string, file string, script string) (string, error) {
	usr, err := s.getUserFromDbByLogin(login)
	if err != nil {
		return "", err
	}
	generatedAnswers := Path{config.PathToAnswers + usr.State + "/" + usr.Login + "/", file}
	template, _ := template.ParseFiles("scripts/" + script)
	var q bytes.Buffer
	template.ExecuteTemplate(&q, script, generatedAnswers)
	w := q.Bytes()
	if _, err := os.Stat(config.PathToAnswers + string(usr.State) + "/" + usr.Login + "/"); os.IsNotExist(err) {
		os.Mkdir(config.PathToAnswers+string(usr.State)+"/", 0777)
		os.Mkdir(config.PathToAnswers+string(usr.State)+"/"+usr.Login+"/", 0777)
	}
	ioutil.WriteFile(config.PathToAnswers+string(usr.State)+"/"+usr.Login+"/"+script, w, 0777)
	return config.PathToAnswers + string(usr.State) + "/" + usr.Login + "/", err
}

func (s *server) executeBash(path string, file string) error {
	cmd := exec.Command("bash", path+file)
	err := cmd.Run()
	return err
}

func (s *server) executeVagrant(path string) error {
	vagrant := exec.Command("/bin/sh", "-c", "cd "+path+"; vagrant up")
	err := vagrant.Run()
	return err
}

func (s *server) executeRm(path string, file string) error {
	cmd := exec.Command("rm", path+file)
	err := cmd.Run()
	return err
}

func (s *server) openFile(path string, filename string, login string, questionNumber string) error {
	bs, err := ioutil.ReadFile(path + filename)
	if err != nil {
		return err
	}

	user, err := s.getUserFromDbByLogin(login)
	if err != nil {
		return err
	}
	if _, err := s.Db.Exec("INSERT INTO `answers` (`id`, `login`, `answer`, `state`, `questionNumber`) VALUES (NULL, ?, ?, '0',?)", user.Login, string(bs), questionNumber); err != nil {
		return err
	}
	return err
}

func (s *server) executeTestGenerator(login string) error {
	path, err := s.makeTestingScripts(login, "temp.txt", "md5.sh")
	if err != nil {
		return err
	}
	err = s.executeBash(path, "md5.sh")
	if err != nil {
		return err
	}
	s.openFile(path, "temp.txt", login, "0")
	if err != nil {
		return err
	}
	s.executeRm(path, "temp.txt")
	if err != nil {
		return err
	}
	s.makeTestingScripts(login, "key.txt", "iso.sh")
	if err != nil {
		return err
	}
	s.executeBash(path, "iso.sh")
	if err != nil {
		return err
	}
	s.openFile(path, "key.txt", login, "1")
	if err != nil {
		return err
	}
	s.executeRm(path, "key.txt")
	if err != nil {
		return err
	}
	s.executeRm(path, "temp.file")
	if err != nil {
		return err
	}
	s.executeRm(path, "key.tar.gz")
	if err != nil {
		return err
	}
	s.makeTestingScripts(login, "text.txt", "find.sh")
	if err != nil {
		return err
	}
	s.executeBash(path, "find.sh")
	if err != nil {
		return err
	}
	if _, err := s.Db.Exec("INSERT INTO `answers` (`id`, `login`, `answer`, `state`, `questionNumber`) VALUES (NULL, ?, ?, '0',?)", login, "AVWEF-EFfQE-wD3FF-asFew-WEFWQ", "2"); err != nil {
		log.Println(err)
		return err
	}
	//s.openFile(path, "key.txt", login, "1")
}

func (s *server) getUserFromDbByLogin(login string) (Users, error) {
	var users Users
	err := s.Db.Get(&users, "SELECT * FROM `users` WHERE login=?", login)
	return users, err
}

func (s *server) usersHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/login/", 302)
		return
	}
	user, err := s.getUserFromDbByLogin(session.Values["login"].(string))
	if err != nil {
		log.Println(err)
		return
	}
	if user.Permission != "admin" {
		http.Redirect(w, r, "/login/", 302)
		return
	}

	vm, err := s.getUsersFromDb()
	if err != nil {
		log.Println(err)
		return
	}
	testTemplate, err := template.ParseFiles("templates/test.html")
	if err != nil {
		log.Println(err)
		return
	}
	if err := testTemplate.Execute(w, vm); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) answersHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/login/", 302)
		return
	}
	user, err := s.getUserFromDbByLogin(session.Values["login"].(string))
	if err != nil {
		log.Println(err)
		return
	}
	if user.Permission != "admin" {
		http.Redirect(w, r, "/login/", 302)
		return
	}

	vm, err := s.getAnswersFromDb()
	if err != nil {
		log.Println(err)
		return
	}
	testTemplate, err := template.ParseFiles("templates/test.html")
	if err != nil {
		log.Println(err)
		return
	}
	if err := testTemplate.Execute(w, vm); err != nil {
		log.Println(err)
		return
	}
}

func (s *server) loginHandler(w http.ResponseWriter, r *http.Request) {
	/*
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/login/", 302)
		return
	}
	user, err := s.getUserFromDbByLogin(session.Values["login"].(string))
	if err != nil {
		log.Println(err)
		return
	}
	if user.Permission != "admin" {
		http.Redirect(w, r, "/login/", 302)
		return
	}
	 */
	session, _ := store.Get(r, "loginData")
	login := r.PostForm.Get("login")
	password := r.PostForm.Get(session.Values["password"].(string))
	state := r.PostForm.Get(session.Values["log in"].(string))
	userInfo, err := s.getUserFromDbByLogin(login)
	if err != nil {
		log.Println(err)
		return
	}
	if state == "1" {
		if password == userInfo.Password {
			session.Values["login"] = login
			session.Save(r, w)
			if userInfo.Permission == "admin" {
				http.Redirect(w, r, "/suggestions/", 302)
			} else {
				http.Redirect(w, r, "/user/", 302)
			}
			return
		}
	}
	testTemplate, err := template.ParseFiles("templates/login.html")
	if err != nil {
		log.Println(err)
		return
	}
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
	log.Println("huy")
	return
}

func (s *server) suggestionsHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/login/", 302)
		return
	}
	user, err := s.getUserFromDbByLogin(session.Values["login"].(string))
	if err != nil {
		log.Println(err)
		return
	}
	if user.Permission != "admin" {
		http.Redirect(w, r, "/login/", 302)
		return
	}
	r.ParseForm()
	solution := r.PostForm.Get(session.Values["solution"].(string))
	id_int, err := strconv.Atoi(r.PostForm.Get("id"))
	if err != nil {
		log.Println(err)
		return
	}
	userOfSuggestion, err := s.getUserFromDbById(id_int)
	if err != nil {
		log.Println(err)
		return
	}
	if solution == "1" {
		if _, err := s.Db.Exec("UPDATE `suggestions` SET status = ? WHERE id = ?", solution, id_int); err != nil {
			log.Println(err)
			return
		}
		err = s.makeVagrantConf(id_int)
		if err != nil {
			log.Println(err)
			return
		}
		err = s.executeTestGenerator(userOfSuggestion.Login)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("Job's done!")
	} else if solution == "2" {
		if _, err := s.Db.Exec("UPDATE `suggestions` SET status = ? WHERE id = ?", solution, id); err != nil {
			log.Println(err)
			return
		}
	}
	var suggestionDb []Suggestionsstr
	if err := s.Db.Select(&suggestionDb, "SELECT * FROM `suggestions` "); err != nil {
		log.Println(err)
		return
	}
	testTemplate, err := template.ParseFiles("templates/suggestions.html")
	if err != nil {
		log.Println(err)
		return
	}
	if err := testTemplate.Execute(w, suggestionDb); err != nil {
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
	vmcstatus := session.Values["createvm"].(string)
	user, err := s.getUserFromDbByLogin(session.Values["login"].(string))
	if err != nil {
		log.Println(err)
		return
	}
	if vmcstatus == "1" {
		if _, err := s.Db.Exec("INSERT INTO `suggestions` (`login`, `state`, `status`) VALUES (?,?,?)", user.Login, user.State, "0"); err != nil {
			log.Println(err)
			return
		}
	}
	testTemplate, err := template.ParseFiles("templates/user.html")
	if err != nil {
		log.Println(err)
		return
	}
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
	user, err := s.getUserFromDbByLogin(session.Values["login"].(string))
	if err != nil {
		log.Println(err)
		return
	}
	submitStatus := session.Values["submit"]
	if submitStatus == "1" {
		answer := session.Values["answer"]
		rightAnswer, err := s.getAnswerFromDbByLogin(session.Values["login"].(string), user.QuestionNumber)
		if err != nil {
			log.Println(err)
			return
		}
		if answer == rightAnswer.Answer {
			if _, err := s.Db.Exec("UPDATE `answers` SET state = ? WHERE questionNumber = ?", "1", user.QuestionNumber); err != nil {
				log.Println(err)
				return
			}
			if _, err := s.Db.Exec("UPDATE `users` SET questionNumber = ? WHERE login = ?", user.QuestionNumber+1, session.Values["login"].(string)); err != nil {
				log.Println(err)
				return
			}
		}
	}
	submitTemplate, err := template.ParseFiles("templates/submit.html")
	if err != nil {
		log.Println(err)
		return
	}
	if err := submitTemplate.Execute(w, user); err != nil {
		log.Println(err)
		return
	}
}

func main() {
	err := loadConfig(*configFile)
	if err != nil {
		panic(err)
	}
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
	err = http.ListenAndServe(":4006", nil)
	if err != nil {
		panic(err)
	}
}
