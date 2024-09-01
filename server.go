package main

import (
	"context"
	"fmt"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"go-graphql/graph"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/go-redis/redis"
)

const defaultPort = "8080"

type Cache struct {
	client redis.UniversalClient
	ttl    time.Duration
}

const apqPrefix = "apq:"

func NewCache(redisAddress string, ttl time.Duration) (*Cache, error) {
	client := redis.NewClient(&redis.Options{
		Addr: redisAddress,
	})

	err := client.Ping().Err()
	if err != nil {
		return nil, fmt.Errorf("could not create cache: %w", err)
	}

	return &Cache{client: client, ttl: ttl}, nil
}

func (c *Cache) Add(ctx context.Context, key string, value interface{}) {
	c.client.Set(apqPrefix+key, value, c.ttl)
}

func (c *Cache) Get(ctx context.Context, key string) (interface{}, bool) {
	s, err := c.client.Get(apqPrefix + key).Result()
	if err != nil {
		return struct{}{}, false
	}
	return s, true
}

func main() {
	// Get the port from environment variables
	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	// Create a new Redis cache with a 24-hour TTL
	redisAddress := "localhost:6379" // Replace with your Redis server address
	cache, err := NewCache(redisAddress, 24*time.Hour)
	if err != nil {
		log.Fatalf("cannot create APQ redis cache: %v", err)
	}

	// Set up the GraphQL handler
	srv := handler.New(graph.NewExecutableSchema(graph.Config{Resolvers: &graph.Resolver{}}))
	srv.AddTransport(transport.POST{})
	srv.Use(extension.AutomaticPersistedQuery{Cache: cache})

	// Set up HTTP routes
	http.Handle("/", playground.Handler("GraphQL playground", "/query"))
	http.Handle("/query", srv)

	// Start the server
	log.Printf("connect to http://localhost:%s/ for GraphQL playground", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
