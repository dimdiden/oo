package oo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
)

// Asset corresponds to Ooyala entity.
type Asset struct {
	Name             string           `json:"name"`
	FileName         string           `json:"original_file_name"`
	EmbedCode        string           `json:"embed_code"`
	AssetType        string           `json:"asset_type"`
	TimeRestrictions TimeRestrictions `json:"time_restrictions"`
	CreatedAt        string           `json:"created_at"`
	UpdatedAt        string           `json:"updated_at"`
	// file is used in upload processes
	file *os.File
	// chunksize is needed in upload processes
	chunksize int
}

// TimeRestrictions is used to set up asset availability by the time
type TimeRestrictions struct {
	Type      string `json:"type"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// CreateAsset sends a POST request creating a new asset in Ooyala account.
// It assigns a videofile and chunksize to itself which is needed in upload actions.
func (c *Client) CreateAsset(file *os.File, name string, chunksize int) (*Asset, error) {
	asset := &Asset{
		file:      file,
		Name:      name,
		chunksize: chunksize,
	}
	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	asset.FileName = stat.Name()
	if asset.Name == "" {
		asset.Name = asset.FileName
	}
	filesize := strconv.FormatInt(stat.Size(), 10)
	// Prepare the request body
	body := fmt.Sprintf(`{"name": "%v",
  "file_name": "%v",
  "asset_type": "video",
  "file_size": "%v",
	"chunk_size": "%v"}`, asset.Name, asset.FileName, filesize, asset.chunksize)

	// Create asset in Backlot
	response, err := c.Post("/v2/assets", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if err := checkServiceError(response, http.StatusOK); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(response.Body)
	if err := dec.Decode(&asset); err != nil {
		return nil, err
	}

	return asset, nil
}

// ReplaceAsset sends a POST request preparing an asset for replacement in Ooyala account.
// It assigns a videofile and chunksize to itself which is needed in upload actions.
func (c *Client) ReplaceAsset(file *os.File, chunksize int, embedCode string) (*Asset, error) {
	asset := &Asset{
		EmbedCode: embedCode,
		file:      file,
		chunksize: chunksize,
	}

	stat, err := file.Stat()
	if err != nil {
		return asset, err
	}
	filesize := strconv.FormatInt(stat.Size(), 10)

	body := fmt.Sprintf(`{"file_size": "%v", "chunk_size": "%v"}`, filesize, asset.chunksize)
	response, err := c.Post("/v2/assets/"+asset.EmbedCode+"/replacement", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if err := checkServiceError(response, http.StatusOK); err != nil {
		return nil, err
	}

	dec := json.NewDecoder(response.Body)
	if err := dec.Decode(&asset); err != nil {
		return nil, err
	}

	return asset, nil
}

// GetAssets retreives an asset list from the Ooyala Account
func (c Client) GetAssets() ([]Asset, error) {
	type data struct {
		Assets []Asset `json:"items"`
	}
	var d data

	r, err := c.Get("/v2/assets")
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	// Read the request body
	if err := decoder.Decode(&d); err != nil {
		return nil, err
	}
	return d.Assets, nil
}

// GetAsset retreives an asset by the embedcode from the Ooyala Account
func (c Client) GetAsset(ec string) (*Asset, error) {
	var asset Asset
	r, err := c.Get("/v2/assets/" + ec)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(r.Body)
	defer r.Body.Close()
	// Read the request body
	if err := decoder.Decode(&asset); err != nil {
		return nil, err
	}
	return &asset, nil
}
