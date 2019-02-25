package oo

import (
	"fmt"
	"io/ioutil"
	"net/http"
)

func checkServiceError(r *http.Response, expected int) error {
	if r.StatusCode != expected {
		result, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return fmt.Errorf("could't check http error: %v", err)
		}
		defer r.Body.Close()
		return fmt.Errorf("service error: [%v] %v", r.StatusCode, string(result))
	}
	return nil
}
