package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/Strubbl/wallabago"
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

func main() {
	log.SetOutput(os.Stdout)
	handleFlags()
	// check for config
	if *verbose {
		log.Println("reading config", *configJSON)
	}
	errorExit(wallabago.ReadConfig(*configJSON))

	// TODO Only get unread entries.
	entries, err := wallabago.GetAllEntries()
	errorExit(err)

	outputDir, err := filepath.Abs(*output)
	errorExit(err)

	errorExit(os.MkdirAll(outputDir, 0755))

	for _, entry := range entries {
		if entry.IsArchived == 1 {
			continue
		}
		data, err := wallabago.ExportEntry(wallabago.APICall, entry.ID, "epub")
		errorExit(err)
		fileName := fmt.Sprintf("%s.epub", entry.Title)
		outputPath := filepath.Join(outputDir, fileName)
		errorExit(ioutil.WriteFile(outputPath, data, 0644))
	}
}
