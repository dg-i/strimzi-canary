//
// Copyright Strimzi authors.
// License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
//

//go:build e2e

package test

import (
	"log"
	"os"
	"strings"
	"testing"

	"github.com/strimzi/strimzi-canary/test/service_manager"
)

var (
	serviceManager *service_manager.ServiceManager

	// Kafka versions the e2e tests run against, it can be overridden with a comma separated list in the E2E_KAFKA_VERSIONS env variable
	kafkaVersions = []string{"4.0.0", "4.1.0", "4.2.0"}
)

const kafkaVersionsEnvVar = "E2E_KAFKA_VERSIONS"

func TestMain(m *testing.M) {
	if versions, ok := os.LookupEnv(kafkaVersionsEnvVar); ok && versions != "" {
		kafkaVersions = strings.Split(versions, ",")
	}

	binaryDir, err := os.MkdirTemp("", "strimzi-canary-e2e")
	if err != nil {
		log.Fatal(err.Error())
	}
	serviceManager = service_manager.CreateManager()
	serviceManager.BuildCanary(binaryDir)

	log.Printf("Starting tests against Kafka versions %v", kafkaVersions)
	code := m.Run()

	// defer has no usage here.
	_ = os.RemoveAll(binaryDir)
	// returning exit code of testing
	os.Exit(code)
}

// runs all the e2e tests against a single node KRaft kafka cluster (on default port 9092) for each Kafka version
func TestKafkaVersions(t *testing.T) {
	for _, kafkaVersion := range kafkaVersions {
		t.Run("Kafka-"+kafkaVersion, func(t *testing.T) {
			serviceManager.StartKafkaContainer(kafkaVersion)
			defer serviceManager.StopKafkaContainer()
			serviceManager.StartCanary()
			defer serviceManager.StopCanary()

			t.Run("CanaryTopicLiveness", testCanaryTopicLiveness)
			t.Run("EndpointsAvailability", testEndpointsAvailability)
			t.Run("MetricServerPrometheusContent", testMetricServerPrometheusContent)
			t.Run("MetricServerCanaryContent", testMetricServerCanaryContent)
		})
	}
}
