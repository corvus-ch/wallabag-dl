package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	stdlog "log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/corvus-ch/wallabag-dl/client"
	"github.com/go-logr/logr"
	"github.com/go-logr/stdr"
	"golang.org/x/term"
)

const version = "0.1"
const defaultConfigJSON = "config.json"

var debug = flag.Bool("d", false, "get debug output (implies verbose mode)")
var v = flag.Bool("v", false, "print version")
var verbose = flag.Bool("verbose", false, "verbose mode")
var configJSON = flag.String("config", defaultConfigJSON, "file name of config JSON file")
var output = flag.String("output", "out", "directory to place exported files")
var format = flag.String("format", "epub", "format of the exported documents")
var archive = flag.Bool("archive", false, "archive entries after download")
var all = flag.Bool("all", false, "download all entries including the ones that are archived")

func handleFlags() logr.Logger {
	flag.Parse()

	log := stdr.NewWithOptions(stdlog.New(os.Stderr, "", stdlog.LstdFlags), stdr.Options{LogCaller: stdr.All})

	if *debug {
		stdr.SetVerbosity(2)
	} else if *verbose {
		stdr.SetVerbosity(1)
	}
	if len(flag.Args()) > 0 {
		log.V(2).Info("handleFlags: non-flag", "args", strings.Join(flag.Args(), " "))
	}
	// version first, because it directly exits here
	if *v {
		fmt.Printf("version %v\n", version)
		os.Exit(0)
	}

	return log.V(1)
}

func errorExit(log logr.Logger, err error, msg string) {
	if err != nil {
		log.Error(err, msg)
		os.Exit(1)
	}
}

type CredentialReader struct{}

func (r *CredentialReader) Username() string {
	fmt.Print("Enter Username: ")
	username, _ := bufio.NewReader(os.Stdin).ReadString('\n')
	return strings.TrimSpace(username)
}

func (r *CredentialReader) Password() string {
	fmt.Print("Enter Password: ")
	password, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(password))
}

type Config struct {
	Url          string `json:"url"`
	ClientId     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func main() {
	log := handleFlags()

	// check for config
	var cfg Config
	log.V(1).Info("read config from file", "path", *configJSON)
	cfgFile, err := os.Open("config.json")
	errorExit(log, json.NewDecoder(cfgFile).Decode(&cfg), "failed to read client configuration")

	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	c := client.New(log.V(1), httpClient, cfg.Url, cfg.ClientId, cfg.ClientSecret, &CredentialReader{})
	entries, err := c.GetEntries(params(*all))
	errorExit(log, err, "failed to instantiate API client")

	outputDir, err := filepath.Abs(*output)
	errorExit(log, err, "failed determine path to output directory")

	errorExit(log, os.MkdirAll(outputDir, 0755), "failed to create output directory")

	for _, entry := range entries {
		entryLog := log.WithValues("id", entry.ID, "title", entry.Title)
		entryLog.Info("Process entry")
		errorExit(entryLog, doExport(entryLog, c, entry, outputDir, *format), "failed to export")
		errorExit(entryLog, doArchive(entryLog, c, entry, *archive), "failed to archive")
	}
}

func params(all bool) url.Values {
	params := url.Values{}
	if !all {
		params.Set("archive", "0")
	}

	return params
}

func doExport(log logr.Logger, c *client.Client, entry client.Item, dir, format string) error {
	fileName := fmt.Sprintf("%s.%s", entry.Title, format)
	outputPath := filepath.Join(dir, fileName)
	log = log.WithValues("path", outputPath)
	var file *os.File
	if _, err := os.Stat(outputPath); err == nil {
		log.V(1).Info("Skip export because output document already exists")
		// File exists so nothing to do. Assumes the file contains the expected data and no upstream changes.
		return nil
	} else if os.IsNotExist(err) {
		log.V(1).Info("Create output file")
		if file, err = os.Create(outputPath); err != nil {
			return nil
		}
	} else {
		return fmt.Errorf("failed to open output path `%s`: %v", outputPath, err)
	}

	defer file.Close()

	log.Info("Export entry")
	return c.ExportEntry(entry.ID, format, file)
}

func doArchive(log logr.Logger, c *client.Client, entry client.Item, archive bool) error {
	if !archive || entry.IsArchived == 1 {
		log.V(1).Info("Skip archive", "archive", archive, "isArchived", entry.IsArchived)
		return nil
	}

	log.Info("Archive entry")
	return c.PatchEntry(entry.ID, map[string]interface{}{
		"archive": 1,
	})
}
