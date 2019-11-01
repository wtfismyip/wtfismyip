package main

import (
	"github.com/gorilla/mux"
	"github.com/oschwald/geoip2-golang"
	"log"
	"net"
	"fmt"
	"time"
	"net/http"
	"strings"
)

var cityReader *geoip2.Reader
var orgReader *geoip2.Reader

type geoText struct {
	org string
	details string
	countryCode string
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

	r := mux.NewRouter()
	r.HandleFunc("/headers", headers)
	r.HandleFunc("/json", json)
	r.HandleFunc("/xml", xml)
	r.HandleFunc("/text", text)
	r.HandleFunc("/js", jsHandle)
	r.HandleFunc("/", wtfHandle)
	r.NotFoundHandler = http.HandlerFunc(custom404)
	http.ListenAndServe(":8080", r)
}

func geoData(ip string) (geoText) {
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

func reverseDNS(ip string) (string) {
	omfg := make(chan string, 1)
	go func() {
		dnsName, err := net.LookupAddr(ip)
		if err != nil {
			omfg <- ip
		}
		if (len(dnsName) == 0) {
			omfg <- ip
		} else {
			hostname := dnsName[0]
			omfg <- hostname[0:len(hostname)-1]
		}
	}()

	select {
	case res := <- omfg :
		return(res)
	case <- time.After(2 * time.Second):
		return(ip)
	}
}

func custom404(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "No such fucking page!")
}

func json(w http.ResponseWriter, r *http.Request) {
	add := r.Header.Get("X-Real-IP")
	hostname := reverseDNS(add)
	geo := geoData(add)
	response := "{\n    \"YourFuckingIPAddress\": \"" + add + "\",\n    \"YourFuckingLocation\": \"" + geo.details + "\",\n    \"YourFuckingHostname\": \"" + hostname + "\",\n    \"YourFuckingISP\": \"" + geo.org + "\",\n    \"YourFuckingTorExit\": false,\n    \"YourFuckingCountryCode\": \"" + geo.countryCode + "\"\n}\n"

	w.Header().Set("X-Hire-Me", "clint@wtfismyip.com")
	w.Header().Set("Content-Type", "application/json");
	w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate");
	w.Header().Set("Pragma", "no-cache");
	w.Header().Set("Expires", "0");
	w.Header().Set("Access-Control-Allow-Origin", "*");
	w.Header().Set("Access-Control-Allow-Methods", "GET");
	fmt.Fprintf(w, response)
}

func text(w http.ResponseWriter, r *http.Request) {
	add := r.Header.Get("X-Real-IP")
	response := add + "\n"
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, response)
}

func jsHandle(w http.ResponseWriter, r *http.Request) {
	add := r.Header.Get("X-Real-IP")
	hostname := reverseDNS(add)
	geo := geoData(add)
	isIPV6 := strings.Contains(add, ":")
	if (isIPV6 && r.Host == "ipv4.wtfismyip.com") {
		w.WriteHeader(http.StatusMisdirectedRequest)
		w.Write([]byte("Fucking protocol error"))
	} else {
		response := "ip='" + add + "';\n";
		response += "hostname='" + hostname + "';\n"
		response += "geolocation='" + geo.details + "';\n"
		response += "document.write('<center><p><h2>Your fucking IPv4 address is:</h2></center>');document.write('<center><p>' + ip + '</center>');document.write('<center><p><h2>Your IPv4 hostname is:</h2></center>');document.write('<center><p>' + hostname + '</center>');document.write('<center><p><h2>Geographic location of your IPv4 address:</h2></center>');document.write('<center><p>' + geolocation + '</center>');";
		fmt.Fprintf(w, response)
	}
}

func xml(w http.ResponseWriter, r *http.Request) {
	add := r.Header.Get("X-Real-IP")
	hostname := reverseDNS(add)
	geo := geoData(add)
	response := "<?xml version=\"1.0\" encoding='UTF-8'?>\n<wtf>\n   <your-fucking-ip-address>" + add + "</your-fucking-ip-address>\n   <your-fucking-location>" + geo.details + "</your-fucking-location>\n   <your-fucking-hostname>" + hostname + "</your-fucking-hostname>\n   <your-fucking-isp>" + geo.org + "</your-fucking-isp>\n   <your-fucking-tor-exit>" + "false" + "</your-fucking-tor-exit>\n   <your-fucking-country-code>" + geo.countryCode + "</your-fucking-country-code>\n</wtf>\n";
	fmt.Fprintf(w, response)
}

func wtfHandle(w http.ResponseWriter, r *http.Request) {
	add := r.Header.Get("X-Real-IP")
	isIPV6 := strings.Contains(add, ":")
	hostname := reverseDNS(add)
	geo := geoData(add)
	response := "<!DOCTYPE HTML PUBLIC \"-//W3C//DTD HTML 4.01 Transitional//EN\" \"http://www.w3.org/TR/html4/loose.dtd\">\n<html><head><link rel=\"canonical\" href=\"https://wtfismyip.com/\"><link rel=\"icon\" href=\"favicon.ico\" type=\"image/x-icon\"><link rel=\"shortcut icon\" href=\"favicon.ico\" type=\"image/x-icon\"><meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\"><meta name=\"msvalidate.01\" content=\"2FAFC220324DC28BC604C74AF6A73153\"><meta name=\"google-site-verification\" content=\"H3adQkdAhp3ae5Oq6hSz9VsRKcGJPhYuAdbt_sW7qJo\" /><meta name=\"keywords\" content=\"wtf, my ip\"><meta name=\"description\" content=\"Tells you WTF your IP address is\"><title>WTF is my IP?!?!??</title><style type=\"text/css\" media=\"screen\">\n*{font-family:sans-serif;margin:0;padding:0}\nbody{background:#fff}\na{text-decoration:none}\na:hover{color:#f90}\n#merryfuckingchristmas{color:green}\n#main{position:relative;top:20px}\n#main h1{font-size:40px;font-weight:400;line-height:40px;letter-spacing:-1px;padding:20px 0}\n#main h2{font-size:24px;font-weight:700;letter-spacing:-1px;padding:2px 0}\n#main p{font-size:15px;line-height:20px;margin:0 0 20px}\n#main ul{padding:0 0 0 20px}\n#main li{list-style-type:square;font-size:15px;line-height:10px;margin:0 0 10px}\n#sidebar{position:relative;top:40px;border-top:1px solid #ccc;text-align:center;width:300px;margin-right:auto;margin-left:auto;padding:20px 20px 0 0}\n#sidebar h2{text-transform:uppercase;font-size:13px;color:#333;letter-spacing:1px;line-height:20px}\n#sidebar ul{list-style-type:none;margin:20px 0}\n#sidebar li{font-size:14px;line-height:20px}\n.blah{margin-bottom:3px}\na:link,a:visited,#tor{color:#f30}\n.halb{padding:20px}\n#local{position:relative;top:20px}\n#local h1{font-size:40px;font-weight:400;line-height:40px;letter-spacing:-1px;padding:20px 0}\n#local h2{font-size:24px;font-weight:700;letter-spacing:-1px;padding:2px 0}\n#local p{font-size:15px;line-height:20px;margin:0 0 20px}\n#local ul{padding:0 0 0 20px}\n#local li{list-style-type:square;font-size:15px;line-height:10px;margin:0 0 10px}</style></script> </head><!--Belive me, you could not write shittier HTML than this even if you tried--><body><div id=\"main\">"

	if isIPV6 {
		response += "<center><p><h2>Your fucking IPv6 address is:</h2></center><center><p>" + add
	} else {
		response += "<center><p><h2>Your fucking IP address is:</h2></center><center><p>" + add
	}
	response += "</center><center><p><h2>Your host name is:</h2></center><center><p>" + hostname

	if (!isIPV6) {
		response += "</center></div><div id=\"local\"></div><div id=\"main\"><script type=\"text/javascript\" src=\"https://wtfismyip.com/js2\"></script><center>"
		response += "</center><center><p><h2>Geographic location of your IP address:</h2></center><center><p>" + geo.details + "</center>"
		response += "<center><p><h2>Your ISP:</h2></center><center><p>" + geo.org + "</center>"
	} else {
		response += "</center><center><p><h2>Geographic location of your IPv6 address:</h2></center><center><p>" + geo.details + "</center>"
		response += "</center></div><div id=\"local\"></div><div id=\"main\"><script type=\"text/javascript\" src=\"https://wtfismyip.com/js2\"></script><center>"
		response += "<script type=\"text/javascript\" src=\"https://ipv4.wtfismyip.com/js\"></script><br><p><center><H3><a href=\"http://ipv4.wtfismyip.com\">WTF is my IPv4 address!?</a></H3>";
	}

	response += "<br><br><center><H3><a href=\"/headers\">What fucking headers are my browser sending?</a></H3></center><center><p><H3>Give me this shit in <a href=\"/xml\">XML</a>, <a href=\"/json\">JSON</a> or <a href=\"/text\">plain text!</a></H3><p></center>"
	response += "</div><div id=\"sidebar\"><h2>Resources</h2><ul><li><p class=\"blah\"><a href=\"/why\">Why wtfismyip.com?</a></p></li><li><p class=\"blah\"><a href=\"/automation\">Automation Policy</a></p><li><p class=\"blah\"><a href=\"/privacy\">Privacy Policy</ali><p class=\"blah\"><a href=\"/donate\">Donate</a></p></p></ul><ul><li>Don't like being tracked? Download the <a href=\"https://www.torproject.org/download/\">Tor Browser Bundle!</a></ul><p class=\"halb\"></div><p><p></body></html>";

	w.Header().Set("Content-Type", "text/html; charset=utf-8");
	w.Header().Set("X-Hire-Me", "clint@wtfismyip.com")
	w.Header().Set("X-Frame-Options", "DENY");
	w.Header().Set("Cache-Control", "no-cache, no-store, max-age=0, must-revalidate");
	w.Header().Set("Pragma", "no-cache");
	w.Header().Set("Expires", "0");
	w.Header().Set("X-OMGWTF", "BBQ");
	w.Header().Set("X-XSS-Protection", "1; mode=block");
	w.Header().Set("X-Content-Type-Options", "nosniff");
	w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src wtfismyip.com; script-src ipv4.wtfismyip.com wtfismyip.com; style-src 'unsafe-inline'");
	w.Header().Set("X-DNS-Prefetch-Control", "off");
	fmt.Fprintf(w, response)
}

func headers(w http.ResponseWriter, r *http.Request) {
	var response string
	for name, val := range r.Header {
		if (name != "X-Real-Ip") && (name != "Connection") {
			response += name + ": " + val[0] + "\n"
		}
	}

	w.Header().Set("X-Hire-Me", "clint@wtfismyip.com")
	w.Header().Set("Content-Type", "text/plain");
	fmt.Fprintf(w, response)
}
