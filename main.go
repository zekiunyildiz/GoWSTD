package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var rnd *renderer.Render
var db *mgo.Database

const (
	hostname       string = "localhost:27017"
	dbName         string = "demo_todo"
	collectionName string = "todo"
	port           string = ":9000"
)

type (
	todoModel struct {
		ID        bson.ObjectId `bson: "_id, omitempyt"`
		Title     string        `bson: "title"`
		Completed bool          `bson: "completed"`
		CreatedAt time.Time     `bson:"createdAt"`
	}

	todo struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Completed string    `json:"completed"`
		CreatedAt time.Time `json:"created_at"`
	}
)

func init() {
	rnd = renderer.New()
	sess, err := mgo.Dial(hostname)
	checkErr(err)
	sess.SetMode(mgo.Monotonic, true)
	db = sess.DB(dbName)
}

func homeHandler(w http.ResponseWriter, r * http.Request)  {
	err := rnd.Template(w, http.StatusOK, []string{"static/home.tpl"}, nil)
	checkErr(err)
}

func fetchTodos(w http.ResponseWriter, r * http.Request)  {
	todos := []todoModel{}
	if err := db.C(collectionName).Find(bson.M{}).All(&todos); err != nil{
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message":"Failed to fetch todo",
			"error":err,
		})
		return
	}
	todoList := []todo{}

	for _,t:=range todos{
		todoList = append(todoList, todo{
			ID: t.ID.Hex(),
			Title: t.Title,
			Completed: t.Completed,
			CreatedAt: t.CreatedAt,
		})
	}
	rnd.JSON(w, http.StatusOK, renderer.M{
		"data":todoList,
	})
}

func creteTodo(w http.ResponseWriter, r *http.Request)  {
	var t todo
	if err:=json.NewDecoder(r.Body).Decode(&t); err!=nil {
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}

	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M {
			"message" : "The title is required",
		})
		return
	}

	tm := todoModel{
		ID: bson.NewObjectId(),
		Title: t.Title,
		Completed: false,
		CreatedAt: time.Now(),
	}

	if err:= db.C(collectionName).Insert(&tm); err !=nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message":"Failed to save todo",
			"error":err,
		})
		return
	}

	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message":"Todo created succesfully",
		"todo_id":tm.ID.Hex(),
	})
	
}

func deleteTodo(w http.ResponseWriter, r *http.Request){
	id:=strings.TrimSpace(chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id) {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "The id is invalid",
		})
		return
	}

	if err:= db.C(collectionName).RemoveId(bson.ObjectIdHex(id)); err!=nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message" : "failed to delete todo",
			"error" : err,
		})
		return
	}
	rnd.JSON(w, http.StatusOK, renderer.M{
		"message":"todo deleted successfully",
	})
}

func main() {
	stopChan := make(chan os.Signal)
	signal.Notify(stopChan, os.Interrupt)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", homeHandler)
	r.Mount("/todo", todoHandlers())

	srv := &http.Server{
		Addr: port,
		Handler: r,
		ReadTimeout: 60*time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout: 60*time.Second,
	}
	go func ()  {
		log.Println("Listening on port", port)
		if err := srv.ListenAndServe(); err !=nil {
			log.Println("Listen:%s\n", err)
		}
	}()

	<-stopChan
	log.Println("shutting down server...", port)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel(
		log.Println("serve gracefully stopped")
	)
}

func todoHandlers() http.Handler {
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router) {
		r.Get("/", fetchTodos)
		r.Post("/", createTodo)
		r.Put("/{id}", updateTodo)
		r.Delete("/{id}", deleteTodo)
	})
	return rg
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
