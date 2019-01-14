package oo

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// BacklotDefaultEndpoint is the default endoint for the requests to Backlot REST API
const BacklotDefaultEndpoint = "https://api.ooyala.com"

// BacklotCDNEndpoint is high performance API endpoint that caches repeated requests to Backlot REST API
const BacklotCDNEndpoint = "https://cdn-api.ooyala.com"

// LiveEndpoint is the default endoint for the requests to Live API
const LiveEndpoint = "https://live.ooyala.com"

// RightsLockerEndpoint is the default endoint for the requests to Rights Locker API
const RightsLockerEndpoint = "https://rl.ooyala.com"

// Client holds secret key, api key, basic url path and expire window in hours
// Client is needed to make basic requests to Ooayla APIs
type Client struct {
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
	// out is used for logging requests
	out io.Writer
}

type Apier interface {
	Get(path string) (*http.Response, error)
	Post(path string, body io.Reader) (*http.Response, error)
	Put(path string, body io.Reader) (*http.Response, error)
	Patch(path string, body io.Reader) (*http.Response, error)
	Delete(path string) (*http.Response, error)
}

// NewClient returns the pointer to the new instance of the Api object
func NewClient(skey, akey, root string, delta int) (*Client, error) {
	if strings.Contains(skey, ".") && !strings.Contains(akey, ".") {
		return nil, errors.New("incorrect order of keys, first should be secrect key, then api key")
	}
	api := &Client{Skey: skey, Akey: akey, Delta: delta}
	u, err := url.Parse(root)
	if err != nil {
		return nil, err
	}
	api.RootUrl = u

	api.out = ioutil.Discard
	return api, nil
}

// SetLogOut sets the writer for the logs
func (c *Client) SetLogOut(out io.Writer) {
	c.out = out
}

// Get makes basic Get request to Ooayla APIs and returns http.Response
func (c Client) Get(path string) (*http.Response, error) {
	res, err := c.sendRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Post makes basic Post request to Ooayla APIs and returns http.Response
func (c Client) Post(path string, body io.Reader) (*http.Response, error) {
	res, err := c.sendRequest(http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Put makes basic Put request to Ooayla APIs and returns http.Response
func (c Client) Put(path string, body io.Reader) (*http.Response, error) {
	res, err := c.sendRequest(http.MethodPut, path, body)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Patch makes basic Patch request to Ooayla APIs and returns http.Response
func (c Client) Patch(path string, body io.Reader) (*http.Response, error) {
	res, err := c.sendRequest(http.MethodPatch, path, body)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// Delete makes basic Delete request to Ooayla APIs and returns http.Response
func (c Client) Delete(path string) (*http.Response, error) {
	res, err := c.sendRequest(http.MethodDelete, path, nil)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func (c Client) sendRequest(method, path string, body io.Reader) (*http.Response, error) {
	req, err := c.NewRequest(method, path, body)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

// NewRequest takes http method in lower or upper case, url query with parameters, and
// body as any Reader and returns *http.Request ready for sending by the http client
func (c Client) NewRequest(method, rawurl string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(strings.ToUpper(method), rawurl, body)
	if err != nil {
		return nil, err
	}
	SignRequest(req, c)

	req.URL = c.RootUrl.ResolveReference(req.URL)
	c.out.Write([]byte("REQUEST: " + req.Method + " " + req.URL.String() + "\n"))
	return req, nil
}

// SignRequest gets a request, adds api_key, expires values,
// generate and adds signature for the query in request.URL
func SignRequest(r *http.Request, c Client) {
	// perform initial string concantination
	sig := c.Skey + strings.ToUpper(r.Method) + r.URL.Path
	// get the query parameters and add api and expires there if they are absent
	q := r.URL.Query()
	if _, ok := q["api_key"]; !ok {
		q.Set("api_key", c.Akey)
	}
	if _, ok := q["expires"]; !ok {
		q.Set("expires", c.expires())
	}
	// sort url parameters by keys alphabetically and contaninate them like a=1b=2c=3
	sig += fmtKeys(q)
	// convert body to string
	if r.Body != nil {
		b, _ := ioutil.ReadAll(r.Body)
		r.ContentLength = int64(len(b))
		sig += string(b)
		// return reader with content back to Body
		r.Body = ioutil.NopCloser(bytes.NewBuffer(b))
	}
	// Generate a SHA-256 digest in base64 and truncate the string to 43 characters
	sum := sha256.Sum256([]byte(sig))
	sig = string(base64.StdEncoding.EncodeToString(sum[:]))[:43]
	// Adding signature to the query parameters and encode them
	q.Set("signature", sig)
	// Reassing query with all parameters to the url again
	r.URL.RawQuery = q.Encode()
}

// Expires genrates expires value based on Delta value
func (c Client) expires() string {
	// Generate the expires value by adding c.Delta to the current time
	timestamp := time.Now().Add(time.Hour * time.Duration(c.Delta)).Unix()
	expires := strconv.FormatInt(timestamp, 10)
	return expires
}

// fmtKeys sort url parameters by keys alphabetically and contaninate them like a=1b=2c=3
func fmtKeys(q url.Values) string {
	var result string
	var keys []string
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		result += k + "=" + q[k][0] // might be issue with [0]
	}
	return result
}
