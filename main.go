package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"text/template"

	"github.com/BigPrettyTuna/testing_system/templates"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/sessions"
	"github.com/jmoiron/sqlx"
)

type server struct {
	Db *sqlx.DB
}
type (
	Users = templates.Users
	SuggestionsStr = templates.SuggestionsStr
	Answer = templates.Answer
)

type VmVariables struct {
	Password string `db:"password"`
	Box      string `db:"box"`
	Hostname string `db:"hostname"`
	Memory   string `db:"memory"`
	Login    string `db:"login"`
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
	generatedVm := VmVariables{Password: usr.Password + "\\n" + usr.Password + "\\n", Box: "fedora-26vm", Hostname: "testvm" + strconv.Itoa(id), Memory: "2048", Login: usr.Login}
	template, err := template.ParseFiles("VagrantConfSample.txt")
	if err != nil {
		return err
	}
	if _, err := os.Stat(config.PathToConfVm + string(usr.State) + "/" + usr.Login + "/"); os.IsNotExist(err) {
		os.Mkdir(config.PathToConfVm+string(usr.State)+"/", 0777)
		os.Mkdir(config.PathToConfVm+string(usr.State)+"/"+usr.Login+"/", 0777)
	}
	file, err := os.Create(config.PathToConfVm + string(usr.State) + "/" + usr.Login + "/Vagrantfile")
	log.Println(config.PathToConfVm + string(usr.State) + "/" + usr.Login + "/Vagrantfile")
	defer file.Close()
	template.ExecuteTemplate(file, "VagrantConfSample.txt", generatedVm)
	s.executeVagrant(config.PathToConfVm + string(usr.State) + "/" + usr.Login + "/")
	return err
}

func (s *server) makeTestingScripts(login string, file string, script string) (string, error) {
	usr, err := s.getUserFromDbByLogin(login)
	if err != nil {
		return "", err
	}
	generatedAnswers := Path{config.PathToAnswers + usr.State + "/" + usr.Login + "/", file}
	template, err := template.ParseFiles("scripts/" + script)
	if err != nil {
		return "", err
	}
	//TODO
	log.Println(config.PathToAnswers + string(usr.State) + "/" + usr.Login + "/" + script)
	if _, err := os.Stat(config.PathToAnswers + string(usr.State) + "/" + usr.Login + "/"); os.IsNotExist(err) {
		os.Mkdir(config.PathToAnswers+string(usr.State)+"/", 0777)
		os.Mkdir(config.PathToAnswers+string(usr.State)+"/"+usr.Login+"/", 0777)
	}
	fileScript, err := os.Create(config.PathToAnswers + string(usr.State) + "/" + usr.Login + "/" + script)
	defer fileScript.Close()
	template.ExecuteTemplate(fileScript, script, generatedAnswers)
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
	err = s.openFile(path, "temp.txt", login, "0")
	if err != nil {
		return err
	}
	err = s.executeRm(path, "temp.txt")
	if err != nil {
		return err
	}
	_, err = s.makeTestingScripts(login, "key.txt", "iso.sh")
	if err != nil {
		return err
	}
	err = s.executeBash(path, "iso.sh")
	if err != nil {
		return err
	}
	err = s.openFile(path, "key.txt", login, "1")
	if err != nil {
		return err
	}
	err = s.executeRm(path, "key.txt")
	if err != nil {
		return err
	}
	err = s.executeRm(path, "temp.file")
	if err != nil {
		return err
	}
	err = s.executeRm(path, "key.tar.gz")
	if err != nil {
		return err
	}
	_, err = s.makeTestingScripts(login, "text.txt", "find.sh")
	if err != nil {
		return err
	}
	err = s.executeBash(path, "find.sh")
	if err != nil {
		return err
	}
	if _, err := s.Db.Exec("INSERT INTO `answers` (`id`, `login`, `answer`, `state`, `questionNumber`) VALUES (NULL, ?, ?, '0',?)", login, "AVWEF-EFfQE-wD3FF-asFew-WEFWQ", "2"); err != nil {
		log.Println(err)
		return err
	}
	return err
}

func (s *server) getUserFromDbByLogin(login string) (Users, error) {
	var users Users
	err := s.Db.Get(&users, "SELECT * FROM `users` WHERE login=?", login)
	return users, err
}

func (s *server) usersHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/", 302)
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
	fmt.Fprint(w, templates.UsersPage(vm))
}

func (s *server) answersHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/", 302)
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
	fmt.Fprint(w, templates.AnswerPage(vm))
}

func (s *server) indexHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Loaded %s page from %s", r.URL.Path, r.Header.Get("X-Real-IP"))
	session, _ := store.Get(r, "loginData")
	r.ParseForm()
	userInfo, err := s.getUserFromDbByLogin(r.PostForm.Get("login"))
	if err != nil {
		log.Println(err)
		//TODO
	}
	log.Println(r.URL.Path)
	if session.Values["login"] != nil && r.URL.Path != "/logout" {
		if userInfo.Permission == "admin" {
			http.Redirect(w, r, "/suggestions/", 302)
			return
		} else {
			http.Redirect(w, r, "/user/", 302)
			return
		}
	}
	switch r.URL.Path {
	case "/login":
		if session.Values["login"] != nil || r.PostForm.Get("password") == "" {
			http.Redirect(w, r, "/", 302)
			return
		}
		if userInfo.Password == r.PostForm.Get("password") {
			session.Values["login"] = userInfo.Login
			session.Save(r, w)
			http.Redirect(w, r, "/", 302)
			return
		}

	case "/logout":
		session.Values["login"] = nil
		session.Save(r, w)
		http.Redirect(w, r, "/", 302)
		return
	}
	fmt.Fprint(w, templates.IndexPage())
}

func (s *server) suggestionsHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/", 302)
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
	solution := r.PostForm.Get("solution")
	id, err := strconv.Atoi(r.PostForm.Get("id_suggestion"))
	log.Println(r.PostForm.Get("id_suggestion"))
	log.Println(id)
	log.Println(solution)
	if err != nil {
		log.Println(err)
	}
	userOfSuggestion, err := s.getUserFromDbById(id)
	if err != nil {
		log.Println(err)
	}
	if solution == "1" {
		if _, err := s.Db.Exec("UPDATE `suggestions` SET status = ? WHERE id = ?", solution, id); err != nil {
			log.Println(err)
			return
		}
		log.Println("vagrantconf")
		err = s.makeVagrantConf(id)
		if err != nil {
			log.Println(err)
			return
		}
		log.Println("executeTestGenerator")
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
	var suggestionDb []SuggestionsStr
	if err := s.Db.Select(&suggestionDb, "SELECT * FROM `suggestions` "); err != nil {
		log.Println(err)
		return
	}
	fmt.Fprint(w, templates.SuggestionsPage(suggestionDb))
}

func (s *server) userHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	r.ParseForm()
	log.Println(session.Values["login"])
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/", 302)
		return
	}
	vmcstatus := r.PostForm.Get("createvm")
	log.Println(vmcstatus)
	user, err := s.getUserFromDbByLogin(session.Values["login"].(string))
	log.Println(session.Values["login"])
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
	fmt.Fprint(w, templates.UserPage(user))
}

func (s *server) submitHandler(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "loginData")
	if session.Values["login"] == nil {
		http.Redirect(w, r, "/", 302)
		return
	}
	user, err := s.getUserFromDbByLogin(session.Values["login"].(string))
	if err != nil {
		log.Println(err)
		return
	}
	r.ParseForm()
	submitStatus := r.PostForm.Get("submit")
	if submitStatus == "1" {
		answer := r.PostForm.Get("answer")
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
	fmt.Fprint(w, templates.SubmitPage(user))
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
	http.HandleFunc("/answers/", s.answersHandler)
	http.HandleFunc("/submit/", s.submitHandler)
	http.HandleFunc("/", s.indexHandler)
	err = http.ListenAndServe(":4006", nil)
	if err != nil {
		panic(err)
	}
}
