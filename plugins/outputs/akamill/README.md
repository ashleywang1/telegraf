# Akamill Output Plugin

This plugin sends metrics in a HTTP message encoded using one of the output
data formats.  For data_formats that support batching, metrics are sent in batch format.

The only difference between this and the HTTP output plugin is that Akamill can't handle
HTTP Post packets that are too large. Here, we ensure that each HTTP Post message is below
that threshold, called max_bytes.

Additionally, we assume that the size of any metrics generated by any input plugins are 
less than max_bytes!

### Configuration:

```toml
# A plugin that can transmit metrics over HTTP
[[outputs.akamill]]
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
  # data_format = "tableprov_csv_delimiting"
  
  ## Additional HTTP headers
  # [outputs.http.headers]
  #   # Should be set manually to "application/json" for json data_format
  #   Content-Type = "text/plain; charset=utf-8"
```
