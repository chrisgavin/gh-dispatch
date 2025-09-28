package client

import (
	"io"
	"log"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/pkg/errors"
)

func NewClient(host string) (*api.RESTClient, error) {
	retryableHTTPClient := retryablehttp.NewClient()
	retryableHTTPClient.RetryMax = 5
	retryableHTTPClient.Logger = log.New(io.Discard, "", log.LstdFlags)
	retryableRoundTripper := retryablehttp.RoundTripper{Client: retryableHTTPClient}

	client, err := api.NewRESTClient(api.ClientOptions{Host: host, Transport: &retryableRoundTripper})
	if err != nil {
		return nil, errors.Wrap(err, "Unable to create GitHub client.")
	}
	return client, nil
}
