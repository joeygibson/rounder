package main

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

func cacheKey(r *http.Request) string {
	return r.URL.String()
}

type cacheTransport struct {
	data              map[string]string
	mu                sync.RWMutex
	originalTransport http.RoundTripper
}

func (c *cacheTransport) Set(r *http.Request, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data[cacheKey(r)] = value
}

func (c *cacheTransport) Get(r *http.Request) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if val, ok := c.data[cacheKey(r)]; ok {
		return val, nil
	}

	return "", errors.New("key not found in cache")
}

func (c *cacheTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	if val, err := c.Get(r); err == nil {
		fmt.Println("\t\t *fetching response from cache")

		return cachedResponse([]byte(val), r)
	}

	// not found in cache
	resp, err := c.originalTransport.RoundTrip(r)
	if err != nil {
		panic(err)
	}

	buf, err := httputil.DumpResponse(resp, true)
	if err != nil {
		panic(err)
	}

	c.Set(r, string(buf))

	fmt.Println("fetching data from real source")

	return resp, nil
}

func (c *cacheTransport) Clear() error {
	c.data = make(map[string]string)
	return nil
}

func cachedResponse(b []byte, r *http.Request) (*http.Response, error) {
	buf := bytes.NewBuffer(b)

	return http.ReadResponse(bufio.NewReader(buf), r)
}

func newTransport() *cacheTransport {
	return &cacheTransport{
		data:              make(map[string]string),
		originalTransport: http.DefaultTransport,
	}
}

func main() {
	cachedTransport := newTransport()

	client := &http.Client{
		Transport: cachedTransport,
		Timeout:   5 * time.Second,
	}

	cacheClearTicker := time.NewTicker(5 * time.Second)
	reqTicker := time.NewTicker(time.Second)

	terminateChan := make(chan os.Signal, 1)
	signal.Notify(terminateChan, syscall.SIGTERM, syscall.SIGHUP)

	req, err := http.NewRequest(http.MethodGet, "http://localhost:8000", strings.NewReader(""))
	if err != nil {
		panic(err)
	}

	for {
		select {
		case <-cacheClearTicker.C:
			cachedTransport.Clear()
		case <-terminateChan:
			cacheClearTicker.Stop()
			reqTicker.Stop()
			return
		case <-reqTicker.C:
			resp, err := client.Do(req)
			if err != nil {
				panic(err)
			}

			buf, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				panic(err)
			}

			fmt.Printf("Response body: %s\n", buf)
		}
	}
}
