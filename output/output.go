package output

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"

	"github.com/XMonetae-DeFi/apollo/db"
	"github.com/XMonetae-DeFi/apollo/generate"
	"github.com/XMonetae-DeFi/apollo/log"
	"github.com/rs/zerolog"
	"github.com/zclconf/go-cty/cty"
)

type OutputOption func(*OutputHandler)

type OutputHandler struct {
	stdout bool
	csv    *CsvHandler
	db     *db.DB
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
	// map of contract to headers, so that we match
	headers map[string][]string
	files   map[string]*csv.Writer
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
	// Remove standard columns
	entries := make([]string, len(header))

	// Remove the standard headers

	for k, v := range res {
		for i, h := range header {
			if k == h {
				entries[i] = fmt.Sprint(v)
			}
		}
	}

	return entries
}
