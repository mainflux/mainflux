# InfluxDB reader

InfluxDB reader provides message repository implementation for InfluxDB.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                         | Description                                         | Default                        |
| -------------------------------- | --------------------------------------------------- | ------------------------------ |
| MF_INFLUX_READER_LOG_LEVEL       | Service log level                                   | info                           |
| MF_INFLUX_READER_HTTP_HOST       | Service HTTP host                                   | localhost                      |
| MF_INFLUX_READER_HTTP_PORT       | Service HTTP port                                   | 9005                           |
| MF_INFLUX_READER_SERVER_CERT     | Service HTTP server cert                            | ""                             |
| MF_INFLUX_READER_SERVER_KEY      | Service HTTP server key                             | ""                             |
| MF_INFLUXDB_PROTOCOL             | InfluxDB protocol                                   | http                           |
| MF_INFLUXDB_HOST                 | InfluxDB host name                                  | localhost                      |
| MF_INFLUXDB_PORT                 | Default port of InfluxDB database                   | 8086                           |
| MF_INFLUXDB_ADMIN_USER           | Default user of InfluxDB database                   | mainflux                       |
| MF_INFLUXDB_ADMIN_PASSWORD       | Default password of InfluxDB user                   | mainflux                       |
| MF_INFLUXDB_NAME                 | InfluxDB database name                              | mainflux                       |
| MF_INFLUXDB_BUCKET               | InfluxDB bucket name                                | mainflux-bucket                |
| MF_INFLUXDB_ORG                  | InfluxDB organization name                          | mainflux                       |
| MF_INFLUXDB_TOKEN                | InfluxDB API token                                  | mainflux-token                 |
| MF_INFLUXDB_DBURL                | InfluxDB database URL                               | ""                             |
| MF_INFLUXDB_USER_AGENT           | InfluxDB user agent                                 | ""                             |
| MF_INFLUXDB_TIMEOUT              | InfluxDB client connection readiness timeout        | 1s                             |
| MF_INFLUXDB_INSECURE_SKIP_VERIFY | InfluxDB insecure skip verify                       | false                          |
| MF_THINGS_AUTH_GRPC_URL          | Things service Auth gRPC URL                        | localhost:7000                 |
| MF_THINGS_AUTH_GRPC_TIMEOUT      | Things service Auth gRPC request timeout in seconds | 1s                             |
| MF_THINGS_AUTH_GRPC_CLIENT_TLS   | Flag that indicates if TLS should be turned on      | false                          |
| MF_THINGS_AUTH_GRPC_CA_CERTS     | Path to trusted CAs in PEM format                   | ""                             |
| MF_AUTH_GRPC_URL                 | Users service gRPC URL                              | localhost:7001                 |
| MF_AUTH_GRPC_TIMEOUT             | Users service gRPC request timeout in seconds       | 1s                             |
| MF_AUTH_GRPC_CLIENT_TLS          | Flag that indicates if TLS should be turned on      | false                          |
| MF_AUTH_GRPC_CA_CERTS            | Path to trusted CAs in PEM format                   | ""                             |
| MF_JAEGER_URL                    | Jaeger server URL                                   | http://jaeger:14268/api/traces |
| MF_SEND_TELEMETRY                | Send telemetry to mainflux call home server         | true                           |
| MF_INFLUX_READER_INSTANCE_ID     | InfluxDB reader instance ID                         |                                |

## Deployment

The service itself is distributed as Docker container. Check the [`influxdb-reader`](https://github.com/mainflux/mainflux/blob/master/docker/addons/influxdb-reader/docker-compose.yml#L17-L40) service section in docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the influxdb-reader
make influxdb-reader

# copy binary to bin
make install

# Set the environment variables and run the service
MF_INFLUX_READER_LOG_LEVEL=[Service log level] \
MF_INFLUX_READER_HTTP_HOST=[Service HTTP host] \
MF_INFLUX_READER_HTTP_PORT=[Service HTTP port] \
MF_INFLUX_READER_HTTP_SERVER_CERT=[Service HTTP server certificate] \
MF_INFLUX_READER_HTTP_SERVER_KEY=[Service HTTP server key] \
MF_INFLUXDB_PROTOCOL=[InfluxDB protocol] \
MF_INFLUXDB_HOST=[InfluxDB database host] \
MF_INFLUXDB_PORT=[InfluxDB database port] \
MF_INFLUXDB_ADMIN_USER=[InfluxDB admin user] \
MF_INFLUXDB_ADMIN_PASSWORD=[InfluxDB admin password] \
MF_INFLUXDB_NAME=[InfluxDB database name] \
MF_INFLUXDB_BUCKET=[InfluxDB bucket] \
MF_INFLUXDB_ORG=[InfluxDB org] \
MF_INFLUXDB_TOKEN=[InfluxDB token] \
MF_INFLUXDB_DBURL=[InfluxDB database URL] \
MF_INFLUXDB_USER_AGENT=[InfluxDB user agent] \
MF_INFLUXDB_TIMEOUT=[InfluxDB timeout] \
MF_INFLUXDB_INSECURE_SKIP_VERIFY=[InfluxDB insecure skip verify] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth gRPC URL] \
MF_THINGS_AURH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MF_THINGS_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MF_THINGS_AUTH_GRPC_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_AUTH_GRPC_URL=[Users service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Users service gRPC request timeout in seconds] \
MF_AUTH_GRPC_CLIENT_TLS=[Flag that indicates if TLS should be turned on] \
MF_AUTH_GRPC_CA_CERTS=[Path to trusted CAs in PEM format] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to mainflux call home server] \
MF_INFLUX_READER_INSTANCE_ID=[InfluxDB reader instance ID] \
$GOBIN/mainflux-influxdb

```

### Using docker-compose

This service can be deployed using docker containers. Docker compose file is
available in `<project_root>/docker/addons/influxdb-reader/docker-compose.yml`.
In order to run all Mainflux core services, as well as mentioned optional ones,
execute following command:

```bash
docker-compose -f docker/docker-compose.yml up -d
docker-compose -f docker/addons/influxdb-reader/docker-compose.yml up -d
```

And, to use the default .env file, execute the following command:

```bash
docker-compose -f docker/addons/influxdb-reader/docker-compose.yml up --env-file docker/.env -d
```

## Usage

Service exposes [HTTP API](https://api.mainflux.io/?urls.primaryName=readers-openapi.yml) for fetching messages.

Comparator Usage Guide:
| Comparator | Usage | Example |  
|----------------------|-----------------------------------------------------------------------------|------------------------------------|
| eq | Return values that are equal to the query | eq["active"] -> "active" |  
| ge | Return values that are substrings of the query | ge["tiv"] -> "active" and "tiv" |  
| gt | Return values that are substrings of the query and not equal to the query | gt["tiv"] -> "active" |  
| le | Return values that are superstrings of the query | le["active"] -> "tiv" |  
| lt | Return values that are superstrings of the query and not equal to the query | lt["active"] -> "active" and "tiv" |

Official docs can be found [here](https://docs.mainflux.io).
