package cache

import (
	"context"

	"google.golang.org/grpc"
)

// CachingModeType - control cache
//                     └── Cacheの制御を行う
type CachingModeType int

const (
	// CachingModeNever - never use cache (キャッシュを一切使わない)
	CachingModeNever CachingModeType = 0

	// CachingModeWriteOnly - only write to cache
	//                          └── キャッシュの書き込みのみを行う
	// Cache is written even if Cache Delete Timing None is used in Cache Delete Timing.
	//    └── CacheDeleteTimingにてCacheDeleteTiming_Noneを使用していた場合でも、キャッシュの書き込みを行う
	CachingModeWriteOnly CachingModeType = 1

	// CachingModeReadOnly - only read cache
	//                         └── キャッシュの読み取りのみを行う
	// Even if there is uncached data, new cache is not written.
	//    └── キャッシュされてないデータがあろうとも、新しくキャッシュの書き込みは行わない
	CachingModeReadOnly CachingModeType = 2

	// CachingModeReadWrite - read and write the cache
	//                          └── キャッシュの読み取りと書き込みを行う
	// Default behavior
	//    └── デフォルトの動作
	CachingModeReadWrite CachingModeType = 3
)

// CachingModeFunc - function that controls cache
//                     └── Cacheの制御を行う関数
type CachingModeFunc = func(
	ctx context.Context,
	method string,
	req,
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) CachingModeType
