package db

import (
	"log"
	"testing"
)

func newDB() *DB {
	db, err := NewDB(DbSettings{
		Host:     "172.17.0.2",
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
	if !db.IsConnected() {
		t.Fatal("not connected")
	}
}

func TestCreateTable(t *testing.T) {
	// db := newDB()
	// schema, err := generate.ParseV2("../schema.v2.yml")
	// if err != nil {
	// 	t.Fatal(err)
	// }

	// ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	// defer cancel()

	// for _, s := range schema.Contracts {
	// 	err = db.CreateTable(ctx, *s)
	// 	if err != nil {
	// 		t.Fatal(err)
	// 	}
	// }

}
