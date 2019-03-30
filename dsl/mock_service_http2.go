package dsl

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"golang.org/x/net/http2"
)

// MockService is the HTTP interface to setup the Pact Mock Service
// See https://github.com/bethesque/pact-mock_service and
// https://gist.github.com/bethesque/9d81f21d6f77650811f4.
type MockServiceHttp2 struct {
	// BaseURL is the base host for the Pact Mock Service.
	BaseURL string

	// Consumer name.
	Consumer string

	// Provider name.
	Provider string

	// PactFileWriteMode specifies how to write to the Pact file, for the life
	// of a Mock Service.
	// "overwrite" will always truncate and replace the pact after each run
	// "update" will append to the pact file, which is useful if your tests
	// are split over multiple files and instantiations of a Mock Server
	// See https://github.com/pact-foundation/pact-ruby/blob/master/documentation/configuration.md#pactfile_write_mode
	PactFileWriteMode string
}

// call sends a message to the Pact service
func (m *MockServiceHttp2) call(method string, url string, content interface{}) error {
	body, err := json.Marshal(content)
	if err != nil {
		fmt.Println(err)
		return err
	}

	client := &http.Client{
		Transport: &http2.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	var req *http.Request
	if method == "POST" {
		req, err = http.NewRequest(method, url, bytes.NewReader(body))
	} else {
		req, err = http.NewRequest(method, url, nil)
	}
	if err != nil {
		return err
	}

	req.Header.Set("X-Pact-Mock-Service", "true")
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return err
	}

	responseBody, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return errors.New(string(responseBody))
	}
	return err
}

// DeleteInteractions removes any previous Mock Service Interactions.
func (m *MockServiceHttp2) DeleteInteractions() error {
	log.Println("[DEBUG] mock service delete interactions")
	url := fmt.Sprintf("%s/interactions", m.BaseURL)
	return m.call("DELETE", url, nil)
}

// AddInteraction adds a new Pact Mock Service interaction.
func (m *MockServiceHttp2) AddInteraction(interaction *Interaction) error {
	log.Println("[DEBUG] mock service add interaction")
	url := fmt.Sprintf("%s/interactions", m.BaseURL)
	return m.call("POST", url, interaction)
}

// Verify confirms that all interactions were called.
func (m *MockServiceHttp2) Verify() error {
	log.Println("[DEBUG] mock service verify")
	url := fmt.Sprintf("%s/interactions/verification", m.BaseURL)
	return m.call("GET", url, nil)
}

// WritePact writes the pact file to disk.
func (m *MockServiceHttp2) WritePact() error {
	log.Println("[DEBUG] mock service write pact")

	if m.Consumer == "" || m.Provider == "" {
		return errors.New("Consumer and Provider name need to be provided")
	}
	if m.PactFileWriteMode == "" {
		m.PactFileWriteMode = "overwrite"
	}

	pact := map[string]interface{}{
		"consumer": map[string]string{
			"name": m.Consumer,
		},
		"provider": map[string]string{
			"name": m.Provider,
		},
		"pactFileWriteMode": m.PactFileWriteMode,
	}

	url := fmt.Sprintf("%s/pact", m.BaseURL)
	return m.call("POST", url, pact)
}
