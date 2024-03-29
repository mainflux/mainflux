# Postgres reader

Postgres reader provides message repository implementation for Postgres.

## Configuration

The service is configured using the environment variables presented in the
following table. Note that any unset variables will be replaced with their
default values.

| Variable                            | Description                                   | Default                        |
| ----------------------------------- | --------------------------------------------- | ------------------------------ |
| MF_POSTGRES_READER_LOG_LEVEL        | Service log level                             | info                           |
| MF_POSTGRES_READER_HTTP_HOST        | Service HTTP host                             | localhost                      |
| MF_POSTGRES_READER_HTTP_PORT        | Service HTTP port                             | 9009                           |
| MF_POSTGRES_READER_HTTP_SERVER_CERT | Service HTTP server cert                      | ""                             |
| MF_POSTGRES_READER_HTTP_SERVER_KEY  | Service HTTP server key                       | ""                             |
| MF_POSTGRES_HOST                    | Postgres DB host                              | localhost                      |
| MF_POSTGRES_PORT                    | Postgres DB port                              | 5432                           |
| MF_POSTGRES_USER                    | Postgres user                                 | mainflux                       |
| MF_POSTGRES_PASS                    | Postgres password                             | mainflux                       |
| MF_POSTGRES_NAME                    | Postgres database name                        | messages                       |
| MF_POSTGRES_SSL_MODE                | Postgres SSL mode                             | disabled                       |
| MF_POSTGRES_SSL_CERT                | Postgres SSL certificate path                 | ""                             |
| MF_POSTGRES_SSL_KEY                 | Postgres SSL key                              | ""                             |
| MF_POSTGRES_SSL_ROOT_CERT           | Postgres SSL root certificate path            | ""                             |
| MF_THINGS_AUTH_GRPC_URL             | Things service Auth gRPC URL                  | localhost:7000                 |
| MF_THINGS_AUTH_GRPC_TIMEOUT         | Things service Auth gRPC timeout in seconds   | 1s                             |
| MF_THINGS_AUTH_GRPC_CLIENT_TLS      | Things service Auth gRPC TLS mode flag        | false                          |
| MF_THINGS_AUTH_GRPC_CA_CERTS        | Things service Auth gRPC CA certificates      | ""                             |
| MF_AUTH_GRPC_URL                    | Users service gRPC URL                        | localhost:7001                 |
| MF_AUTH_GRPC_TIMEOUT                | Users service gRPC request timeout in seconds | 1s                             |
| MF_AUTH_GRPC_CLIENT_TLS             | Users service gRPC TLS mode flag              | false                          |
| MF_AUTH_GRPC_CA_CERTS               | Users service gRPC CA certificates            | ""                             |
| MF_JAEGER_URL                       | Jaeger server URL                             | http://jaeger:14268/api/traces |
| MF_SEND_TELEMETRY                   | Send telemetry to mainflux call home server   | true                           |
| MF_POSTGRES_READER_INSTANCE_ID      | Postgres reader instance ID                   |                                |

## Deployment

The service itself is distributed as Docker container. Check the [`postgres-reader`](https://github.com/mainflux/mainflux/blob/master/docker/addons/postgres-reader/docker-compose.yml#L17-L41) service section in
docker-compose to see how service is deployed.

To start the service, execute the following shell script:

```bash
# download the latest version of the service
git clone https://github.com/mainflux/mainflux

cd mainflux

# compile the postgres writer
make postgres-writer

# copy binary to bin
make install

# Set the environment variables and run the service
MF_POSTGRES_READER_LOG_LEVEL=[Service log level] \
MF_POSTGRES_READER_HTTP_HOST=[Service HTTP host] \
MF_POSTGRES_READER_HTTP_PORT=[Service HTTP port] \
MF_POSTGRES_READER_HTTP_SERVER_CERT=[Service HTTPS server certificate path] \
MF_POSTGRES_READER_HTTP_SERVER_KEY=[Service HTTPS server key path] \
MF_POSTGRES_HOST=[Postgres host] \
MF_POSTGRES_PORT=[Postgres port] \
MF_POSTGRES_USER=[Postgres user] \
MF_POSTGRES_PASS=[Postgres password] \
MF_POSTGRES_NAME=[Postgres database name] \
MF_POSTGRES_SSL_MODE=[Postgres SSL mode] \
MF_POSTGRES_SSL_CERT=[Postgres SSL cert] \
MF_POSTGRES_SSL_KEY=[Postgres SSL key] \
MF_POSTGRES_SSL_ROOT_CERT=[Postgres SSL Root cert] \
MF_THINGS_AUTH_GRPC_URL=[Things service Auth GRPC URL] \
MF_THINGS_AUTH_GRPC_TIMEOUT=[Things service Auth gRPC request timeout in seconds] \
MF_THINGS_AUTH_GRPC_CLIENT_TLS=[Things service Auth gRPC TLS mode flag] \
MF_THINGS_AUTH_GRPC_CA_CERTS=[Things service Auth gRPC CA certificates] \
MF_AUTH_GRPC_URL=[Users service gRPC URL] \
MF_AUTH_GRPC_TIMEOUT=[Users service gRPC request timeout in seconds] \
MF_AUTH_GRPC_CLIENT_TLS=[Users service gRPC TLS mode flag] \
MF_AUTH_GRPC_CA_CERTS=[Users service gRPC CA certificates] \
MF_JAEGER_URL=[Jaeger server URL] \
MF_SEND_TELEMETRY=[Send telemetry to mainflux call home server] \
MF_POSTGRES_READER_INSTANCE_ID=[Postgres reader instance ID] \
$GOBIN/mainflux-postgres-reader
```

## Usage

Starting service will start consuming normalized messages in SenML format.

Comparator Usage Guide:

| Comparator | Usage                                                                       | Example                            |
| ---------- | --------------------------------------------------------------------------- | ---------------------------------- |
| eq         | Return values that are equal to the query                                   | eq["active"] -> "active"           |
| ge         | Return values that are substrings of the query                              | ge["tiv"] -> "active" and "tiv"    |
| gt         | Return values that are substrings of the query and not equal to the query   | gt["tiv"] -> "active"              |
| le         | Return values that are superstrings of the query                            | le["active"] -> "tiv"              |
| lt         | Return values that are superstrings of the query and not equal to the query | lt["active"] -> "active" and "tiv" |

Official docs can be found [here](https://docs.mainflux.io).
