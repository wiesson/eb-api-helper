package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
	"sort"
)

var sensorTypes = []string{
	"main",
	"ct",
}

type SamplesResponse struct {
	Sample []SamplesResponseData `json:"data"`
	Meta struct {
		SampleInterval uint `json:"sample_interval"`
	} `json:"meta"`
	Links struct {
		NextURL string `json:"next"`
	} `json:"links"`
}

type SamplesResponseData struct {
	Type string `json:"type"`
	Id   string `json:"id"`
	Attributes struct {
		Timestamp             int64            `json:"timestamp"`
		SystemTemperature     float32          `json:"system_temperature"`
		PowerResponseSamples  []ResponseSample `json:"power"`
		EnergyResponseSamples []ResponseSample `json:"energy"`
	} `json:"attributes"`
}

type ResponseSample struct {
	SensorID string  `json:"sensor_id"`
	Value    float64 `json:"value"`
}

type Reading float64

func (r Reading) String() string {
	return strconv.FormatFloat(float64(r), 'f', 8, 64)
}

type Sample struct {
	Timestamp int64
	DateTime  time.Time
	Samples   map[string]Reading
	Values    []float64
	Energy    []ResponseSample
}

type Data []Sample

func (d *Data) AddItem(value SamplesResponseData) {
	DateTime := time.Unix(value.Attributes.Timestamp, 0)

	row := &Sample{
		Timestamp: value.Attributes.Timestamp,
		DateTime:  DateTime,
		Samples:   make(map[string]Reading),
	}

	for _, sample := range value.Attributes.EnergyResponseSamples {
		row.Samples[sample.SensorID] = Reading(sample.Value)
		row.Values = append(row.Values, sample.Value)
		row.Energy = append(row.Energy, sample)
	}

	*d = append(*d, *row)
}

type API struct {
	baseUrl    string
	dataLogger string
	site       string
	timeFrom   int64
	timeTo     int64
	SensorType string
}

func sumSamples(d Data) (map[string]float64, int) {
	m := make(map[string]float64)
	amount := len(d)

	for _, sample := range d {
		for _, energy := range sample.Energy {
			m[energy.SensorID] += energy.Value
			m["total"] += energy.Value
		}
	}

	return m, amount
}

func formatCommandlineOutput(d Data, aggregationLevel string) string {
	sumValues, samplesCount := sumSamples(d)
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

func (a *API) GetRequestPath(path string, aggregationLevel string) string {
	if path != "" {
		return path
	}

	payload := url.Values{}
	payload.Set("aggregation_level", aggregationLevel)
	payload.Add("filter[from]", strconv.FormatInt(a.timeFrom, 10))
	payload.Add("filter[to]", strconv.FormatInt(a.timeTo, 10))
	payload.Add("filter[samples]", "timestamp,power,energy")

	if a.dataLogger != "" {
		payload.Add("filter[data_logger]", a.dataLogger)
	} else {
		payload.Add("filter[site]", a.site)
	}

	if a.SensorType != "" {
		payload.Add("filter[type]", a.SensorType)
	}

	return "/v2/samples/?" + payload.Encode()
}

func (a *API) Get(url string) (SamplesResponse, error) {
	res, err := http.Get(a.baseUrl + url)
	defer res.Body.Close()
	if err != nil {
		return SamplesResponse{}, err
	}

	s := &SamplesResponse{}
	err = json.NewDecoder(res.Body).Decode(s)
	if err != nil {
		return SamplesResponse{}, err
	}

	return *s, nil
}

func (a *API) GetSamples(aggregationLevel string, ch chan<- string) {
	d := &Data{}

	nextUrl := a.GetRequestPath("", aggregationLevel)
	hasNext := true

	for hasNext {
		s, err := a.Get(nextUrl)
		if err != nil {
			panic(err)
		}

		for _, value := range s.Sample {
			d.AddItem(value)
		}

		nextUrl = s.Links.NextURL
		if nextUrl == "" {
			hasNext = false
			break
		}
	}

	ch <- formatCommandlineOutput(*d, aggregationLevel)
}

func Bod(t time.Time) time.Time {
	year, month, day := t.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, t.Location())
}

func inSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func main() {
	start := Bod(time.Now().AddDate(0, 0, -2))
	from := start.AddDate(0, 0, 1)

	cmdFrom := flag.String("from", start.Format("2006-1-2"), "The lower date")
	cmdTo := flag.String("to", from.Format("2006-1-2"), "The upper date")
	logger := flag.String("logger", "", "Id of the data-logger")
	site := flag.String("site", "", "Id of the site")
	tz := flag.String("tz", "UTC", "The identifier of the timezone, Europe/Berlin")
	sensorType := flag.String("type", "main", "SensorType - main, ct")

	flag.Parse()

	if *logger == "" && *site == "" {
		fmt.Println("Please enter a logger id -logger <LOGGER_ID> or a site id -site <SITE_ID>")
		os.Exit(0)
	}

	if *sensorType != "main" {
		if inSlice(*sensorType, sensorTypes) == false {
			fmt.Println("Please use a valid energy type", sensorTypes)
			os.Exit(0)
		}
	}

	var loc, _ = time.LoadLocation(*tz)

	lower, _ := time.ParseInLocation("2006-1-2", *cmdFrom, loc)
	upper, _ := time.ParseInLocation("2006-1-2", *cmdTo, loc)

	fmt.Println(lower, upper)

	api := API{
		baseUrl:    "https://api.internetofefficiency.com",
		dataLogger: *logger,
		site:       *site,
		timeFrom:   lower.Unix(),
		timeTo:     upper.Unix(),
		SensorType: *sensorType,
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
