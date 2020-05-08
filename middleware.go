package cache

import (
	"context"
	"log"

	"golang.org/x/xerrors"

	"google.golang.org/genproto/googleapis/datastore/v1"
	"google.golang.org/grpc"
)

// Middleware - キャッシュを管理するミドルウェア。
type Middleware struct {
	cache             Cache           // Cache - Cache interfaceを満たす構造体。これによりキャッシュを実現する。
	CacheDeleteTiming DeleteTiming    // DeleteTiming - Cacheの削除を行うタイミング。
	CachingModeFunc   CachingModeFunc // CachingModeFunc - Cacheの削除や状態を個別に管理する関数。
	Logger            *log.Logger
}

// UnaryClientMethod - Datastoreの呼び出しメソッド
type UnaryClientMethod = string

const (
	// UnaryClientMethodLookup - Get, GetMultiなどの問い合せにより呼ばれる
	UnaryClientMethodLookup UnaryClientMethod = "/google.datastore.v1.Datastore/Lookup"

	// UnaryClientMethodCommit - Put, PutMulti, Delete, DeleteMulti, Mutateなどの問い合わせにより呼ばれる
	UnaryClientMethodCommit UnaryClientMethod = "/google.datastore.v1.Datastore/Commit"
)

// NewMiddleware - Datastoreの操作をキャッシュするMiddlewareを初期化する。
// cache - Cacheインターフェイスを満たす構造体である必要がある。
// cacheインターフェイスを操作し、キャッシュを実現する。
//
// デフォルト動作:
//   - cacheは更新動作の前後で削除される
//   - 全ての要素をキャッシュする
func NewMiddleware(cache Cache) *Middleware {
	return &Middleware{
		cache:             cache,
		CacheDeleteTiming: DeleteTimingBeforeAndAfterCommit,
		CachingModeFunc:   nil,
	}
}

// UnaryClientInterceptor - datastoreのgRPCから呼ばれる
func (m *Middleware) UnaryClientInterceptor(
	ctx context.Context,
	method string, req,
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) (err error) {
	cachingMode := CachingModeReadWrite
	if m.CachingModeFunc != nil {
		cachingMode = m.CachingModeFunc(ctx, method, req, reply, cc, invoker, opts...)
	}

	switch method {
	case UnaryClientMethodLookup:
		// キャッシュの参照
		return m.lookup(
			ctx,
			cachingMode,
			method,
			req.(*datastore.LookupRequest),
			reply.(*datastore.LookupResponse),
			cc,
			invoker,
			opts...,
		)
	case UnaryClientMethodCommit:
		// キャッシュの削除
		return m.commit(
			ctx,
			cachingMode,
			method,
			req.(*datastore.CommitRequest),
			reply.(*datastore.CommitResponse),
			cc,
			invoker,
			opts...,
		)
	default:
		return invoker(ctx, method, req, invoker, cc, opts...)
	}
}

// lookup - DatastoreのLookupのときの処理
func (m *Middleware) lookup(
	ctx context.Context,
	cachingMode CachingModeType,
	method string,
	req *datastore.LookupRequest,
	reply *datastore.LookupResponse,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) (err error) {
	// トランザクションが有効であれば対象としない
	if req.GetReadOptions().GetTransaction() != nil || cachingMode == CachingModeNever {
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	// キャッシュの取得
	if cachingMode&CachingModeReadOnly != 0 {
		err = m.beforeLookup(ctx, req, reply)
		if err != nil {
			err = xerrors.Errorf("search on cache before Lookup failed: %w", err)
			m.logPrintError(err)
		}
	}

	if len(req.Keys) < 1 {
		return nil
	}

	// 本来の処理
	invokerReply := new(datastore.LookupResponse)
	err = invoker(ctx, method, req, invokerReply, cc, opts...)
	if err != nil {
		return err
	}

	// キャッシュの保存
	if cachingMode&CachingModeWriteOnly != 0 {
		err = m.afterLookup(ctx, req, invokerReply)
		err = xerrors.Errorf("cache after Lookup failed: %w", err)
		m.logPrintError(err)
	}

	reply.Found = append(reply.Found, invokerReply.Found...)
	reply.Missing = invokerReply.Missing
	reply.Deferred = invokerReply.Deferred

	return nil
}

// beforeLookup - Lookup前に呼ばれる
func (m *Middleware) beforeLookup(
	ctx context.Context,
	req *datastore.LookupRequest,
	reply *datastore.LookupResponse,
) (err error) {
	cacheKeys := req.Keys
	items, err := m.cache.GetMulti(ctx, req.ProjectId, cacheKeys)
	if err != nil {
		return err
	}

	if len(items) != len(req.Keys) {
		return xerrors.Errorf("cache middleware should return %d, but returned %d", len(req.Keys), len(items))
	}

	nonCachedKeys := make([]*datastore.Key, 0, len(req.Keys))
	for i := range items {
		if items[i] == nil {
			nonCachedKeys = append(nonCachedKeys, req.Keys[i])
		}
	}

	index := 0
	for i := range items {
		if items[i] != nil {
			items[index] = items[i]
			index++
		}
	}
	items = items[:index]

	reply.Found = items
	req.Keys = nonCachedKeys

	return nil
}

// afterLookup - Lookup後に呼ばれる
func (m *Middleware) afterLookup(
	ctx context.Context,
	req *datastore.LookupRequest,
	reply *datastore.LookupResponse,
) (err error) {
	entities := reply.GetFound()
	if len(entities) < 1 {
		return nil
	}
	return m.cache.SetMulti(ctx, req.ProjectId, entities)
}

// commit - Commitのときの処理
func (m *Middleware) commit(
	ctx context.Context,
	cachingMode CachingModeType,
	method string,
	req *datastore.CommitRequest,
	reply *datastore.CommitResponse,
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) (err error) {
	// キャッシュの削除
	err = m.beforeCommit(ctx, req, cachingMode)
	if err != nil {
		err = xerrors.Errorf("cache before commit failed: %w", err)
		m.logPrintError(err)
		return err
	}

	// 本来の処理
	err = invoker(ctx, method, req, reply, cc, opts...)
	if err != nil {
		return err
	}

	// キャッシュの削除
	err = m.afterCommit(ctx, req, cachingMode)
	if err != nil {
		err = xerrors.Errorf("cache after commit failed: %w", err)
		m.logPrintError(err)
	}
	return nil
}

// beforeCommit - Commit前に呼ばれる
func (m *Middleware) beforeCommit(
	ctx context.Context,
	req *datastore.CommitRequest,
	cachingMode CachingModeType,
) (err error) {
	if m.CacheDeleteTiming&DeleteTimingBeforeCommit == 0 && cachingMode&CachingModeWriteOnly == 0 {
		return nil
	}
	return m.deleteCache(ctx, req)
}

// afterCommit - Commit後に呼ばれる
func (m *Middleware) afterCommit(
	ctx context.Context,
	req *datastore.CommitRequest,
	cachingMode CachingModeType,
) (err error) {
	if m.CacheDeleteTiming&DeleteTimingAfterCommit == 0 && cachingMode&CachingModeWriteOnly == 0 {
		return nil
	}
	return m.deleteCache(ctx, req)
}

// deleteCache - CacheをDeleteさせる
func (m *Middleware) deleteCache(ctx context.Context, req *datastore.CommitRequest) (err error) {
	deleteKeys := make([]*datastore.Key, 0)

	for _, m := range req.GetMutations() {
		entities := []*datastore.Entity{
			m.GetInsert(),
			m.GetUpdate(),
			m.GetUpsert(),
		}
		for _, e := range entities {
			if e == nil {
				continue
			}
			deleteKeys = append(deleteKeys, e.Key)
		}

		if m.GetDelete() == nil {
			continue
		}
		deleteKeys = append(deleteKeys, m.GetDelete())
	}

	return m.cache.DeleteMulti(ctx, req.ProjectId, deleteKeys)
}

func (m *Middleware) logPrintError(err error) {
	if m.Logger == nil {
		return
	}
	m.Logger.Printf("%v\n", err)
}
