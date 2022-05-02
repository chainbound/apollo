package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/XMonetae-DeFi/apollo/chainservice"
	"github.com/XMonetae-DeFi/apollo/generate"
	_ "github.com/lib/pq"
)

var (
	ErrNotConnected = errors.New("not connected to database, use Connect() or check if DB is accessible")
)

type DbSettings struct {
	User     string
	Password string
	Name     string
	Host     string
}

type DB struct {
	Settings DbSettings
	pdb      *sql.DB
	connStr  string
}

func NewDB(s DbSettings) *DB {
	connStr := fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable", s.User, s.Password, s.Host, s.Name)

	return &DB{
		Settings: s,
		connStr:  connStr,
	}
}

func (db *DB) Connect() (*DB, error) {
	pdb, err := sql.Open("postgres", db.connStr)
	if err != nil {
		return nil, err
	}

	db.pdb = pdb
	return db, nil
}

func (db *DB) Ping(ctx context.Context) error {
	return db.pdb.PingContext(ctx)
}

func (db *DB) IsConnected() bool {
	if db.pdb == nil {
		return false
	} else {
		return db.pdb.Ping() == nil
	}
}

func (db DB) CreateTable(ctx context.Context, s generate.ContractSchemaV2) error {
	ddl, err := generate.GenerateDDL(s)
	if err != nil {
		return err
	}
	_, err = db.pdb.ExecContext(ctx, ddl)
	if err != nil {
		return err
	}
	return nil
}

func (db DB) InsertResult(ctx context.Context, res chainservice.CallResult) error {
	// _, err := db.pdb.ExecContext(ctx, ddl)
	// if err != nil {
	// 	return err
	// }
	// return nil
	return nil
}
