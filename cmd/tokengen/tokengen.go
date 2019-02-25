package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/dimdiden/oo"
)

func main() {
	secret := flag.String("s", "", "specify secret key")
	api := flag.String("a", "", "specify api key")
	expires := flag.String("t", "", "specify expires value")
	embed_code := flag.String("e", "", "specify embed code")

	flag.Parse()

	if *expires == "" || *secret == "" || *api == "" || *embed_code == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Common part
	pcode := strings.Split(*api, ".")[0]
	path := "/sas/embed_token/" + pcode + "/" + *embed_code
	path = path + "?override_syndication_group=override_synd_groups_in_backlot"

	ooClient, _ := oo.NewClient(*secret, *api, "//player.ooyala.com", 15)

	token, err := ooClient.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(token.URL)
}
