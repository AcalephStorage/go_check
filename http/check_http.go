package main

import (
	"github.com/newrelic/go_nagios"
	"gopkg.in/alecthomas/kingpin.v1"
)

var (
	userAgent     = kingpin.Flag("user-agent", "Specify a USER-AGENT").Default("Go-HTTP-Check").String()
	url           = kingpin.Flag("url", "A URL to connect to").String()
	host          = kingpin.Flag("host", "A HOSTNAME to connect to").String()
	port          = kingpin.Flag("port", "Select another port").Int()
	requestUri    = kingpin.Flag("request-uri", "Specify a uri path").String()
	header        = kingpin.Flag("name", "Check for a HEADER").String()
	ssl           = kingpin.Flag("ssl", "Enabling SSL connections").Default("false").Bool()
	insecure      = kingpin.Flag("insecure", "Enabling insecure connections")
	username      = kingpin.Flag("username", "A username to connect as").String()
	password      = kingpin.Flag("password", "A password to use for the username").String()
	cert          = kingpin.Flag("cert", "Cert to use").String()
	cacert        = kingpin.Flag("cacert", "A CA Cert to use").String()
	expiry        = kingpin.Flag("expiry", "Warn EXPIRE days before cert expires").Int()
	query         = kingpin.Flag("query", "Query for a specific pattern").String()
	timeout       = kingpin.Flag("timeout", "Set the timeout").Int()
	redirectOk    = kingpin.Flag("redirect-ok", "Check if a redirect is ok").Bool()
	redirectTo    = kingpin.Flag("redirect-to", "Redirect to another page").Bool()
	responseBytes = kingpin.Flag("response-bytes", "Print BYTES of the output").Int()
	requireBytes  = kingpin.Flag("require-bytes", "Check the response contains exactly BYTES bytes").Bytes()
	responseCode  = kingpin.Flag("response-code", "Check for a specific response code").Int()
)

func main() {
	kingpin.Version("1.0.0")
	kingpin.Parse()
	checkHttp()
}

func checkHttp() {

}
