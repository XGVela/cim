// Copyright 2020 Mavenir
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This package is responsible for making client support on client side
// Here types.ConfigDetails is a struct which contains some variables
// for http/2 implementation.

package httpUtility

import (
	"bytes"
	"cim/types"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"golang.org/x/net/http2"
)

// Http_client is a basic default http client.
// will be updating this default client so that it can support HTTP/2 as well.
var Http_client http.Client
var Http2_client http.Client

// This function is responsible for making http client based on configuration.
// if http2_enabled is true in configuration this function will prepare http2 client.
// else default http client will be prepared
// InSecureSkipVerify will inform that verification skip..

////// TODO ///////////////////////////
// This function can be modularized in future to support certificates
// in that case allow_http and key file path can be taken from configuration file
func InitializeHttpClient() {
	config := &tls.Config{
		InsecureSkipVerify: true,
	}
	Http2_client = http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				return net.Dial(network, addr)
			},
		},
		Timeout: 10 * time.Second,
	}
	Http_client = http.Client{
		Transport: &http.Transport{TLSClientConfig: config},
		Timeout:   10 * time.Second,
	}
}

// HttpDo function acts as a API for external use.
// arguments
// method can be any of PUT/GET/DELETE/POST
// url should be string
func HttpDo(method string, url string, content_type string, body []byte, opts ...string) (resp *http.Response, err error) {
	switch method {
	case http.MethodGet:
		resp, err = HttpGet(url, content_type, body, opts...)
	case http.MethodPost:
		resp, err = HttpPost(url, content_type, body, opts...)
	case http.MethodPut:
		resp, err = HttpPut(url, content_type, body, opts...)
	case http.MethodDelete:
		resp, err = HttpDelete(url, content_type, body, opts...)
	}
	return
}

func getProtocol(opts ...string) string {
	var protocol string
	if len(opts) >= 1 {
		protocol = opts[0]
	} else {
		if types.CimConfigObj.CimConfig.Subjects.Http2Enabled {
			protocol = types.ProtocolHTTP2
		} else {
			protocol = types.ProtocolHTTP
		}
	}
	return protocol
}

// HttpGet function acts as API for HTTP GET request
// Arguments and description is same as HttpDo except method.
func HttpGet(url string, content_type string, body []byte, opts ...string) (resp *http.Response, err error) {
	protocol := getProtocol(opts...)
	if protocol == types.ProtocolHTTP2 {
		return Http2_client.Get(url)
	}
	return Http_client.Get(url)
}

// HttpPost function acts as API for HTTP POST request
// Arguments and description is same as HttpDo except method.
func HttpPost(url string, content_type string, body []byte, opts ...string) (resp *http.Response, err error) {
	protocol := getProtocol(opts...)
	if protocol == types.ProtocolHTTP2 {
		return Http2_client.Post(url, content_type, bytes.NewBuffer(body))
	}
	return Http_client.Post(url, content_type, bytes.NewBuffer(body))
}

// HttpPut function acts as API for HTTP PUT request
// Arguments and description is same as HttpDo except method.
func HttpPut(url string, content_type string, body []byte, opts ...string) (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return
	}
	protocol := getProtocol(opts...)
	if protocol == types.ProtocolHTTP2 {
		return Http2_client.Do(req)
	}
	return Http_client.Do(req)
}

// HttpDelete function acts as API for HTTP DELETE request
// Arguments and description is same as HttpDo except method.
func HttpDelete(url string, content_type string, body []byte, opts ...string) (resp *http.Response, err error) {
	req, err := http.NewRequest(http.MethodDelete, url, bytes.NewBuffer(body))
	if err != nil {
		return
	}
	protocol := getProtocol(opts...)
	if protocol == types.ProtocolHTTP2 {
		return Http2_client.Do(req)
	}
	return Http_client.Do(req)
}

// ShutDownApp sends shutdown signal to the app
func ShutDownApp(reason string) (*http.Response, error) {
	if len(types.AppShutDownCause) != 0 {
		return nil, fmt.Errorf("Shutdown request already sent to the application with reason %s", types.AppShutDownCause[0])
	}
	response, err := HttpPost(
		"http://localhost:"+types.AppPort+"/api/v1/_operations/shutdown",
		"application/json",
		nil,
	)
	if err != nil {
		return nil, err
	}
	types.AppShutDownCause = append(types.AppShutDownCause, reason)
	return response, nil
}
