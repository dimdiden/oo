package oo

// Clent holds secret key, api key, basic url path and expire window in hours
// Client is needed to make basic requests to Ooayla APIs
type Client struct {
	// Sk is the secret key for Ooyala account
	Sk string
	// Ak is the api key for Ooyala account
	Ak string
	// Url is the root url for making requests:
	// For example https://api.ooyala.com for Backlot REST api
	Url string
	// Exp is the number of hours the request should stay valid
	Exp int
}
