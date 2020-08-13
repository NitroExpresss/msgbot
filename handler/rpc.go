package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

var (
	httpClient = http.Client{
		Timeout: 30 * time.Second,
	}
)

func (h *Handler) RPC(method, url string, payload, response interface{}, token ...string) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return errors.Wrap(err, "failed to marshal a payload")
	}
	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrap(err, "failed to create an http request")
	}

	req.Header.Set("Content-Type", "application/json")
	if len(token) > 0 {
		req.Header.Add("Authorization", "Bearer "+token[0])
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to make a %s request", method))
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Errorf("%s request to %s failed with status: %d", method, url, resp.StatusCode)
		}
		return errors.Errorf("%s request to %s failed with status: %d and body: %s", method, url, resp.StatusCode, string(body))
	}
	if response != nil {
		return json.NewDecoder(resp.Body).Decode(response)
	}
	return nil
}
