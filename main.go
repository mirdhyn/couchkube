package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

type Database struct {
	Name *string `json:"name"`
}

var (
	namespace         = "couchdb"
	kubeHost          = "130.211.100.80"
	kubeUser          = "admin"
	kubePassword      = "u7M56qMNEWkWZts2"
	couchdbUser       = "admin"
	couchdbPassword   = "admin"
	getServiceTimeout = 15

	kube *client.Client
)

func createDatabaseHandler(w http.ResponseWriter, r *http.Request) {
	var db Database
	body, _ := ioutil.ReadAll(r.Body)

	err := json.Unmarshal(body, &db)
	if err != nil || db.Name == nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad request"))
		return
	}

	// Fail if db already exists (but with retrun code "200 OK")
	_, err = kube.Services(namespace).Get(*db.Name)
	if err == nil {
		w.Write([]byte("Database already exists"))
		return
	}

	// Create new deployemnt
	err = CreateDeployment(db)
	if err != nil {
		log.Println("could not create deployment controller '%s': %s", *db.Name, err)
	}
	log.Printf("deployment controller '%s' created", *db.Name)

	// Create new service
	err = CreateService(db)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Could not create database."))
		log.Printf("could not create service '%s': %s", *db.Name, err)
		return
	}
	log.Printf("service '%s' created", *db.Name)
	w.WriteHeader(http.StatusCreated)
}

func getServiceHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	if _, err := GetService(name); err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	for i := 0; i <= getServiceTimeout; i++ {
		endpoint, err := GetService(name)
		if err == nil && len(endpoint) > 0 {
			w.Write([]byte(endpoint))
			return
		}
		time.Sleep(1 * time.Second)
	}

	w.Write([]byte("Pending endpoint assignment..."))

}

func main() {

	config := &restclient.Config{
		Host:     kubeHost,
		Username: kubeUser,
		Password: kubePassword,
		Insecure: true,
	}

	var err error
	kube, err = client.New(config)
	if err != nil {
		log.Fatalf("could not connect to Kubernetes API: %s", err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/{name}", getServiceHandler).Methods("GET")
	r.HandleFunc("/", createDatabaseHandler).Methods("POST")

	log.Println("Starting...")
	log.Fatal(http.ListenAndServe(":8080", r))

}
