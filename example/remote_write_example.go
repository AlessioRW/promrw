package promrw

import (
	"fmt"
	"time"

	promrw "github.com/AlessioRW/promrw/pkg"
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
