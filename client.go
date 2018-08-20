package oo

import "net/url"

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
	// Exp is the number of hours the request should stay valid
	Expires int
}

// NewClient returns the pointer to the new instance of the client
func NewClient(secret, api, root string, expires int) (*Client, error) {
	client := &Client{Secret: secret, Api: api, Expires: expires}
	url, err := url.Parse(root)
	if err != nil {
		return nil, err
	}
	client.Url = url
	return client, nil
}
