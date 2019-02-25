package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/dimdiden/oo"
)

type Channel struct {
	Name string
}

func main() {
	// Flag block
	secret := flag.String("s", "", "specify secret key")
	api := flag.String("a", "", "specify api key")
	channel := flag.String("c", "", "specify channel id for renaming")
	name := flag.String("n", "", "specify the new channel name")
	verbose := flag.Bool("v", false, "verbose mode")
	flag.Parse()

	if *secret == "" || *api == "" || *channel == "" || *name == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	if strings.ContainsAny(*name, " ") {
		log.Fatal("Only numbers, characters, and underscores allowed for the chanel name")
	}

	ooClient, _ := oo.NewClient(*secret, *api, oo.LiveEndpoint, 15)
	if *verbose {
		ooClient.SetLogOut(os.Stdout)
	}

	body := fmt.Sprintf(`{"name": "%s"}`, *name)
	response, err := ooClient.Patch("/v2/channels/"+*channel, strings.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}
	result, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Fatalf("Error: [%v] %v", response.StatusCode, string(result))
	}
	// Parse data to Channel struct
	var renamedChannel Channel
	err = json.Unmarshal(result, &renamedChannel)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Channel %s has been renamed to %s\n", *channel, renamedChannel.Name)
}
