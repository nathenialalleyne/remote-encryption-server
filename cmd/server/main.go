package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/go-redis/redis/v8"

	"github.com/nathenialalleyne/remote-encryption-service/internal/handlers"
	"github.com/nathenialalleyne/remote-encryption-service/pkg/helpers"
)

var ctx = context.Background()

//TODO: Close and destroy redis instance when program closes/crashes
//TODO: API for creating processes

func main() {
	// TODO: Connect to existing Redis instance if open
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	helpers.HandleError(err)
	defer cli.Close()

	port := findAvailablePort(6379)

	startRedis(cli, port)

	hostAddress := fmt.Sprintf("localhost:%d", port)
	rdb := redis.NewClient(&redis.Options{
		Addr:     hostAddress,
		Password: "",
		DB:       0,
	})

	err = waitForRedis("localhost", strconv.Itoa(port), 30*time.Second, rdb)
	helpers.HandleError(err)

	fmt.Printf("Connected to Redis on port %d\n Starting server on port 9000\n", port)

	http.HandleFunc("/encrypt", handlers.EncryptionHandler())

	http.ListenAndServe(":9000", nil)
}

func startRedis(cli *client.Client, port int) {
	//TODO: Add DB Username and Password with cat'ed redis config file
	//TODO: Check if port is used prior to creating a new container
	fmt.Printf("Starting Redis on port %d via Docker...\n", port)

	reader, err := cli.ImagePull(ctx, "docker.io/library/redis", image.PullOptions{})
	helpers.HandleError(err)
	defer reader.Close()

	portStr := strconv.Itoa(port)
	binding := nat.Port(portStr + "/tcp")

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "redis",
		Cmd:   []string{"redis-server", "--port", portStr},
		Tty:   false,
		ExposedPorts: nat.PortSet{
			binding: struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			binding: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: portStr,
				},
			},
		},
	}, nil, nil, "")
	helpers.HandleError(err)

	err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil && strings.Contains(err.Error(), "port is already allocated") {
		fmt.Printf("Port %d is already allocated, trying next port\n", port)
		return
	}
	helpers.HandleError(err)

	fmt.Printf("Redis started on port %d\n", port)
}

func waitForRedis(host, port string, timeout time.Duration, rdb *redis.Client) error {
	address := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)

	delay := 500 * time.Millisecond
	maxDelay := 5 * time.Second

	for time.Now().Before(deadline) {
		_, err := rdb.Ping(ctx).Result()
		if err == nil {
			return nil
		}

		fmt.Printf("Waiting for Redis to be ready, retrying in %v...\n", delay)
		time.Sleep(delay)

		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
		}
	}

	return fmt.Errorf("timeout: Redis did not become available on %s within %v", address, timeout)
}

func findAvailablePort(startPort int) int {
	port := startPort
	for isPortTaken(port) {
		port++
	}
	return port
}

func isPortTaken(port int) bool {
	address := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		fmt.Printf("Port %d is in use\n", port)
		return true
	}
	defer listener.Close()
	return false
}

