package redis

import (
	"context"
	"strings"
	"testing"

	redigo "github.com/gomodule/redigo/redis"
	"github.com/google/go-cmp/cmp"
	"github.com/rafaeljusto/redigomock"
	"google.golang.org/genproto/googleapis/datastore/v1"
)

const (
	projectID = "project-id"
)

func initRedis(t *testing.T) (*redigomock.Conn, *Redis) {
	t.Helper()

	conn := redigomock.NewConn()

	pool := &redigo.Pool{
		Dial: func() (redigo.Conn, error) {
			return conn, nil
		},
		MaxIdle: 10,
	}

	return conn, NewRedis(pool)
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
					"str": {
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
					"str": {
						ValueType:          &datastore.Value_StringValue{StringValue: "id 10 version 1"},
						Meaning:            0,
						ExcludeFromIndexes: false,
					},
				},
			},
			Version: 1,
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
							IdType: &datastore.Key_PathElement_Name{Name: "10"},
						},
					},
				},
				Properties: map[string]*datastore.Value{
					"str": {
						ValueType:          &datastore.Value_StringValue{StringValue: "name 10 version 1"},
						Meaning:            0,
						ExcludeFromIndexes: false,
					},
				},
			},
			Version: 1,
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
							IdType: &datastore.Key_PathElement_Id{Id: 11},
						},
					},
				},
				Properties: map[string]*datastore.Value{
					"integer": {
						ValueType:          &datastore.Value_IntegerValue{IntegerValue: 10},
						Meaning:            0,
						ExcludeFromIndexes: false,
					},
				},
			},
			Version: 1,
		},
	}
)

func setEntityResults(t *testing.T, conn *redigomock.Conn, r *Redis) {
	t.Helper()

	conn.Command("MULTI").Expect("ok")

	var redisResults []interface{}
	for i, res := range entityResults {
		encoded, err := encodeEntity(res)

		if err != nil {
			t.Fatalf("failed to encode %dth entity: %+v", i, err)
		}

		conn.Command("ZADD", calcKeyForEntity(projectID, res.Entity.Key), res.Version, encoded).Expect([]byte("queued"))

		redisResults = append(redisResults, "1")
	}

	conn.Command("EXEC").ExpectSlice(redisResults...)

	if err := r.SetMulti(context.Background(), projectID, entityResults); err != nil {
		t.Fatalf("failed to SetMulti for entities: %+v", err)
	}
}

func TestRedis_setThenGet(t *testing.T) {
	conn, r := initRedis(t)

	setEntityResults(t, conn, r)

	encode := func(entity *datastore.EntityResult) []interface{} {
		b, err := encodeEntity(entity)

		if err != nil {
			t.Fatalf("failed to encode entity: %+v", err)
		}

		return []interface{}{b}
	}

	redisResults := []interface{}{
		[]interface{}{},
		encode(entityResults[1]),
		encode(entityResults[2]),
		encode(entityResults[3]),
	}

	expectedResults := []*datastore.EntityResult{
		entityResults[1],
		entityResults[2],
		entityResults[3],
	}

	keys := []*datastore.Key{
		{ // not found key
			PartitionId: &datastore.PartitionId{
				ProjectId:   "project-id",
				NamespaceId: "namespace-id",
			},
			Path: []*datastore.Key_PathElement{
				{
					Kind:   "kind",
					IdType: &datastore.Key_PathElement_Id{Id: 100},
				},
			},
		},

		entityResults[0].Entity.Key,
		entityResults[2].Entity.Key,
		entityResults[3].Entity.Key,
	}

	conn.Command("MULTI").Expect("ok")

	for _, key := range keys {
		conn.Command("ZREVRANGE", calcKeyForEntity(projectID, key), 0, 0).Expect("queued")
	}

	conn.Command("EXEC").ExpectSlice(redisResults...)

	items, err := r.GetMulti(context.Background(), "project", keys)

	if err != nil {
		t.Fatalf("failed to GetMulti entites: %+v", err)
	}

	filter := cmp.FilterPath(func(path cmp.Path) bool {
		return !strings.HasPrefix(path.Last().String(), "XXX_")
	}, cmp.Ignore())

	if diff := cmp.Diff(expectedResults, items, filter); diff != "" {
		t.Errorf("returned values from GetMulti differed: %s", diff)
	}
}

func TestRedis_setThenDelete(t *testing.T) {
	conn, r := initRedis(t)

	setEntityResults(t, conn, r)

	redisResults := []interface{}{
		0,
		1,
		1,
		1,
	}

	keys := []*datastore.Key{
		{ // not found key
			PartitionId: &datastore.PartitionId{
				ProjectId:   "project-id",
				NamespaceId: "namespace-id",
			},
			Path: []*datastore.Key_PathElement{
				{
					Kind:   "kind",
					IdType: &datastore.Key_PathElement_Id{Id: 100},
				},
			},
		},

		entityResults[0].Entity.Key,
		entityResults[2].Entity.Key,
		entityResults[3].Entity.Key,
	}

	conn.Command("MULTI").Expect("ok")

	for _, key := range keys {
		conn.Command("DEL", calcKeyForEntity(projectID, key)).Expect("queued")
	}

	conn.Command("EXEC").ExpectSlice(redisResults...)

	err := r.DeleteMulti(context.Background(), projectID, keys)

	if err != nil {
		t.Fatalf("failed to DeleteMulti entites: %+v", err)
	}
}
