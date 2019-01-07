package tableprov

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"regexp"
	"strings"
)

var tableprovDataTypes = map[string]int{
	"str":     0,
	"string":  0,
	"int":     0,
	"integer": 0,
	"reg":     0,
	"region":  0,
	"time":    0,
	"ip":      0,
	"ipaddr":  0,
	"ipv6":    0,
	"ll":      0,
	"null":    0,
}

var tableprov2DataTypes = map[string]int{
	"str":   0,
	"int":   0,
	"float": 0,
	"time":  0,
	"ip":    0,
	"ipv6":  0,
	"null":  0,
}

// validate goes through the entire csv file to check that it is valid
func (tp *Tableprov) validate(fileContents bytes.Buffer, filepath string, tbl *TblInfo) error {

	csvReader := csv.NewReader(strings.NewReader(fileContents.String()))
	csvReader.FieldsPerRecord = -1
	csvReader.TrimLeadingSpace = true
	csvReader.ReuseRecord = true
	records, err := csvReader.ReadAll()
	if err != nil {
		return err
	}

	if len(records) < 5 {
		err := fmt.Errorf("file[%s] - missing metadata", filepath)
		return err
	}
	tbl.version = strings.Join(records[0], ",")
	// If the number of fields in the names, types, or data lines don't match,
	// the file is invalid.
	var columns = len(records[2])

	tbl.cols = columns
	if columns == 0 {
		return fmt.Errorf("file[%s] - cannot have 0 columns", filepath)
	}
	if len(records[3]) != columns {
		return fmt.Errorf("file[%s] - column types[%d] != columns[%d]",
			filepath, len(records[3]), columns)
	}
	// Check that we have correct tableprov data types
	if tbl.csvfilefmt == 1 {
		for i := 0; i < len(records[3]); i++ {
			t := records[3][i]
			if _, ok := tableprovDataTypes[t]; !ok {
				return fmt.Errorf("file[%s] - invalid data type[%s] for Tableprov CSV", filepath, t)
			}
		}
	} else if tbl.csvfilefmt == 2 {
		for i := 0; i < len(records[3]); i++ {
			t := parseDataType(records[3][i])
			if _, ok := tableprov2DataTypes[t]; !ok {
				return fmt.Errorf("file[%s] - invalid data type[%s] for Tableprov CSV2", filepath, t)
			}
		}
	}
	var rows int
	for i := 5; i < len(records); i++ {
		if len(records[i]) != columns {
			return fmt.Errorf("file[%s] - data line[%d] fields[%d] != columns[%d]",
				filepath, i, len(records[i]), columns)
		}
		rows++
	}
	tbl.rows = rows
	err = checkNames(filepath, records[2])
	if err != nil {
		return err
	}

	return nil
}

func parseDataType(str string) string {
	start := strings.Index(str, ")")
	end := strings.Index(str, "?")
	if strings.Index(str, "!") > end {
		end = strings.Index(str, "!")
	}
	if end == -1 {
		end = len(str)
	}
	return str[start+1 : end]
}

// checkNames ensures that the table and column names cannot contain any
// non-alphanumeric characters other than underscores ("_").
// Additionally, names cannot start with a digit and should not start with
// an underscore. Finally, table names should not be any of query's reserved words.
func checkNames(filepath string, columnNames []string) error {
	filename := basename(filepath)
	re := regexp.MustCompile("^[a-zA-Z]+[a-zA-Z0-9_]*$")
	if !re.MatchString(filename) {
		return fmt.Errorf("file[%s] - table name not allowed", filepath)
	}
	if _, ok := reservedwords[filename]; ok {
		return fmt.Errorf("file[%s] - table name is a reserved word", filepath)
	}
	for _, columnName := range columnNames {
		if !re.MatchString(columnName) {
			return fmt.Errorf("file[%s] - ColumnName[%s] not allowed", filepath, columnName)
		}
		// This is technically okay, although it causes problems later on.
		// if _, ok := reservedwords[columnName]; ok {
		// 	return fmt.Errorf("file[%s] - ColumnName[%s] is a reserved word", filepath, columnName)
		// }
	}
	return nil
}
