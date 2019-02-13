package oo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
)

type Asset struct {
	File             *os.File
	Name             string           `json:"name"`
	FileName         string           `json:"original_file_name"`
	EmbedCode        string           `json:"embed_code"`
	AssetType        string           `json:"asset_type"`
	TimeRestrictions TimeRestrictions `json:"time_restrictions"`
	CreatedAt        string           `json:"created_at"`
	UpdatedAt        string           `json:"updated_at"`
	chunkSize        int
	uploadMap        map[string][]byte
}

type TimeRestrictions struct {
	Type      string `json:"type"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

// type Content struct {
// 	File      *os.File
// 	chunkSize int
// 	Name      string
// }

type Uploader interface {
	Upload(m map[string][]byte) error
}

type concUploader struct{}

func NewConcUploader() Uploader {
	return &concUploader{}
}

func (u *concUploader) Upload(m map[string][]byte) error {
	var wg sync.WaitGroup

	errs := make(chan error, 1)
	done := make(chan bool, 1)
	// perfrom uploading for an each of the chunks
	for url, chunk := range m {
		// increment wait counter
		wg.Add(1)

		contentLength := int64(len(chunk))

		reader := bytes.NewReader(chunk)
		// chunk upload in a separate goroutine
		go func(reader io.Reader, contentLength int64, url string) {
			UploadChunk(reader, contentLength, url, &wg, errs)
		}(reader, contentLength, url)
	}
	// wait until the last goutine in waitgroup is finished and close the finished channel
	go func() {
		wg.Wait()
		close(done)
	}()
	// This select will block until one of the two channels returns a value.
	select {
	case <-done:
	case err := <-errs:
		return err
	}
	return nil
}

func (c Client) CreateAsset(file *os.File, name string, chunksize int) (*Asset, error) {
	var asset Asset
	asset.File = file
	asset.Name = name
	asset.chunkSize = chunksize

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	asset.FileName = stat.Name()
	if name == "" {
		asset.Name = asset.FileName
	}

	// asset.chunkSize = chunksize
	filesize := strconv.FormatInt(stat.Size(), 10)
	// Prepare the request body
	body := fmt.Sprintf(`{"name": "%v",
  "file_name": "%v",
  "asset_type": "video",
  "file_size": "%v",
	"chunk_size": "%v"}`, asset.Name, asset.FileName, filesize, asset.chunkSize)
	// Create asset in Backlot
	response, err := c.Post("/v2/assets", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	result, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed: [%v] [%v]", response.StatusCode, string(result))
	}
	// Receive the json and fetch the embed_code
	err = json.Unmarshal(result, &asset)
	if err != nil {
		return nil, err
	}

	// if err := c.getUploadMap(&asset); err != nil {
	if err := c.getUploadMap(&asset, false); err != nil {
		return &asset, err
	}
	return &asset, nil
}

func (c Client) ReplaceAsset(embedCode string, file *os.File, chunksize int) (*Asset, error) {
	var asset Asset
	asset.EmbedCode = embedCode
	asset.File = file
	asset.chunkSize = chunksize

	stat, err := file.Stat()
	if err != nil {
		return &asset, err
	}
	filesize := strconv.FormatInt(stat.Size(), 10)
	// Prepare the request body
	body := fmt.Sprintf(`{"file_size": "%v", "chunk_size": "%v"}`, filesize, asset.chunkSize)

	response, err := c.Post("/v2/assets/"+asset.EmbedCode+"/replacement", strings.NewReader(body))
	if err != nil {
		return &asset, err
	}
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return &asset, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return &asset, fmt.Errorf("request failed: [%v] [%v]", response.StatusCode, string(result))
	}

	// if err := c.getReplaceMap(&asset); err != nil {
	if err := c.getUploadMap(&asset, true); err != nil {
		return &asset, err
	}
	return &asset, nil
}

func (c Client) getUploadMap(asset *Asset, replace bool) error {
	m := make(map[string][]byte)

	q := "/v2/assets/" + asset.EmbedCode + "/uploading_urls"
	if replace {
		q = "/v2/assets/" + asset.EmbedCode + "/replacement/uploading_urls"
	}
	response, err := c.Get(q)
	if err != nil {
		return err
	}

	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: [%v] [%v]", response.StatusCode, string(result))
	}

	var urls []string
	err = json.Unmarshal(result, &urls)
	if err != nil {
		return err
	}

	chunks, err := asset.splitFile()
	if err != nil {
		return err
	}

	for i, chunk := range chunks {
		m[urls[i]] = chunk
	}

	asset.uploadMap = m
	return nil
}

func (c Client) UploadContent(file *os.File, name string, chunksize int, uploader Uploader) (*Asset, error) {

	defer file.Close()

	asset, err := c.CreateAsset(file, name, chunksize)
	if err != nil {
		return nil, err
	}

	if err := uploader.Upload(asset.uploadMap); err != nil {
		return asset, err
	}

	if err := c.TriggerNewProcessing(asset); err != nil {
		return asset, err
	}
	return asset, nil
}

func (c Client) ReplaceContent(embedCode string, file *os.File, chunksize int, uploader Uploader) error {

	defer file.Close()

	asset, err := c.ReplaceAsset(embedCode, file, chunksize)
	if err != nil {
		return err
	}

	if err := uploader.Upload(asset.uploadMap); err != nil {
		return err
	}

	if err := c.TriggerReplaceProcessing(asset); err != nil {
		return err
	}
	return nil
}

func (c Client) TriggerNewProcessing(asset *Asset) error {
	if err := c.triggerProcessing(asset, false); err != nil {
		return err
	}
	return nil
}

func (c Client) TriggerReplaceProcessing(asset *Asset) error {
	if err := c.triggerProcessing(asset, true); err != nil {
		return err
	}
	return nil
}

func (c Client) triggerProcessing(asset *Asset, replace bool) error {
	q := "/v2/assets/" + asset.EmbedCode + "/upload_status"
	if replace {
		q = "/v2/assets/" + asset.EmbedCode + "/replacement/upload_status"
	}
	response, err := c.Put(q, strings.NewReader(`{"status":"uploaded"}`))
	if err != nil {
		return err
	}
	result, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("request failed: [%v] [%v]", response.StatusCode, string(result))
	}
	return nil
}

func (asset *Asset) splitFile() ([][]byte, error) {
	// Transfer data from file to buffer
	buf, err := ioutil.ReadAll(asset.File)
	if err != nil {
		return nil, err
	}
	// make slice of chunks by deviding buffer
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/asset.chunkSize+1)
	for len(buf) >= asset.chunkSize {
		chunk, buf = buf[:asset.chunkSize], buf[asset.chunkSize:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:len(buf)])
	}
	return chunks, nil
}

func UploadChunk(reader io.Reader, contentLength int64, url string, wg *sync.WaitGroup, errs chan error) {
	// decrement counter after this function is finished
	defer wg.Done()

	request, err := http.NewRequest("PUT", url, reader)
	if err != nil {
		errs <- err
		return
	}
	request.ContentLength = contentLength

	// send a request, receive response
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		errs <- err
		return
	}
	// there is an issue if StatusCode is not 204
	if response.StatusCode != 204 {
		message, _ := ioutil.ReadAll(response.Body)
		err := fmt.Errorf("Error: [%v] %v", response.StatusCode, string(message))
		defer response.Body.Close()
		errs <- err
		return
	}
}

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
