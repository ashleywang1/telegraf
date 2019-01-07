package tableprov

import (
	"bufio"
	"log"
	"os"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
)

// readConfig will read the Tableprov Config file and parse the index files
func (tp *Tableprov) readConfig(acc telegraf.Accumulator, tables map[string]*TblInfo, indices map[string]*TblInfo) error {
	// Open the tableprov config file
	file, err := os.Open(tp.Config)
	if err != nil {
		acc.AddError(err)
		return err
	}
	defer file.Close()

	// Scan config file for index files
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		l := scanner.Text()
		if l == "[watch]" {
			scanner.Scan()
			indexName := strings.TrimPrefix(scanner.Text(), "indexname = ")
			scanner.Scan()
			index := strings.TrimPrefix(scanner.Text(), "index = ")
			scanner.Scan()
			indexDir := strings.TrimSuffix(strings.TrimPrefix(scanner.Text(), "dir = "), "/")
			scanner.Scan()
			csvfilefmtstr := strings.TrimPrefix(scanner.Text(), "csvfilefmt = ")
			csvfilefmt := 1
			if csvfilefmtstr == "tableprov2" {
				csvfilefmt = 2
			}

			tp.populateTablesFrom(indexName, index, indexDir, csvfilefmt, tables, indices)
		}
	}
	if err := scanner.Err(); err != nil {
		acc.AddError(err)
	}

	return nil
}

// updateConfig will reread the configuration file and index files, and update
// the metadata by adding and removing the indexes or tables to watch.
func (tp *Tableprov) updateConfig(acc telegraf.Accumulator) {
	newTables := make(map[string]*TblInfo)
	newIndices := make(map[string]*TblInfo)
	tp.readConfig(acc, newTables, newIndices)

	// Update the indices
	for file, newIdx := range newIndices {
		if _, ok := tp.Indices[file]; !ok {
			// Register a new index
			tp.Indices[file] = newIdx
			log.Printf("[inputs.tableprov]: Registered a new index (%s)", newIdx.name)
		}
	}
	for file, oldIdx := range tp.Indices {
		if _, ok := newIndices[file]; !ok {
			// Remove a deleted index
			delete(tp.Indices, file)
			log.Printf("[inputs.tableprov]: Deleted index (%s)", oldIdx.name)
		}
	}

	// Update the tables
	for file, newTable := range newTables {
		if _, ok := tp.Tables[file]; !ok {
			// Register a new table
			tp.Tables[file] = newTable
			log.Printf("[inputs.tableprov]: Registered a new table: %s", newTable.name)
		}
	}
	for file, oldTable := range tp.Tables {
		if _, ok := newTables[file]; !ok {
			// Remove a deleted table
			delete(tp.Tables, file)
			log.Printf("[inputs.tableprov]: Deleted table: %s", oldTable.name)
		}
	}
}

// populateTablesFrom the index file
// Line format should be: TABLE [, [FILE]]
//                        OR
//                        TABLE,[FILE],CHECKFILE
// If FILE isn't provided, it defaults to TABLE.csv
func (tp *Tableprov) populateTablesFrom(indexName string, indexPath string, dir string,
	v int, tables map[string]*TblInfo, indices map[string]*TblInfo) {
	// If we have no data on filemod times, the default is epoch time
	minTime := time.Unix(0, 0)

	// track index files even if they don't exist
	indexFile, err := os.Open(indexPath)
	if err != nil {
		if idx, ok := indices[indexPath]; ok {
			idx.errors++
			idx.rows = -1
		} else {
			indices[indexPath] = &TblInfo{
				name: indexName, errors: 0, usingbackup: false, rows: -1, cols: 0,
				version: "", timestamp: minTime, csvfilefmt: 0, valid: true,
			}
		}
		log.Printf("[inputs.tableprov]: could not find index file at %s\n", indexPath)
		return
	}
	defer indexFile.Close()
	fileInfo, err := os.Stat(indexPath)
	if err != nil {
		log.Printf("[inputs.tableprov]: could not stat index file at %s\n", indexPath)
		return
	}

	// track all tableprov tables listed in index files
	scanner := bufio.NewScanner(indexFile)
	scanner.Scan()
	version := scanner.Text()
	tablesFound := 0
	for scanner.Scan() {
		l := scanner.Text()
		if l == "" {
			continue
		}
		tableInfo := strings.Split(l, ",")
		var tableName string
		var tableFile string
		var pidFile string
		if len(tableInfo) == 1 {
			tableName = tableInfo[0]
			tableFile = tableInfo[0] + ".csv"
		} else if len(tableInfo) == 2 {
			tableName = tableInfo[0]
			tableFile = tableInfo[1]
		} else if len(tableInfo) == 3 {
			tableName = tableInfo[0]
			tableFile = tableInfo[1]
			pidFile = tableInfo[2]
		} else {
			log.Printf("[inputs.tableprov]: Invalid index file %s\n", indexPath)
			return
		}
		tables[dir+"/"+basename(tableFile)+".csv"] = &TblInfo{
			name: tableName, errors: 0, usingbackup: false, rows: -1, cols: -1,
			version: version, timestamp: minTime, pidFile: pidFile, csvfilefmt: v, valid: true,
		}
		tablesFound++
	}

	// Keep index file information for tableprov_tables
	indices[indexPath] = &TblInfo{
		name: indexName, errors: 0, usingbackup: false, rows: tablesFound, cols: 0,
		version: "", timestamp: fileInfo.ModTime(), csvfilefmt: 0, valid: true,
	}
}
