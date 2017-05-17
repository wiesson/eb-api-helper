package main

import (
	"net/http"
	"fmt"
	"encoding/json"
	"net/url"
	"strconv"
	"time"
	"flag"
	"os"
	"sort"
)

type SamplesResponse struct {
	Sample []Sample `json:"data"`
}

type Sample struct {
	Type       string `json:"type"`
	Id         string `json:"id"`
	Attributes Attributes `json:"attributes"`
}

type Attributes struct {
	Timestamp         int64 `json:"timestamp"`
	SystemTemperature float32 `json:"system_temperature"`
	EnergySamples     []Samples `json:"energy"`
	PowerSamples      []Samples `json:"power"`
}

type Samples struct {
	SensorId string `json:"sensor_id"`
	Value    float64 `json:"value"`
}

type SamplesRequest struct {
	baseUrl    string
	dataLogger string
	timeFrom   int64
	timeTo     int64
}

func sumSamples(s SamplesResponse) (map[string]float64, int) {
	m := make(map[string]float64)
	amount := len(s.Sample)

	for _, sample := range s.Sample {
		for _, energy := range sample.Attributes.EnergySamples {
			m[energy.SensorId] += energy.Value
			m["total"] += energy.Value
		}
	}

	return m, amount
}

func formatCommandlineOutput(s SamplesResponse, aggregationLevel string) string {
	sumValues, samplesCount := sumSamples(s)

	var output string

	var sensorIds []string
	for k := range sumValues {
		sensorIds = append(sensorIds, k)
	}

	sort.Strings(sensorIds)
	for _, k := range sensorIds {
		output += fmt.Sprintf("%s: %.4f kWh, ", k, sumValues[k])
	}

	output += fmt.Sprintf("%s, count: %d", aggregationLevel, samplesCount)

	return output
}

func (a *SamplesRequest) GetSamples(aggregationLevel string, ch chan<- string) {
	s := &SamplesResponse{}

	payload := url.Values{}
	payload.Set("filter[data_logger]", a.dataLogger)
	payload.Add("filter[type]", "main")
	payload.Add("filter[from]", strconv.FormatInt(a.timeFrom, 10))
	payload.Add("filter[to]", strconv.FormatInt(a.timeTo, 10))
	payload.Add("aggregation_level", aggregationLevel)

	res, err := http.Get(a.baseUrl + "?" + payload.Encode())

	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(s)
	ch <- formatCommandlineOutput(*s, aggregationLevel)
}

func Bod(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func main() {
	start := Bod(time.Now().AddDate(0, 0, -2))
	from := start.AddDate(0, 0, 1)

	cmdFrom := flag.String("from", start.Format("2006-1-2"), "The lower date")
	cmdTo := flag.String("to", from.Format("2006-1-2"), "The upper date")
	logger := flag.String("logger", "", "Id of the data-logger")
	tz := flag.String("tz", "UTC", "The identifier of the timezone, Europe/Berlin")

	flag.Parse()

	if *logger == "" {
		fmt.Println("Please enter a logger id --logger=")
		os.Exit(0)
	}

	var loc, _ = time.LoadLocation(*tz)

	lower, _ := time.ParseInLocation("2006-1-2", *cmdFrom, loc)
	upper, _ := time.ParseInLocation("2006-1-2", *cmdTo, loc)

	fmt.Println(lower, upper)

	api := &SamplesRequest{
		baseUrl:    "https://api.internetofefficiency.com/v2/samples",
		dataLogger: *logger,
		timeFrom:   lower.Unix(),
		timeTo:     upper.Unix(),
	}

	ch := make(chan string)

	aggregationLevels := [4]string{"days_1", "hours_1", "minutes_15", "minutes_1"}

	for _, i := range aggregationLevels {
		go api.GetSamples(i, ch)
	}

	for range aggregationLevels {
		fmt.Println(<-ch)
	}
}