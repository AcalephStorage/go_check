package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/AcalephStorage/go_check/Godeps/_workspace/src/github.com/newrelic/go_nagios"
	"github.com/AcalephStorage/go_check/Godeps/_workspace/src/gopkg.in/alecthomas/kingpin.v1"
)

var (
	userAgent     = kingpin.Flag("user-agent", "Specify a USER-AGENT").Default("Go-HTTP-Check").String()
	urlArg        = kingpin.Flag("url", "A URL to connect to").String()
	host          = kingpin.Flag("host", "A HOSTNAME to connect to").String()
	port          = kingpin.Flag("port", "Select another port").Int()
	requestUri    = kingpin.Flag("request-uri", "Specify a uri path").String()
	query         = kingpin.Flag("query", "request query (without ?)").String()
	header        = kingpin.Flag("name", "Check for a HEADER").String()
	ssl           = kingpin.Flag("ssl", "Enabling SSL connections").Default("false").Bool()
	insecure      = kingpin.Flag("insecure", "Enabling insecure connections").Bool()
	username      = kingpin.Flag("username", "A username to connect as").String()
	password      = kingpin.Flag("password", "A password to use for the username").String()
	certFile      = kingpin.Flag("cert-file", "Cert to use").String()
	keyFile       = kingpin.Flag("key-file", "Key to use").String()
	cacert        = kingpin.Flag("cacert", "A CA Cert to use").String()
	expiry        = kingpin.Flag("expiry", "Warn EXPIRE days before cert expires").Int()
	pattern       = kingpin.Flag("query", "Query for a specific pattern").String()
	timeout       = kingpin.Flag("timeout", "Set the timeout").Default("15").Int()
	redirectOk    = kingpin.Flag("redirect-ok", "Check if a redirect is ok").Bool()
	redirectTo    = kingpin.Flag("redirect-to", "Redirect to another page").String()
	responseBytes = kingpin.Flag("response-bytes", "Print BYTES of the output").Int()
	requireBytes  = kingpin.Flag("require-bytes", "Check the response contains exactly BYTES bytes").Int()
	responseCode  = kingpin.Flag("response-code", "Check for a specific response code").Int()
)

func main() {
	kingpin.Version("1.0.0")
	kingpin.Parse()
	checkHttp()
}

func checkHttp() {

	request := &http.Request{
		Method: "GET",
		URL:    createUrl(),
		Close:  true,
	}

	var config *tls.Config
	requestTimeout := (time.Duration(*timeout) * time.Second)
	if *ssl {

		certificates := make([]tls.Certificate, 1)
		if *certFile != "" {
			certificates[0], _ = tls.LoadX509KeyPair(*certFile, *keyFile)
		}

		var clientCAs *x509.CertPool
		if *cacert != "" {
			clientCAs := x509.NewCertPool()
			data, err := ioutil.ReadFile(*cacert)
			if err != nil {
				nagios.Unknown(err.Error())
			}
			clientCAs.AppendCertsFromPEM(data)
		}

		if *expiry > 0 {
			certExpiry := certificates[0].Leaf.NotAfter
			daysDifference := int(certExpiry.Sub(time.Now()).Hours() / 24)
			if daysDifference <= *expiry {
				nagios.Warning(fmt.Sprintf("Certificate will expire %v", certExpiry))
			}
		}

		config = &tls.Config{
			InsecureSkipVerify: *insecure,
			Certificates:       certificates,
			RootCAs:            clientCAs,
		}

	}

	if *username != "" && *password != "" {
		request.SetBasicAuth(*username, *password)
	}

	transport := &http.Transport{
		TLSClientConfig:       config,
		ResponseHeaderTimeout: requestTimeout,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   requestTimeout,
	}

	request.Header = http.Header{}
	if *userAgent != "" {
		request.Header.Set("User-Agent", *userAgent)
	}
	if *header != "" {
		headers := strings.Split(*header, ",")
		for _, h := range headers {
			kv := strings.Split(h, ":")
			request.Header.Set(kv[0], kv[1])
		}
	}

	response, err := client.Do(request)
	if err != nil {
		nagios.Critical(err)
	}
	defer response.Body.Close()

	code := response.StatusCode

	var body string
	if *responseBytes > 0 {
		b := make([]byte, *responseBytes)
		if n, err := request.Body.Read(b); n == 0 || err != nil {
			nagios.Critical(err)
		}
		body = "\n" + string(b)
	}

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		nagios.Unknown(err.Error())
	}
	fullBody := string(b)
	size := len(b)

	if *requireBytes > 0 {
		if size != *requireBytes {
			err := errors.New(fmt.Sprintf("Response was %d bytes instead of %d%s", size, *requireBytes, body))
			nagios.Critical(err)
		}
	}

	switch {
	case code/200 == 1:
		if *redirectTo != "" {
			err := errors.New(fmt.Sprintf("Expected redirect to %s but got %d%s", *redirectTo, code, body))
			nagios.Critical(err)
		} else if *pattern != "" {
			r, err := regexp.Compile(*pattern)
			if err != nil {
				nagios.Unknown(err.Error())
			}
			if r.MatchString(fullBody) {
				nagios.Ok(fmt.Sprintf("%d, found /%s/ in %d bytes%s", code, *pattern, size, body))
			} else {
				err := errors.New(fmt.Sprintf("%d, did not found /%s/ in %d bytes%s", code, *pattern, size, body))
				nagios.Critical(err)
			}
		} else {
			nagios.Ok(fmt.Sprintf("%d, %d bytes%s", code, size, body))
		}
	case code/300 == 1:
		if *redirectOk || *redirectTo != "" {
			if *redirectOk {
				nagios.Ok(fmt.Sprintf("%d, %d bytes%s", code, size, body))
			} else {
				err := errors.New(fmt.Sprintf("Expected redirect to %s instead redirected to %s", *redirectTo, response.Request.URL.String()))
				nagios.Critical(err)
			}
		} else {
			nagios.Warning(fmt.Sprintf("%d %s", code, body))
		}
	case code/400 == 1, code/500 == 1:
		if *responseCode == code {
			nagios.Ok(fmt.Sprintf("%d, %d bytes%s", code, size, body))
		} else {
			err := errors.New(fmt.Sprintf("%d%s", code, body))
			nagios.Critical(err)
		}
	}
}

func createUrl() *url.URL {
	if *urlArg != "" {
		if u, err := url.Parse(*urlArg); err != nil {
			nagios.Unknown(err.Error())
		} else {
			return u
		}
	}

	scheme := "http"
	if *ssl {
		scheme += "s"
	}
	hostname := ""
	if *host != "" {
		hostname = *host
	}
	if *port > 0 {
		hostname += fmt.Sprintf(":%d", *port)
	} else if *ssl {
		hostname += ":443"
	} else {
		hostname += ":80"
	}

	return &url.URL{
		Scheme:   scheme,
		Host:     hostname,
		Path:     *requestUri,
		RawQuery: *query,
	}
}
