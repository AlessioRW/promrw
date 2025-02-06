package example

import (
	"fmt"
	"time"

	promrw "github.com/AlessioRW/promrw/pkg"
)

func main() {
	promClient, err := promrw.NewClient(
		"remote_write_endpoint",
		"promrw-example/1.0.0",
		[]promrw.Label{
			{Name: "label", Value: "label_example_value"},
		},
	)

	if err != nil {
		fmt.Println(err)
		return
	}

	metric, err := promrw.NewMetric(
		"promrw_example",
		[]promrw.Label{},
	)
	if err != nil {
		fmt.Printf("error creating metric, error: %v \n", err)
	}

	err = metric.AddSample(20, time.Now().UnixMilli())
	if err != nil {
		fmt.Println(err)
	}
	err = promClient.PushMetric(metric)
	if err != nil {
		fmt.Println(err)
	}
}
