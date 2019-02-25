package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"time"

	"github.com/dimdiden/oo"
)

const shortForm = "2006-Jan-02"

type Program struct {
	Id        string
	Name      string
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Channel   string `json:"channel_id"`
	EmbedCode string `json:"embed_code"`
}

func (p Program) String() string {
	str := fmt.Sprintf("Id: %s\nName: %s\nStartTime: %s\nEndTime: %s\nChannel: %s\nEmbedCode: %s\n",
		p.Id, p.Name, p.StartTime, p.EndTime, p.Channel, p.EmbedCode)
	return str
}

type Item struct {
	Program Program
}

type Events struct {
	Items []Item
}

func main() {
	// Flag block
	secret := flag.String("s", "", "specify secret key")
	api := flag.String("a", "", "specify api key")
	search := flag.String("n", "", "specify embed_code or a name of the event")
	stime := flag.String("st", "", "specify start time in format 2018-May-18")
	etime := flag.String("et", "", "specify end time in format 2018-May-18")
	verbose := flag.Bool("v", false, "verbose mode")
	flag.Parse()

	if *secret == "" || *api == "" || *search == "" || *stime == "" || *etime == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	_, err := time.Parse(shortForm, *stime)
	if err != nil {
		log.Fatal(err)
	}
	_, err = time.Parse(shortForm, *etime)
	if err != nil {
		log.Fatal(err)
	}

	ooClient, _ := oo.NewClient(*secret, *api, oo.LiveEndpoint, 15)
	if *verbose {
		ooClient.SetLogOut(os.Stdout)
	}

	response, err := ooClient.Get("/v3/events?exclude=attr&from_date=" + *stime + "&to_date=" + *etime)
	if err != nil {
		log.Fatal(err)
	}
	res, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Fatalf("Error: [%v] %v", response.StatusCode, string(res))
	}

	var events Events
	err = json.Unmarshal(res, &events)
	if err != nil {
		log.Fatal(err)
	}

	// Search section
	var programs []Program

	for _, i := range events.Items {
		if i.Program.EmbedCode == *search || strings.Contains(i.Program.Name, *search) {
			programs = append(programs, i.Program)
		}
	}

	if len(programs) > 0 {
		fmt.Println("Events have been found")
		fmt.Println("======================")
		for _, p := range programs {
			fmt.Println(p)
		}
		os.Exit(0)
	}
	fmt.Println("No events found")
}
