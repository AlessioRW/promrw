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
		{Name: "label", Value: "label_example_value"}, // these labels will be applied to every metric pushed via this client
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
