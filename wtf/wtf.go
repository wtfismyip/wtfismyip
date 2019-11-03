package main

import (
	"crypto/tls"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/oschwald/geoip2-golang"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"
)

var cityReader *geoip2.Reader
var orgReader *geoip2.Reader
var templateHTML *template.Template
var templateJSON *template.Template
var templateXML *template.Template

type geoText struct {
	org         string
	details     string
	countryCode string
}

type wtfResponse struct {
	IPv6        bool
	Address     string
	Hostname    string
	Geo         string
	ISP         string
	CountryCode string
}

func main() {
	var err error

	cityReader, err = geoip2.Open("/usr/local/wtf/GeoIP/GeoIP2-City.mmdb")
	if err != nil {
		log.Fatal(err)
	}

	orgReader, err = geoip2.Open("/usr/local/wtf/GeoIP/GeoIP2-ISP.mmdb")
	if err != nil {
		log.Fatal(err)
	}

	defer cityReader.Close()
	defer orgReader.Close()

	templateHTML, err = template.ParseFiles("/usr/local/wtf/static/html.template")
	if err != nil {
		log.Fatal(err)
	}
	templateJSON, err = template.ParseFiles("/usr/local/wtf/static/json.template")
	if err != nil {
		log.Fatal(err)
	}
	templateXML, err = template.ParseFiles("/usr/local/wtf/static/xml.template")
	if err != nil {
		log.Fatal(err)
	}

	if err != nil {
		log.Fatal(err)
	}

	r := mux.NewRouter()
	r.HandleFunc("/headers", headers)
	r.HandleFunc("/json", json)
	r.HandleFunc("/xml", xml)
	r.HandleFunc("/text", text)
	r.HandleFunc("/js", jsHandle)
	r.HandleFunc("/", wtfHandle)
	r.NotFoundHandler = http.HandlerFunc(custom404)

	cfg := &tls.Config{
		MinVersion:               tls.VersionTLS10,
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		},
	}

	srvHTTPS := &http.Server{
		ReadTimeout:  16 * time.Second,
		WriteTimeout: 24 * time.Second,
		Addr:         ":443",
		Handler:      r,
		TLSConfig:    cfg,
	}

	srvHTTP := &http.Server{
		ReadTimeout:  16 * time.Second,
		WriteTimeout: 24 * time.Second,
		Handler:      r,
		Addr:         ":80",
	}

	srvHTTP.SetKeepAlivesEnabled(false)
	go srvHTTP.ListenAndServe()

	srvHTTPS.ListenAndServeTLS("/docker/certs/wtf.ecc.cert.pem", "/docker/certs/wtf.ecc.key.pem")
}

func geoData(ip string) geoText {
	var details string
	var state string
	address := net.ParseIP(ip)
	record, err := cityReader.City(address)
	if err != nil {
		log.Println(err)
	}
	foo, err := orgReader.ISP(address)
	if err != nil {
		log.Println(err)
	}
	if len(record.Subdivisions) > 0 {
		state = record.Subdivisions[0].IsoCode
	}
	city, isCityPresent := record.City.Names["en"]
	country, _ := record.Country.Names["en"]
	code := record.Country.IsoCode
	if isCityPresent {
		if len(state) > 0 {
			details = city + ", " + state + ", " + country
		} else {
			details = city + ", " + country
		}
	} else {
		if len(country) > 0 {
			details = country
		} else {
			details = "Unknown"
		}
	}
	if len(code) == 0 {
		code = "Unknown"
	}

	return geoText{foo.ISP, details, code}
}

func reverseDNS(ip string) string {
	omfg := make(chan string, 1)
	go func() {
		dnsName, err := net.LookupAddr(ip)
		if err != nil {
			omfg <- ip
		}
		if len(dnsName) == 0 {
			omfg <- ip
		} else {
			hostname := dnsName[0]
			omfg <- hostname[0 : len(hostname)-1]
		}
	}()

	select {
	case res := <-omfg:
		return (res)
	case <-time.After(2 * time.Second):
		return (ip)
	}
}

func custom404(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join("/usr/local/wtf/static", filepath.Clean(r.URL.Path))
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Fprintf(w, "No such fucking page!")
	}
	w.Write(contents)
}

func json(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	hostname := reverseDNS(add)
	geo := geoData(add)
	isIPv6 := strings.Contains(add, ":")
	resp := wtfResponse{isIPv6, add, hostname, geo.details, geo.org, geo.countryCode}
	w.Header().Set("Content-Type", "application/json")
	templateJSON.Execute(w, resp)
}

func text(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	response := add + "\n"
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, response)
}

func jsHandle(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	hostname := reverseDNS(add)
	geo := geoData(add)
	isIPv6 := strings.Contains(add, ":")
	if isIPv6 && r.Host == "ipv4.wtfismyip.com" {
		w.WriteHeader(http.StatusMisdirectedRequest)
		w.Write([]byte("Fucking protocol error"))
	} else {
		response := "ip='" + add + "';\n"
		response += "hostname='" + hostname + "';\n"
		response += "geolocation='" + geo.details + "';\n"
		response += "document.write('<center><p><h2>Your fucking IPv4 address is:</h2></center>');document.write('<center><p>' + ip + '</center>');document.write('<center><p><h2>Your IPv4 hostname is:</h2></center>');document.write('<center><p>' + hostname + '</center>');document.write('<center><p><h2>Geographic location of your IPv4 address:</h2></center>');document.write('<center><p>' + geolocation + '</center>');"
		fmt.Fprintf(w, response)
	}
}

func xml(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	hostname := reverseDNS(add)
	geo := geoData(add)
	isIPv6 := strings.Contains(add, ":")
	resp := wtfResponse{isIPv6, add, hostname, geo.details, geo.org, geo.countryCode}
	w.Header().Set("Content-Type", "application/xml")
	templateXML.Execute(w, resp)
}

func wtfHandle(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	isIPv6 := strings.Contains(add, ":")
	hostname := reverseDNS(add)
	geo := geoData(add)
	resp := wtfResponse{isIPv6, add, hostname, geo.details, geo.org, geo.countryCode}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("X-Hire-Me", "clint@wtfismyip.com")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	w.Header().Set("X-OMGWTF", "BBQ")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src wtfismyip.com; script-src ipv4.wtfismyip.com wtfismyip.com; style-src 'unsafe-inline'")
	w.Header().Set("X-DNS-Prefetch-Control", "off")
	templateHTML.Execute(w, resp)
}

func headers(w http.ResponseWriter, r *http.Request) {
	var response string
	for name, val := range r.Header {
		if (name != "X-Real-Ip") && (name != "Connection") {
			response += name + ": " + val[0] + "\n"
		}
	}
	w.Header().Set("X-Hire-Me", "clint@wtfismyip.com")
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, response)
}

func getAddress(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "0.0.0.0"
	}
	return ip
}
