package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"k8s.io/kubernetes/pkg/client/restclient"
	client "k8s.io/kubernetes/pkg/client/unversioned"
)

// Database struct for database creation request json decoding
type Database struct {
	Name *string `json:"name"`
}

var (
	namespace         = "couchdb"
	kubeHost          = "localhost:8080"
	kubeUser          = "admin"
	kubePassword      = "admin"
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

	// Fail if db already exists (but still return code "200 OK")
	_, err = kube.Services(namespace).Get(*db.Name)
	if err == nil {
		w.Write([]byte("Database already exists"))
		return
	}

	// Create new deployemnt
	err = CreateDeployment(db)
	if err != nil {
		log.Printf("could not create deployment controller '%s': %s", *db.Name, err)
	}
	log.Printf("deployment controller '%s' created", *db.Name)

	// Create new service (and return "201 Created" if successful)
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

	// Loop until either the LoadBalancer endpoint has been assigned
	// or timeout is exceeded
	for i := 0; i <= getServiceTimeout; i++ {
		endpoint, err := GetService(name)
		// Return LoadBalancer endpoint if found
		if err == nil && len(endpoint) > 0 {
			w.Write([]byte(endpoint))
			return
		}
		time.Sleep(1 * time.Second)
	}

	w.Write([]byte("Pending endpoint assignment..."))

}

func main() {

	// Setting-up environment variables
	if envNamespace := os.Getenv("COUCHKUBE_NAMESPACE"); len(envNamespace) > 0 {
		namespace = envNamespace
	}

	if envKubeHost := os.Getenv("COUCHKUBE_HOST"); len(envKubeHost) > 0 {
		kubeHost = envKubeHost
	}

	if envKubeUser := os.Getenv("COUCHKUBE_USER"); len(envKubeUser) > 0 {
		kubeUser = envKubeUser
	}

	if envKubePassword := os.Getenv("COUCHKUBE_PASSWORD"); len(envKubePassword) > 0 {
		kubePassword = envKubePassword
	}

	if envCouchdbUser := os.Getenv("COUCHKUBE_COUCHDB_USER"); len(envCouchdbUser) > 0 {
		couchdbUser = envCouchdbUser
	}

	if envCouchdbPassword := os.Getenv("COUCHKUBE_COUCHDB_PASSWORD"); len(envCouchdbPassword) > 0 {
		couchdbPassword = envCouchdbPassword
	}

	if envTimeout := os.Getenv("COUCHKUBE_TIMEOUT"); len(envTimeout) > 0 {
		if t, err := strconv.Atoi(envTimeout); err != nil {
			getServiceTimeout = t
		}
	}

	// Setting connection to Kubernetes
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

	if err = CreateNamespace(); err != nil {
		log.Fatalf("unable to create namespace '%s': %s", namespace, err)
	}

	// Routing
	r := mux.NewRouter()
	r.HandleFunc("/{name}", getServiceHandler).Methods("GET")
	r.HandleFunc("/", createDatabaseHandler).Methods("POST")

	log.Println("Listening...")
	log.Fatal(http.ListenAndServe(":8080", r))

}
