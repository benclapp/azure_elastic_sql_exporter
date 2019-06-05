package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"sync"

	"gopkg.in/yaml.v2"

	_ "github.com/denisenkom/go-mssqldb"
	logg "github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Version of azure_elastic_sql_exporter. Set at build time.
	Version = "0.0.0.dev"

	logger logg.Logger

	listenAddress = flag.String("web.listen-address", ":9596", "Address to listen on for web interface and telemetry.")
	metricsPath   = flag.String("web.telemetry-path", "/metrics", "Path under which to expose metrics.")
	configFile    = flag.String("config.file", "./config.yaml", "Specify the config file with the database credentials.")
)

const namespace = "azure_elastic_sql"

// Exporter implements prometheus.Collector.
type Exporter struct {
	dbs            []Database
	mutex          sync.RWMutex
	up             prometheus.Gauge
	cpuPercent     *prometheus.GaugeVec
	dataIO         *prometheus.GaugeVec
	logIO          *prometheus.GaugeVec
	memoryPercent  *prometheus.GaugeVec
	workPercent    *prometheus.GaugeVec
	sessionPercent *prometheus.GaugeVec
	storagePercent *prometheus.GaugeVec
	dbUp           *prometheus.GaugeVec
}

// NewExporter returns an initialized MS SQL Exporter.
func NewExporter(dbs []Database) *Exporter {
	return &Exporter{
		dbs:            dbs,
		up:             newGuage("up", "Was the last scrape of Azure SQL successful."),
		cpuPercent:     newGuageVec("cpu_percent", "Average compute utilization in percentage of the limit of the service tier."),
		dataIO:         newGuageVec("data_io", "Average I/O utilization in percentage based on the limit of the service tier."),
		logIO:          newGuageVec("log_io", "Average write resource utilization in percentage of the limit of the service tier."),
		memoryPercent:  newGuageVec("memory_percent", "Average Memory Usage In Percent"),
		workPercent:    newGuageVec("worker_percent", "Maximum concurrent workers (requests) in percentage based on the limit of the database’s service tier."),
		sessionPercent: newGuageVec("session_percent", "Maximum concurrent sessions in percentage based on the limit of the database’s service tier."),
		storagePercent: newGuageVec("storage_percent", "Amount of storage utilised in percent."),
		dbUp:           newGuageVec("db_up", "Is the database is accessible."),
	}
}

// Describe describes all the metrics exported by the MS SQL exporter.
func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.cpuPercent.Describe(ch)
	e.dataIO.Describe(ch)
	e.logIO.Describe(ch)
	e.memoryPercent.Describe(ch)
	e.workPercent.Describe(ch)
	e.sessionPercent.Describe(ch)
	e.storagePercent.Describe(ch)
	e.dbUp.Describe(ch)
	e.up.Describe(ch)
}

// Collect fetches the stats from MS SQL and delivers them as Prometheus metrics. It implements prometheus.Collector.
func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	for _, db := range e.dbs {
		// logger.Log("Scraping", db.String())
		go e.scrapeDatabase(db)
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.cpuPercent.Collect(ch)
	e.dataIO.Collect(ch)
	e.logIO.Collect(ch)
	e.memoryPercent.Collect(ch)
	e.workPercent.Collect(ch)
	e.sessionPercent.Collect(ch)
	e.storagePercent.Collect(ch)
	e.dbUp.Collect(ch)
	e.up.Set(1)
}

func (e *Exporter) scrapeDatabase(d Database) {
	conn, err := sql.Open("mssql", d.DSN())
	if err != nil {
		e.mutex.Lock()
		defer e.mutex.Unlock()
		logger.Log("Failed to access database %s: %s", d, err)
		e.dbUp.WithLabelValues(d.Server).Set(0)
		return
	}
	defer conn.Close()
	query := "SELECT TOP 1 avg_cpu_percent, avg_data_io_percent, avg_log_write_percent, avg_instance_memory_percent, max_session_percent, max_worker_percent, avg_storage_percent FROM sys.elastic_pool_resource_stats ORDER BY end_time DESC"
	var cpu, data, logio, memory, session, worker, storage float64
	err = conn.QueryRow(query).Scan(&cpu, &data, &logio, &memory, &session, &worker, &storage)
	if err != nil {
		e.mutex.Lock()
		defer e.mutex.Unlock()
		logger.Log("Failed to query database %s: %s", d, err)
		e.dbUp.WithLabelValues(d.Server, "master").Set(0)
		return
	}
	e.mutex.Lock()
	defer e.mutex.Unlock()
	e.cpuPercent.WithLabelValues(d.Server, "master").Set(cpu)
	e.dataIO.WithLabelValues(d.Server, "master").Set(data)
	e.logIO.WithLabelValues(d.Server, "master").Set(logio)
	e.memoryPercent.WithLabelValues(d.Server, "master").Set(memory)
	e.workPercent.WithLabelValues(d.Server, "master").Set(worker)
	e.sessionPercent.WithLabelValues(d.Server, "master").Set(session)
	e.storagePercent.WithLabelValues(d.Server, "master").Set(storage)
	e.dbUp.WithLabelValues(d.Server, "master").Set(1)
}

// Database represents a MS SQL database connection.
type Database struct {
	Server   string
	User     string
	Password string
	Port     uint
}

// DSN returns the data source name as a string for the DB connection.
func (d Database) DSN() string {
	return fmt.Sprintf("server=%s;user id=%s;password=%s;port=%d;database=master", d.Server, d.User, d.Password, d.Port)
}

// DSN returns the data source name as a string for the DB connection with the password hidden for safe log output.
func (d Database) String() string {
	return fmt.Sprintf("server=%s;user id=%s;password=******;port=%d;database=master", d.Server, d.User, d.Port)
}

// Config contains all the required information for connecting to the databases.
type Config struct {
	Databases []Database
}

// NewConfig creates an instance of Config from a local YAML file.
func NewConfig(path string) (Config, error) {
	fh, err := ioutil.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("unable to read file %s: %s", path, err)
	}
	var config Config
	err = yaml.Unmarshal(fh, &config)
	if err != nil {
		return Config{}, fmt.Errorf("unable to unmarshal file %s: %s", path, err)
	}
	return config, nil
}

func newGuageVec(metricsName, docString string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      metricsName,
			Help:      docString,
		},
		[]string{"server", "database"},
	)
}

func newGuage(metricsName, docString string) prometheus.Gauge {
	return prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      metricsName,
			Help:      docString,
		},
	)
}

func init() {
	logger = logg.NewLogfmtLogger(logg.NewSyncWriter(os.Stderr))
	logger = logg.With(logger, "ts", logg.DefaultTimestampUTC, "caller", logg.DefaultCaller)
	logger.Log("msg", "Logger initialised")
}

func main() {
	flag.Parse()
	config, err := NewConfig(*configFile)
	if err != nil {
		log.Fatalf("Cannot open config file %s: %s", *configFile, err)
	}
	exporter := NewExporter(config.Databases)
	prometheus.MustRegister(exporter)
	http.Handle(*metricsPath, prometheus.Handler())
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
                <head><title>Azure Elastic SQL Exporter</title></head>
                <body>
                   <h1>Azure Elastic SQL Exporter</h1>
                   <p><a href='` + *metricsPath + `'>Metrics</a></p>
                   </body>
                </html>
              `))
	})
	logger.Log("Starting Server", *listenAddress)
	log.Fatal(http.ListenAndServe(*listenAddress, nil))
}
