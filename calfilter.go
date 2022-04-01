package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type RuleMode int

const (
	ruleAnd = iota
	ruleOr
)

type RuleGroup struct {
	Mode  RuleMode          `json:"mode"`
	Rules map[string]string `json:"rules"`
}

type Config struct {
	Port       string      `json:"port"`
	CalUrl     string      `json:"cal_url"`
	Key        string      `json:"key"`
	RuleGroups []RuleGroup `json:"rule_groups"`
}

var conf Config

func downloadCal(url string) (io.ReadCloser, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("received %v status code when fetching calendar", res.StatusCode)
	}
	contentType := res.Header["Content-Type"][0]
	expectedContentType := "text/calendar;charset=UTF-8"
	if contentType != expectedContentType {
		return nil, fmt.Errorf("expected content-type %q, instead got: %q", expectedContentType, contentType)
	}

	return res.Body, nil
}

func calHandler(w http.ResponseWriter, r *http.Request) {
	// Validate key
	q := r.URL.Query()
	if q.Get("key") != conf.Key {
		http.Error(w, "Specified key does not match!", http.StatusUnauthorized)
		return
	}

	body, err := downloadCal(conf.CalUrl)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer func(body io.ReadCloser) {
		err := body.Close()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}(body)

	w.Header().Set("content-type", "text/calendar")
	w.Header().Set("content-disposition", "attachment; filename=calendar.ics")

	cf := NewParser(body)
	_, err = cf.Parse(w, conf.RuleGroups)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func handleRequests(port string) {
	http.HandleFunc("/filter", calHandler)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", port), nil))
}

func readConfig(filename string) (Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return Config{}, err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Fatalf("Closing config file failed with error: %q", err)
		}
	}(f)

	content, err := ioutil.ReadAll(f)
	if err != nil {
		return Config{}, err
	}

	var config Config
	err = json.Unmarshal(content, &config)
	if err != nil {
		return Config{}, err
	}

	return config, nil
}

func main() {
	var err error
	conf, err = readConfig("./config.json")
	if err != nil {
		log.Fatalf("Reading config file failed with error: %v", err)
	}

	handleRequests(conf.Port)
}
