# Azure SQL Exporter



[Prometheus](https://prometheus.io/) exporter for Azure SQL metrics.

Databases are only queries when fetching /metrics from the exporter so that you may control the interval from your scrape_config section in Prometheus. Recommended to be no more frequent than `15s`, this is the frequency that the data is updated at.

## Install

```bash
go get -u github.com/benclapp/azure_elastic_sql_exporter
```

## Usage
```bash
Usage of azure_elastic_sql_exporter:
  -config.file string
    	Specify the config file with the database credentials. (default "./config.yaml")
  -web.listen-address string
    	Address to listen on for web interface and telemetry. (default ":9139")
  -web.telemetry-path string
    	Path under which to expose metrics. (default "/metrics")
```

## Configuration

This exporter requires a configuration file. By default, it will look for the config.yaml file in the PWD and can be specified with the -config.file parameter.

The file is in YAML format and contains the information for connecting to the databases you want to export. This file will contain sensitive information so make sure your configuration management locks down access to this file (chmod [46]00) and it is encouraged to create an SQL user with the least amount of privilege.

```yaml
databases:
  - user: prometheus
    port: 1433
    password: str0ngP@sswordG0esHere
    server: salesdb.database.windows.net

  - user: prometheus
    port: 1433
    password: str0ngP@sswordG0esHere
    server: inventorydb.database.windows.net
```


## Binary releases

Pre-compiled versions may be found in the [release section](https://github.com/benclapp/azure_elastic_sql_exporter/releases).

## Docker

Images are available on [Docker Hub](https://hub.docker.com/r/benclapp/azure_elastic_sql_exporter/). Example:

```bash
docker run -d -p 9139:9139 -v ./config.yaml:/config/config.yaml benclapp/azure_elastic_sql_exporter:latest -config.file /config/config.yaml
```