package tableprov

import (
	"bytes"
	"strconv"
	"time"

	"github.com/influxdata/telegraf"
)

type usesBackup bool

func (b usesBackup) String() string {
	if b {
		return "1"
	}
	return "0"
}

// TblInfo contains all the metadata about one tableprov table
type TblInfo struct {
	name        string
	errors      int
	usingbackup usesBackup
	rows        int
	cols        int
	version     string
	status      string
	timestamp   time.Time
	pidFile     string
	csvfilefmt  int
	valid       bool
}

// createTableprovTablesMetric makes a summary of all the tableprov tables processed
func (tp *Tableprov) createTableprovTablesMetric(acc telegraf.Accumulator) {
	var b bytes.Buffer

	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	b.WriteString(timestamp + "\n" +
		"An overview of all tables provided by tableprov on this machine\n" +
		"ip,table,errors,usingbackup,rows,cols,version,file,status,lastreadtime\n" +
		"ip,string,int,int,int,int,string,string,string,time\n" +
		"ip, tablename, total errors since startup, using backup, " +
		"rows, cols, version, file name, process status, last read time (GMT)\n")

	for file, tbl := range tp.Tables {
		b.WriteString(tp.HostIP + "," +
			tbl.name + "," +
			strconv.Itoa(tbl.errors) + "," +
			tbl.usingbackup.String() + "," +
			strconv.Itoa(tbl.rows) + "," +
			strconv.Itoa(tbl.cols) + "," +
			tbl.version + "," +
			file + "," +
			tbl.status + "," +
			strconv.FormatInt(tbl.timestamp.Unix(), 10) +
			"\n")
	}

	for file, idx := range tp.Indices {
		b.WriteString(tp.HostIP + "," +
			idx.name + "," +
			strconv.Itoa(idx.errors) + "," +
			idx.usingbackup.String() + "," +
			strconv.Itoa(idx.rows) + "," +
			strconv.Itoa(idx.cols) + "," +
			idx.version + "," +
			file + "," +
			idx.status + "," +
			strconv.FormatInt(idx.timestamp.Unix(), 10) +
			"\n")
	}

	fields := make(map[string]interface{})
	fields["tableprov"] = b.String()
	acc.AddFields("tableprov_tables", fields, nil, time.Now())
}
