package cache

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/gcp-kit/datastore-cache-go/mock"
	"github.com/golang/mock/gomock"
	"google.golang.org/genproto/googleapis/datastore/v1"
	"google.golang.org/grpc"
)

var (
	testKeys = []*datastore.Key{
		{
			PartitionId: &datastore.PartitionId{
				ProjectId:   "a",
				NamespaceId: "a",
			},
			Path: []*datastore.Key_PathElement{
				{
					Kind: "a",
					IdType: &datastore.Key_PathElement_Id{
						Id: 100,
					},
				},
			},
		},
	}
	testKeys2 = []*datastore.Key{
		{
			PartitionId: &datastore.PartitionId{
				ProjectId:   "a",
				NamespaceId: "a",
			},
			Path: []*datastore.Key_PathElement{
				{
					Kind: "a",
					IdType: &datastore.Key_PathElement_Id{
						Id: 100,
					},
				},
			},
		},
		{
			PartitionId: &datastore.PartitionId{
				ProjectId:   "b",
				NamespaceId: "b",
			},
			Path: []*datastore.Key_PathElement{
				{
					Kind: "b",
					IdType: &datastore.Key_PathElement_Name{
						Name: "b",
					},
				},
			},
		},
		{
			PartitionId: &datastore.PartitionId{
				ProjectId:   "c",
				NamespaceId: "c",
			},
			Path: []*datastore.Key_PathElement{
				{
					Kind: "c",
					IdType: &datastore.Key_PathElement_Id{
						Id: 500,
					},
				},
			},
		},
		{
			PartitionId: &datastore.PartitionId{
				ProjectId:   "d",
				NamespaceId: "d",
			},
			Path: []*datastore.Key_PathElement{
				{
					Kind: "d",
					IdType: &datastore.Key_PathElement_Name{
						Name: "d",
					},
				},
			},
		},
	}
)

const (
	projectID = "project-id"
)

func TestCacheMiddleware_beforeLookup(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mock.NewMockCache(ctrl)

	ctx := context.Background()

	returnData := []*datastore.EntityResult{
		{
			Entity: &datastore.Entity{
				Key: testKeys2[0],
			},
		},
		{
			Entity: &datastore.Entity{
				Key: testKeys2[1],
			},
		},
		nil,
		nil,
	}

	m.EXPECT().
		GetMulti(ctx, projectID, testKeys2).
		Return(returnData, nil)

	c := NewMiddleware(m)

	testData := &datastore.LookupRequest{
		ProjectId: projectID,
		Keys:      testKeys2,
	}

	reply := new(datastore.LookupResponse)
	err := c.beforeLookup(ctx, testData, reply)
	if err != nil {
		t.Fatal(err)
	}
	if len(reply.Found) != 2 {
		t.Fatalf("TestCacheMiddleware_beforeLookup reply number was invalid value(%d != %d)\n", 2, len(reply.Found))
	}
	if len(testData.Keys) != 2 {
		t.Fatalf("TestCacheMiddleware_beforeLookup testData.Keys number was invalid value(%d != %d)\n", 2, len(testData.Keys))
	}
}

func TestCacheMiddleware_deleteCache(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mock.NewMockCache(ctrl)

	ctx := context.Background()

	m.EXPECT().
		DeleteMulti(ctx, projectID, testKeys2).
		Return(nil)

	c := NewMiddleware(m)

	testData := &datastore.CommitRequest{
		ProjectId: projectID,
		Mutations: []*datastore.Mutation{
			{
				Operation: &datastore.Mutation_Insert{
					Insert: &datastore.Entity{
						Key: testKeys2[0],
					},
				},
			},
			{
				Operation: &datastore.Mutation_Update{
					Update: &datastore.Entity{
						Key: testKeys2[1],
					},
				},
			},
			{
				Operation: &datastore.Mutation_Upsert{
					Upsert: &datastore.Entity{
						Key: testKeys2[2],
					},
				},
			},
			{
				Operation: &datastore.Mutation_Delete{
					Delete: testKeys2[3],
				},
			},
		},
	}

	err := c.deleteCache(ctx, testData)
	if err != nil {
		t.Fatal(err)
	}
}

func TestCacheMiddleware_beforeCommit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mock.NewMockCache(ctrl)

	ctx := context.Background()

	m.EXPECT().
		DeleteMulti(ctx, projectID, testKeys).
		Return(fmt.Errorf("e"))

	req := &datastore.CommitRequest{
		ProjectId: projectID,
		Mutations: []*datastore.Mutation{
			{
				Operation: &datastore.Mutation_Delete{
					Delete: testKeys[0],
				},
			},
		},
	}

	c := NewMiddleware(m)
	c.CacheDeleteTiming = DeleteTimingAfterCommit
	err := c.beforeCommit(ctx, req, CachingModeNever)
	if err != nil {
		t.Fatal(err)
	}

	c = NewMiddleware(m)
	c.CacheDeleteTiming = DeleteTimingBeforeAndAfterCommit
	err = c.beforeCommit(ctx, req, CachingModeNever)
	if err == nil || err.Error() != "e" {
		t.Fatal(err)
	}
}

func TestCacheMiddleware_afterCommit(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	m := mock.NewMockCache(ctrl)

	ctx := context.Background()

	m.EXPECT().
		DeleteMulti(ctx, projectID, testKeys).
		Return(fmt.Errorf("e"))

	req := &datastore.CommitRequest{
		ProjectId: projectID,
		Mutations: []*datastore.Mutation{
			{
				Operation: &datastore.Mutation_Delete{
					Delete: testKeys[0],
				},
			},
		},
	}

	c := NewMiddleware(m)
	c.CacheDeleteTiming = DeleteTimingBeforeCommit

	err := c.afterCommit(ctx, req, CachingModeNever)
	if err != nil {
		t.Fatal(err)
	}

	c = NewMiddleware(m)
	c.CacheDeleteTiming = DeleteTimingBeforeAndAfterCommit

	err = c.afterCommit(ctx, req, CachingModeNever)
	if err == nil || err.Error() != "e" {
		t.Fatal(err)
	}
}

func TestCacheMiddleware_lookup(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mock.NewMockCache(ctrl)

		ctx := context.Background()
		keys := []*datastore.Key{
			{
				PartitionId: &datastore.PartitionId{
					ProjectId:   "a",
					NamespaceId: "a",
				},
				Path: []*datastore.Key_PathElement{
					{
						Kind: "a",
						IdType: &datastore.Key_PathElement_Id{
							Id: 100,
						},
					},
				},
			},
		}

		invoker := func(
			ctx context.Context,
			method string,
			req,
			reply interface{},
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			return nil
		}

		req := &datastore.LookupRequest{
			ProjectId: projectID,
			Keys:      keys,
		}
		reply := new(datastore.LookupResponse)

		m.EXPECT().
			GetMulti(ctx, projectID, keys).
			Return(nil, fmt.Errorf("e"))

		c := NewMiddleware(m)

		err := c.lookup(ctx, CachingModeNever, "", req, reply, new(grpc.ClientConn), func(
			ctx context.Context,
			method string,
			req,
			reply interface{},
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			return fmt.Errorf("ee")
		}, nil)
		if err == nil || err.Error() != "ee" {
			t.Fatal(err)
		}

		c = NewMiddleware(m)
		w := new(bytes.Buffer)
		c.Logger = log.New(w, "", 0)

		err = c.lookup(ctx, CachingModeReadOnly, "", req, reply, new(grpc.ClientConn), invoker, nil)
		errorLog := w.String()
		if err != nil || errorLog != "search on cache before Lookup failed: e\n" {
			t.Fatalf("error: %v, log: %s\n", err, errorLog)
		}

		founds := []*datastore.EntityResult{
			{
				Entity: &datastore.Entity{
					Key: keys[0],
				},
				Version: 99,
			},
		}
		invoker = func(
			ctx context.Context,
			method string,
			req,
			reply interface{},
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			reply.(*datastore.LookupResponse).Found = founds

			return nil
		}
		m.EXPECT().
			SetMulti(ctx, projectID, founds).
			Return(fmt.Errorf("eee"))

		c = NewMiddleware(m)
		w = new(bytes.Buffer)
		c.Logger = log.New(w, "", 0)

		err = c.lookup(ctx, CachingModeWriteOnly, "", req, reply, new(grpc.ClientConn), invoker, nil)
		errorLog = w.String()
		if err != nil || errorLog != "cache after Lookup failed: eee\n" {
			t.Fatalf("error: %v, log: %s\n", err, errorLog)
		}
	})

	t.Run("cache_hit_all", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		m := mock.NewMockCache(ctrl)

		ctx := context.Background()
		keys := []*datastore.Key{
			{
				PartitionId: &datastore.PartitionId{
					ProjectId:   "a",
					NamespaceId: "a",
				},
				Path: []*datastore.Key_PathElement{
					{
						Kind: "a",
						IdType: &datastore.Key_PathElement_Id{
							Id: 100,
						},
					},
				},
			},
		}

		invoker := func(
			ctx context.Context,
			method string,
			req,
			reply interface{},
			cc *grpc.ClientConn,
			opts ...grpc.CallOption,
		) error {
			return nil
		}

		req := &datastore.LookupRequest{
			ProjectId: projectID,
			Keys:      keys,
		}
		reply := new(datastore.LookupResponse)

		cacheReply := []*datastore.EntityResult{
			{
				Entity: &datastore.Entity{
					Key: keys[0],
				},
				Version: 99,
			},
		}

		m.EXPECT().
			GetMulti(ctx, projectID, keys).
			Return(cacheReply, nil)

		c := NewMiddleware(m)

		err := c.lookup(ctx, CachingModeReadWrite, "", req, reply, new(grpc.ClientConn), invoker, nil)
		if err != nil {
			t.Fatalf("lookup failed: %+v", err)
		}

		if len(reply.Found) != 1 {
			t.Fatalf("lookup returned %d items(expected: %d)", len(reply.Found), 1)
		}

		if reply.Found[0] == nil {
			t.Fatalf("lookup returned nil item")
		}
	})
}
