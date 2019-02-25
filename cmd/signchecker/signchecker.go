package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/dimdiden/oo"
)

func main() {
	// Flag block
	skey := flag.String("s", "", "specify secret key")
	akey := flag.String("a", "", "specify api key")
	query := flag.String("q", "", "specify url query needed to be signed")
	method := flag.String("m", "", "specify the http method for the request")
	body := flag.String("b", "", "specify either JSON or the path to the binary file")
	flag.Parse()

	if *skey == "" || *akey == "" || *query == "" || *method == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Parse and check the provided query
	u, err := url.Parse(*query)
	if err != nil {
		log.Fatal(err)
	}
	q := u.Query()
	if val, ok := q["api_key"]; ok && len(val) == 1 && val[0] != *akey {
		log.Fatal("api_key value in query differs from specified")
	}
	if _, ok := q["signature"]; ok {
		log.Fatal("signature is already present in the provided query")
	}
	// Create the new ooyala client
	ooClient, err := oo.NewClient(*skey, *akey, "", 15)
	if err != nil {
		log.Fatal(err)
	}
	// Check if body is a path to a file or JSON string
	// and convert this to string
	b, err := checkBody(*body)
	if err != nil {
		log.Fatal(err)
	}

	r, err := http.NewRequest(*method, *query, b)
	if err != nil {
		log.Fatal(err)
	}
	// Signing the query
	oo.SignRequest(r, *ooClient)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("=========================")
	fmt.Println("SIGNED REQUEST: ", r.URL.String())
}

// Check if body is a path to a file or JSON string
// and convert this to string. Otherwise return error
func checkBody(b string) (io.Reader, error) {
	if b == "" {
		return nil, nil
	}

	var reader io.Reader
	if f, err := os.Open(b); err == nil {
		defer f.Close()
		b, _ := ioutil.ReadAll(f)
		reader = bytes.NewBuffer(b)
	} else if isJSON(b) {
		reader = strings.NewReader(b)
	} else {
		return nil, fmt.Errorf("specified body neither JSON string nor a path to the existing file")
	}
	return reader, nil
}

// Check if string is JSON
func isJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}
