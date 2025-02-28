# promrw - A Golang Prometheus Remote Write Package

## Overview
`promrw` is a Go package that provides a client to simplify sending metrics via Prometheus Remote Write. This is limited to Timeseries metrics as of now.

## Installation
To install `promrw`, use:
```sh
go get github.com/AlessioRW/promrw
```

## Example

```
package promrw

import (
	"fmt"
	"time"

	promrw "github.com/AlessioRW/promrw/promrw"
)

func promrwExample() {
	prometheusUrl := ""
	userAgent := ""
	globalLabels := []promrw.Label{
		{Name: "label", Value: "label_example_value"},
	}

	promClient, err := promrw.NewClient(
		prometheusUrl,
		userAgent,
		globalLabels,
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	err = promClient.PushMetric(
		"example_metric_name",
		[]promrw.Sample{
			{Value: 17, Timestamp: time.Now().UnixMilli()},
		},
		[]promrw.Label{},
	)
	if err != nil {
		fmt.Println(err)
	}
}

```