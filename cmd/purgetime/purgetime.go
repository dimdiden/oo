package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/dimdiden/oo"
)

// var timer = time.NewTimer(2 * time.Second)

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
		if err := purgeTime(ooClient, embedCode); err != nil {
			log.Fatalf("could not process asset %v: %v", embedCode, err)
		}
		fmt.Printf("asset %v has been processed\n", embedCode)
	}
}

func purgeTime(oo oo.Apier, embedCode string) error {
	response, err := oo.Patch("/v2/assets/"+embedCode, strings.NewReader(`{"time_restrictions": null}`))
	if err != nil {
		return err
	}

	credits, err := strconv.Atoi(response.Header.Get("X-RateLimit-Credits"))
	if err != nil {
		return err
	}

	fmt.Println("credits left: ", response.Header.Get("X-RateLimit-Credits"))
	result, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return err
	}
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: [%v] [%v]", response.StatusCode, string(result))
	}

	timer := time.NewTimer(2 * time.Minute)
	if credits < 100 {
		fmt.Println("Only 100 credits left. Waiting for 2 min...")
		<-timer.C
	}
	return nil
}
