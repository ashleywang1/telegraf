package blacklist

import (
	"encoding/xml"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/plugins/processors"
)

var sampleConfig = `
  [[processors.blacklist]]
    config = "/usr/local/akamai/goblin_telegraf/conf/goblin.restriction.conf"
`

// Restriction is the top xml struct for the restriction file
type Restriction struct {
	Config       string
	lastReadTime time.Time

	Group     []Group `xml:"group"`
	Blacklist map[string]struct{}
	Whitelist map[string]struct{}
}

// Group contains the restrictions imposed by an interested party
type Group struct {
	Owner       string     `xml:"owner,attr"`
	Criteria    []Criteria `xml:"criteria"`
	Whitelisted struct {
		Tables []string `xml:"tablename,attr"`
	} `xml:"whitelist>table"`
	Blacklisted struct {
		Tables []string `xml:"tablename,attr"`
	} `xml:"blacklist>table"`
}

// Criteria determines whether this grouping applies or not
type Criteria struct {
	Network string `xml:"network,attr"`
}

func (r *Restriction) SampleConfig() string {
	return sampleConfig
}

func (r *Restriction) Description() string {
	return "Drop metrics that are blacklisted."
}

func (r *Restriction) Apply(in ...telegraf.Metric) []telegraf.Metric {
	info, err := os.Stat(r.Config)
	if err == nil {
		if r.lastReadTime != info.ModTime() {
			r.readConfig()
			r.lastReadTime = info.ModTime()
		}
	}
	res := make([]telegraf.Metric, 0, 0)
	for _, metric := range in {
		if _, ok := r.Blacklist[metric.Name()]; ok {
			continue
		}
		if _, ok := r.Whitelist[metric.Name()]; !ok {
			continue
		}
		res = append(res, metric)
	}
	return res
}

func (r *Restriction) readConfig() {

	blacklist := make(map[string]struct{})
	whitelist := make(map[string]struct{})
	rfile, err := ioutil.ReadFile(r.Config)
	if err != nil {
		// We have an empty blacklist and whitelist
		r.Blacklist = blacklist
		r.Whitelist = whitelist
		return
	}
	v := Restriction{}
	if err := xml.Unmarshal(rfile, &v); err != nil {
		// Invalid restriction file results in no changes made.
		return
	}
	x := struct{}{}
	for _, group := range v.Group {
		if !r.applicable(group.Criteria) {
			continue
		}
		for _, t := range group.Whitelisted.Tables {
			tname := strings.ToLower(strings.TrimSpace(t))
			whitelist[tname] = x
			log.Printf("[processors.blacklist] whitelisting %s for %s\n", tname, group.Owner)
		}
		for _, t := range group.Blacklisted.Tables {
			tname := strings.ToLower(strings.TrimSpace(t))
			blacklist[tname] = x
			log.Printf("[processors.blacklist] blacklisting %s for %s\n", tname, group.Owner)
		}
	}
	r.Blacklist = blacklist
	r.Whitelist = whitelist
}

// TODO(awang) be able to restrict by network
func (r *Restriction) applicable(criteria []Criteria) bool {
	return true
}

func init() {
	processors.Add("blacklist", func() telegraf.Processor {
		return &Restriction{}
	})
}
