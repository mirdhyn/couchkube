package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

type Database struct {
	Name *string `json:"name"`
}

var (
	namespace       = "couchdb"
	kubeHost        = "104.155.15.42"
	kubeUser        = "admin"
	kubePassword    = "wf6invLPq78prAer"
	couchdbUser     = "admin"
	couchdbPassword = "admin"

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
		log.Fatalf("could not create deployment controller: %s", err)
	}
	log.Println("deployment controller created")

	// Create new service
	err = CreateService(db)
	if err != nil {
		log.Fatalf("could not create service: %s", err)
	}
	log.Println("service created")

	//w.Write([]byte(*db.Name))
	// return 201 Created
}

func getServiceHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]

	endpoint, err := GetService(name)
	if err != nil {
		// error
	}
	w.Write([]byte(endpoint))
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
