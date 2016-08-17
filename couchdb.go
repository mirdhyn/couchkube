package main

import "github.com/patrickjuchli/couch"

func createDB(endpoint string, name string) {
	cred := couch.NewCredentials(couchdbUser, couchdbPassword)
	s := couch.NewServer(endpoint, cred)
	db := s.Database(name)

	if !db.Exists() {
		db.Create()
	}
}
