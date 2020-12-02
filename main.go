package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/corvus-ch/wallabag-dl/client"
	"golang.org/x/term"
)

const version = "0.1"
const defaultConfigJSON = "config.json"

var debug = flag.Bool("d", false, "get debug output (implies verbose mode)")
var debugDebug = flag.Bool("dd", false, "get even more debug output like data (implies debug mode)")
var v = flag.Bool("v", false, "print version")
var verbose = flag.Bool("verbose", false, "verbose mode")
var configJSON = flag.String("config", defaultConfigJSON, "file name of config JSON file")
var output = flag.String("output", "out", "directory to place exported files")

func handleFlags() {
	flag.Parse()
	if *debug && len(flag.Args()) > 0 {
		log.Printf("handleFlags: non-flag args=%v", strings.Join(flag.Args(), " "))
	}
	// version first, because it directly exits here
	if *v {
		fmt.Printf("version %v\n", version)
		os.Exit(0)
	}
	// test verbose before debug because debug implies verbose
	if *verbose && !*debug && !*debugDebug {
		log.Printf("verbose mode")
	}
	if *debug && !*debugDebug {
		log.Printf("handleFlags: debug mode")
		// debug implies verbose
		*verbose = true
	}
	if *debugDebug {
		log.Printf("handleFlags: debug² mode")
		// debugDebug implies debug
		*debug = true
		// and debug implies verbose
		*verbose = true
	}
}

func errorExit(err error) {
	if err != nil {
		fmt.Println(err.Error())
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
	log.SetOutput(os.Stdout)
	handleFlags()
	// check for config
	if *verbose {
		log.Println("reading config", *configJSON)
	}
	var cfg Config
	cfgFile, err := os.Open("config.json")
	errorExit(json.NewDecoder(cfgFile).Decode(&cfg))

	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	c := client.New(httpClient, cfg.Url, cfg.ClientId, cfg.ClientSecret, &CredentialReader{})
	entries, err := c.GetEntries(url.Values{
		"archive": {"0"},
	})
	errorExit(err)

	outputDir, err := filepath.Abs(*output)
	errorExit(err)

	errorExit(os.MkdirAll(outputDir, 0755))

	for _, entry := range entries {
		if entry.IsArchived == 1 {
			continue
		}
		fileName := fmt.Sprintf("%s.pdf", entry.Title)
		outputPath := filepath.Join("out", fileName)

		file, err := os.Create(outputPath)
		errorExit(err)
		defer file.Close()

		errorExit(c.ExportEntry(entry.ID, "pdf", file))
	}
}
