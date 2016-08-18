package main

import (
	"log"
	"time"

	"github.com/patrickjuchli/couch"
)

func createDB(name string) error {
	log.Printf("Waiting for '%s' endpoint...", name)
	for {
		endpoint, err := GetService(name)
		if err == nil && len(endpoint) != 0 {
			s := couch.NewServer(endpoint, couch.NewCredentials(couchdbUser, couchdbPassword))
			db := s.Database(name)

			if !db.Exists() {
				err = db.Create()
				if err != nil {
					log.Printf("unable to create couchdb database '%s': %s", name, err)
					return err
				}
				log.Printf("database '%s' created", name)
			}
			setReplication(name, db)
			break
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

func setReplication(name string, db *couch.Database) {
	for _, service := range GetServicesName() {
		if service != name {
			endpoint, err := GetService(service)
			if err != nil {
				continue
			}
			s := couch.NewServer(endpoint, couch.NewCredentials(couchdbUser, couchdbPassword))
			target := s.Database(name)
			replication, err := db.ReplicateTo(target, true)
			if err != nil {
				log.Printf("unable to replicate db '%s' to target '%s' (%s)", name, service, endpoint)
			}
			active, _ := replication.IsActive()
			continuous := replication.Continuous()

			if active && continuous {
				log.Printf("continuous replication active: '%s' -> '%s' ", name, service)
			}
		}
	}
}
