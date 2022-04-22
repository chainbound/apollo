package db

import (
	"log"
	"testing"
)

func newDB() *DB {
	db, err := NewDB(DbSettings{
		Host:     "localhost",
		User:     "chainreader",
		Password: "chainreader",
		Name:     "postgres",
	}).Connect()

	if err != nil {
		log.Fatal(err)
	}

	return db
}

func TestConnect(t *testing.T) {
	db := newDB()
	if !db.isConnected() {
		t.Fatal("not connected")
	}

}
