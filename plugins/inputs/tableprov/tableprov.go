// Package tableprov implements an input Telegraf plugin that
// watches a list of Tableprov csv files and converts the csv
// file into a metric, with appropriate tags and metadata.
//
// Directory path: telegraf/plugins/inputs/tableprov
// Files: README.md             description of package configuration.
//        tableprov.go          contains the part of the package that is public
//                              and exposed to the rest of Telegraf.
//        tableprov_config.go   contains the code for reading in the tableprov
//                              config file.
//        file_reader.go        contains the code for reading in and chunking
//                              large tableprov CSV files.
//        validator.go          contains the code for validating tableprov CSV files
//        reservedwords.go      contains the list of query2 words not allowed in
//                              table and column names
//
package tableprov

import (
	"errors"
	"log"
	"os"
	"sync"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/inputs"
	"github.com/influxdata/telegraf/plugins/parsers"
	"github.com/influxdata/telegraf/utils"
)

// Tableprov is the parent struct for both Tableprov and Tableprov2
type Tableprov struct {
	Config         string
	MaxMetricBytes int
	parser         parsers.Parser

	HostIP  string
	Tables  map[string]*TblInfo
	Indices map[string]*TblInfo
}

const defaultTableChunkSize = 900000

// SampleConfig describes the expected configuration parameters
func (tp *Tableprov) SampleConfig() string {
	return `
	## Tableprov Config filepath
	config = "/usr/local/akamai/etc/staticinfo/tableprov.conf"
	max_metric_bytes = 900000
	`
}

// Description of the plugin
func (tp *Tableprov) Description() string {
	return "Convert Tableprov csv files into metrics"
}

// Start sets up the list of Tableprov csv files to monitor
func (tp *Tableprov) Start(acc telegraf.Accumulator) error {
	// Set default MaxMetricBytes
	if tp.MaxMetricBytes == 0 {
		tp.MaxMetricBytes = defaultTableChunkSize
	}
	// Get IP address
	tp.HostIP = utils.GetIP()
	if tp.HostIP == "" {
		return errors.New("[inputs.tableprov]: couldn't find machine's IP address")
	}
	// Keep track of each Tableprov csv file listed
	tp.Tables = make(map[string]*TblInfo)
	tp.Indices = make(map[string]*TblInfo)
	// Create a backup directory
	err := os.MkdirAll(backupDirectory, 0755)
	if err != nil {
		return errors.New("[inputs.tableprov]: unable to create backup directory")
	}

	return nil
}

// Gather runs once every interval, but the plugin will only pick up files
// that have been modified.
func (tp *Tableprov) Gather(acc telegraf.Accumulator) error {
	tp.updateConfig(acc)

	var wg sync.WaitGroup
	wg.Add(len(tp.Tables))
	log.Printf("[inputs.tableprov]: Starting Cycle\n")
	for file := range tp.Tables {
		go tp.scanTableprovFile(file, acc, &wg)
	}
	go func() {
		wg.Wait()
		tp.createTableprovTablesMetric(acc)
	}()
	return nil
}

// Stop is a noop, but required for the ServiceInput interface
func (tp *Tableprov) Stop() {
}

// init initializes the package.
func init() {
	inputs.Add("tableprov", func() telegraf.Input {
		return &Tableprov{}
	})
}
