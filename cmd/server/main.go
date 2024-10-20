package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()

func startRedis(cli *client.Client) {
	fmt.Println("Starting Redis via Docker")

	reader, err := cli.ImagePull(ctx, "docker.io/library/redis", image.PullOptions{})

	if err != nil{
		panic(err)
	}

	defer reader.Close()

	io.Copy(os.Stdout, reader)

	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "redis",
		Cmd: []string{},
		Tty: false,

	}, &container.HostConfig{
		PortBindings: map[nat.Port][]nat.PortBinding{
			"6379/tcp": {
				{
					HostIP:   "0.0.0.0",
					HostPort: "6379",
				},
			},
		},
	}, nil, nil, "")

	if err != nil{
		panic(err)
	}

	
	if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil{
		panic(err)
	}
	
	if err := waitForRedis("localhost", "6379", 30*time.Second); err != nil {
		log.Fatalf("Redis did not start within the expected time: %v", err)
	}
}	

func main(){
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())

	if err != nil{
		panic(err)
	}

	defer cli.Close()

	filterArgs := filters.NewArgs()
	filterArgs.Add("ancestor", "redis")

	containers, err := cli.ContainerList(ctx, container.ListOptions{Filters: filterArgs})

	if err != nil{
		panic(err)
	}

	if len(containers) == 0 {
		defer startRedis(cli) 
	}

	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
	})

	pong, err := rdb.Ping(ctx).Result()

	if err != nil{
		panic("Couldn't connect to Redis")
	}

	fmt.Println("Connected to Redis:", pong)
}

func waitForRedis(host, port string, timeout time.Duration) error {
	address := net.JoinHostPort(host, port)
	deadline := time.Now().Add(timeout)

	initialDelay := 500 * time.Millisecond
	maxDelay := 5 * time.Second
	delay := initialDelay

	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 1*time.Second)
		if err == nil {
			conn.Close()
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