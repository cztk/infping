/*
The MIT License (MIT)

Copyright (c) 2016 Tor Hveem              https://github.com/torhve/infping
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
	"bytes"
	"fmt"
	"github.com/influxdata/influxdb1-client/v2"
	"github.com/spf13/viper"
	"log"
	"net/url"
	"os"
	"reflect"
	"strings"
	"text/template"
)

func main() {
	setDefaults()
	readConfiguration()

	influxClient := createInfluxClient()

	sendPingToInflux(influxClient)
	createDatabaseIfNotExist(influxClient)

	hosts := viper.GetStringSlice("hosts")
	hostname := viper.GetString("hostname")
	fpingConfig := prepareFpingConfiguration()

	log.Printf("Launching fping with hosts: %s", strings.Join(hosts, ", "))
	err := runAndRead(hosts, influxClient, fpingConfig, hostname)

	if err != nil {
		log.Fatal("Failed when obtaining and storing pings", err)
	}
}


type prefixTemplateParams struct {
	Hostname string
	ReverseHostname string
}

func parsePrefixTemplate(tplString string) (string, error) {
	tpl, err := template.New("prefix-template").Parse(tplString)
	if err != nil {
		return "", err
	}

	params := prefixTemplateParams{}
	params.Hostname = strings.ToLower(viper.GetString("hostname"))

	splitHostname := strings.Split(params.Hostname, ".")
	reverseAny(splitHostname)
	params.ReverseHostname = strings.Join(splitHostname, ".")

	out := bytes.NewBufferString("")
	err = tpl.Execute(out, params)
	if err != nil {
		return "", err
	}

	return out.String(), nil
}

func reverseAny(s interface{}) {
	// https://stackoverflow.com/questions/28058278/how-do-i-reverse-a-slice-in-go
	n := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, n - 1; i < j; i, j = i + 1, j - 1 {
		swap(i, j)
	}
}

func setDefaults() {
	viper.SetDefault("influx.host", "localhost")
	viper.SetDefault("influx.port", "8086")
	viper.SetDefault("influx.user", "")
	viper.SetDefault("influx.pass", "")
	viper.SetDefault("influx.secure", false)
	viper.SetDefault("influx.db", "infping")
	viper.SetDefault("influx.measurement", "infping")
	viper.SetDefault("fping.backoff", "1")
	viper.SetDefault("fping.retries", "0")
	viper.SetDefault("fping.tos", "0")
	viper.SetDefault("fping.summary", "10")
	viper.SetDefault("fping.period", "1000")
	viper.SetDefault("fping.custom", map[string]string{})
	viper.SetDefault("hosts", []string{"localhost"})
	viper.SetDefault("hostname", mustHostname())
}

func readConfiguration() {
	viper.SetConfigName("infping")
	viper.AddConfigPath("/etc/")
	viper.AddConfigPath("/etc/infping/")
	viper.AddConfigPath("/usr/local/etc/")
	viper.AddConfigPath("/usr/local/etc/infping/")
	viper.AddConfigPath("/config/")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("Unable to read config file", err)
	}
}

func createInfluxClient() *InfluxClient {
	influxScheme := "https"
	if !viper.GetBool("influx.secure") {
		influxScheme = "http"
	}

	influxHost := viper.GetString("influx.host")
	influxPort := viper.GetString("influx.port")
	influxUser := viper.GetString("influx.user")
	influxPass := viper.GetString("influx.pass")
	influxDB := viper.GetString("influx.db")
	influxMeasurement, err := parsePrefixTemplate(viper.GetString("influx.measurement"))
	influxRetPolicy := viper.GetString("influx.policy")

	if err != nil {
		log.Fatal("Unable to parse measurement template", err)
	}

	u, err := url.Parse(fmt.Sprintf("%s://%s:%s", influxScheme, influxHost, influxPort))
	if err != nil {
		log.Fatal("Unable to build valid Influx URL", err)
	}

	conf := client.HTTPConfig {
		Addr:      u.String(),
		Username:  influxUser,
		Password:  influxPass,
		UserAgent: "infping",
	}

	rawClient, err := client.NewHTTPClient(conf)
	if err != nil {
		log.Fatal("Failed to create Influx client", err)
	}

	return NewInfluxClient(rawClient, influxDB, influxMeasurement, influxRetPolicy)
}

func sendPingToInflux(influxClient *InfluxClient) {
	dur, version, err := influxClient.Ping()
	if err != nil {
		log.Fatal("Unable to ping InfluxDB", err)
	}
	log.Printf("Pinged InfluxDB (version %s) in %v", version, dur)
}

func createDatabaseIfNotExist(influxClient *InfluxClient) {
	influxDB := viper.GetString("influx.db")
	q := client.Query { Command: "SHOW DATABASES" }
	databases, err := influxClient.Query(q)
	if err != nil {
		log.Fatal("Unable to list databases", err)
	}
	if len(databases.Results) != 1 {
		log.Fatalf("Expected 1 result in response, got %d", len(databases.Results))
	}
	if len(databases.Results[0].Series) != 1 {
		log.Fatalf("Expected 1 series in result, got %d", len(databases.Results[0].Series))
	}

	found := false
	for i := 0; i < len(databases.Results[0].Series[0].Values); i++ {
		if databases.Results[0].Series[0].Values[i][0] == influxDB {
			found = true
		}
	}

	if !found {
		q = client.Query{
			Command: fmt.Sprintf("CREATE DATABASE %s", influxDB),
		}
		_, err := influxClient.Query(q)
		if err != nil {
			log.Fatalf("Failed to create database %s %v", influxDB, err)
		}
		log.Printf("Created new database %s", influxDB)
	}
}

func prepareFpingConfiguration() map[string]string {
	fpingBackoff := viper.GetString("fping.backoff")
	fpingRetries := viper.GetString("fping.retries")
	fpingTos := viper.GetString("fping.tos")
	fpingSummary := viper.GetString("fping.summary")
	fpingPeriod := viper.GetString("fping.period")
	fpingConfig := map[string]string{
		"-B": fpingBackoff,
		"-r": fpingRetries,
		"-O": fpingTos,
		"-Q": fpingSummary,
		"-p": fpingPeriod,
		"-l": "",
		"-D": "",
	}

	fpingCustom := viper.GetStringMapString("fping.custom")
	for k, v := range fpingCustom {
		fpingConfig[k] = v
	}

	return fpingConfig
}

// mustHostname returns the local hostname or throws an error
func mustHostname() string {
	name, err := os.Hostname()
	if err != nil {
		panic("unable to find hostname " + err.Error())
	}
	return strings.ToLower(name)
}
