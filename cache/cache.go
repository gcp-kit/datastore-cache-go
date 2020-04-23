/*
Package cache - このパッケージではキャッシュに関する操作を提供します。

キャッシュはDatastoreとのやり取りに関して、取得・削除・更新などのgRPC通信にインジェクションされ、中間でリクエスト・レスポンスを書き換えることで実現します。
*/
package cache

import (
	"context"

	"google.golang.org/genproto/googleapis/datastore/v1"
)

//go:generate mockgen -source=$GOFILE -destination=mock_$GOFILE -package=mock -self_package=github.com/gcp-kit/datastore-cache-go/cache -destination ./mock/mock_cache.go

// Cache - データをキャッシュする機構。
// このインターフェイスを満たしたものをmiddlewareに渡すことで動作する。
type Cache interface {
	// GetMulti - キャッシュを取得する。
	// datastoreのキーの配列が渡され、それに対応するデータを取得する。
	// キャッシュに存在しないキーが渡された場合にはnilを詰め、len(items)がlen(keys)と一致している必要がある
	GetMulti(ctx context.Context, projectID string, keys []*datastore.Key) (items []*datastore.EntityResult, err error)

	// SetMulti - キャッシュする。
	// datastoreのEntityの配列が渡されるので、それらのデータをキャッシュする。
	SetMulti(ctx context.Context, projectID string, items []*datastore.EntityResult) (err error)

	// DeleteMulti - キャッシュから削除する。
	// 更新や削除などでキャッシュを破棄する場合に呼ばれる。
	DeleteMulti(ctx context.Context, projectID string, keys []*datastore.Key) (err error)
}
