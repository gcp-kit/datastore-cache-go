//+abuild emulator,redis
package cache_test

import (
	"context"
	"os"
	"testing"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/gcp-kit/datastore-cache-go/cache"
	"github.com/gcp-kit/datastore-cache-go/cache/redis"
	redigo "github.com/gomodule/redigo/redis"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func initMiddleware(t *testing.T) *cache.Middleware {
	t.Helper()

	addr := os.Getenv("REDIS_ADDR")

	if len(addr) == 0 {
		t.Fatalf("$REDIS_ADDR is not set")
	}

	pool := &redigo.Pool{
		MaxIdle:     3,
		MaxActive:   0,
		IdleTimeout: 240 * time.Second,
		Dial:        func() (redigo.Conn, error) { return redigo.Dial("tcp", addr) },
	}

	return cache.NewMiddleware(redis.NewRedis(pool))
}

func initClient(t *testing.T) *datastore.Client {
	t.Helper()

	middleware := initMiddleware(t)

	if os.Getenv("DATASTORE_EMULATOR_HOST") == "" {
		os.Setenv("DATASTORE_EMULATOR_HOST", "localhost:8000")
	}

	os.Setenv("DATASTORE_PROJECT_ID", "project-id-in-google")

	client, err := datastore.NewClient(
		context.Background(),
		"",
		option.WithGRPCDialOption(
			grpc.WithUnaryInterceptor(middleware.UnaryClientInterceptor),
		),
	)

	if err != nil {
		t.Fatalf("failed to initialize datastore client: %+v", err)
	}

	return client
}

func initNonCachedClient(t *testing.T) *datastore.Client {
	t.Helper()

	os.Setenv("DATASTORE_PROJECT_ID", "project-id-in-google")

	client, err := datastore.NewClient(
		context.Background(),
		"",
	)

	if err != nil {
		t.Fatalf("failed to initialize datastore client: %+v", err)
	}

	return client
}

type TestData struct {
	Name string
}

func TestEmulator_SetGet(t *testing.T) {
	client := initClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	key := datastore.IncompleteKey("test", nil)

	data := &TestData{Name: "foo"}

	key, err := client.Put(ctx, key, data)

	if err != nil {
		t.Fatalf("failed to put new data: %+v", err)
	}

	var ret TestData

	if err := client.Get(ctx, key, &ret); err != nil {
		t.Fatalf("failed to get data: %+v", err)
	}

	if ret.Name != data.Name {
		t.Errorf("retrieved data differed: %s(expected: %s)", ret.Name, data.Name)
	}

	ret.Name = ""

	if err := client.Get(ctx, key, &ret); err != nil {
		t.Fatalf("failed to get data from cache: %+v", err)
	}

	if ret.Name != data.Name {
		t.Errorf("retrieved data FROM CACHE differed: %s(expected: %s)", ret.Name, data.Name)
	}
}

// TestEmulator_ConfirmCache - データをPutし、一度Getしキャッシュさせた後に直接datastoreから削除し、その後Getすることでキャッシュされていることを確かめる
func TestEmulator_ConfirmCache(t *testing.T) {
	client := initClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	key := datastore.IncompleteKey("test", nil)

	data := &TestData{Name: "foo"}

	key, err := client.Put(ctx, key, data)

	if err != nil {
		t.Fatalf("failed to put new data: %+v", err)
	}

	var ret TestData

	if err := client.Get(ctx, key, &ret); err != nil {
		t.Fatalf("failed to get data: %+v", err)
	}

	if ret.Name != data.Name {
		t.Errorf("retrieved data differed: %s(expected: %s)", ret.Name, data.Name)
	}

	// 実態をdatastoreからは削除することでRedisから取得できていることを確認する
	if err := initNonCachedClient(t).Delete(ctx, key); err != nil {
		t.Errorf("failed to delete data from datastore: %+v", err)
	}

	ret.Name = ""

	if err := client.Get(ctx, key, &ret); err != nil {
		t.Fatalf("failed to get data from cache: %+v", err)
	}

	if ret.Name != data.Name {
		t.Errorf("retrieved data FROM CACHE differed: %s(expected: %s)", ret.Name, data.Name)
	}
}

// TestEmulator_ConfirmCache - データをPutし、一度Getしキャッシュさせた後にPutし、その後Getすることでキャッシュが削除されていることを確かめる
func TestEmulator_ClearCache(t *testing.T) {
	client := initClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	key := datastore.IncompleteKey("test", nil)

	data := &TestData{Name: "foo"}

	key, err := client.Put(ctx, key, data)

	if err != nil {
		t.Fatalf("failed to put new data: %+v", err)
	}

	var ret TestData

	if err := client.Get(ctx, key, &ret); err != nil {
		t.Fatalf("failed to get data: %+v", err)
	}

	if ret.Name != data.Name {
		t.Errorf("retrieved data differed: %s(expected: %s)", ret.Name, data.Name)
	}

	data.Name = "bar"

	if _, err := client.Put(ctx, key, data); err != nil {
		t.Fatalf("failed to update entity: %+v", err)
	}

	ret.Name = ""

	if err := client.Get(ctx, key, &ret); err != nil {
		t.Fatalf("failed to get data from cache: %+v", err)
	}

	if ret.Name != data.Name {
		t.Errorf("retrieved data FROM CACHE differed: %s(expected: %s)", ret.Name, data.Name)
	}
}
