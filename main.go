package main

import (
	"flag"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/andygrunwald/cachet"
	"github.com/prometheus/alertmanager/template"
)

const (
	landingPage = `<html>
	<head>
		<title>Prometheus Cachet</title>
		<style>
			body { color: #ffffff; background-color: #26282b; font-family: monospace; padding: 1%% }
			a:-webkit-any-link { color: #ffffff; text-decoration: underline }
			a:hover { color: #ff5959 }
			.footer { position: fixed; bottom: 0; text-align: center }
		</style>
	</head>
	<body>
		<h2>Prometheus Cachet <small>Integration</small></h2>
		<p>Small go based microservice to receive Prometheus Alertmanager triggers and update corresponding incidents in Cachet.</p>
		<p class="footer">GitHub: %s</p>
	</body>
</html>`
	github = "https://github.com/gregdhill/prometheus-cachet"
)

var (
	// Flag assignment
	portNumber = flag.String("port", "8080", "The port number to listen on for HTTP requests.")
	address    = flag.String("address", "0.0.0.0", "The address to listen on for HTTP requests.")

	logLevel   = flag.String("level", "info", "The level of logs to log")	
)

type alerts struct {
	client    *cachet.Client
	incidents map[string]int
	mutex     sync.Mutex
}

func loglevel(opt string) {
	switch opt {
	case "error":
		log.SetLevel(log.ErrorLevel)
	case "warn":
		log.SetLevel(log.WarnLevel)
	case "info":
		log.SetLevel(log.InfoLevel)
	case "debug":
		log.SetLevel(log.DebugLevel)
	default:
		log.Warnln("Unrecognized log level, will default to `info` log level")
	}
}

func (alt *alerts) cachetAlert(status, name, message string) {
	if _, ok := alt.incidents[name]; ok {
		if strings.ToUpper(status) == "RESOLVED" {
			log.Infof("Resolving alert \"%s\"", name)
			incidentResolve := &cachet.IncidentUpdate{
				Message: "This incdent has been resolved, we apologise for any inconvenience caused.",
				Status:  cachet.IncidentStatusFixed,
			}
			alt.client.IncidentUpdates.Create(alt.incidents[name],incidentResolve)
			alt.mutex.Lock()
			delete(alt.incidents, name)
			alt.mutex.Unlock()
		} else {
			log.Infof("Alert \"%s\" already reported.", name)
		}
		return
	}

	incident := &cachet.Incident{
		Name:    name,
		Message: message,
		Status:  cachet.IncidentStatusInvestigating,
	}
	newIncident, _, _ := alt.client.Incidents.Create(incident)

	log.Infof("Incident reported: %s", newIncident.Name)

	id := newIncident.ID
	alt.mutex.Lock()
	alt.incidents[name] = id
	alt.mutex.Unlock()

	log.Debugf("Incident created with ID: %d", newIncident.ID)
}

func (alt *alerts) prometheusAlert(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	log.Info("Receiving incoming alert")
	data := template.Data{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		log.Errorf("Error decoding alert: %s", err.Error())
		log.Debugf("Body: ",r.Body)
		return
	}
	status := data.Status
	for _, alert := range data.Alerts {
		log.Debugf("Alert: status=%s,Labels=%v,Annotations=%v", alert.Status, alert.Labels, alert.Annotations)
		alt.cachetAlert(status, alert.Labels["alertname"], alert.Annotations["summary"])
	}
}

func (alt *alerts) health(w http.ResponseWriter, r *http.Request) {
	pong, _, err := alt.client.General.Ping()

	if(pong == "Pong!" && err == nil) {
		fmt.Fprint(w, "Healthy.")
	} else {
		log.Errorf("Cachet API issue. Response: %s, ERR: %s", pong, err)

		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "Unhealthy.")
	}
}

func landing(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(fmt.Sprintf(landingPage, github)))
}

func main() {
	flag.Parse()

	loglevel(*logLevel)

	statusPage := os.Getenv("CACHET_URL")
	if len(statusPage) == 0 {
		log.Fatalf("Cachet URL not provided, please set enviroment variable 'CACHET_URL'.")
	}
	client, err := cachet.NewClient(statusPage, nil)
	if err != nil {
		log.Fatalf("Failed to initialize cachet client sdk: %s", err)
	}
	apiKey := os.Getenv("CACHET_KEY")
	if len(apiKey) == 0 {
		log.Fatalf("Cachet API Token not provided, please set enviroment variable 'CACHET_KEY'.")
	}
	client.Authentication.SetTokenAuth(apiKey)

	alerts := alerts{incidents: make(map[string]int), client: client}
	http.HandleFunc("/", landing)
	http.HandleFunc("/health", alerts.health)
	http.HandleFunc("/webhook", alerts.prometheusAlert)

	log.Infof("Listening for requests on %s:%s", *address, *portNumber)
	log.Fatalf("Failed to start web server: %s", http.ListenAndServe(fmt.Sprintf("%s:%s", *address, *portNumber), nil))
}
