package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"text/tabwriter"

	"github.com/dimdiden/oo"
)

const shortForm = "2006-Jan-02"

type LogItem struct {
	User         string `json:"user"`
	CreationTime string `json:"creation_time"`
	EmbedCode    string `json:"embed_code"`
	ErrorMessage string `json:"error_message"`
	FileType     string `json:"file_type"`
	Status       string `json:"status"`
	ID           string `json:"id"`
	FileID       string `json:"file_id"`
	FileName     string `json:"file_name"`
}

func (li LogItem) String() string {
	str := fmt.Sprintf("%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\t%v\n",
		li.User, li.CreationTime, li.EmbedCode, li.ErrorMessage, li.FileName, li.Status, li.ID, li.FileID, li.FileName)
	return str
}

func main() {
	// Flag block
	secret := flag.String("s", "", "specify secret key")
	api := flag.String("a", "", "specify api key")
	search := flag.String("n", "", "specify name of uploaded file")
	verbose := flag.Bool("v", false, "verbose mode")
	flag.Parse()

	if *secret == "" || *api == "" || *search == "" {
		fmt.Println("Incorrect usage, please specify the required parameters")
		flag.PrintDefaults()
		os.Exit(1)
	}

	ooClient, _ := oo.NewClient(*secret, *api, oo.BacklotDefaultEndpoint, 15)
	if *verbose {
		ooClient.SetLogOut(os.Stdout)
	}

	response, err := ooClient.Get("/v2/ingestion/logs?file_name=" + *search)
	if err != nil {
		log.Fatal(err)
	}
	res, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if response.StatusCode != 200 {
		log.Fatalf("Error: [%v] %v", response.StatusCode, string(res))
	}

	type data struct {
		Results []LogItem
	}

	var d data
	err = json.Unmarshal(res, &d)
	if err != nil {
		log.Fatal(err)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 0, ' ', tabwriter.Debug)
	fmt.Fprint(w, "User\tCreationTime\tEmbedCode\tErrorMessage\tFileType\tStatus\tID\tFileID\tFileName\n")
	for _, li := range d.Results {
		fmt.Fprint(w, li)
	}
	w.Flush()
}

// Check this one
// /v2/ingestion/logs?period=start=2018-10-10;end=2018-10-20

// {
//  "filter_params": {
//   "period": "start=2018-10-10T00:00:00+00:00;end=2018-10-21T00:00:00+00:00"
//  },
//  "next_page_url": "https://api.ooyala.com/v2/ingestion/logs?api_key=F0OWwyOo4T6H27--qop_GKi9c_ea.zjuQK&expires=1539961964&paging_state=ABQABAABtMsAAAoyMDE4LTEwLTE5AABSAAgAAAFmi0GNSAAAE2luZy5nYWxheGlhQGN2bWMuZXMAACRhNWJiYmVmOS1lZjM5LTRmYTYtOGJlNi02ZTdiMDU4MWY2ODMAAAdmaWxlX2lkAH___5s%3D&period=start%3D2018-10-10T00%3A00%3A00%2B00%3A00%3Bend%3D2018-10-21T00%3A00%3A00%2B00%3A00&signature=xz7Wj%2Bb2FAFCd0CF7s09DhQ6BLhsUTbKkgwsEbLyrC8",
//  "results": [   {
//    "creation_time": "2018-10-19T11:21:10+00:00",
//    "file_type": "video",
//    "file_name": "P0300110-000801.mp4",
//    "status": "metadata_waiting",
//    "embed_code": null,
//    "file_id": "456d41c502d145383c6cfe3006d63190",
//    "user": "gthau_dalet_cvmc",
//    "id": null,
//    "error_message": null
//   },   {
//    "creation_time": "2018-10-19T11:21:09+00:00",
//    "file_type": "thumbnail",
//    "file_name": "APUNTDEFAULT.jpg",
//    "status": "done",
//    "embed_code": "JueGVnZzE6OLZlWNlhNGV9Fikhbu5vKY",
//    "file_id": "3479eec45bdee0d673c7522f5a934d59",
//    "user": "gthau_dalet_cvmc",
//    "id": "9YGlTUIeo0qcayy_EMVXFJHly50=",
//    "error_message": null
//   },   {
//    "creation_time": "2018-10-19T11:21:06+00:00",
//    "file_type": "manifest",
//    "file_name": "P0300110-000801.xml",
//    "status": "done",
//    "embed_code": null,
//    "file_id": "7d105a36a515e0886adcaaf9ab407ade",
//    "user": "gthau_dalet_cvmc",
//    "id": "unavailable",
//    "error_message": null
//   },   {
//    "creation_time": "2018-10-19T11:20:49+00:00",
//    "file_type": "manifest",
//    "file_name": "P0300110-000801.xml",
//    "status": "received",
//    "embed_code": null,
//    "file_id": "7d105a36a515e0886adcaaf9ab407ade",
//    "user": "gthau_dalet_cvmc",
//    "id": "unavailable",
//    "error_message": null
//   },
