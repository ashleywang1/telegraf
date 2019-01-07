# Tableprov Input Plugin

The Tableprov plugin monitors a list of Tableprov csv files and turns them into metrics.

### Configuration:

```toml
# Gather Tableprov Metrics
[[inputs.tableprov]]
	## Tableprov Config filepath
	config = "/usr/local/akamai/etc/staticinfo/tableprov.conf"
	max_metric_bytes = 900000
```

### Tableprov CSV files:

These files are self-describing files containing both the schema and the data for a table to go into query. The name of the table is the name of the file itself, less the .csv extension.

The file adheres to the well-defined CSV format, which should make it easy to edit in Excel or gnumeric if so desired. Basically, fields are comma separated, optionally enclosed in quotes. All characters must be 7-bit printable ASCII, ' ' 0x20 - '~' 0x7E. If you want a comma in a field, the field must be quoted. For a quote in the field, quote the whole field and double the quote (whew!). Spaces in fields do not need a quoted field.

The first five lines are metadata. The lines are interpreted as:

Line 1: Version: Line will be used to monitor that tableprov is providing the correct versions of tables. No validation done or format enforced. 

Line 2: Table Description: Description of table. No validation for format enforced. 

Line 3: Column Names: Columns names. The aforementioned rules regarding legal table names also apply to column names. 

Line 4: Column Types: (see below) 

Line 5: Column Descriptions: Description of each column. No validation or format enforced. 

Lines 6-end: Data: one row per line, as many rows as you want

If the number of fields in the names, types, or data lines don't match, the file is invalid. 

For more information, see: https://collaborate.akamai.com/confluence/display/DDC/Query2+Tableprov+Description+and+Usage

### Tableprov 2 CSV files:

These files are the same as Tableprov 2 files except for Line 4, which describes the data types. In the original Tableprov CSV file format, the valid data types are:
* DATA: string, str, int, integer, region, reg, time, ip, ipv6, ll, null

For the Tableprov2 CSV file format, we will only accept one correct spelling for a data type, not multiple. We will also add float as an allowed data type, depreciate region, reg, and ll. The valid data types are now:
* DATA: str, int, float, time, ip, ipv6

The fourth line should be the column datatypes, in the form:
* [(AGG)]TYPE[NULL/MERGE]

The possible values for each type are:
		* AGG: sum, min, max, none
		* TYPE: int, ip, ipv6, time, string
		* NULL: true, false
		* MERGE: null, none

The defaults types are:
		* AGG: none
		* TYPE: int
		* NULL: false
		* MERGE: noneg

The possible combinations for NULL/MERGE are:
		* ? for NULL=true and MERGE=none
		* ! for NULL=true and MERGE=null
		*   for NULL=false and MERGE=none

If the datatype has the default case, then it can be represented in [(AGG)TYPE[NULL/MERGE]] as an empty string.
