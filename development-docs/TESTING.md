# Testing

The available tests are grouped using two different build tags:

* `unit_test`: for the unit tests alongside the different Go files.
* `e2e`: for running an end-to-end test using a single node Kafka cluster in KRaft mode (via Docker compose, see `test/compose-kafka-kraft.yaml`) and the canary running alongside it, in order to verify that it's able to exchange messages.

Use the `go test` command in order to run the tests.

## Unit Tests

For running the unit tests:

```shell
go test ./internal/... --tags=unit_test
```

## End-to-end Tests

For running the e2e tests:

```shell
go test ./test/... --tags=e2e
```

The whole e2e test suite runs once for each supported Kafka version (4.0.0, 4.1.0 and 4.2.0), each time with a fresh Kafka cluster and canary instance.
The canary is configured with the same `KAFKA_VERSION` as the Kafka cluster.
For running the e2e tests against a different set of Kafka versions, use the `E2E_KAFKA_VERSIONS` env variable with a comma separated list:

```shell
E2E_KAFKA_VERSIONS=4.2.0 go test ./test/... --tags=e2e
```
