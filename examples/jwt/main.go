package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"

	jira "github.com/andygrunwald/go-jira"
	jwt "github.com/rbriski/atlassian-jwt"
)

// IssueEvent holds all issue change data
// This isn't even close to the full struct that
// JIRA returns.  This is a sample only.
type IssueEvent struct {
	Timestamp          int64  `json:"timestamp"`
	WebhookEvent       string `json:"webhookEvent"`
	IssueEventTypeName string `json:"issue_event_type_name"`
	Issue              struct {
		ID   string `json:"id"`
		Self string `json:"self"`
		Key  string `json:"key"`
	} `json:"issue"`
}

// SecurityContext holds the information from the installation handshake
type SecurityContext struct {
	Key            string `json:"key"`
	ClientKey      string `json:"clientKey"`
	PublicKey      string `json:"publicKey"`
	SharedSecret   string `json:"sharedSecret"`
	ServerVersion  string `json:"serverVersion"`
	PluginsVersion string `json:"pluginsVersion"`
	BaseURL        string `json:"baseUrl"`
	ProductType    string `json:"productType"`
	Description    string `json:"description"`
	EventType      string `json:"eventType"`
}

func main() {
	var (
		port    = flag.String("port", "8080", "web server port")
		baseURL = flag.String("baseurl", os.Getenv("BASE_URL"), "local base url")
	)
	flag.Parse()

	log.Printf("Example server - running on port:%v\nYou should spin up ngrok to expose this to your Jira dev instance", *port)

	// This is the JWT config.  I just pass it around everywhere here but I'd
	// probably use a context or something if this was a real app
	config := &jwt.Config{}

	http.HandleFunc("/atlassian-connect.json", atlassianConnect(*baseURL, "config"))

	http.HandleFunc("/installed", installed(config))

	http.HandleFunc("/issue_event", handleIssueEvent(config))

	http.HandleFunc("/uninstalled", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	http.ListenAndServe(":"+*port, nil)
}

func atlassianConnect(baseURL string, templateName string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		lp := path.Join("./templates", "atlassian-connect.json")
		vals := map[string]string{
			"BaseURL": baseURL,
		}
		tmpl, err := template.ParseFiles(lp)
		if err != nil {
			log.Fatalf("%v", err)
			http.Error(w, err.Error(), 500)
			return
		}
		tmpl.ExecuteTemplate(w, templateName, vals)
	}
}

func installed(c *jwt.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatalf("Can't read request:%v\n", err)
			http.Error(w, err.Error(), 500)
			return
		}
		var sc SecurityContext
		json.Unmarshal(body, &sc)

		// Set during install
		// This is only for the example.  If this was a real app you
		// should probably save the security context somewhere
		// and create the config from that.  This app only works if you install
		// it and never shut it down.
		c.Key = sc.Key
		c.ClientKey = sc.ClientKey
		c.SharedSecret = sc.SharedSecret
		c.BaseURL = sc.BaseURL

		json.NewEncoder(w).Encode([]string{"OK"})
	}
}

func handleIssueEvent(c *jwt.Config) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Fatalf("Can't read request:%v\n", err)
			http.Error(w, err.Error(), 500)
			return
		}

		var ie IssueEvent
		json.Unmarshal(body, &ie)

		jiraClient, _ := jira.NewClient(c.Client(), c.BaseURL)
		issue, _, err := jiraClient.Issue.Get(ie.Issue.Key, nil)
		if err != nil {
			log.Fatalf("failed to get issue: %v\n", err)
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Print("ISSUE INFO:\n")
		issueJSON, err := json.MarshalIndent(issue, "", "    ")
		if err != nil {
			log.Fatalf("failed to marshal issue JSON: %v\n", err)
			http.Error(w, err.Error(), 500)
			return
		}
		fmt.Print(string(issueJSON))

		json.NewEncoder(w).Encode([]string{"OK"})
	}
}
