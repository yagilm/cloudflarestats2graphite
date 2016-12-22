package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/marpaia/graphite-golang"
	flag "github.com/ogier/pflag"
)

// Lasttimeserie is the global store to remember which is the last timeserie we
// sent to statsd. When not assigned, it is '0001-01-01 00:00:00 +0000 UTC',
// thus earlier of any possible timeserie
var Lasttimeserie time.Time

// Configuration options
type Configuration struct {
	headerXAuthEmail string
	headerXAuthKey   string
	zone             string
	zonedomain       string
	Graphite         struct {
		Host string
		Port int
	}
}

// Config keeps the configuration
var Config Configuration

// Check if configuration is invalid
func (conf Configuration) configurationInvalid() bool {
	return conf.headerXAuthEmail == "" ||
		conf.headerXAuthKey == "" ||
		conf.zone == "" ||
		conf.zonedomain == "" ||
		conf.Graphite.Host == "" ||
		conf.Graphite.Port == 0
}

// Response describes the parts we want from cloudflare's json response
type Response struct {
	Result struct {
		Timeseries []struct {
			Since    time.Time `json:"since"`
			Until    time.Time `json:"until"`
			Requests struct {
				HTTPStatusCodes map[string]int `json:"http_status"`
			}
		} `json:"timeseries"`
	} `json:"result"`
}

// init() runs before the main function as described in:
// https://golang.org/doc/effective_go.html#init
func init() {
	flag.StringVar(&Config.headerXAuthEmail, "email", "", "X-Auth-Email for cloudflare's API")
	flag.StringVar(&Config.headerXAuthKey, "auth", "", "X-Auth-Key for cloudflare's API")
	flag.StringVar(&Config.zone, "zone", "", "Cloudflare's zone")
	flag.StringVar(&Config.zonedomain, "zonedomain", "", "Domain of the zone")
	flag.StringVar(&Config.Graphite.Host, "ghost", "", "Graphite host")
	flag.IntVar(&Config.Graphite.Port, "gport", 0, "Graphite port")
	flag.Usage = func() {
		fmt.Printf("Usage: cloudflareanalytics [options]\nRequired options:\n")
		flag.PrintDefaults()
	}
	flag.Parse()
	// Replace domain's dot with underscore so graphite doesn't devide it
	Config.zonedomain = strings.Replace(Config.zonedomain, ".", "_", -1)
	if Config.configurationInvalid() {
		flag.Usage()
		os.Exit(1)
	}
}

// getzoneanalytics calls cloudflare API with the provided options and saves the response
// Requires Headers: X-Auth-Email, X-Auth-Key, "Content-Type: application/json", since=-MINUTES
func getzoneanalytics() []byte {
	// make the request with the appropriate headers
	req, _ := http.NewRequest("GET",
		fmt.Sprintf(
			"https://api.cloudflare.com/client/v4/zones/%s/analytics/dashboard?since=-30",
			Config.zone),
		nil)
	req.Header.Set("X-Auth-Email", Config.headerXAuthEmail)
	req.Header.Set("X-Auth-Key", Config.headerXAuthKey)
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	res, _ := client.Do(req)
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Print("Something went wrong requesting the json in cloudflare's API:", err)
	}
	return body
}

func sendtographite(res []byte) {
	// initialize connection to graphite
	Graphite, err := graphite.NewGraphite(Config.Graphite.Host, Config.Graphite.Port)
	if err != nil {
		fmt.Printf("Something went wrong connecting to Graphite")
	}
	response := Response{}
	json.Unmarshal([]byte(res), &response)
	// Iterate through the timeseries
	for _, timeserie := range response.Result.Timeseries {
		if timeserie.Until.After(Lasttimeserie) {
			metrictime := timeserie.Until.Unix()
			for key, value := range timeserie.Requests.HTTPStatusCodes {
				key = fmt.Sprintf("stats.cloudflare.%s.%s", Config.zonedomain, key)
				Graphite.SendMetric(graphite.NewMetric(key, fmt.Sprintf("%d", value), metrictime))
			}
			// fmt.Println("sent smth to graphite")
			Lasttimeserie = timeserie.Until
		}
		// break
	}
}

func main() {
	timer := time.NewTicker(time.Minute * 2)
	// infinite loop
	for {
		sendtographite(getzoneanalytics())
		<-timer.C
	}
}
