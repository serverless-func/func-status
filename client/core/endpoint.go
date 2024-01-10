package core

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Masterminds/semver/v3"
	"github.com/serverless-aliyun/func-status/client/util"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type EndpointType string

const (
	// HostHeader is the name of the header used to specify the host
	HostHeader = "Host"

	// ContentTypeHeader is the name of the header used to specify the content type
	ContentTypeHeader = "Content-Type"

	// UserAgentHeader is the name of the header used to specify the request's user agent
	UserAgentHeader = "User-Agent"

	// GatusUserAgent is the default user agent that Gatus uses to send requests.
	GatusUserAgent = "Gatus/1.0"

	EndpointTypeDNS     EndpointType = "DNS"
	EndpointTypeHTTP    EndpointType = "HTTP"
	EndpointTypeVERSION EndpointType = "VERSION"
	EndpointTypeUNKNOWN EndpointType = "UNKNOWN"
)

var (
	// ErrEndpointWithNoCondition is the error with which Gatus will panic if an endpoint is configured with no conditions
	ErrEndpointWithNoCondition = errors.New("you must specify at least one condition per endpoint")

	// ErrEndpointWithNoURL is the error with which Gatus will panic if an endpoint is configured with no url
	ErrEndpointWithNoURL = errors.New("you must specify an url for each endpoint")

	// ErrEndpointWithNoName is the error with which Gatus will panic if an endpoint is configured with no name
	ErrEndpointWithNoName = errors.New("you must specify a name for each endpoint")

	// ErrEndpointWithInvalidNameOrGroup is the error with which Gatus will panic if an endpoint has an invalid character where it shouldn't
	ErrEndpointWithInvalidNameOrGroup = errors.New("endpoint name and group must not have \" or \\")

	// ErrUnknownEndpointType is the error with which Gatus will panic if an endpoint has an unknown type
	ErrUnknownEndpointType = errors.New("unknown endpoint type")

	// ErrInvalidConditionFormat is the error with which Gatus will panic if a condition has an invalid format
	ErrInvalidConditionFormat = errors.New("invalid condition format: does not match '<VALUE> <COMPARATOR> <VALUE>'")

	// ErrInvalidVersionFormat is the error with which version not match Semantic Versions
	ErrInvalidVersionFormat = errors.New("invalid condition format: does not match '<VALUE> <COMPARATOR> <VALUE>'")
)

// Endpoint is the configuration of a monitored
type Endpoint struct {
	// Enabled defines whether to enable the monitoring of the endpoint
	Enabled *bool `yaml:"enabled,omitempty"`

	// Name of the endpoint. Can be anything.
	Name string `yaml:"name"`

	// URL to send the request to
	URL string `yaml:"url"`

	// DNS is the configuration of DNS monitoring
	DNS *DNS `yaml:"dns,omitempty"`

	// Method of the request made to the url of the endpoint
	Method string `yaml:"method,omitempty"`

	// Body of the request
	Body string `yaml:"body,omitempty"`

	// GraphQL is whether to wrap the body in a query param ({"query":"$body"})
	GraphQL bool `yaml:"graphql,omitempty"`

	// Headers of the request
	Headers map[string]string `yaml:"headers,omitempty"`

	// Version of Current Release | Package
	Version string `yaml:"version,omitempty"`

	// Conditions used to determine the health of the endpoint
	Conditions []Condition `yaml:"conditions"`
}

// IsEnabled returns whether the endpoint is enabled or not
func (endpoint Endpoint) IsEnabled() bool {
	if endpoint.Enabled == nil {
		return true
	}
	return *endpoint.Enabled
}

// Type returns the endpoint type
func (endpoint Endpoint) Type() EndpointType {
	switch {
	case endpoint.DNS != nil:
		return EndpointTypeDNS
	case strings.HasPrefix(endpoint.URL, "http://") || strings.HasPrefix(endpoint.URL, "https://"):
		if endpoint.Version != "" {
			return EndpointTypeVERSION
		}
		return EndpointTypeHTTP
	default:
		return EndpointTypeUNKNOWN
	}
}

// ValidateAndSetDefaults validates the endpoint's configuration and sets the default value of args that have one
func (endpoint *Endpoint) ValidateAndSetDefaults() error {
	if len(endpoint.Method) == 0 {
		endpoint.Method = http.MethodGet
	}
	if len(endpoint.Headers) == 0 {
		endpoint.Headers = make(map[string]string)
	}
	// Automatically add user agent header if there isn't one specified in the endpoint configuration
	if _, userAgentHeaderExists := endpoint.Headers[UserAgentHeader]; !userAgentHeaderExists {
		endpoint.Headers[UserAgentHeader] = GatusUserAgent
	}
	// Automatically add "Content-Type: application/json" header if there's no Content-Type set
	// and endpoint.GraphQL is set to true
	if _, contentTypeHeaderExists := endpoint.Headers[ContentTypeHeader]; !contentTypeHeaderExists && endpoint.GraphQL {
		endpoint.Headers[ContentTypeHeader] = "application/json"
	}
	if len(endpoint.Name) == 0 {
		return ErrEndpointWithNoName
	}
	if strings.ContainsAny(endpoint.Name, "\"\\") {
		return ErrEndpointWithInvalidNameOrGroup
	}
	if len(endpoint.URL) == 0 {
		return ErrEndpointWithNoURL
	}
	if len(endpoint.Conditions) == 0 {
		return ErrEndpointWithNoCondition
	}
	for _, c := range endpoint.Conditions {
		if err := c.Validate(); err != nil {
			return fmt.Errorf("%v: %w", ErrInvalidConditionFormat, err)
		}
	}
	if endpoint.DNS != nil {
		return endpoint.DNS.validateAndSetDefault()
	}
	if endpoint.Type() == EndpointTypeVERSION {
		if _, err := semver.NewVersion(endpoint.Version); err != nil {
			return fmt.Errorf("%v: %w", ErrInvalidVersionFormat, err)
		}
	}
	if endpoint.Type() == EndpointTypeUNKNOWN {
		return ErrUnknownEndpointType
	}
	// Make sure that the request can be created
	_, err := http.NewRequest(endpoint.Method, endpoint.URL, bytes.NewBuffer([]byte(endpoint.Body)))
	if err != nil {
		return err
	}
	return nil
}

// DisplayName returns an identifier made up of the Name and, if not empty, the Group.
func (endpoint Endpoint) DisplayName() string {
	return endpoint.Name
}

// Key returns the unique key for the Endpoint
func (endpoint Endpoint) Key() string {
	return util.ConvertEndpointNameToKey(endpoint.Name)
}

// EvaluateHealth sends a request to the endpoint's URL and evaluates the conditions of the endpoint.
func (endpoint *Endpoint) EvaluateHealth() *Result {
	result := &Result{Success: true, Errors: []string{}}
	// Parse or extract hostname from URL
	if endpoint.DNS != nil {
		result.Hostname = strings.TrimSuffix(endpoint.URL, ":53")
	} else {
		urlObject, err := url.Parse(endpoint.URL)
		if err != nil {
			result.AddError(err.Error())
		} else {
			result.Hostname = urlObject.Hostname()
		}
	}
	if endpoint.Type() == EndpointTypeVERSION {
		result.Version = endpoint.Version
	}
	// Retrieve IP if necessary
	if endpoint.needsToRetrieveIP() {
		endpoint.getIP(result)
	}
	// Call the endpoint (if there's no errors)
	if len(result.Errors) == 0 {
		endpoint.call(result)
	} else {
		result.Success = false
	}
	// Evaluate the conditions
	for _, condition := range endpoint.Conditions {
		success := condition.evaluate(result, false)
		if !success {
			result.Success = false
		}
	}
	result.Timestamp = time.Now()
	return result
}

func (endpoint *Endpoint) getIP(result *Result) {
	if ips, err := net.LookupIP(result.Hostname); err != nil {
		result.AddError(err.Error())
		return
	} else {
		result.IP = ips[0].String()
	}
}

func (endpoint *Endpoint) call(result *Result) {
	var request *http.Request
	var response *http.Response
	var err error
	var certificate *x509.Certificate
	endpointType := endpoint.Type()
	if endpointType == EndpointTypeHTTP || endpointType == EndpointTypeVERSION {
		request = endpoint.buildHTTPRequest()
	}
	startTime := time.Now()
	if endpointType == EndpointTypeDNS {
		endpoint.DNS.query(endpoint.URL, result)
		result.Duration = time.Since(startTime)
	} else {
		response, err = util.GetHTTPClient().Do(request)
		result.Duration = time.Since(startTime)
		if err != nil {
			result.AddError(err.Error())
			return
		}
		defer response.Body.Close()
		if response.TLS != nil && len(response.TLS.PeerCertificates) > 0 {
			certificate = response.TLS.PeerCertificates[0]
			result.CertificateExpiration = time.Until(certificate.NotAfter)
		}
		result.HTTPStatus = response.StatusCode
		result.Connected = response.StatusCode > 0
		// Only read the Body if there's a condition that uses the BodyPlaceholder
		if endpoint.needsToReadBody() {
			result.Body, err = io.ReadAll(response.Body)
			if err != nil {
				result.AddError("error reading response body:" + err.Error())
			}
		}
	}
}

// Close HTTP connections between watchdog and endpoints to avoid dangling socket file descriptors
// on configuration reload.
// More context on https://github.com/TwiN/gatus/issues/536
func (endpoint *Endpoint) Close() {
	if endpoint.Type() == EndpointTypeHTTP {
		util.GetHTTPClient().CloseIdleConnections()
	}
}

func (endpoint *Endpoint) buildHTTPRequest() *http.Request {
	var bodyBuffer *bytes.Buffer
	if endpoint.GraphQL {
		graphQlBody := map[string]string{
			"query": endpoint.Body,
		}
		body, _ := json.Marshal(graphQlBody)
		bodyBuffer = bytes.NewBuffer(body)
	} else {
		bodyBuffer = bytes.NewBuffer([]byte(endpoint.Body))
	}
	request, _ := http.NewRequest(endpoint.Method, endpoint.URL, bodyBuffer)
	for k, v := range endpoint.Headers {
		request.Header.Set(k, v)
		if k == HostHeader {
			request.Host = v
		}
	}
	return request
}

// needsToReadBody checks if there's any condition that requires the response Body to be read
func (endpoint *Endpoint) needsToReadBody() bool {
	for _, condition := range endpoint.Conditions {
		if condition.hasBodyPlaceholder() {
			return true
		}
	}
	return false
}

// needsToRetrieveIP checks if there's any condition that requires an IP lookup
func (endpoint *Endpoint) needsToRetrieveIP() bool {
	for _, condition := range endpoint.Conditions {
		if condition.hasIPPlaceholder() {
			return true
		}
	}
	return false
}
