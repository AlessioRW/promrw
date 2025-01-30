package promrw

import (
    "bytes"
    "compress/gzip"
    "context"
    "fmt"
    "io"
    "log/slog"
    "net/http"
    "time"

    "github.com/prometheus/prometheus/prompb"
    "github.com/robfig/cron"
)

type RemoteWriteClient struct {
    userAgent     string
    prometheusURL string
    metrics       []prompb.TimeSeries
    cron          *cron.Cron
}

type Label struct {
    Name  string
    Value string
}

// Create a new Remote Write Client.
// Write frequency is a Cron Schedule (see https://pkg.go.dev/github.com/robfig/cron?utm_source=godoc for help.
func NewClient(remoteWriteURL string, userAgent string, writeFrequency string) (RemoteWriteClient, error) {

    client := RemoteWriteClient{
        prometheusURL: remoteWriteURL,
        userAgent:     userAgent,
        metrics:       []prompb.TimeSeries{},
    }

    client.cron = cron.New()
    err := client.cron.AddFunc(writeFrequency, func() {
        err := client.pushMetrics()

    })
    if err != nil {
        slog.Error("error scheduling cron job", slog.Any("error", err))
        return RemoteWriteClient{}, err
    }
    client.cron.Start()

    return client, nil
}

// Add a Metric to be pushed to Prometheus.
func (client *RemoteWriteClient) AddMetric(name string, labels []Label) {
    pLabels := []prompb.Label{}
    for _, label := range labels {
        pLabels = append(pLabels, prompb.Label{Name: label.Name, Value: label.Value})
    }

    metric := prompb.TimeSeries{
        Labels:  append(pLabels, prompb.Label{Name: "__name__", Value: name}),
        Samples: []prompb.Sample{},
    }

    client.metrics = append(client.metrics, metric)
}

func (client *RemoteWriteClient) getMetric(metricName string) (prompb.TimeSeries, error) {
    for _, metric := range client.metrics {
        for _, label := range metric.Labels {
            if label.Name == "__name__" {
                if label.Value == metricName {
                    return metric, nil
                }
            }
        }
    }
    return prompb.TimeSeries{}, fmt.Errorf("could not find metric with name \"%v\". add one using the AddMetric method", metricName)
}

// Add a Timeseries point to a Metric, these will be cleared every run of the Cron passed when initalising the client.
// Timestamp is a Millisecond value from the Unix Epoch.
func (client *RemoteWriteClient) AddMetricPoint(metricName string, value float64, timestamp int64) error {

    newSample := prompb.Sample{
        Value:     value,
        Timestamp: timestamp,
    }

    metric, err := client.getMetric(metricName)
    if err != nil {
        return err
    }

    metric.Samples = append(metric.Samples, newSample)

    return nil
}

func (client *RemoteWriteClient) pushMetrics() {

    writeReq := prompb.WriteRequest{
        Timeseries: client.metrics,
    }

    data, err := writeReq.Marshal()
    if err != nil {
        fmt.Println("error marshalling timeseries data, error: %v,", err)
        return
    }

    var buffer bytes.Buffer
    gzipWriter := gzip.NewWriter(&buffer)
    _, err = gzipWriter.Write(data)
    if err != nil {
        fmt.Println("error creating http request, error: %v,", err)
    }
    gzipWriter.Close()

    req, err := http.NewRequestWithContext(context.Background(), "POST", client.prometheusURL, &buffer)
    if err != nil {
        fmt.Println("error creating http request, error: %v,", err)
        return
    }
    req.Header.Set("Content-Type", "application/x-protobuf")
    req.Header.Set("User-Agent", client.userAgent)
    req.Header.Set("Content-Encoding", "gzip")
    req.Header.Set("X-Prometheus-Remote-Write-Version", "0.1.0")

    httpClient := &http.Client{Timeout: 10 * time.Second}
    res, err := httpClient.Do(req)
    defer res.Body.Close()

    if res.StatusCode/100 != 2 || err != nil {
        stringErr, convErr := io.ReadAll(res.Body)
        if convErr != nil {
            fmt.Println("remote write request failed with status code: %d, error: %v,", res.StatusCode, err)
        }
        fmt.Println("remote write request failed with status code: %d, error: %v, error returned from prometheus", res.StatusCode, err, string(stringErr))
        return
    }

    client.clearMetricSamples()
}

// clear samples so we don't send repeating data
func (client *RemoteWriteClient) clearMetricSamples() {
    for _, metric := range client.metrics {
        metric.Samples = []prompb.Sample{}
    }
}

