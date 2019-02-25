package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/dimdiden/oo"
)

func main() {
	// Flag block
	secret := flag.String("s", "", "specify secret key")
	api := flag.String("a", "", "specify api key")
	verbose := flag.Bool("v", false, "verbose mode")
	flag.Parse()

	if *secret == "" || *api == "" {
		fmt.Println("Incorrect usage, please specify the required parameters")
		flag.PrintDefaults()
		os.Exit(1)
	}

	ooClient, _ := oo.NewClient(*secret, *api, oo.BacklotDefaultEndpoint, 15)
	if *verbose {
		ooClient.SetLogOut(os.Stdout)
	}

	for {
		response, err := ooClient.Get(`/v2/assets?where=status='live'+AND+metadata.video='test'+AND+updated_at>'2018-12-05T09:00:00Z'`)
		if err != nil {
			log.Fatal(err)
		}
		defer response.Body.Close()

		if response.StatusCode != http.StatusOK {
			result, _ := ioutil.ReadAll(response.Body)
			log.Fatalf("request failed: [%v] [%v]", response.StatusCode, string(result))
		}

		type data struct {
			Assets []oo.Asset `json:"items"`
		}
		var d data

		decoder := json.NewDecoder(response.Body)
		if err := decoder.Decode(&d); err != nil {
			log.Fatal(err)
		}

		if len(d.Assets) > 0 {
			log.Println(d.Assets)
			break
		}

		credits, err := strconv.Atoi(response.Header.Get("X-RateLimit-Credits"))
		if err != nil {
			log.Fatal(err)
		}

		timer := time.NewTimer(1 * time.Minute)
		if credits < 100 {
			fmt.Println("Only 100 credits left. Waiting for 1 min...")
			<-timer.C
		}
	}
}
