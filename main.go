package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"
	"time"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"
	"gopkg.in/yaml.v2"
)

// RawConfig
type RawConfig struct {
	GlobalConfig    GlobalConfig    `yaml:"global"`
	RequestsConfigs []RequestConfig `yaml:"requests"`
	NotiConfigs     NotiConfig      `yaml:"notification"`
}

// GlobalConfig
type GlobalConfig struct {
	Github  GithubConfig `yaml:"github"`
	Timeout int          `yaml:"timeout"`
}

// GitHubConfig
type GithubConfig struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo"`
}

// RequestConfig
type RequestConfig struct {
	Url    string `yaml:"url"`
	Method string `yaml:"method"`
}

// NotiConfig
type NotiConfig struct {
	Github NotiGitHubConfig `yaml:"github"`
}

// NotiGitHubConfig
type NotiGitHubConfig struct {
	Assignees []string `yaml:"assignees"`
	Labels    []string `yaml:"labels"`
}

// RequestResult
type RequestResult struct {
	Request    RequestConfig
	StatusCode int
	Duration   time.Duration
	IsSucceed  bool   // true when 200, false when not
	IsFailed   bool   // true when timeout or any failure occurs, false when otherwise
	ErrorMsg   string // error message if !IsSucceed | IsFailed
}

var logger *log.Logger
var config RawConfig

func init() {
	// Setup default logger
	logger = log.New(os.Stdout, "INFO: ", log.LstdFlags)

	// Read config from yaml file
	fnConfig, _ := filepath.Abs("./config.yaml")
	yamlFile, err := ioutil.ReadFile(fnConfig)
	if err == nil {
		err = yaml.Unmarshal(yamlFile, &config)
		if err != nil {
			logger.Fatalln(err)
		}

		log.Println("Configs: ", config)
		log.Println("Global configs: ", config.GlobalConfig)
		log.Println("Requests: ", config.RequestsConfigs)
	}
}

func main() {
	// TODO: Validate configs before making channel

	// Make channel for reponses
	nReqs := len(config.RequestsConfigs)
	resChans := make(chan RequestResult, nReqs)
	log.Println("Making channel for responses: ", nReqs)

	// Make WaitGroup for aggregate responses
	resWg := sync.WaitGroup{}
	resWg.Add(nReqs)

	// Do request & parse response per config in parallel
	for _, RequestsConfig := range config.RequestsConfigs {
		if RequestsConfig.Url == "" || RequestsConfig.Method == "" {
			resChans <- RequestResult{Request: RequestsConfig, StatusCode: 0, Duration: -1, IsSucceed: false, IsFailed: true, ErrorMsg: "Invalid request config"}
			resWg.Done()
			continue
		}

		// Do in parallel using goroutine
		go func(RequestsConfig RequestConfig, timeout int, logger *log.Logger) {
			defer resWg.Done()

			// Create context withTimeout
			ctx, cancel := context.WithTimeout(context.Background(), time.Second*time.Duration(timeout))
			defer cancel()

			// Create request
			req, err := http.NewRequestWithContext(ctx, RequestsConfig.Method, RequestsConfig.Url, nil)
			if err != nil {
				logger.Println("Error creating request: ", err)
				resChans <- RequestResult{Request: RequestsConfig, StatusCode: 0, Duration: -1, IsSucceed: false, IsFailed: true, ErrorMsg: err.Error()}
				return
			}

			// Send request
			reqStart := time.Now()
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				logger.Println("Error sending request: ", err)
				resChans <- RequestResult{Request: RequestsConfig, StatusCode: 0, Duration: -1, IsSucceed: false, IsFailed: true, ErrorMsg: err.Error()}
				return
			}
			resp.Body.Close()
			reqDuration := time.Since(reqStart)

			// Parse response
			var reqResult RequestResult
			reqResult.Request = RequestsConfig
			reqResult.StatusCode = resp.StatusCode
			reqResult.Duration = reqDuration
			reqResult.IsSucceed = (resp.StatusCode == 200)
			reqResult.IsFailed = false

			// Send result to channel
			resChans <- reqResult
		}(RequestsConfig, config.GlobalConfig.Timeout, logger)
	}

	// Wait for aggregating reqResults
	resWg.Wait()

	// Aggregate reqResults
	var reqResults []RequestResult
	for i := 0; i < nReqs; i++ {
		reqResults = append(reqResults, <-resChans)
	}
	logger.Println("Request Results: ", reqResults)

	// Read template from md file
	tmplTitle, err := template.ParseFiles("./template_title.md")
	if err != nil {
		logger.Fatalln(err)
	}
	var issueTitle bytes.Buffer
	if err := tmplTitle.Execute(&issueTitle, time.Now().Format(time.RFC3339Nano)); err != nil {
		logger.Fatalln(err)
	}

	tmplBody, err := template.ParseFiles("./template_body.md")
	if err != nil {
		logger.Fatalln(err)
	}
	var issueBody bytes.Buffer
	tmplBody.Execute(&issueBody, reqResults)

	// Submit request results to GitHub issue
	GITHUB_TOKEN := os.Getenv("GH_TOKEN")
	ghCtx := context.Background()
	ghTokn := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: GITHUB_TOKEN})
	ghAuth := oauth2.NewClient(ghCtx, ghTokn)
	ghClient := github.NewClient(ghAuth)

	_, _, err = ghClient.Issues.Create(ghCtx, config.GlobalConfig.Github.Owner, config.GlobalConfig.Github.Repo, &github.IssueRequest{
		Title:     github.String(issueTitle.String()),
		Body:      github.String(issueBody.String()),
		Labels:    &config.NotiConfigs.Github.Labels,
		Assignees: &config.NotiConfigs.Github.Assignees,
	})
	if err != nil {
		logger.Fatalln("Error on submitting issue: ", err)
	}

	// TODO: Save results for statistics
}
