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

// Clent holds secret key, api key, basic url path and expire window in hours
// Client is needed to make basic requests to Ooayla APIs
type Client struct {
	// Secret is the secret key for Ooyala account
	Secret string
	// Api is the api key for Ooyala account
	Api string
	// Url is the root url for making requests:
	// For example https://api.ooyala.com for Backlot REST api
	Url *url.URL
	// Delta is the number of hours the request should stay valid
	Delta int
}

// NewClient returns the pointer to the new instance of the client
func NewClient(secret, api, root string, delta int) (*Client, error) {
	client := &Client{Secret: secret, Api: api, Delta: delta}
	url, err := url.Parse(root)
	if err != nil {
		return nil, err
	}
	client.Url = url
	return client, nil
}

// Get performs basic Get request to Ooayla APIs.
func (c Client) Do(method, path string) (*http.Response, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}

	u = c.Url.ResolveReference(u)

	// fmt.Println(u.Host, u.Scheme)

	req, err := http.NewRequest(method, u.String(), nil) // <= stringer!!!!
	if err != nil {
		return nil, err
	}

	// req, err := c.NewRequest(method, path, nil) // <= stringer!!!!
	// if err != nil {
	//   return nil, err
	// }

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (c Client) NewRequest(method, path string, body io.Reader) (*http.Request, error) {
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	// fmt.Println("PATH: ", u.Path)
	q := u.Query()
	q.Set("api_key", c.Api)
	q.Set("expires", c.Expires())

	u.RawQuery = q.Encode()
	sig, err := c.Sign(method, u, body)
	if err != nil {
		return nil, err
	}
	q.Set("signature", sig)

	u.RawQuery = q.Encode()
	u = c.Url.ResolveReference(u)

	req, err := http.NewRequest(method, u.String(), body) // <= stringer!!!!
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c Client) Expires() string {
	// Get the expires value by adding c.Delta to the current time
	timestamp := time.Now().Add(time.Hour * time.Duration(c.Delta)).Unix()
	expires := strconv.FormatInt(timestamp, 10)
	return expires
}

func (c Client) Sign(method string, u *url.URL, body io.Reader) (string, error) {
	// Perform initial string concantination
	sig := c.Secret + strings.ToUpper(method) + u.Path
	// Get the query parameters
	q := u.Query()
	// Sort url parameters by keys alphabetically and contaninate them like a=1b=2c=3
	var keys []string
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		sig += k + "=" + q[k][0] // might be issue with [0]
	}
	// Convert body to strings
	if body != nil {
		var buf bytes.Buffer
		buf.ReadFrom(body)
		sig += buf.String()
	}
	// Adding body, generate a SHA-256 digest in base64 and truncate the string to 43 characters
	// sig += buf.String()
	sum := sha256.Sum256([]byte(sig))
	sig = string(base64.StdEncoding.EncodeToString(sum[:]))[:43]
	return sig, nil
}
