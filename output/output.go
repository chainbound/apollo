package output

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/chainbound/apollo/db"
	"github.com/chainbound/apollo/generate"
	"github.com/chainbound/apollo/log"
	"github.com/rs/zerolog"
	"github.com/zclconf/go-cty/cty"
)

type OutputHandler struct {
	stdout bool
	csv    *CsvHandler
	db     *db.DB
	// tables keeps track of which tables have been created
	tables map[string]bool
	logger zerolog.Logger
}

func NewOutputHandler() *OutputHandler {
	var (
		defaultDB *db.DB = nil
	)

	handler := &OutputHandler{
		db:     defaultDB,
		tables: make(map[string]bool),
		logger: log.NewLogger("output"),
	}

	return handler
}

func (o *OutputHandler) WithDB(db *db.DB) *OutputHandler {
	o.logger.Trace().Str("name", db.Settings.Name).Msg("running with db output")
	o.db = db
	return o
}

func (o *OutputHandler) WithStdOut() *OutputHandler {
	o.logger.Trace().Msg("running with stdout output")
	o.stdout = true
	return o
}

func (o *OutputHandler) WithCsv(csv *CsvHandler) *OutputHandler {
	o.logger.Trace().Msg("running with csv output")
	o.csv = csv
	return o
}

func (o OutputHandler) LogMap(m map[string]cty.Value) {
	fmt.Println()
	for k, v := range convertCtyMap(m) {
		o.logger.Info().Msg(fmt.Sprintf("%s: %s", k, v))
	}
}

func convertCtyMap(m map[string]cty.Value) map[string]string {
	new := make(map[string]string)
	for k, v := range m {
		switch v.Type() {
		case cty.Number:
			new[k] = v.AsBigFloat().String()

		case cty.String:
			new[k] = v.AsString()
		}
	}

	return new
}

// HandleResult takes a map of the final results (from the `save` block), and writes
// it to the preferred output options. If DB output is selected, it will create
// the table if it doesn't exist yet. If CSV is selected, it will create the file.
func (o OutputHandler) HandleResult(name string, res map[string]cty.Value) error {
	if o.stdout {
		o.LogMap(res)
	}

	strRes := convertCtyMap(res)

	if o.db != nil {
		if ok := o.tables[name]; !ok {
			err := o.db.CreateTable(context.Background(), name, res)
			if err != nil {
				return err
			}

			o.tables[name] = true
		}

		if err := o.db.InsertResult(name, strRes); err != nil {
			return err
		}
	}

	if o.csv != nil {
		csv, ok := o.csv.files[name]
		if !ok {
			err := o.csv.AddCsv(name, res)
			if err != nil {
				return err
			}

			csv = o.csv.files[name]
		}

		err := csv.Write(o.csv.generateCsvEntry(name, strRes))
		if err != nil {
			return err
		}

		csv.Flush()
	}

	return nil
}

type CsvHandler struct {
	// headers maps queries to header names, for matching
	headers map[string][]string
	// files maps queries to csv writers
	files map[string]*csv.Writer
}

func NewCsvHandler() *CsvHandler {
	return &CsvHandler{
		headers: make(map[string][]string),
		files:   make(map[string]*csv.Writer),
	}
}

func (c *CsvHandler) AddCsv(name string, cols map[string]cty.Value) error {
	f, err := os.Create(name + ".csv")
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)

	header := generate.GenerateCsvHeader(cols)
	w.Write(header)
	w.Flush()

	c.files[name] = w
	c.headers[name] = header

	return nil
}

func (c CsvHandler) generateCsvEntry(name string, res map[string]string) []string {
	header := c.headers[name]
	entries := make([]string, len(header))

	// This loop makes sure the entries (which are not of a set order)
	// are written in the correct order determined by the header.
	for k, v := range res {
		for i, h := range header {
			if k == h {
				entries[i] = fmt.Sprint(v)
			}
		}
	}

	return entries
}
