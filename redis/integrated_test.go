// +build redis

package redis_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gcp-kit/datastore-cache-go/redis"
	redigo "github.com/gomodule/redigo/redis"
	"github.com/google/go-cmp/cmp"
	"google.golang.org/genproto/googleapis/datastore/v1"
)

const (
	projectID = "project-id"
)

func newRedis(t *testing.T) *redis.Redis {
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

	return redis.NewRedis(pool)
}

var (
	entityResults = []*datastore.EntityResult{
		{
			Entity: &datastore.Entity{
				Key: &datastore.Key{
					PartitionId: &datastore.PartitionId{
						ProjectId:   "project-id",
						NamespaceId: "namespace-id",
					},
					Path: []*datastore.Key_PathElement{
						{
							Kind:   "kind",
							IdType: &datastore.Key_PathElement_Id{Id: 10},
						},
					},
				},
				Properties: map[string]*datastore.Value{
					"str": &datastore.Value{
						ValueType:          &datastore.Value_StringValue{StringValue: "id 10 version 0"},
						Meaning:            0,
						ExcludeFromIndexes: false,
					},
				},
			},
			Version: 0,
		},
		{
			Entity: &datastore.Entity{
				Key: &datastore.Key{
					PartitionId: &datastore.PartitionId{
						ProjectId:   "project-id",
						NamespaceId: "namespace-id",
					},
					Path: []*datastore.Key_PathElement{
						{
							Kind:   "kind",
							IdType: &datastore.Key_PathElement_Id{Id: 10},
						},
					},
				},
				Properties: map[string]*datastore.Value{
					"str": &datastore.Value{
						ValueType:          &datastore.Value_StringValue{StringValue: "id 10 version 1"},
						Meaning:            0,
						ExcludeFromIndexes: false,
					},
				},
			},
			Version: 1,
		},
	}
)

// TestRedis_SetOldAfterNew - 新しいバージョンをSetした後に古いバージョンをSetしても上書きされないことをテスト
func TestRedis_SetOldAfterNew(t *testing.T) {
	r := newRedis(t)

	ctx := context.Background()

	// 新しいVersion
	if err := r.SetMulti(ctx, projectID, []*datastore.EntityResult{entityResults[1]}); err != nil {
		t.Fatalf("SetMulti failed: %+v", err)
	}

	// 古いVersion
	if err := r.SetMulti(ctx, projectID, []*datastore.EntityResult{entityResults[0]}); err != nil {
		t.Fatalf("SetMulti failed: %+v", err)
	}

	results, err := r.GetMulti(ctx, projectID, []*datastore.Key{entityResults[0].Entity.Key})

	if err != nil {
		t.Fatalf("GetMulti failed: %+v", err)
	}

	if len(results) != 1 {
		t.Fatalf("GetMulti returned %d results", len(results))
	}

	if results[0] == nil {
		t.Fatalf("GetMulti returned no results")
	}

	filter := cmp.FilterPath(func(path cmp.Path) bool {
		return !strings.HasPrefix(path.Last().String(), "XXX_")
	}, cmp.Ignore())

	if diff := cmp.Diff(entityResults[1], results[0], filter); diff != "" {
		t.Errorf("returned values from GetMulti differed: %s", diff)
	}
}

// TestRedis_SetOldAfterNew - 新しいバージョンをSetした後に古いバージョンをSetしても上書きされないことをテスト
func TestRedis_GetAfterDelete(t *testing.T) {
	r := newRedis(t)

	ctx := context.Background()

	if err := r.SetMulti(ctx, projectID, []*datastore.EntityResult{entityResults[0]}); err != nil {
		t.Fatalf("SetMulti failed: %+v", err)
	}

	if err := r.DeleteMulti(ctx, projectID, []*datastore.Key{entityResults[0].Entity.Key}); err != nil {
		t.Fatalf("DeleteMulti failed: %+v", err)
	}

	results, err := r.GetMulti(ctx, projectID, []*datastore.Key{entityResults[0].Entity.Key})

	if err != nil {
		t.Fatalf("GetMulti failed: %+v", err)
	}

	if len(results) != 1 {
		t.Fatalf("GetMulti returned %d results: %v", len(results), results)
	}

	if results[0] != nil {
		t.Fatalf("GetMulti returned %d results: %v", len(results), results)
	}
}
