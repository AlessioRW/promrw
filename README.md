# promrw - A Golang Prometheus Remote Write

## Overview
`promrw` is a Go package that provides a client for sending metrics to a Prometheus Remote Write endpoint. It allows users to create metrics, add time series data points, and push them to Prometheus efficiently using HTTP and gzip compression.

## Features
- Create a Remote Write client with configurable labels.
- Define and manage Prometheus metrics.
- Add timestamped samples to metrics.
- Push metrics to a Prometheus Remote Write endpoint.
- Efficient data transmission using gzip compression.

## Installation
To install `promrw`, use:
```sh
go get github.com/AlessioRW/promrw
