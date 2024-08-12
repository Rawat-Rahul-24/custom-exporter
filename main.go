package main

import (
    "encoding/csv"
    "log"
    "math"
    "net/http"
    "os"
    "os/exec"
    "strconv"
    "strings"
    "time"

    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
    frequencyDesc = prometheus.NewDesc(
        "raspberry_pi_frequency_hertz",
        "CPU Frequency of the Raspberry Pi",
        nil, nil,
    )
    utilizationDesc = prometheus.NewDesc(
        "raspberry_pi_cpu_utilization",
        "CPU Utilization of the Raspberry Pi",
        nil, nil,
    )
    powerDesc = prometheus.NewDesc(
        "raspberry_pi_power_watts",
        "Estimated power consumption of the Raspberry Pi",
        nil, nil,
    )

    powerData = make(map[string]float64)
)

type Exporter struct{}

func NewExporter() *Exporter {
    return &Exporter{}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
    ch <- frequencyDesc
    ch <- utilizationDesc
    ch <- powerDesc
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
    frequency := getFrequency()
    utilization := getCPUUtilization()

    power := estimatePower(frequency, utilization)

    ch <- prometheus.MustNewConstMetric(frequencyDesc, prometheus.GaugeValue, frequency)
    ch <- prometheus.MustNewConstMetric(utilizationDesc, prometheus.GaugeValue, utilization)
    ch <- prometheus.MustNewConstMetric(powerDesc, prometheus.GaugeValue, power)
}

func getFrequency() float64 {
    out, err := exec.Command("/usr/bin/vcgencmd", "measure_clock", "arm").Output()
    if err != nil {
        log.Println("Error measuring frequency:", err)
        return 0
    }

    output := string(out)
    parts := strings.Split(output, "=")
    if len(parts) == 2 {
        freqStr := strings.TrimSpace(parts[1])
        freq, err := strconv.ParseFloat(freqStr, 64)
        if err == nil {
            freqKHz := freq / 1e3
            log.Println("CPU Frequency:", freqKHz) // Convert to MHz
            return freqKHz
        }
    }

    log.Println("Error parsing frequency:", output)
    return 0
}

func getCPUUtilization() float64 {
    out, err := exec.Command("sh", "-c", "top -bn2 | grep 'Cpu(s)' | tail -n 1 | awk '{print $2 + $4 + $6}'").Output()
    if err != nil {
        log.Println("Error measuring CPU utilization:", err)
        return 0
    }

    utilization, err := strconv.ParseFloat(strings.TrimSpace(string(out)), 64)
    if err != nil {
        log.Println("Error parsing CPU utilization:", err)
        return 0
    }

    log.Println("CPU Utilization:", utilization)
    return utilization
}

func estimatePower(frequency float64, utilization float64) float64 {
    // Round frequency to nearest 100 MHz and utilization to nearest 10%
    roundedFreq := roundToNearest(frequency, 100000)
    roundedUtilization := roundToNearest(utilization, 10)

    key := strconv.FormatFloat(roundedFreq, 'f', 0, 64) + "_" + strconv.FormatFloat(roundedUtilization, 'f', 0, 64)

    if power, exists := powerData[key]; exists {
        return power
    }

    log.Println("No power data for frequency:", roundedFreq, "and utilization:", roundedUtilization)
    return 0
}

func roundToNearest(value float64, nearest float64) float64 {
    return math.Round(value/nearest) * nearest
}

func loadPowerData() {
    file, err := os.Open("/usr/local/bin/cpu_changing_workload_df.csv")
    if err != nil {
        log.Fatal("Error opening power data file:", err)
    }
    defer file.Close()

    reader := csv.NewReader(file)
    records, err := reader.ReadAll()
    if err != nil {
        log.Fatal("Error reading power data file:", err)
    }

    for _, record := range records {
        frequency, _ := strconv.ParseFloat(record[0], 64)
        utilization, _ := strconv.ParseFloat(record[1], 64)
        power, _ := strconv.ParseFloat(record[2], 64)

        key := strconv.FormatFloat(frequency, 'f', 0, 64) + "_" + strconv.FormatFloat(utilization, 'f', 0, 64)
        powerData[key] = power
    }
}

func main() {
    loadPowerData()

    exporter := NewExporter()
    prometheus.MustRegister(exporter)

    http.Handle("/metrics", promhttp.Handler())
    go func() {
        for {
            time.Sleep(20 * time.Second)
        }
    }()

    log.Println("Beginning to serve on port :8080")
    log.Fatal(http.ListenAndServe(":8080", nil))
}
