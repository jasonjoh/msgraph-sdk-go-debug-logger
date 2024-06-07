package graphdebug

import (
	"bytes"
	"compress/gzip"
	"io"
	"log"
	"net/http"
	"strings"

	kiota "github.com/microsoft/kiota-http-go"
)

type GraphDebugLogMiddleware struct {
	logger       *log.Logger
	showToken    bool
	showPayloads bool
}

func NewGraphDebugLogMiddleware(
	logger *log.Logger,
	showToken bool,
	showPayloads bool) *GraphDebugLogMiddleware {

	middleware := &GraphDebugLogMiddleware{
		logger:       logger,
		showToken:    showToken,
		showPayloads: showPayloads,
	}

	return middleware
}

func (mw *GraphDebugLogMiddleware) Intercept(
	pipeline kiota.Pipeline,
	middlewareIndex int,
	req *http.Request) (*http.Response, error) {

	mw.logger.Println("REQUEST")

	// Request line
	mw.logger.Printf("%s %s\n", req.Method, req.URL)

	// Headers
	for key, val := range req.Header {
		if !mw.showToken && strings.EqualFold(key, "authorization") {
			mw.logger.Printf("%s: %s\n", key, "***")
		} else {
			mw.logger.Printf("%s: %s\n", key, strings.Join(val, ","))
		}
	}

	// Payload
	if req.ContentLength > 0 && mw.showPayloads {
		contentType := req.Header.Get("Content-Type")
		if contentType == "application/octet-stream" {
			mw.logger.Print("Binary content")
		} else {
			payload, err := io.ReadAll(req.Body)
			if err != nil {
				mw.logger.Printf("Error reading request body: %v\n", err)
				return pipeline.Next(req, middlewareIndex)
			}

			// Reset request body
			req.Body = io.NopCloser(bytes.NewBuffer(payload))

			byteReader := bytes.NewReader(payload)
			gzReader, err := gzip.NewReader(byteReader)
			if err != nil {
				mw.logger.Printf("Error creating gzip reader: %v", err)
				return pipeline.Next(req, middlewareIndex)
			}

			decompressed, err := io.ReadAll(gzReader)
			if err != nil {
				mw.logger.Printf("Error decompressing request payload: %v", err)
				return pipeline.Next(req, middlewareIndex)
			}

			mw.logger.Printf("Payload: %s\n", string(decompressed))
		}
	}

	response, pipelineErr := pipeline.Next(req, middlewareIndex)

	mw.logger.Println("RESPONSE")

	mw.logger.Printf("Status: %s", response.Status)

	for key, val := range response.Header {
		mw.logger.Printf("%s: %s\n", key, strings.Join(val, ","))
	}

	if response.ContentLength != 0 && mw.showPayloads {
		responsePayload, err := io.ReadAll(response.Body)
		if err != nil {
			mw.logger.Printf("Error reading response body: %v\n", err)
			return response, pipelineErr
		}

		// Reset response body
		response.Body = io.NopCloser(bytes.NewBuffer(responsePayload))

		mw.logger.Printf("Payload: %s\n", string(responsePayload))
	}

	return response, pipelineErr
}
