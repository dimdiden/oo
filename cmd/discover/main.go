package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strconv"

	"github.com/dimdiden/oo"
)

// SELECT updated_at, embed_code, name
// FROM movies
// WHERE provider_id = 96623
// ORDER BY updated_at DESC
// LIMIT 10

func main() {
	// input data
	var (
		akey          string
		skey          string
		inFile        string
		excludeLabels string
		scoreType     string
		profile       string
		limit         int
		updatedAt     string
		verbose       bool
	)
	// Flag block
	flag.StringVar(&akey, "a", "", "specify api key")
	flag.StringVar(&skey, "s", "", "specify secret key")
	flag.StringVar(&inFile, "in", "./input.csv", "specify input file")
	flag.StringVar(&excludeLabels, "l", "", "specify label to be excluded from recommendations")
	flag.StringVar(&scoreType, "t", "", "specify score type")
	flag.StringVar(&profile, "p", "", "specify discovery profile")
	flag.IntVar(&limit, "n", 0, "specify the number of recommendations (default 10)")
	flag.StringVar(&updatedAt, "u", "", "specify the timestamp for condition \"< updated_at\" in format '2019-01-18T07:00:00")
	flag.BoolVar(&verbose, "v", false, "verbose mode")
	flag.Parse()

	if skey == "" || akey == "" {
		fmt.Println("Incorrect usage, please specify the required parameters")
		flag.PrintDefaults()
		os.Exit(1)
	}
	// ooClientStaging, _ := oo.NewClient(skey, akey, "https://api-staging.ooyala.com", 15)
	ooClient, _ := oo.NewClient(skey, akey, oo.BacklotDefaultEndpoint, 15)
	if verbose {
		// ooClientStaging.SetLogOut(os.Stdout)
		ooClient.SetLogOut(os.Stdout)
	}

	v := url.Values{}
	if excludeLabels != "" {
		v.Add("exclude_labels", excludeLabels)
	}
	if scoreType != "" {
		v.Add("score_type", scoreType)
	}
	if profile != "" {
		v.Add("discovery_profile_id", profile)
	}
	if limit != 0 {
		v.Add("limit", strconv.Itoa(limit))
	}
	if updatedAt != "" {
		val := fmt.Sprintf("updated_at<'%v'", updatedAt)
		v.Add("where", val)
	}

	targets, err := loadDataFromCSV(inFile)
	if err != nil {
		log.Fatal(err)
	}

	// pairs, err := getPairs(targets, ooClientStaging, v)
	pairs, err := getPairs(targets, ooClient, v)
	if err != nil {
		log.Fatal(err)
	}

	aggrFile, err := os.Create("./result_aggregate.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer aggrFile.Close()

	if err := pairs.renderAggregateResult(aggrFile); err != nil {
		log.Fatal(err)
	}

	resultFile, err := os.Create("./result.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer aggrFile.Close()

	fmt.Fprintln(resultFile, v)
	if err := pairs.renderCommonResult(resultFile, limit); err != nil {
		log.Fatal(err)
	}

}
