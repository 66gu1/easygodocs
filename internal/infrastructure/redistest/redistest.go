//go:build testutil

package redistest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	tc "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// StartRedis starts a Redis container and returns its address.
func StartRedis() (addr string, cleanup func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	container, err := tc.GenericContainer(ctx, tc.GenericContainerRequest{
		ContainerRequest: tc.ContainerRequest{
			Image:        "redis:7-alpine",
			ExposedPorts: []string{"6379/tcp"},
			WaitingFor: wait.ForListeningPort("6379/tcp").
				WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		panic(fmt.Sprintf("failed to start redis container: %v", err))
	}

	host, err := container.Host(ctx)
	if err != nil {
		_ = container.Terminate(context.Background()) //nolint:errcheck
		panic(fmt.Sprintf("get container host: %v", err))
	}
	port, err := container.MappedPort(ctx, "6379/tcp")
	if err != nil {
		_ = container.Terminate(context.Background()) //nolint:errcheck
		panic(fmt.Sprintf("get mapped port: %v", err))
	}

	cleanup = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		_ = container.Terminate(ctx) //nolint:errcheck
	}

	return fmt.Sprintf("%s:%s", host, port.Port()), cleanup
}

// NewClient returns a client for the address, closed when the test ends.
func NewClient(t testing.TB, addr string) *redis.Client {
	t.Helper()

	client := redis.NewClient(&redis.Options{Addr: addr, ContextTimeoutEnabled: true})
	t.Cleanup(func() { _ = client.Close() }) //nolint:errcheck
	return client
}
