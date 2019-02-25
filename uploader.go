package oo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Uploader provides features to upload and replace assets
type Uploader struct {
	client      *Client
	replacement bool
	wg          *sync.WaitGroup
	requests    chan *http.Request
	// startFunc, filterFunc, deferFunc are used to hook into a process of chunks upload
	// can be usefull to visualize this process
	startFunc  func() error
	filterFunc func(*http.Request) (*http.Request, error)
	deferFunc  func()
}

// NewUploader returns a new Uploader with a given client
func NewUploader(client *Client) *Uploader {
	return &Uploader{
		client:      client,
		replacement: false,
		wg:          &sync.WaitGroup{},
		requests:    make(chan *http.Request),
	}
}

// SetStartFunc sets a function which will be executed before a process of chunks upload
func (u *Uploader) SetStartFunc(f func() error) {
	u.startFunc = f
}

// SetFilterFunc sets a function which is used to modify requests with chunks before seding them.
// Can be usefull to visualize this process
func (u *Uploader) SetFilterFunc(f func(*http.Request) (*http.Request, error)) {
	u.filterFunc = f
}

// SetDeferFunc sets a function which will be executed after a process of chunks upload
func (u *Uploader) SetDeferFunc(f func()) {
	u.deferFunc = f
}

// CreateUploadAsset creates an asset in Ooyala account,
// uploads file for this asset and triggers the transcoding job
func (u *Uploader) CreateUploadAsset(file *os.File, name string, chunksize int) (*Asset, error) {
	u.replacement = false
	asset, err := u.client.CreateAsset(file, name, chunksize)
	if err != nil {
		return nil, fmt.Errorf("couldn't create asset: %v", err)
	}

	if err := u.Upload(asset); err != nil {
		return nil, fmt.Errorf("couldn't upload asset: %v", err)
	}

	if err := u.TriggerProcessing(asset.EmbedCode); err != nil {
		return nil, fmt.Errorf("couldn't trigger asset processing: %v", err)
	}

	return asset, nil
}

// ReplaceUploadAsset prepares an asset for replacement in Ooyala account,
// uploads file for this asset and triggers the transcoding job
func (u *Uploader) ReplaceUploadAsset(file *os.File, chunksize int, embedCode string) (*Asset, error) {
	u.replacement = true
	asset, err := u.client.ReplaceAsset(file, chunksize, embedCode)
	if err != nil {
		return nil, fmt.Errorf("couldn't replace asset: %v", err)
	}

	if err := u.Upload(asset); err != nil {
		return nil, fmt.Errorf("couldn't upload asset: %v", err)
	}

	if err := u.TriggerProcessing(asset.EmbedCode); err != nil {
		return nil, fmt.Errorf("couldn't trigger asset processing: %v", err)
	}

	return asset, nil
}

// Upload uploads file for the asset
func (u *Uploader) Upload(asset *Asset) error {
	if u.deferFunc != nil {
		defer u.deferFunc()
	}
	defer asset.file.Close()

	urls, err := u.getURLs(asset.EmbedCode)
	if err != nil {
		return fmt.Errorf("couldn't get uploading urls: %v", err)
	}

	u.wg.Add(len(urls))

	done := make(chan bool, 1)
	errs := make(chan error, 1)

	go u.pushRequests(asset.file, urls, errs)

	go func() {
		u.wg.Wait()
		close(done)
	}()

	if u.startFunc != nil {
		u.startFunc()
	}

	for request := range u.requests {
		go u.uploadChunk(request, errs)
	}

	for {
		select {
		case <-done:
			return nil
		case err := <-errs:
			return err
		}
	}
}

func (u *Uploader) getURLs(embedCode string) ([]*url.URL, error) {
	q := "/v2/assets/" + embedCode + "/uploading_urls"
	if u.replacement {
		q = "/v2/assets/" + embedCode + "/replacement/uploading_urls"
	}
	response, err := u.client.Get(q)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if err := checkServiceError(response, http.StatusOK); err != nil {
		return nil, err
	}

	var rawurls []string
	dec := json.NewDecoder(response.Body)
	if err := dec.Decode(&rawurls); err != nil {
		return nil, err
	}

	var urls []*url.URL
	for _, rawurl := range rawurls {
		url, err := url.Parse(rawurl)
		if err != nil {
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

func (u *Uploader) pushRequests(r io.Reader, urls []*url.URL, errs chan error) {
	defer close(u.requests)

	if len(urls) == 0 {
		errs <- fmt.Errorf("no uploading urls to perform upload")
		return
	}
	if r == nil {
		errs <- fmt.Errorf("no file attached")
		return
	}

	for _, url := range urls {
		chunk, err := getChunk(r, url)
		if err != nil {
			errs <- err
			return
		}
		request, err := http.NewRequest(http.MethodPut, url.String(), chunk)
		if err != nil {
			errs <- err
			return
		}
		request.ContentLength = int64(chunk.Len())
		if u.filterFunc != nil {
			r, err := u.filterFunc(request)
			if err != nil {
				errs <- err
			}
			request = r
		}
		u.requests <- request
	}
}

func getChunk(r io.Reader, url *url.URL) (*bytes.Buffer, error) {
	q := url.Query()
	chunksize := q.Get("filesize")
	i, err := strconv.Atoi(chunksize)
	if err != nil {
		return nil, fmt.Errorf("couldn't get chunksize from url: %v", err)
	}
	chunk := make([]byte, i)
	_, err = r.Read(chunk)
	if err != nil {
		// TODO: check if EOF check is needed
		return nil, fmt.Errorf("couldn't read chunk from reader: %v", err)
	}
	return bytes.NewBuffer(chunk), nil
}

func (u *Uploader) uploadChunk(request *http.Request, errs chan error) {
	defer u.wg.Done()
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		errs <- err
		return
	}
	if err := checkServiceError(response, http.StatusNoContent); err != nil {
		errs <- err
	}
}

// TriggerProcessing starts the transcoding process for an asset by the given embed code
func (u *Uploader) TriggerProcessing(embedCode string) error {
	q := "/v2/assets/" + embedCode + "/upload_status"
	if u.replacement {
		q = "/v2/assets/" + embedCode + "/replacement/upload_status"
	}
	response, err := u.client.Put(q, strings.NewReader(`{"status":"uploaded"}`))
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if err := checkServiceError(response, http.StatusOK); err != nil {
		return err
	}
	return nil
}

// UploadImage uploads a thumbnail image for an asset by the given embed code
func (u *Uploader) UploadImage(file *os.File, embedCode string) error {
	response, err := u.client.Post("/v2/assets/"+embedCode+"/preview_image_files", file)
	if err != nil {
		return err
	}
	if err := checkServiceError(response, http.StatusOK); err != nil {
		return err
	}
	return nil
}
