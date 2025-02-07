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
	metrics       []*prompb.TimeSeries
	labels        []prompb.Label
	httpClient    *http.Client
}

type metric struct {
	Labels  []prompb.Label
	Samples []prompb.Sample
}

type Label struct {
	Name  string
	Value string
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
		metrics:       []*prompb.TimeSeries{},
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

// Create a new metric to be pushed to Prometheus.
// name Parameter is the value of the "__name__" label of the metric.
// Label names and "name" parameter, must match this pattern: ^[a-zA-Z_:][a-zA-Z0-9_:]*$.
func NewMetric(name string, labels []Label) (*metric, error) {

	pLabels := []prompb.Label{}
	for _, label := range labels {
		pLabels = append(pLabels, prompb.Label{Name: label.Name, Value: label.Value})
	}

	metric := metric{
		Labels:  append(pLabels, prompb.Label{Name: "__name__", Value: name}),
		Samples: []prompb.Sample{},
	}

	err := regexCheckLabels(metric.Labels)
	if err != nil {
		return nil, err
	}

	return &metric, nil

}

// Add a Timeseries point to a Metric, these will be cleared every run of PushMetric.
// Timestamp is a Millisecond value from the Unix Epoch.
func (metric *metric) AddSample(value float64, timestamp int64) error {

	newSample := prompb.Sample{
		Value:     value,
		Timestamp: timestamp,
	}

	metric.Samples = append(metric.Samples, newSample)

	return nil
}

func (client *RemoteWriteClient) PushMetric(metric *Metric) error {
	// TODO clean up this function

	allLabels := []prompb.Label{}
	// add client specific labels
	for _, label := range client.labels {
		allLabels = append(allLabels, prompb.Label{
			Name:  label.Name,
			Value: label.Value,
		})
	}

	// add metric labels
	allLabels = append(allLabels, metric.Labels...)

	prompbMetric := prompb.TimeSeries{
		Labels:  allLabels,
		Samples: metric.Samples,
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

	clearMetricSamples(metric)

	return nil
}

// clear samples so we don't send repeating data
func clearMetricSamples(metric *metric) {
	metric.Samples = []prompb.Sample{}
}
