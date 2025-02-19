package promrw

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"time"

	"github.com/prometheus/prometheus/prompb"
)

var (
	labelRegex = "^[a-zA-Z_:][a-zA-Z0-9_:]*$"
)

type RemoteWriteClient struct {
	userAgent     string
	prometheusURL string
	labels        []prompb.Label
	httpClient    *http.Client
}

type Label struct {
	Name  string
	Value string
}

type Sample struct {
	Timestamp int64
	Value     float64
}

// Create a new Remote Write Client.
// Labels passed into this function will be applied to every metric pushed via this client.
// Label names must match this pattern: ^[a-zA-Z_:][a-zA-Z0-9_:]*$.
func NewClient(remoteWriteURL string, userAgent string, labels []Label) (*RemoteWriteClient, error) {

	pLabels := []prompb.Label{}
	for _, label := range labels {
		pLabels = append(pLabels, prompb.Label{Name: label.Name, Value: label.Value})
	}

	err := regexCheckLabels(pLabels)
	if err != nil {
		return nil, err
	}

	client := RemoteWriteClient{
		prometheusURL: remoteWriteURL,
		userAgent:     userAgent,
		labels:        pLabels,
		httpClient:    &http.Client{Timeout: 10 * time.Second},
	}

	return &client, nil
}

func regexCheckLabels(labels []prompb.Label) error {
	for _, label := range labels {
		match, err := regexp.MatchString(labelRegex, label.Name)
		if err != nil {
			return fmt.Errorf("error matching the regex for label, error: %v", err)
		}
		if !match {
			return fmt.Errorf("label name \"%v\" does not match the required regex: %v", label.Name, labelRegex)
		}

		if label.Name == "__name__" {
			match, err := regexp.MatchString(labelRegex, label.Value)
			if err != nil {
				return fmt.Errorf("error matching the regex for label value, error: %v", err)
			}
			if !match {
				return fmt.Errorf("label value \"%v\" does not match the required regex: %v", label.Value, labelRegex)
			}
		}

	}

	return nil
}

// Push a metric to Prometheus.
// metricName Parameter is the value of the "__name__" label of the metric.
// Label names and "name" parameter, must match this pattern: ^[a-zA-Z_:][a-zA-Z0-9_:]*$.
func (client *RemoteWriteClient) PushMetric(metricName string, samples []Sample, labels []Label) error {

	prompbLabels := []prompb.Label{
		{Name: "__name__", Value: metricName},
	}
	// add client specific labels
	for _, label := range client.labels {
		prompbLabels = append(prompbLabels, prompb.Label{
			Name:  label.Name,
			Value: label.Value,
		})
	}

	// add metric labels
	for _, label := range labels {
		prompbLabels = append(prompbLabels, prompb.Label{
			Name:  label.Name,
			Value: label.Value,
		})
	}

	// add samples
	prompbSamples := []prompb.Sample{}
	for _, sample := range samples {
		prompbSamples = append(prompbSamples, prompb.Sample{
			Timestamp: sample.Timestamp,
			Value:     sample.Value,
		})
	}

	err := regexCheckLabels(prompbLabels)
	if err != nil {
		return err
	}

	prompbMetric := prompb.TimeSeries{
		Labels:  prompbLabels,
		Samples: prompbSamples,
	}

	writeReq := prompb.WriteRequest{
		Timeseries: []prompb.TimeSeries{prompbMetric},
	}

	data, err := writeReq.Marshal()
	if err != nil {
		return fmt.Errorf("error marshalling timeseries data, error: %v,", err)
	}

	var buffer bytes.Buffer
	gzipWriter := gzip.NewWriter(&buffer)
	_, err = gzipWriter.Write(data)
	if err != nil {
		return fmt.Errorf("error creating http request, error: %v,", err)
	}
	gzipWriter.Close()
	req, err := http.NewRequestWithContext(context.Background(), "POST", client.prometheusURL, &buffer)
	if err != nil {
		return fmt.Errorf("error creating http request, error: %v,", err)

	}
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("User-Agent", client.userAgent)
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

	res, err := client.httpClient.Do(req)
	if err != nil {
		fmt.Printf("error sending http request to prometheus endpoint, error: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode/100 != 2 || err != nil {
		stringErr, convErr := io.ReadAll(res.Body)
		if convErr != nil {
			return fmt.Errorf("remote write request failed with status code: %d, error: %v,", res.StatusCode, err)
		}

		return fmt.Errorf("remote write request failed with status code: %d, error: %v, error returned from prometheus: %v", res.StatusCode, err, string(stringErr))
	}

	return nil
}
