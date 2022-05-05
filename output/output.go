package output

import (
	"encoding/csv"
	"fmt"
	"os"

	"github.com/XMonetae-DeFi/apollo/chainservice"
	"github.com/XMonetae-DeFi/apollo/db"
	"github.com/XMonetae-DeFi/apollo/generate"
)

type OutputOption func(*OutputHandler)

type OutputHandler struct {
	stdout bool
	csv    *CsvHandler
	db     *db.DB
}

func NewOutputHandler(opts ...OutputOption) *OutputHandler {
	var (
		defaultDB *db.DB = nil
	)

	handler := &OutputHandler{
		db: defaultDB,
		// out: os.Stdout,
	}

	for _, opt := range opts {
		opt(handler)
	}

	return handler
}

func (o *OutputHandler) WithDB(db *db.DB) *OutputHandler {
	o.db = db
	return o
}

func (o *OutputHandler) WithStdOut() *OutputHandler {
	o.stdout = true
	return o
}

func (o *OutputHandler) WithCsv(csv *CsvHandler) *OutputHandler {
	o.csv = csv
	return o
}

func (o OutputHandler) HandleResult(res chainservice.CallResult) error {
	if o.stdout {
		fmt.Println(res)
	}

	if o.db != nil {
		if err := o.db.InsertResult(res); err != nil {
			return err
		}
	}

	if o.csv != nil {
		csv := o.csv.files[res.ContractName]
		err := csv.Write(o.csv.generateCsvEntry(res))
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

func (c *CsvHandler) AddCsv(cs generate.ContractSchemaV2) error {
	f, err := os.Create(cs.Name() + ".csv")
	if err != nil {
		return err
	}

	w := csv.NewWriter(f)

	header := generate.GenerateCsvHeader(cs)
	w.Write(header)
	w.Flush()

	c.files[cs.Name()] = w
	c.headers[cs.Name()] = header

	return nil
}

func (c CsvHandler) generateCsvEntry(res chainservice.CallResult) []string {
	header := c.headers[res.ContractName]
	// Remove standard columns
	header = header[4:]
	entries := make([]string, len(header))

	// Remove the standard headers

	for k, v := range res.Inputs {
		for i, h := range header {
			if k == h {
				entries[i] = fmt.Sprint(v)
			}
		}
	}

	for k, v := range res.Outputs {
		for i, h := range header {
			if k == h {
				entries[i] = fmt.Sprint(v)
			}
		}
	}

	row := []string{
		fmt.Sprint(res.Timestamp),
		fmt.Sprint(res.BlockNumber),
		string(res.Chain),
		res.ContractAddress.String(),
	}

	row = append(row, entries...)

	return row
}
