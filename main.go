package main

import (
	"encoding/json"
	"fmt"
	"github.com/serverless-aliyun/func-status/client/config"
	"github.com/serverless-aliyun/func-status/client/gen"
	r "github.com/serverless-aliyun/func-status/client/result"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	cfg, err := config.LoadApolloConfiguration()
	if err != nil {
		log.Panicln(err)
		return
	}
	err = r.ConnectToDB(cfg.DSN)
	if err != nil {
		return
	}
	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "pong")
	})

	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		check(cfg)
		_, _ = fmt.Fprintf(w, "done")
	})

	port := os.Getenv("FC_SERVER_PORT")
	if port == "" {
		port = "9000"
	}

	log.Println("Listening on :" + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func check(cfg *config.Config) {
	var endpoints []gen.EndpointGen
	for _, endpoint := range cfg.Endpoints {
		if endpoint.IsEnabled() {
			time.Sleep(777 * time.Millisecond)
			result := endpoint.EvaluateHealth()
			url := endpoint.URL
			if endpoint.Version != "" {
				url = "Running Version: " + endpoint.Version
			}
			endpoints = append(endpoints, gen.EndpointGen{
				Key:     endpoint.Key(),
				Name:    endpoint.Name,
				URL:     url,
				Results: r.SaveToDB(endpoint, result, cfg.MaxDays),
			})
			if cfg.Debug {
				rb, _ := json.Marshal(result)
				fmt.Println(string(rb))
			}
		}
	}
}
