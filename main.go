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
var format = flag.String("format", "epub", "format of the exported documents")
var archive = flag.Bool("archive", false, "archive entries after download")

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
		errorExit(doExport(c, entry, outputDir, *format))
		errorExit(doArchive(c, entry, *archive))
	}
}

func doExport(c *client.Client, entry client.Item, dir, format string) error {
	fileName := fmt.Sprintf("%s.%s", entry.Title, format)
	outputPath := filepath.Join(dir, fileName)
	var file *os.File
	if _, err := os.Stat(outputPath); err == nil {
		// File exists so nothing to do. Assumes the file contains the expected data and no upstream changes.
		return nil
	} else if os.IsNotExist(err) {
		if file, err = os.Create(outputPath); err != nil {
			return nil
		}
	} else {
		return fmt.Errorf("failed to open output path `%s`: %v", outputPath, err)
	}

	defer file.Close()

	return c.ExportEntry(entry.ID, format, file)
}

func doArchive(c *client.Client, entry client.Item, archive bool) error {
	if !archive {
		return nil
	}

	return c.PatchEntry(entry.ID, map[string]interface{}{
		"archive": 1,
	})
}
