package ooutils

import (
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

func ReplaceVideo(a oo.Apier, embedCode, file string, chunklimit int) (*http.Response, error) {
	// get fileinfo to use it for the further queries
	fileInfo, err := getFileStat(file)
	if err != nil {
		return nil, err
	}
	// send request and receive the respose with asset
	err = replaceAsset(a, embedCode, fileInfo, chunklimit)
	if err != nil {
		return nil, err
	}
	// get the needed urls for uploading
	urls, err := getReplaceUrl(a, embedCode)
	if err != nil {
		return nil, err
	}
	// split file up into chunks based on the limit
	chunks, err := splitFile(file, chunklimit)
	if err != nil {
		return nil, err
	}
	// errchannel used to track erorrs in goroutines
	errchannel := make(chan error, 1)
	// finished channel indicates that all goroutines are finished
	finished := make(chan bool, 1)
	// perfrom uploading for an each of the chunks
	// waitgroup waits for a collection of goroutines to finish
	var wg sync.WaitGroup
	// pool of the progress bars
	pool, err := pb.StartPool()
	if err != nil {
		return nil, err
	}
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
	// Stop displaing a pool of progress bars
	pool.Stop()
	// final query to mark an asset as uploaded
	res, err := a.Put("/v2/assets/"+embedCode+"/replacement/upload_status", strings.NewReader(`{"status":"uploaded"}`))
	if err != nil {
		return nil, err
	}
	return res, nil
}

func replaceAsset(a oo.Apier, embedCode string, stat os.FileInfo, chunklimit int) error {
	filesize := strconv.FormatInt(stat.Size(), 10)
	// Prepare the request body
	body := fmt.Sprintf(`{"file_size": "%v", "chunk_size": "%v"}`, filesize, chunklimit)
	// Replace asset in Backlot
	response, err := a.Post("/v2/assets/"+embedCode+"/replacement", strings.NewReader(body))
	if err != nil {
		return err
	}
	result, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return err
	}
	// Receive the json and fetch the embed_code
	var asset Asset
	err = json.Unmarshal(result, &asset) // <=== bee, need to redesign
	if err != nil {
		return err
	}
	return nil
}

func getReplaceUrl(a oo.Apier, embedCode string) ([]string, error) {
	var urls []string
	response, err := a.Get("/v2/assets/" + embedCode + "/replacement/uploading_urls")
	if err != nil {
		return nil, err
	}

	result, err := ioutil.ReadAll(response.Body)
	defer response.Body.Close()
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(result, &urls)
	if err != nil {
		return nil, err
	}
	return urls, nil
}
