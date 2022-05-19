package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/XMonetae-DeFi/apollo/generate"
	_ "github.com/lib/pq"
	"github.com/zclconf/go-cty/cty"
)

var (
	ErrNotConnected = errors.New("not connected to database, use Connect() or check if DB is accessible")
)

type DbSettings struct {
	User           string `yaml:"user"`
	Password       string `yaml:"password"`
	Name           string `yaml:"name"`
	Host           string `yaml:"host"`
	DefaultTimeout time.Duration
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

// CreateTable drop and creates the table if it exists, otherwise just creates it.
func (db DB) CreateTable(ctx context.Context, name string, cols map[string]cty.Value) error {
	ddl, err := generate.GenerateCreateDDL(name, cols)
	if err != nil {
		return err
	}

	_, err = db.pdb.ExecContext(ctx, ddl)
	if err != nil {
		return err
	}
	return nil
}

func (db DB) InsertResult(name string, toInsert map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), db.Settings.DefaultTimeout)
	defer cancel()

	ddl := generate.GenerateInsertSQL(name, toInsert)

	_, err := db.pdb.ExecContext(ctx, ddl)
	if err != nil {
		return err
	}
	return nil
}
