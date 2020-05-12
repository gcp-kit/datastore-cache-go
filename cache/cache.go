/*
Package cache - This package provides operations related to cache.
The Cache is injected into gRPC communication such as Get/Delete/Update regarding the exchange with Datastore.
And is realized by rewriting the Request/Response in the middle.
...
このパッケージではキャッシュに関する操作を提供します。
キャッシュはDatastoreとのやり取りに関して、取得・削除・更新などのgRPC通信にインジェクションされ、中間でリクエスト・レスポンスを書き換えることで実現します。
*/
package cache

import (
	"context"

	"google.golang.org/genproto/googleapis/datastore/v1"
)

//go:generate mockgen -source=$GOFILE -package=mock -destination=./mock/mock_$GOFILE -self_package=github.com/gcp-kit/datastore-cache-go

// Cache - Mechanism for caching data.
//           └── データをキャッシュする機構。
// It works by passing the one that satisfies this interface to middleware.
//    └── このインターフェイスを満たしたものをmiddlewareに渡すことで動作する
type Cache interface {
	// GetMulti - get the cache.
	//              └── キャッシュを取得する
	// An array of Datastore keys is passed and the corresponding data is obtained.
	//    └── Datastoreのキーの配列が渡され、それに対応するデータを取得する
	// If a key that does not exist in the cache is passed, it must be filled with nil and len(items) must match len(keys).
	//    └── キャッシュに存在しないキーが渡された場合にはnilを詰め、len(items)がlen(keys)と一致している必要がある
	GetMulti(ctx context.Context, projectID string, keys []*datastore.Key) (items []*datastore.EntityResult, err error)

	// SetMulti - Set to cache
	//              └── キャッシュする
	// Since an array of Entity of Datastore is passed, those data are cached.
	//    └── datastoreのEntityの配列が渡されるので、それらのデータをキャッシュする。
	SetMulti(ctx context.Context, projectID string, items []*datastore.EntityResult) (err error)

	// DeleteMulti - Delete from cache.
	//                 └── キャッシュから削除する。
	// It is called when the cache is destroyed by updating or deleting.
	//    └── 更新や削除などでキャッシュを破棄する場合に呼ばれる。
	DeleteMulti(ctx context.Context, projectID string, keys []*datastore.Key) (err error)
}
