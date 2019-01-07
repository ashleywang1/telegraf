package tableprov

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/influxdata/telegraf"
	ps "github.com/mitchellh/go-ps"
)

const backupDirectory = "/usr/local/akamai/goblin_telegraf/tableprov/"
const bakExt = ".valid"
const tmpExt = ".invalid"

// scanTableprovFile turns all of the data in the file into a metric.
func (tp *Tableprov) scanTableprovFile(file string, acc telegraf.Accumulator, wg *sync.WaitGroup) {
	defer wg.Done()
	tbl := tp.Tables[file]

	// Check the process to see if it's running
	if !tp.checkPIDFile(tbl.pidFile) {
		return
	}

	// Check the file to see if we should use a backup file
	usingbackup, changed, err := tp.checkForChanges(file)
	tbl.usingbackup = usingbackup
	if err != nil {
		// We couldn't find any file to open and scan, not even a backup
		log.Printf("[inputs.tableprov]: could not find tableprov table at %s", file)
		tbl.errors++
		tbl.valid = false
		return
	}

	// Optimization - If a table is unchanged, and we have previously
	// found it invalid, we won't re-scan it since we wouldn't have sent it anyway.
	if !changed && !tbl.valid {
		log.Printf("[inputs.tableprov]: skip same invalid table: %s\n", tbl.name)
		return
	}

	// Assign the file we use to the variable tbpvFile
	var tbpvFile string
	if changed {
		// update the timestamp
		fileInfo, err := os.Stat(file)
		if err == nil {
			tbl.timestamp = fileInfo.ModTime()
		}
		// use a temporary file
		err = copyToTmp(file)
		if err != nil {
			// Can't create tmp file, probably due to a permissions error
			// Just use the regular file.
			acc.AddError(err)
			tbpvFile = file
		} else {
			tbpvFile = tmp(file)
		}
	} else if usingbackup {
		// The file hasn't been changed, use the backup file that
		// is guaranteed to be valid.
		tbpvFile = bak(file)
	}

	// Open and read tbpvFile
	f, err := os.Open(tbpvFile)
	if err != nil {
		acc.AddError(err)
		tbl.errors++
		tbl.valid = false
		return
	}
	defer f.Close()
	var fileContents bytes.Buffer
	var chunkBoundaries = []int{}
	fileContents, err = tp.read(f, tbl, &chunkBoundaries)
	if err != nil {
		acc.AddError(err)
		tbl.errors++
		tbl.valid = false
		return
	}

	// Only validate changed files
	if changed {
		if err = tp.validate(fileContents, file, tbl); err != nil {
			log.Printf("[inputs.tableprov]: checked tableprov table: %s - INVALID %s \n",
				tbl.name, err.Error())
			tbl.errors++
			tbl.valid = false
			return
		}
		log.Printf("[inputs.tableprov]: checked tableprov table: %s - OK\n", tbl.name)
		tbl.valid = true

		// Move validated file to backup
		os.Rename(tmp(file), bak(file))
	}

	// Send the metric
	fields := make(map[string]interface{})
	tags := make(map[string]string)
	timestamp := time.Now()
	start := 0
	for chunkNumber, end := range chunkBoundaries {
		fields["tableprov"] = fileContents.String()[start:end]
		tags["chunkNumber"] = strconv.Itoa(chunkNumber)
		tags["isLast"] = strconv.FormatBool((end == fileContents.Len()))
		acc.AddFields(tbl.name, fields, tags, timestamp)
		start = end
	}
}

// checkPIDFile will check a PID file for the process id associated with the table.
// If the process is not running, we return an error.
func (tp *Tableprov) checkPIDFile(PIDfile string) bool {
	if PIDfile == "" {
		return true
	}
	f, err := os.Open(PIDfile)
	if err == nil {
		scanner := bufio.NewScanner(f)
		scanner.Scan()
		pidTxt := scanner.Text()
		f.Close()
		if err := scanner.Err(); err != nil {
			log.Printf("[inputs.tableprov]: %v", err)
			return false
		}
		pidData := strings.Split(pidTxt, ",")
		if len(pidData) == 1 {
			pid, err := strconv.Atoi(pidData[0])
			if err != nil {
				log.Printf("[inputs.tableprov]: %v", err)
				return false
			}
			p, err := ps.FindProcess(pid)
			if p == nil || err != nil {
				log.Printf("[inputs.tableprov]: %s - No process found with PID %d", PIDfile, pid)
				return false
			}
		} else if len(pidData) == 2 {
			pid, err := strconv.Atoi(pidData[0])
			if err != nil {
				log.Printf("[inputs.tableprov]: %v", err)
				return false
			}
			p, err := ps.FindProcess(pid)
			if p == nil || err != nil || p.Executable() != pidData[1] {
				log.Printf("[inputs.tableprov]: No process found with PID %d and name %s", pid, pidData[1])
				return false
			}
		}
	}
	return true
}

// checkForChanges will decide if we should use a backup file
// and detect if the table file has changed.
// Returns: usingbackup,
//          changed (whether the file has changed, default is true),
//          error
func (tp *Tableprov) checkForChanges(file string) (usesBackup, bool, error) {
	f, err := os.Stat(file)
	_, bErr := os.Stat(bak(file))
	tf, tErr := os.Stat(tmp(file))

	// Does the file even exist?
	if err != nil { // No
		// Check for a backup file
		if bErr != nil {
			return false, false, err
		}
		return true, false, nil
	}

	// Does a tmpfile exist?
	if tErr == nil { // Yes, we probably crashed while reading this file.
		// Has a new file arrived?
		if f.ModTime().After(tf.ModTime()) {
			// Yes
			return false, true, nil
		}
		// No, check for a backup file
		if bErr != nil {
			return false, false, bErr
		}
		return true, false, nil
	}

	// Does the backup file exist?
	if bErr != nil { // No
		return false, true, nil
	}

	// Check if the table file has been modified
	if f.ModTime() == tp.Tables[file].timestamp {
		// No, use the backup file
		return true, false, nil
	}
	return false, true, nil
}

// read will chunk the file into strings ending on a row boundary, less than MaxMetricBytes
func (tp *Tableprov) read(f *os.File, tbl *TblInfo, chunkBoundaries *[]int) (bytes.Buffer, error) {
	var buf bytes.Buffer
	scanner := bufio.NewScanner(f)
	ln := 0
	bytesInPreviousChunks := 0
	for scanner.Scan() {
		line := scanner.Text()
		var newLine string
		if line == "" && ln < 5 {
			// The encoding/csv library silently removes empty lines, so in order to
			// correctly count the number of metadata lines, we replace these empty lines with a comma.
			// Future checks will determine if we are missing important metadata or not.
			newLine = ",\n"
		} else {
			newLine = line + "\n"
		}
		// Record where the chunk boundaries are, so we can send chunked metrics
		if buf.Len()-bytesInPreviousChunks+len(newLine) > tp.MaxMetricBytes {
			*chunkBoundaries = append(*chunkBoundaries, buf.Len())
			bytesInPreviousChunks = buf.Len()
		}
		buf.WriteString(newLine)
		ln++
	}
	if err := scanner.Err(); err != nil {
		return buf, err
	}
	*chunkBoundaries = append(*chunkBoundaries, buf.Len())

	return buf, nil
}

// copyToTmp will copy the contents of the input file into a new temporary file
func copyToTmp(file string) error {
	from, err := os.Open(file)
	if err != nil {
		return err
	}
	defer from.Close()

	to, err := os.OpenFile(tmp(file), os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer to.Close()

	_, err = io.Copy(to, from)
	if err != nil {
		return err
	}
	return nil
}

// basename removes directory components and a trailing .suffix.
// e.g., a => a, a.go => a, a/b/c.go => c, a/b.c.go => b.c
func basename(s string) string {
	slash := strings.LastIndex(s, "/") // -1 if "/" not found
	s = s[slash+1:]
	if dot := strings.LastIndex(s, "."); dot >= 0 {
		s = s[:dot]
	}
	return s
}

func tmp(s string) string {
	return backupDirectory + basename(s) + tmpExt
}

func bak(s string) string {
	return backupDirectory + basename(s) + bakExt
}
