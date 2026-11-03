//
// Copyright Strimzi authors.
// License: Apache License 2.0 (see the file LICENSE or http://apache.org/licenses/LICENSE-2.0.html).
//

package service_manager

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/IBM/sarama"
	"github.com/strimzi/strimzi-canary/internal/config"
)

// Implementation of Service Manager
type ServiceManager struct {
	CanaryConfig
	Paths
	canaryCmd    *exec.Cmd
	canaryExited chan struct{}
}

// Configurations of canary that is manipulated in e2e tests
type CanaryConfig struct {
	ReconcileIntervalTime string
	TopicTestName         string
	KafkaBrokerAddress    string
	// Kafka version used for the broker container image and the canary KAFKA_VERSION
	KafkaVersion string
}

// paths to services ( kafka docker compose, Canary application main method and the built Canary binary)
type Paths struct {
	pathDockerComposeKafka string
	pathToCanaryMain       string
	pathToCanaryBinary     string
}

const (
	canaryTestTopicName         = "__strimzi_canary_test_topicv123"
	kafkaBrokerAddress          = "127.0.0.1:9092"
	canaryReconcileIntervalTime = "1000"

	pathToDockerComposeImage = "compose-kafka-kraft.yaml"
	pathToMainMethod         = "../cmd/main.go"

	// env variable used by the docker compose file for selecting the Kafka image version
	kafkaImageVersionEnvVar = "KAFKA_IMAGE_VERSION"
)

func (c *ServiceManager) StartKafkaContainer(kafkaVersion string) {
	log.Printf("Starting kafka %s (KRaft)", kafkaVersion)
	c.KafkaVersion = kafkaVersion
	os.Setenv(kafkaImageVersionEnvVar, kafkaVersion)

	errComposingContainers := c.executeDockerCompose(
		"start kafka container using docker compose",
		"-f", c.pathDockerComposeKafka, "up", "-d", "--remove-orphans",
	)

	if errComposingContainers != nil {
		log.Fatal(errComposingContainers.Error())
	}
	log.Println("Kafka container created")
	// after creation of container we still have to wait for some time before successful communication with Kafka
	c.waitForBroker()
}

func (c *ServiceManager) StopKafkaContainer() {
	log.Println("Stopping kafka")
	errStoppingContainers := c.executeDockerCompose(
		"stop kafka container using docker compose",
		"-f", c.pathDockerComposeKafka, "down", "--remove-orphans",
	)
	if errStoppingContainers != nil {
		log.Fatal(errStoppingContainers.Error())
	}
}

func CreateManager() *ServiceManager {
	manager := &ServiceManager{}

	manager.ReconcileIntervalTime = canaryReconcileIntervalTime
	manager.TopicTestName = canaryTestTopicName
	manager.pathToCanaryMain = pathToMainMethod
	manager.pathDockerComposeKafka = pathToDockerComposeImage
	manager.KafkaBrokerAddress = kafkaBrokerAddress
	return manager
}

// BuildCanary builds the Canary binary once into the provided directory, so that it can be started (and stopped) for each Kafka version
func (c *ServiceManager) BuildCanary(dir string) {
	log.Println("Building Canary")
	c.pathToCanaryBinary = filepath.Join(dir, "strimzi-canary")
	buildCmd := exec.Command("go", "build", "-o", c.pathToCanaryBinary, c.pathToCanaryMain)
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		log.Fatalf("failed to build canary: %v\n%s", err, output)
	}
}

func (c *ServiceManager) StartCanary() {
	log.Println("Starting Canary")
	c.setUpCanaryParamsViaEnv()
	var wg sync.WaitGroup
	wg.Add(1)

	myCmd := exec.Command(c.pathToCanaryBinary)

	// merge stdout and stderr, so that canary logs keep being drained and are visible in the test output
	output, outputWriter := io.Pipe()
	myCmd.Stdout = outputWriter
	myCmd.Stderr = outputWriter

	go func() {
		started := false
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			msg := scanner.Text()
			log.Printf("[canary] %s", msg)
			if !started && strings.Contains(msg, "Starting canary manager") {
				started = true
				wg.Done()
			}
		}
	}()

	if err := myCmd.Start(); err != nil {
		log.Fatal(err.Error())
	}

	c.canaryCmd = myCmd
	c.canaryExited = make(chan struct{})
	go func() {
		err := myCmd.Wait()
		log.Printf("canary process exited: %v", err)
		_ = outputWriter.Close()
		close(c.canaryExited)
	}()

	if waitTimeout(&wg, time.Second*30) {
		_ = myCmd.Process.Kill()
		log.Fatal("canary failed to start within allowed time")
	}
	log.Println("Canary is ready")
}

// StopCanary gracefully stops the Canary, killing it if it doesn't exit within the allowed time
func (c *ServiceManager) StopCanary() {
	if c.canaryCmd == nil {
		return
	}
	log.Println("Stopping Canary")
	_ = c.canaryCmd.Process.Signal(syscall.SIGTERM)
	select {
	case <-c.canaryExited:
	case <-time.After(30 * time.Second):
		log.Println("canary didn't stop within allowed time, killing it")
		_ = c.canaryCmd.Process.Kill()
		<-c.canaryExited
	}
	c.canaryCmd = nil
}

// per se it means waiting for container's broker to communicate correctly
func (c *ServiceManager) waitForBroker() {
	log.Println("start waiting for broker")
	timeout := time.After(120 * time.Second)
	brokerIsReadyChannel := make(chan bool)

	go func() {
		configuration := sarama.NewConfig()
		brokers := []string{c.KafkaBrokerAddress}

		for {
			// if we can create Cluster Admin, broker can communicate
			if _, err := sarama.NewClusterAdmin(brokers, configuration); err != nil {
				log.Println("waiting for broker's start")
				time.Sleep(time.Millisecond * 500)
				continue
			}
			break
		}

		brokerIsReadyChannel <- true
	}()

	select {
	case <-timeout:
		log.Println("Broker isn't ready within expected timeout")
		errObtainingLogs := c.executeDockerCompose(
			"obtain logs from kafka container",
			"-f", c.pathDockerComposeKafka, "logs",
		)
		if errObtainingLogs != nil {
			log.Println("Problem obtaining logs from kafka container")
			log.Fatal(errObtainingLogs.Error())
		}
		log.Fatal("containers are not in suitable state")
	case <-brokerIsReadyChannel:
		log.Println("Container (Broker) is ready")
	}
}

// prefer the docker compose plugin (v2) and fall back to the standalone docker-compose binary
func (c *ServiceManager) executeDockerCompose(commandDescription string, commandArgs ...string) error {
	if exec.Command("docker", "compose", "version").Run() == nil {
		return c.executeCmdWithLogging(commandDescription, "docker", append([]string{"compose"}, commandArgs...)...)
	}
	return c.executeCmdWithLogging(commandDescription, "docker-compose", commandArgs...)
}

func (c *ServiceManager) executeCmdWithLogging(commandDescription, commandName string, commandArgs ...string) error {

	cmd := exec.Command(commandName, commandArgs...)
	var stdout, stderr bytes.Buffer
	// redirect Stdout and Stderr of command to buffers
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	// execute command
	err := cmd.Run()
	// log Stdout Stderr from command (stored within buffer)
	outStr, errStr := string(stdout.Bytes()), string(stderr.Bytes())
	log.Printf("cmd description: %s\n", commandDescription)
	log.Printf("execute cmd: %s %s\n", commandName, strings.Join(commandArgs, " "))
	log.Printf("cmd stdout:\n%s", outStr)
	log.Printf("cmd stderr:\n%s", errStr)
	return err
}

func (c *ServiceManager) setUpCanaryParamsViaEnv() {
	log.Println("Setting up environment variables")
	os.Setenv(config.ReconcileIntervalEnvVar, c.ReconcileIntervalTime)
	os.Setenv(config.TopicEnvVar, c.TopicTestName)
	os.Setenv(config.BootstrapServersEnvVar, c.KafkaBrokerAddress)
	os.Setenv(config.KafkaVersionEnvVar, c.KafkaVersion)
}

func waitTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	c := make(chan struct{})
	go func() {
		defer close(c)
		wg.Wait()
	}()
	select {
	case <-c:
		return false
	case <-time.After(timeout):
		return true
	}
}
