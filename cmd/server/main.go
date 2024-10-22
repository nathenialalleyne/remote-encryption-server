package main

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

func main(){
	//TODO: Connect to existing redis instance if open
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil{
		panic(err)
	}

	defer cli.Close()

	port := 6379

	for isPortTaken(port){
		port++
	}

	startRedis(cli, port)

	hostAddress := fmt.Sprintf("localhost:%d", port)

	rdb := redis.NewClient(&redis.Options{
		Addr: hostAddress,
		Password: "",
		DB: 0,
	})

	if err != nil{
		panic(err)
	}

	if err := waitForRedis("localhost", string(port), 30*time.Second, rdb); err != nil {
		panic("Redis did not start within the expected time")
	}
	
	fmt.Println("Connected to Redis")
}

func startRedis(cli *client.Client, port int) bool {
	//TODO: Add DB Username and Password with cat'ed redis config file
	//TODO: Check if port is used prior to creating a new container
	fmt.Println("Starting Redis via Docker")

	reader, err := cli.ImagePull(ctx, "docker.io/library/redis", image.PullOptions{})
	if err != nil {
		panic(err)
	}
	defer reader.Close()

	var binding nat.Port = nat.Port(fmt.Sprintf("%d/tcp", port))

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "redis",
		Cmd:   []string{"redis-server", "--port", strconv.Itoa(port)},
		Tty:   false,
		ExposedPorts: nat.PortSet{
			binding: struct{}{},
		},
	}, &container.HostConfig{
		PortBindings: nat.PortMap{
			binding: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: strconv.Itoa(port),
				},
			},
		},
	}, nil, nil, "")

	if err != nil {
		panic(err)
	}

	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		if strings.Contains(err.Error(), "port is already allocated") {
			fmt.Printf("Port %d is already allocated, trying port %d\n", port, port+1)
			return false
		} else {
			panic(err)
		}
	}
	return true
}

func waitForRedis(host, port string, timeout time.Duration, redisClient *redis.Client) error {
	//TODO: Figure out why it wont connect to freshly created containers
	address := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)

	initialDelay := 500 * time.Millisecond
	maxDelay := 5 * time.Second
	delay := initialDelay

	for time.Now().Before(deadline) {
		_, err := redisClient.Ping(ctx).Result()
		
		if err == nil{
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

func isPortTaken(port int) bool{
	address := fmt.Sprintf(":%d", port)

	listener, err := net.Listen("tcp", address)
	if err != nil{
		fmt.Printf("Port %d is in use. Attempting to connect to port %d\n", port, port+1)
		return true
	}
	listener.Close()
	return false
}