package cache

import (
	"context"

	"google.golang.org/grpc"
)

// CachingModeType - Cacheの制御を行う
type CachingModeType int

const (
	// CachingModeNever - キャッシュを一切使わない
	CachingModeNever CachingModeType = 0

	// CachingModeWriteOnly - キャッシュの書き込みのみを行う。
	// CacheDeleteTimingにてCacheDeleteTiming_Noneを使用していた場合でも、キャッシュの書き込みを行う。
	CachingModeWriteOnly CachingModeType = 1

	// CachingModeReadOnly - キャッシュの読み取りのみを行う。
	// キャッシュされてないデータがあろうとも、新しくキャッシュの書き込みは行わない。
	CachingModeReadOnly CachingModeType = 2

	// CachingModeReadWrite - キャッシュの読み取りと書き込みを行う。
	// デフォルトの動作。
	CachingModeReadWrite CachingModeType = 3
)

// CachingModeFunc - Cacheの制御を行う関数。
type CachingModeFunc = func(
	ctx context.Context,
	method string,
	req,
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) CachingModeType
