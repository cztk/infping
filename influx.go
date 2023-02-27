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
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"golang.org/x/net/context"
	"time"
)

// Client is a generic interface to write single-metric ping data
type Client interface {
	Write(point Point) error
	Ping() (bool, error)
	//Query(q client.Query) (*client.Response, error)
}

// NewInfluxClient creates a concrete InfluxDB Writer
func NewInfluxClient(client influxdb2.Client, influxOrg string, influxMeasurement string, influxBucket string, retPolicy string, tags map[string]interface{}) *InfluxClient {
	return &InfluxClient{
		influx:      client,
		org:         influxOrg,
		measurement: influxMeasurement,
		bucket:      influxBucket,
		retPolicy:   retPolicy,
		tags:        tags,
	}
}

// InfluxClient implements the Client interface to provide a metrics client
// backed by InfluxDB
type InfluxClient struct {
	influx      influxdb2.Client
	org         string
	bucket      string
	measurement string
	retPolicy   string
	tags        map[string]interface{}
}

// Ping calls Ping on the underlying influx client
func (i *InfluxClient) Ping() (bool, error) {
	return i.influx.Ping(context.Background())
}

// Query calls Query on	the underlying influx client
//func (i *InfluxClient) Query(q client.Query) (*client.Response, error) {
//	return i.influx.Query(q)
//}

// Write writes a single point to influx
func (i *InfluxClient) Write(point Point) error {
	writeAPI := i.influx.WriteAPIBlocking(i.org, i.bucket)

	if point.Min != 0 && point.Avg != 0 && point.Max != 0 {
		p := influxdb2.NewPointWithMeasurement(i.measurement).
			AddTag("tx_host", point.TxHost).
			AddTag("rx_host", point.RxHost).
			AddField("tx_host", point.RxHost).
			AddField("rx_host", point.TxHost).
			AddField("loss", point.LossPercent).
			AddField("min", point.Min).
			AddField("avg", point.Avg).
			AddField("max", point.Max).
			SetTime(time.Now())
		addTags(p, i.tags)
		err := writeAPI.WritePoint(context.Background(), p)
		if err != nil {
			return err
		}
	} else {
		p := influxdb2.NewPointWithMeasurement(i.measurement).
			AddTag("tx_host", point.TxHost).
			AddTag("rx_host", point.RxHost).
			AddField("loss", point.LossPercent).
			SetTime(time.Now())
		addTags(p, i.tags)
		err := writeAPI.WritePoint(context.Background(), p)
		if err != nil {
			return err
		}
	}

	return nil
}

func addTags(p *write.Point, tags map[string]interface{}) {
	for k, v := range tags {
		s := v.(string)
		p.AddTag(k, s)
	}
}
