package main

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"
)

// Hashes represents translations hashes indexed by domain
type Hashes map[string]string

// TranslationsProvider is an interface for providing translation Hashes
type TranslationsProvider interface {
	// Fetch translation hashes and return a pointer to the Hashes struct.
	// The Hashes pointer must remain the same after future calls.
	Fetch() *Hashes
}

// LocoProvider is the Loco implementation of the translation TranslationsProvider
type LocoProvider struct {
	clients map[string]*LocoClient
	hashes  *Hashes
}

// NewLocoProvider creates a new LocoProvider instance
func NewLocoProvider(clients map[string]*LocoClient) *LocoProvider {
	return &LocoProvider{clients, &Hashes{}}
}

// Fetch loco translations by domain
func (p *LocoProvider) Fetch() *Hashes {

	var wg sync.WaitGroup

	errChan := make(chan error)
	defer close(errChan)

	// Compute hashes for all translation domains in parallel
	for k, v := range p.clients {
		wg.Add(1)

		go func(domain string, client *LocoClient) {
			defer wg.Done()

			body, err := client.exportAll()
			if err != nil {
				errChan <- err
			}

			defer body.Close()

			buf := new(bytes.Buffer)
			buf.ReadFrom(body)

			hash := sha1.New()
			hash.Write(buf.Bytes())

			(*p.hashes)[domain] = hex.EncodeToString(hash.Sum(nil))
		}(k, v)
	}

	wgDone := make(chan bool)
	go func() {
		wg.Wait()
		close(wgDone)
	}()

	select {
	case <-wgDone:
		break
	case err := <-errChan:
		close(errChan)
		log.Fatal(err)
	}

	log.Printf("Refreshed hashes: %+v\n", p.hashes)

	return p.hashes
}

// LocoClient represents a Loco client
type LocoClient struct {
	http       *http.Client
	project    string
	apiKey     string
	baseURI    string
	apiVersion string
}

// NewClient creates a new Client
func NewClient(project string, apiKey string) *LocoClient {
	http := &http.Client{}

	return &LocoClient{
		http,
		project,
		apiKey,
		"https://localise.biz",
		"1.0.25",
	}
}

// CreateLocoClients creates loco clients for every entry of the domain => api key map
func CreateLocoClients(apiKeys map[string]string) map[string]*LocoClient {
	// Create Loco clients
	var clients = make(map[string]*LocoClient)
	for k, v := range apiKeys {
		clients[k] = NewClient(k, v)
	}

	// Check API connectivity (parallel)
	log.Println("Checking connectivity to Loco API...")
	for domain, client := range clients {
		err := client.authVerify()
		if err != nil {
			log.Fatal(errors.New("Authentication to Loco API for domain " + domain + " failed: " + err.Error()))
		}
	}

	return clients
}

// AuthVerify will check the authentication credentials
func (c LocoClient) authVerify() error {
	resp, err := c.call("GET", "/api/auth/verify")
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return errors.New("Authentication failed")
	}

	return nil
}

// ExportAll returns the export of all traslations
func (c LocoClient) exportAll() (io.ReadCloser, error) {
	resp, err := c.call("GET", "/api/export/all")
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (c LocoClient) call(method string, path string) (*http.Response, error) {
	req, err := http.NewRequest(method, c.baseURI+path+"?key="+c.apiKey, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-Api-Version", c.apiVersion)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}

	return resp, nil
}
