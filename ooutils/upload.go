package ooutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/dimdiden/oo"
	pb "gopkg.in/cheggaaa/pb.v1"
)

type Asset struct {
	Embed_code string
}

// Type image is used in upload/get preview image file for the asset
type Image struct {
	Id  string `json:"id"`
	Url string `json:"url"`
}

// UploadVideo is the main method to upload file. It takes path to a file,
// an instance of the OO object for making queries and max chank for dividing the file
func UploadVideo(a oo.Apier, file, name string, chunklimit int) (*http.Response, error) {
	// get fileinfo to use it for the further queries
	fileInfo, err := getFileStat(file)
	if err != nil {
		return nil, err
	}
	// send request and receive the respose with asset
	asset, err := getAsset(a, fileInfo, name, chunklimit)
	if err != nil {
		return nil, err
	}
	// get the needed urls for uploading
	urls, err := getUploadUrl(a, asset.Embed_code)
	if err != nil {
		return nil, err
	}
	// split file up into chunks based on the limit
	chunks, err := splitFile(file, chunklimit)
	if err != nil {
		return nil, err
	}
	// waitgroup waits for a collection of goroutines to finish
	var wg sync.WaitGroup
	// pool of the progress bars
	pool, err := pb.StartPool()
	if err != nil {
		return nil, err
	}
	// errchannel used to track erorrs in goroutines
	errchannel := make(chan error, 1)
	// finished channel indicates that all goroutines are finished
	finished := make(chan bool, 1)
	// perfrom uploading for an each of the chunks
	for i, b := range chunks {
		// increment wait counter
		wg.Add(1)
		// set up bar for one chunk
		bar := pb.StartNew(len(b)).SetUnits(pb.U_BYTES)
		pool.Add(bar)
		// chunk upload in a separate goroutine
		go uploadChunk(b, bar, urls[i], &wg, errchannel)
	}
	// wait until the last goutine in waitgroup is finished and close the finished channel
	go func() {
		wg.Wait()
		pool.Stop()
		close(finished)
	}()
	// This select will block until one of the two channels returns a value.
	select {
	case <-finished:
	case err := <-errchannel:
		if err != nil {
			return nil, err
		}
	}
	// final query to mark an asset as uploaded
	res, err := a.Put("/v2/assets/"+asset.Embed_code+"/upload_status", strings.NewReader(`{"status":"uploaded"}`))
	if err != nil {
		return nil, err
	}
	return res, nil
}

// getAsset performes the needed request and receives the embed_code for an asset
func getAsset(a oo.Apier, stat os.FileInfo, name string, chunklimit int) (Asset, error) {
	filename := stat.Name()
	if name == "" {
		name = filename
	}
	filesize := strconv.FormatInt(stat.Size(), 10)
	// Prepare the request body
	body := fmt.Sprintf(`{"name": "%v",
  "file_name": "%v",
  "asset_type": "video",
  "file_size": "%v",
	"chunk_size": "%v"}`, name, filename, filesize, chunklimit)
	// Create asset in Backlot
	response, err := a.Post("/v2/assets", strings.NewReader(body))
	if err != nil {
		return Asset{}, err
	}
	result, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return Asset{}, err
	}
	// Receive the json and fetch the embed_code
	var asset Asset
	err = json.Unmarshal(result, &asset)
	if err != nil {
		return Asset{}, err
	}
	return asset, nil
}

// getUploadUrl queries Backlot api to receive the needed routes for upload
func getUploadUrl(a oo.Apier, embedCode string) ([]string, error) {
	var urls []string
	response, err := a.Get("/v2/assets/" + embedCode + "/uploading_urls")
	if err != nil {
		return nil, err
	}

	result, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()

	err = json.Unmarshal(result, &urls)
	if err != nil {
		return nil, err
	}
	// fmt.Println(urls)
	return urls, nil
}

func getFileStat(file string) (os.FileInfo, error) {
	// Open file and get it statistics
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	return stat, nil
}

func splitFile(file string, lim int) ([][]byte, error) {
	// Transfer data from file to buffer
	buf, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}
	// make slice of chunks by deviding buffer
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:len(buf)])
	}
	return chunks, nil
}

// uploadChunk uploads chunk to a given url
func uploadChunk(b []byte, bar *pb.ProgressBar, url string, wg *sync.WaitGroup, errchannel chan error) {
	// decrement counter after this function is finished
	defer wg.Done()
	// create readers
	reader := bytes.NewReader(b)
	readerBar := bar.NewProxyReader(reader)
	// Construct request
	request, err := http.NewRequest("PUT", url, readerBar)
	if err != nil {
		errchannel <- err
		return
	}
	// request is not sent without ContentLength due to pb proxy reader
	request.ContentLength = int64(len(b))
	// send a request, receive response
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		errchannel <- err
		return
	}
	// Stop displaing bar
	bar.Finish()
	// there is an issue if StatusCode is not 204
	if response.StatusCode != 204 {
		message, _ := ioutil.ReadAll(response.Body)
		err := fmt.Errorf("Error: [%v] %v", response.StatusCode, string(message))
		defer response.Body.Close()
		errchannel <- err
		return
	}
}

// UploadImage is used to upload preview image files for the asset with given embed code
func UploadImage(a oo.Apier, file, embed_code string) (*Image, error) {
	// Transfer data from file to buffer
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	res, err := a.Post("/v2/assets/"+embed_code+"/preview_image_files", f)
	if err != nil {
		return nil, err
	}

	// Handle unsuccessful attempt
	if res.StatusCode != 200 {
		result, err := ioutil.ReadAll(res.Body)
		defer res.Body.Close()
		if err != nil {
			return nil, err
		}
		err = fmt.Errorf("Error: [%v] %v", res.StatusCode, string(result))
		return nil, err
	}
	// Get the data for the uploaded image from the server
	res, err = a.Get("/v2/assets/" + embed_code + "/preview_image_files")
	if err != nil {
		return nil, err
	}

	result, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}
	if res.StatusCode != 200 {
		err := fmt.Errorf("Error: [%v] %v", res.StatusCode, string(result))
		return nil, err
	}
	// Parse data to the Image struct and return it
	var images []Image
	err = json.Unmarshal(result, &images)
	if err != nil {
		return nil, err
	}
	return &images[0], nil
}
