package oo

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Api holds secret key, api key, basic url path and expire window in hours
// Api is needed to make basic requests to Ooayla APIs
type Api struct {
	// Skey is the secret key for Ooyala account
	Skey string
	// Akey is the api key for Ooyala account
	Akey string
	// RootUrl is the root url for making requests:
	// For example https://api.ooyala.com for Backlot REST api
	RootUrl *url.URL
	// Delta is the number of hours the request should stay valid
	// Required further to generate expires value for a request
	Delta int
}

// NewApi returns the pointer to the new instance of the Api object
func NewApi(skey, akey, root string, delta int) (*Api, error) {
	api := &Api{Skey: skey, Akey: akey, Delta: delta}
	u, err := url.Parse(root)
	if err != nil {
		return nil, err
	}
	api.RootUrl = u
	return api, nil
}

// NewRequest takes http method in lower or upper case, url query with parameters, and
// body as any Reader and returns *http.Request ready for sending by the http client
func (c Api) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	// Convert body to bytes.Buffer for passing it as string to Sign
	var buf bytes.Buffer
	if body != nil {
		buf.ReadFrom(body)
	}
	// Get the signed url.URL
	u, err := c.Sign(method, path, buf.String())
	// Make full url with subpath and query
	u = c.RootUrl.ResolveReference(u)
	// Make a simple request
	req, err := http.NewRequest(method, u.String(), &buf)
	if err != nil {
		return nil, err
	}
	return req, nil
}

// Sign gets a raw url query, adds api_key and expires values,
// generate and adds signatture for the query an returns url.URL adding the new values
func (c Api) Sign(method, path string, body string) (*url.URL, error) {
	// Convert string path to url.URL value
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	// Perform initial string concantination
	sig := c.Skey + strings.ToUpper(method) + u.Path
	// Get the query parameters and add api and expires there if they are absent
	q := u.Query()
	if _, ok := q["api_key"]; !ok {
		q.Set("api_key", c.Akey)
	}
	if _, ok := q["expires"]; !ok {
		q.Set("expires", c.expires())
	}
	// Sort url parameters by keys alphabetically and contaninate them like a=1b=2c=3
	var keys []string
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sig += k + "=" + q[k][0] // might be issue with [0]
	}
	// Adding body, generate a SHA-256 digest in base64 and truncate the string to 43 characters
	sig += body
	sum := sha256.Sum256([]byte(sig))
	sig = string(base64.StdEncoding.EncodeToString(sum[:]))[:43]
	// Adding signature to the query parameters and encode them
	q.Set("signature", sig)
	// Reassing query with all parameters to the url again
	u.RawQuery = q.Encode()
	return u, nil
}

// Expires genrates expires value based on Delta value
func (c Api) expires() string {
	// Generate the expires value by adding c.Delta to the current time
	timestamp := time.Now().Add(time.Hour * time.Duration(c.Delta)).Unix()
	expires := strconv.FormatInt(timestamp, 10)
	return expires
}
