package main

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/caddyserver/certmagic"
	"github.com/gorilla/mux"
	"github.com/oschwald/geoip2-golang"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	metrics "github.com/slok/go-http-metrics/metrics/prometheus"
	"github.com/slok/go-http-metrics/middleware"
)

var cityReader *geoip2.Reader
var orgReader *geoip2.Reader
var templateHTML *template.Template
var templateJSON *template.Template
var templateXML *template.Template
var templateClean *template.Template

type geoText struct {
	org         string
	details     string
	countryCode string
	city        string
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
	templateClean, err = template.ParseFiles("/usr/local/wtf/static/clean.template")
	if err != nil {
		log.Fatal(err)
	}

	mdlw := middleware.New(middleware.Config{
		Recorder: metrics.NewRecorder(metrics.Config{}),
	})

	r := mux.NewRouter()
	h := mdlw.Handler("", r)

	r.Host("ipv5.wtfismyip.com").HandlerFunc(ipv5Handler)
	r.Host("ipv7.wtfismyip.com").HandlerFunc(ipv5Handler)
	r.Host("text.wtfismyip.com").HandlerFunc(text)
	r.HandleFunc("/headers", headers)
	r.HandleFunc("/test", test)
	r.HandleFunc("/json", json)
	r.HandleFunc("/xml", xml)
	r.HandleFunc("/text", text)
	r.HandleFunc("/text/isp", textisp)
	r.HandleFunc("/text/geo", textgeo)
	r.HandleFunc("/text/city", textcity)
	r.HandleFunc("/text/country", textcountry)
	r.HandleFunc("/text/ip", text)
	r.HandleFunc("/text", text)
	r.HandleFunc("/js", jsHandle)
	r.HandleFunc("/jsclean", jscleanHandle)
	r.HandleFunc("/js2", js2Handle)
	r.HandleFunc("/js2clean", js2cleanHandle)
	r.HandleFunc("/clean", cleanHandle)
	r.HandleFunc("/traffic.png", trafficHandle)
	r.HandleFunc("/", wtfHandle).Methods("GET")
	r.HandleFunc("/", miscHandle).Methods("POST")
	r.HandleFunc("/", miscHandle).Methods("PUT")
	r.HandleFunc("/", miscHandle).Methods("DELETE")
	r.HandleFunc("/", miscHandle).Methods("TRACE")
	r.HandleFunc("/admin", adminHandle)
	r.HandleFunc("/administrator", adminHandle)
	r.HandleFunc("/metrics", metricsHandle)
	r.HandleFunc("/{foo:.*log$}", miscHandle)
	r.HandleFunc("/{foo:.*bak$}", miscHandle)
	r.HandleFunc("/{foo:.*swp$}", miscHandle)
	r.HandleFunc("/{foo:.*~$}", miscHandle)
	r.HandleFunc("/{foo:.*sql$}", sqlHandle)
	r.HandleFunc("/{foo:.*zip$}", zipHandle)
	r.HandleFunc("/{foo:.*gz$}", gzHandle)
	r.HandleFunc("/{foo:.*ini$}", iniHandle)
	r.HandleFunc("/{foo:.*php$}", trollHandle)
	r.HandleFunc("/{foo:.*asp$}", trollHandle)
	r.HandleFunc("/{foo:.*aspx$}", trollHandle)
	r.NotFoundHandler = http.HandlerFunc(custom404)

	config := certmagic.NewDefault()
	tags := []string{}
	config.CacheUnmanagedCertificatePEMFile("/docker/certs/wtf.ecc.cert.pem", "/docker/certs/wtf.ecc.key.pem",tags)
	tlsConfig := config.TLSConfig()

	srvHTTPS := &http.Server{
		ReadTimeout:  16 * time.Second,
		WriteTimeout: 24 * time.Second,
		Addr:         ":10443",
		Handler:      h,
		TLSConfig:    tlsConfig,
	}

	srvHTTP := &http.Server{
		ReadTimeout:  16 * time.Second,
		WriteTimeout: 24 * time.Second,
		Handler:      h,
		Addr:         ":10080",
	}

	go srvHTTP.ListenAndServe()
	srvHTTPS.ListenAndServeTLS("","")
}

func geoData(ip string) geoText {
	var details string
	var state string

	address := net.ParseIP(ip)
	isp, err := orgReader.ISP(address)
	if err != nil {
		log.Println(err)
	}

	record, err := cityReader.City(address)
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

	return geoText{isp.ISP, details, code, city}
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
	case <-time.After(5 * time.Second):
		return (ip)
	}
}

func custom404(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join("/usr/local/wtf/static", filepath.Clean(r.URL.Path))
	contents, err := ioutil.ReadFile(path)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such fucking page!")
	}
	w.Write(contents)
}

func js2Handle(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadFile("/usr/local/wtf/static/js2")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such fucking page!")
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(contents)
}


func js2cleanHandle(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadFile("/usr/local/wtf/static/js2clean")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such fucking page!")
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(contents)
}

func miscHandle(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadFile("/usr/local/wtf/static/evil.log")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such fucking page!")
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(contents)
}

func sqlHandle(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadFile("/usr/local/wtf/static/evil.sql")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such fucking page!")
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(contents)
}

func zipHandle(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadFile("/usr/local/wtf/static/evil.zip")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such fucking page!")
	}
	w.Header().Set("Content-Type", "application/zip")
	w.Write(contents)
}

func gzHandle(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadFile("/usr/local/wtf/static/evil.tar.gz")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such fucking page!")
	}
	w.Header().Set("Content-Type", "application/gzip")
	w.Write(contents)
}

func iniHandle(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadFile("/usr/local/wtf/static/evil.ini")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such fucking page!")
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write(contents)
}

func adminHandle(w http.ResponseWriter, r *http.Request) {
	contents, err := ioutil.ReadFile("/usr/local/wtf/static/admin.html")
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "No such fucking page!")
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(contents)
}

func trollHandle(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><head><meta http-equiv=\"Refresh\" content=\"0; url=https://www.youtube.com/watch?v=sTSA_sWGM44\" /></head><body><p>TROLOLOLOL!</p></body></html>")
}

// lets add some really rudimentary and shitty IP whitelisting to block access to explicit metrics
func metricsHandle(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	if add == "51.159.58.189" {
		promhttp.Handler().ServeHTTP(w,r)
	} else {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("sorry dude"))
	}
}

func json(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	hostname := reverseDNS(add)
	geo := geoData(add)
	isIPv6 := strings.Contains(add, ":")
	resp := wtfResponse{isIPv6, add, hostname, geo.details, geo.org, geo.countryCode}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	templateJSON.Execute(w, resp)
}

func text(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	response := add + "\n"
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	fmt.Fprintf(w, response)
}


func textisp(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	response := geoData(add).org + "\n"
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	fmt.Fprintf(w, response)
}

func textgeo(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	response := geoData(add).details + "\n"
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	fmt.Fprintf(w, response)
}

func textcountry(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	response := geoData(add).countryCode+ "\n"
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	fmt.Fprintf(w, response)
}

func textcity(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	response := geoData(add).city + "\n"
	w.Header().Set("Content-Type", "text/plain")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	fmt.Fprintf(w, response)
}

func test(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "Yes, the website is fucking running\n")
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
		response += "document.write('<center><p><h2>Your fucking IPv4 address is:</h2></center>');document.write('<center><p>' + ip + '</center>');document.write('<center><p><h2>Your fucking IPv4 hostname is:</h2></center>');document.write('<center><p>' + hostname + '</center>');document.write('<center><p><h2>Geographic location of your fucking IPv4 address:</h2></center>');document.write('<center><p>' + geolocation + '</center>');"
		fmt.Fprintf(w, response)
	}
}

func jscleanHandle(w http.ResponseWriter, r *http.Request) {
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
                response += "document.write('<center><p><h2>Your IPv4 address is:</h2></center>');document.write('<center><p>' + ip + '</center>');document.write('<center><p><h2>Your IPv4 hostname is:</h2></center>');document.write('<center><p>' + hostname + '</center>');document.write('<center><p><h2>Geographic location of your IPv4 address:</h2></center>');document.write('<center><p>' + geolocation + '</center>');"
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

func cleanHandle(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	isIPv6 := strings.Contains(add, ":")
	hostname := reverseDNS(add)
	geo := geoData(add)
	resp := wtfResponse{isIPv6, add, hostname, geo.details, geo.org, geo.countryCode}
	templateClean.Execute(w, resp)
}

func wtfHandle(w http.ResponseWriter, r *http.Request) {
	add := getAddress(r)
	isIPv6 := strings.Contains(add, ":")
	hostname := reverseDNS(add)
	geo := geoData(add)
	resp := wtfResponse{isIPv6, add, hostname, geo.details, geo.org, geo.countryCode}
	if r.TLS == nil {
		http.Redirect(w, r, "https://wtfismyip.com/", 301)
	} else {
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
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Commentary", "I really set most of these headers to get an A at securityheaders.io. Yes, I understand that most of these are completely unnecessary for this fucking website.")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		templateHTML.Execute(w, resp)
	}
}

func headers(w http.ResponseWriter, r *http.Request) {
	var response string
	for name, val := range r.Header {
		response += name + ": " + val[0] + "\n"
	}
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, response)
}

func ipv5Handler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
        fmt.Fprintf(w, "No such fucking protocol")
}

func trafficHandle(w http.ResponseWriter, r *http.Request) {
        contents, err := ioutil.ReadFile("/docker/metrics/omgwtfbbq.png")
        if err != nil {
                w.WriteHeader(http.StatusNotFound)
                fmt.Fprintf(w, "No such fucking page!")
        }
        w.Header().Set("Content-Type", "image/png")
        w.Write(contents)
}

func getAddress(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "0.0.0.0"
	}
	return ip
}
