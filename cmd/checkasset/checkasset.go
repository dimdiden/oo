package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dimdiden/oo"
)

func main() {
	// Flag block
	secret := flag.String("s", "", "specify secret key")
	api := flag.String("a", "", "specify api key")
	path := flag.String("f", "", "specify path to file")
	verbose := flag.Bool("v", false, "verbose mode")
	flag.Parse()

	if *secret == "" || *api == "" || *path == "" {
		fmt.Println("Incorrect usage, please specify the required parameters")
		flag.PrintDefaults()
		os.Exit(1)
	}

	file, err := os.Open(*path)
	if err != nil {
		log.Fatal("could not open file: ", err)
	}
	defer file.Close()

	ooClient, _ := oo.NewClient(*secret, *api, oo.BacklotDefaultEndpoint, 15)
	if *verbose {
		ooClient.SetLogOut(os.Stdout)
	}

	r := csv.NewReader(file)
	lines, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for i, line := range lines {
		if i == 0 {
			continue
		}
		embedCode := line[2]
		asset, err := ooClient.GetAsset(embedCode)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("Asset: %v; %v\n", asset.EmbedCode, asset.TimeRestrictions)
	}
}
