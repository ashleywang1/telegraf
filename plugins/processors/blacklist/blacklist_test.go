package blacklist

import (
	"testing"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/metric"
	"github.com/stretchr/testify/assert"
)

// Test config reader
func TestReadConfig(t *testing.T) {
	r := Restriction{
		Config: "test/test1.xml",
	}
	r.readConfig()
	assert.Equal(t, 2, len(r.Whitelist))
	assert.Equal(t, 2, len(r.Blacklist))
}

func newMetric(name string) telegraf.Metric {
	m1, _ := metric.New(name,
		map[string]string{},
		map[string]interface{}{},
		time.Now(),
	)
	return m1
}

// TestApplyWhitelist
func TestApplyWhitelist(t *testing.T) {
	r := Restriction{}
	r.Whitelist = make(map[string]struct{})

	// If there is no whitelist, drop all metrics
	processed := r.Apply(newMetric("table0"))
	assert.Equal(t, 0, len(processed))

	// If there is a whitelist, only allow the whitelisted metrics through
	r.Whitelist["table1"] = struct{}{}
	r.Whitelist["table2"] = struct{}{}
	m1 := newMetric("table1")
	processed = r.Apply(m1)
	assert.Equal(t, 1, len(processed))
	assert.Equal(t, m1, processed[0])
	m2 := newMetric("table2")
	processed = r.Apply(m2)
	assert.Equal(t, 1, len(processed))
	assert.Equal(t, m2, processed[0])
	processed = r.Apply(newMetric("table3"))
	assert.Equal(t, 0, len(processed))
}
