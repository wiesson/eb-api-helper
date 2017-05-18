# eb-api-helper

Fetches and calculates the total energy data in kWh for a given timeframe from the energybox API.

### Installation

```
go get github.com/energybox/eb-api-helper
cd $GOPATH/src/github.com/energybox/eb-api-helper
go install
```

### Usage

`eb-api-helper -from=2017-05-15 -to=2017-05-16 -logger=577bd2ba65622d0635000015`

### Required Arguments

#### loggerId
`-logger=577bd2ba65622d0635000015`

### Additional arguments

#### from (default start of yesterday)
`-from=2017-05-15`

#### to (default end of yesterday)
`-to=2017-05-16`

#### Timezone (default UTC)
`-tz=Europe/Berlin` or `-tz=America/Los_Angeles`

#### SensorType (default main)
`-type=main` or `-type=ct`

### Example Output:

```
❯❯❯ eb-api-helper -from=2017-05-15 -to=2017-05-16 -logger=577bd2ba65622d0635000015 -tz=America/Los_Angeles
2017-05-15 00:00:00 -0700 PDT 2017-05-16 00:00:00 -0700 PDT
5859022f65622d3ba70001e9: 0.5592 kWh, 5859022f65622d3ba70001ea: 3.1209 kWh, 5859022f65622d3ba70001eb: 3.7796 kWh, total: 7.4597 kWh, days_1, count: 1
5859022f65622d3ba70001e9: 0.4687 kWh, 5859022f65622d3ba70001ea: 2.9058 kWh, 5859022f65622d3ba70001eb: 2.8199 kWh, total: 6.1944 kWh, hours_1, count: 24
5859022f65622d3ba70001e9: 0.4687 kWh, 5859022f65622d3ba70001ea: 2.9058 kWh, 5859022f65622d3ba70001eb: 2.8199 kWh, total: 6.1944 kWh, minutes_15, count: 96
5859022f65622d3ba70001e9: 0.4687 kWh, 5859022f65622d3ba70001ea: 2.9058 kWh, 5859022f65622d3ba70001eb: 2.8199 kWh, total: 6.1944 kWh, minutes_1, count: 1440
```
