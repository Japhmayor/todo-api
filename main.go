package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
)

var user = "root"
var password = "bontusfavor1994?"
var port = "3306"
var dbName = "todo_db"
var host = "127.0.0.1"
var connectionString = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local", user, password, host, port, dbName)
var db, ferr = gorm.Open("mysql", connectionString)

type TodoModel struct {
	Id          int `gorm:"primary_key"`
	Description string
	Completed   bool
}

// Returns {"alive": true} whenever called.
func Health(w http.ResponseWriter, r *http.Request) {
	log.Info("Api health is ok")
	w.Header().Set("Content-Type", "application/json")
	io.WriteString(w, `{"alive": true}`)
}

func init() {
	log.SetFormatter(&log.TextFormatter{})
	log.SetReportCaller(true)
}

func main() {
	log.Info("Starting todo api server")
	if ferr != nil {
		fmt.Printf("Cannot connect to %s database", dbName)
		log.Fatal("This is the error:", ferr)
	} else {
		fmt.Printf("We are connected to the %s database", dbName)
	}
	// auto migrate mysql database
	db.Debug().DropTableIfExists(&TodoModel{})
	db.Debug().AutoMigrate(&TodoModel{})

	router := mux.NewRouter()
	router.HandleFunc("/health", Health).Methods("GET")
	router.HandleFunc("/todo", CreateItem).Methods("POST")
	router.HandleFunc("/todo/{id}", DeleteItem).Methods("DELETE")
	router.HandleFunc("/todo/{id}", UpdateItem).Methods("POST")
	router.HandleFunc("/todo-completed", GetCompletedItems).Methods("GET")
	router.HandleFunc("/todo-incomplete", GetIncompleteItems).Methods("GET")
	handler := cors.New(cors.Options{AllowedMethods: []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"}}).Handler(router)
	http.ListenAndServe(":8000", handler)
}

func CreateItem(w http.ResponseWriter, r *http.Request) {
	description := r.FormValue("description")
	log.WithFields(log.Fields{"description": description}).Info("Add new todoitem. Saving to database")
	todo := &TodoModel{Description: description, Completed: true}
	db.Create(&todo)
	// query the database
	result := db.Last(&todo)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result.Value)
}

func UpdateItem(w http.ResponseWriter, r *http.Request) {
	// get url parameter
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	// test if the todo item is in the database
	err := GetItemById(id)
	if err != true {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"updated": false, "error": "Record not found"}`)
	} else {
		completed, _ := strconv.ParseBool(r.FormValue("completed"))
		log.WithFields(log.Fields{"id": id, "completed": completed}).Info("Updating Item")
		todo := &TodoModel{}
		db.First(&todo, id)
		todo.Completed = completed
		db.Save(&todo)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"updated": true}`)
	}
}

func GetItemById(id int) bool {
	todo := &TodoModel{}
	result := db.First(&todo, id)
	if result.Error != nil {
		log.Warn("Item not found in database")
		return false
	}
	return true
}

func DeleteItem(w http.ResponseWriter, r *http.Request) {
	// Get URL parameter from mux
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	// Test if the TodoItem exist in DB
	err := GetItemById(id)
	if err == false {
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"deleted": false, "error": "Record Not Found"}`)
	} else {
		log.WithFields(log.Fields{"Id": id}).Info("Deleting TodoItem")
		todo := &TodoModel{}
		db.First(&todo, id)
		db.Delete(&todo)
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"deleted": true}`)
	}
}

func GetCompletedItems(w http.ResponseWriter, r *http.Request) {
	log.Info("Get completed items")
	completedItems := GetItems(true)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(completedItems)
}

func GetIncompleteItems(w http.ResponseWriter, r *http.Request) {
	log.Info("Get Incomplete TodoItems")
	IncompleteTodoItems := GetItems(false)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(IncompleteTodoItems)
}

func GetItems(completed bool) interface{} {
	var todos []TodoModel
	TodoItems := db.Where("completed = ?", completed).Find(&todos).Value
	return TodoItems
}
