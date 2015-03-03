package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"time"
)

var apiVersion = "v1.18"

type RemoteClient struct {
	transport   *http.Transport
	proto, addr string
}

func NewRemoteClient(host string) (*RemoteClient, error) {

	protoAndAddr := strings.SplitN(host, "://", 2)
	if len(protoAndAddr) != 2 {
		return nil, fmt.Errorf("Bad format for host: %s", host)
	}
	proto := protoAndAddr[0]
	addr := protoAndAddr[1]

	tr := http.Transport{}
	timeout := 10 * time.Second

	switch proto {
	case "unix":
		// no need in compressing for local communications
		tr.DisableCompression = true
		tr.Dial = func(_, _ string) (net.Conn, error) {
			return net.DialTimeout("unix", addr, timeout)
		}
	case "tcp":
		tr.Proxy = http.ProxyFromEnvironment
		tr.Dial = (&net.Dialer{Timeout: timeout}).Dial
	default:
		return nil, fmt.Errorf("Unsupported protocol: ", proto)
	}

	return &RemoteClient{transport: &tr,
		proto: proto,
		addr:  addr}, nil
}

func (c *RemoteClient) Get(path string) (io.ReadCloser, int, error) {
	req, err := http.NewRequest("GET", "/"+apiVersion+path, nil)
	if err != nil {
		return nil, -1, err
	}

	req.URL.Host = c.addr
	req.URL.Scheme = "http"

	httpClient := &http.Client{Transport: c.transport}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, -1, err
	}

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, -1, err
		}
		return nil, resp.StatusCode, fmt.Errorf("RemoteClient Error (%d): %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	return resp.Body, resp.StatusCode, nil
}

func (c *RemoteClient) GetJSON(path string, result interface{}) error {
	output, _, err := c.Get(path)
	if err != nil {
		return err
	}
	defer output.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(output)

	err = json.Unmarshal(buf.Bytes(), result)
	if err != nil {
		return err
	}

	return nil
}

func (c *RemoteClient) Exists(path string) (bool, error) {
	output, status, err := c.Get(path)

	if status == 404 {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		output.Close()
		return true, nil
	}
}

func (c *RemoteClient) Post(path string, bodyType string, body io.Reader) error {
	req, err := http.NewRequest("POST", "/"+apiVersion+path, body)
	if err != nil {
		return err
	}

	req.URL.Host = c.addr
	req.URL.Scheme = "http"

	httpClient := &http.Client{Transport: c.transport}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("RemoteClient Error (%d): %s", resp.StatusCode, bytes.TrimSpace(body))
	}

	return nil
}
