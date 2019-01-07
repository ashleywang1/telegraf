package akamill

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/influxdata/telegraf"
	"github.com/influxdata/telegraf/internal"
	"github.com/influxdata/telegraf/internal/tls"
	"github.com/influxdata/telegraf/plugins/outputs"
	"github.com/influxdata/telegraf/plugins/serializers"
)

var sampleConfig = `
  ## URL is the address to send metrics to
  url = "http://127.0.0.1:8080/metric"

  ## Timeout for HTTP message
  # timeout = "5s"

  ## HTTP method, one of: "POST" or "PUT"
  # method = "POST"

  ## Akamill message limit, default of 1MB
  # max_bytes = 1000000

  ## HTTP Basic Auth credentials
  # username = "username"
  # password = "pa$$word"

  ## Optional TLS Config
  # tls_ca = "/etc/telegraf/ca.pem"
  # tls_cert = "/etc/telegraf/cert.pem"
  # tls_key = "/etc/telegraf/key.pem"
  ## Use TLS but skip chain & host verification
  # insecure_skip_verify = false

  ## Data format to output.
  ## Each data format has it's own unique set of configuration options, read
  ## more about them here:
  ## https://github.com/influxdata/telegraf/blob/master/docs/DATA_FORMATS_OUTPUT.md
  # data_format = "influx"
  
  ## Additional HTTP headers
  # [outputs.http.headers]
  #   # Should be set manually to "application/json" for json data_format
  #   Content-Type = "text/plain; charset=utf-8"
`

const (
	defaultClientTimeout = 5 * time.Second
	defaultContentType   = "text/plain; charset=utf-8"
	defaultMethod        = http.MethodPost
)

type HTTP struct {
	URL                string            `toml:"url"`
	Timeout            internal.Duration `toml:"timeout"`
	Method             string            `toml:"method"`
	MaxBytes           int               `toml:"max_bytes"`
	Username           string            `toml:"username"`
	Password           string            `toml:"password"`
	TLSCA              string            `toml:"tls_ca"`
	TLSCert            string            `toml:"tls_cert"`
	TLSKey             string            `toml:"tls_key"`
	InsecureSkipVerify bool              `toml:"insecure_skip_verify"`
	tls.ClientConfig
	Headers map[string]string `toml:"headers"`

	client     *http.Client
	serializer serializers.Serializer
}

func (h *HTTP) SetSerializer(serializer serializers.Serializer) {
	h.serializer = serializer
}

func (h *HTTP) Connect() error {
	h.ClientConfig = tls.ClientConfig{
		TLSCA:              h.TLSCA,
		TLSCert:            h.TLSCert,
		TLSKey:             h.TLSKey,
		InsecureSkipVerify: h.InsecureSkipVerify,
	}
	if h.MaxBytes <= 0 {
		h.MaxBytes = 1000000
	}
	if h.Method == "" {
		h.Method = http.MethodPost
	}
	h.Method = strings.ToUpper(h.Method)
	if h.Method != http.MethodPost && h.Method != http.MethodPut {
		return fmt.Errorf("invalid method [%s] %s", h.URL, h.Method)
	}

	if h.Timeout.Duration == 0 {
		h.Timeout.Duration = defaultClientTimeout
	}

	tlsCfg, err := h.ClientConfig.TLSConfig()
	if err != nil {
		return err
	}

	h.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
			Proxy:           http.ProxyFromEnvironment,
		},
		Timeout: h.Timeout.Duration,
	}

	return nil
}

func (h *HTTP) Close() error {
	return nil
}

func (h *HTTP) Description() string {
	return "A plugin that can transmit metrics over HTTP"
}

func (h *HTTP) SampleConfig() string {
	return sampleConfig
}

func (h *HTTP) Write(metrics []telegraf.Metric) error {
	start := time.Now()
	var bytesWritten int
	var numPostMessages int
	var reqBody bytes.Buffer

	for _, m := range metrics {
		body, err := h.serializer.Serialize(m)
		if reqBody.Len()+len(body) > h.MaxBytes {
			if err := h.write(reqBody.Bytes()); err != nil {
				log.Printf("[outputs.akamill]: %v", err)
			}
			numPostMessages++
			bytesWritten += reqBody.Len()
			reqBody.Reset()
		}
		_, err = reqBody.Write(body)
		if err != nil {
			log.Printf("[outputs.akamill]: %v", err)
		}
	}
	if reqBody.Len() > 0 {
		if reqBody.Len() > h.MaxBytes {
			log.Printf("[outputs.akamill]: WARNING! an input plugin is creating metrics larger than %d bytes", h.MaxBytes)
		}
		if err := h.write(reqBody.Bytes()); err != nil {
			log.Printf("[outputs.akamill]: %v", err)
		}
		numPostMessages++
		bytesWritten += reqBody.Len()
	}

	elapsed := time.Since(start)
	log.Printf("[outputs.akamill]: %d bytes written in %d post messages in %s \n", bytesWritten, numPostMessages, elapsed)

	return nil
}

func (h *HTTP) write(reqBody []byte) error {
	req, err := http.NewRequest(h.Method, h.URL, bytes.NewBuffer(reqBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", defaultContentType)
	for k, v := range h.Headers {
		req.Header.Set(k, v)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = ioutil.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("when writing to [%s] received status code: %d", h.URL, resp.StatusCode)
	}

	return nil
}

func init() {
	outputs.Add("akamill", func() telegraf.Output {
		return &HTTP{
			Timeout: internal.Duration{Duration: defaultClientTimeout},
			Method:  defaultMethod,
		}
	})
}
