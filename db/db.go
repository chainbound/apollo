package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/chainbound/apollo/generate"
	"github.com/chainbound/apollo/log"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
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
	logger   zerolog.Logger
}

func NewDB(s DbSettings) *DB {
	connStr := fmt.Sprintf("postgresql://%s:%s@%s/%s?sslmode=disable", s.User, s.Password, s.Host, s.Name)

	return &DB{
		Settings: s,
		connStr:  connStr,
		logger:   log.NewLogger("db"),
	}
}

func (db *DB) Connect() (*DB, error) {
	pdb, err := sql.Open("postgres", db.connStr)
	if err != nil {
		return nil, err
	}

	db.pdb = pdb
	if !db.IsConnected() {
		return nil, fmt.Errorf("can't connect to db at %s", db.connStr)
	}

	db.logger.Debug().Str("conn_str", db.connStr).Msg("connected to db")

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

	db.logger.Trace().Str("ddl", ddl).Msg("generated create stmt")
	_, err = db.pdb.ExecContext(ctx, ddl)
	if err != nil {
		return fmt.Errorf("creating table: %w", err)
	}

	db.logger.Debug().Str("table_name", name).Msg("created table")

	return nil
}

func (db DB) InsertResult(name string, toInsert map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), db.Settings.DefaultTimeout)
	defer cancel()

	ddl := generate.GenerateInsertSQL(name, toInsert)
	db.logger.Trace().Str("ddl", ddl).Msg("generated insert stmt")

	_, err := db.pdb.ExecContext(ctx, ddl)
	if err != nil {
		return fmt.Errorf("inserting result: %w", err)
	}

	db.logger.Debug().Str("table_name", name).Msg("inserted result")
	return nil
}
