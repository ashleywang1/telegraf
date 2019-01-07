package tableprov

import (
	"bytes"
	"io/ioutil"
	"testing"
)

// TestValidateTableprovFile ensures that we only accept files that tableprov would accept,
// and reject files that tableprov would reject
func TestValidateTableprovFile(t *testing.T) {
	tp := &Tableprov{}
	var ValidateTableprovFile = tp.validate
	var filetests = []struct {
		filepath    string
		expectError bool
	}{
		{"test/correct.csv", false},
		{"test/empty.csv", true},
		{"test/missingcolumns.csv", true},
		{"test/nocolumnnames.csv", true},
		{"test/nocolumntypes.csv", true},
		{"test/nocolumndesc.csv", false},
		{"test/badcolumnnames.csv", true},
		{"test/wrongdatatype.csv", true},
	}
	for _, ft := range filetests {
		var fileContents bytes.Buffer
		fileBytes, err := ioutil.ReadFile(ft.filepath)
		if err != nil {
			t.Errorf("Couldn't read file %s", ft.filepath)
		}
		fileContents.Write(fileBytes)
		err = ValidateTableprovFile(fileContents, ft.filepath, &TblInfo{csvfilefmt: 1})
		if ft.expectError {
			if err == nil {
				t.Errorf("ValidateTableprovFile(%q) => nil, wanted error", ft.filepath)
			}
		} else {
			if err != nil {
				t.Errorf("ValidateTableprovFile(%q) => %q, wanted no errors", ft.filepath, err)
			}
		}
	}
}

// TestValidateTableprov2Files ensures that we are correctly reading in Tableprov2 CSV files
func TestValidateTableprov2Files(t *testing.T) {
	tp := &Tableprov{}
	var ValidateTableprovFile = tp.validate
	var filetests = []struct {
		filepath    string
		expectError bool
	}{
		{"test/tableprov2.csv", false},
		{"test/nocolumndesc.csv", true},
	}
	for _, ft := range filetests {
		var fileContents bytes.Buffer
		fileBytes, err := ioutil.ReadFile(ft.filepath)
		if err != nil {
			t.Errorf("Couldn't read file %s", ft.filepath)
		}
		fileContents.Write(fileBytes)
		err = ValidateTableprovFile(fileContents, ft.filepath, &TblInfo{csvfilefmt: 2})
		if ft.expectError {
			if err == nil {
				t.Errorf("ValidateTableprovFile(%q) => nil, wanted error", ft.filepath)
			}
		} else {
			if err != nil {
				t.Errorf("ValidateTableprovFile(%q) => %q, wanted no errors", ft.filepath, err)
			}
		}
	}
}

// TestCheckNames ensures we only allow tableprov table and column names
func TestCheckNames(t *testing.T) {
	var CheckNames = checkNames
	var nametests = []struct {
		tableName   string
		columnNames []string
		expectError bool
	}{
		{"alphanum3r1c", nil, false},
		{"contains_underscores", nil, false},
		{"contains-!@#$%^&*()-_=+`~", nil, true},
		{"contains space", nil, true},
		{"¢ontains_n¤n_as¢ii", nil, true},
		{"0starts_with_digit", nil, true},
		{"1starts_with_digit", nil, true},
		{"tables", nil, true}, // reserved word
		{"x", []string{"alphanum3r1c", "column", "names"}, false},
		{"x", []string{"containing_", "underscores_"}, false},
		{"x", []string{"containing!", "speci@l", "ch@r@cter5"}, true},
		{"x", []string{"containing!", "n¤n", "as¢ii"}, true},
		{"x", []string{"1starting", "2with", "3digits"}, true},
		{"x", []string{"some", "reserved", "words"}, false},
	}

	for _, nt := range nametests {
		err := CheckNames(nt.tableName, nt.columnNames)
		if nt.expectError {
			if err == nil {
				t.Errorf("CheckNames(%q, %q) => nil, wanted error", nt.tableName, nt.columnNames)
			}
		} else {
			if err != nil {
				t.Errorf("CheckNames(%q, %q) => %q, wanted no errors", nt.tableName, nt.columnNames, err)
			}
		}
	}
}
