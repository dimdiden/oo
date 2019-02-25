package oo

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
)

// Similars represents a set of assets returned from Ooyala
// as a recommendation to the specific asset
type Similars struct {
	Assets []Similar `json:"results"`
}

// Len, Swap, Less are needed to satisfy Sort interface
// Sort alphabetically
func (s Similars) Len() int {
	return len(s.Assets)
}
func (s Similars) Swap(i, j int) {
	s.Assets[i], s.Assets[j] = s.Assets[j], s.Assets[i]
}
func (s Similars) Less(i, j int) bool {
	return strings.ToLower(s.Assets[i].EmbedCode) < strings.ToLower(s.Assets[j].EmbedCode)
}

// Similar is a single recommendation for a particular asset
type Similar struct {
	Asset
	Reason     string `json:"reason"`
	BucketInfo string `json:"bucket_info"`
}

// GetNewSimilars fetches list of the recommendations for the given embedCode.
func GetNewSimilars(ci ClientInterface, embedCode string, values url.Values) (*Similars, error) {
	query := fmt.Sprintf("/v2/discover/similar/assets/%v?%v", embedCode, values.Encode())
	response, err := ci.Get(query)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		result, _ := ioutil.ReadAll(response.Body)
		log.Fatalf("request failed: [%v] [%v]", response.StatusCode, string(result))
	}

	var similars Similars
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&similars); err != nil {
		return nil, err
	}
	return &similars, nil
}

// Deflate decodes BucketInfo from the response
func (s Similar) Deflate() (string, error) {
	d := struct {
		Encoded string `json:"encoded"`
	}{}

	decoder := json.NewDecoder(strings.NewReader(s.BucketInfo[1:]))

	err := decoder.Decode(&d)
	if err != nil {
		return "", err
	}

	str := strings.Replace(d.Encoded, "\\n", "\n", -1)

	decoded, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return "", err
	}

	b := bytes.NewReader(decoded)
	r, err := zlib.NewReader(b)
	if err != nil {
		return "", err
	}

	buf := &bytes.Buffer{}
	buf.ReadFrom(r)
	r.Close()

	return buf.String(), nil
}
