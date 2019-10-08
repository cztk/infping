/*
The MIT License (MIT)

Copyright (c) 2017 Nicholas Van Wiggeren  https://github.com/nickvanw/infping
Copyright (c) 2018 Michael Newton         https://github.com/miken32/infping

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal 
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package main

import (
	"github.com/influxdata/influxdb1-client/v2"
	"time"
)

// Client is a generic interface to write single-metric ping data
type Client interface {
	Write(point Point) error
	Ping() (time.Duration, string, error)
	Query(q client.Query) (*client.Response, error)
}

// NewInfluxClient creates a concrete InfluxDB Writer
func NewInfluxClient(client client.Client, db, measurement, retPolicy string) *InfluxClient {
	return &InfluxClient{
		influx:      client,
		db:          db,
		retPolicy:   retPolicy,
		measurement: measurement,
	}
}

// InfluxClient implements the Client interface to provide a metrics client
// backed by InfluxDB
type InfluxClient struct {
	influx      client.Client
	db          string
	retPolicy   string
	measurement string
}

// Ping calls Ping on the underlying influx client
func (i *InfluxClient) Ping() (time.Duration, string, error) {
	return i.influx.Ping(time.Second)
}

// Query calls Query on	the underlying influx client
func (i *InfluxClient) Query(q client.Query) (*client.Response, error) {
	return i.influx.Query(q)
}

// Write writes a single point to influx
func (i *InfluxClient) Write(point Point) error {
	var fields map[string]interface{}
	if point.Min != 0 && point.Avg != 0 && point.Max != 0 {
		fields = map[string]interface{}{
			"loss": point.LossPercent,
			"min":  point.Min,
			"avg":  point.Avg,
			"max":  point.Max,
		}
	} else {
		fields = map[string]interface{}{
			"loss": point.LossPercent,
		}
	}

	pt, err := client.NewPoint(
		i.measurement,
		map[string]string{
			"rx_host": point.RxHost,
			"tx_host": point.TxHost,
		},
		fields,
		point.Time,
	)

	if err != nil {
		return err
	}

	batchConfig := client.BatchPointsConfig{Database: i.db, Precision: "s"}
	bp, err := client.NewBatchPoints(batchConfig)
	if err != nil {
		return err
	}

	bp.AddPoint(pt)
	if i.retPolicy != "" {
		bp.SetRetentionPolicy(i.retPolicy)
	}

	return i.influx.Write(bp)
}
