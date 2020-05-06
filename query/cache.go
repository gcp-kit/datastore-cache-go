package querycache

import (
	"context"

	"cloud.google.com/go/datastore"
	"golang.org/x/xerrors"
)

// GetALl - キャッシュが利用されるようなqueryの実行
func GetAll(
	ctx context.Context,
	client *datastore.Client,
	q *datastore.Query,
	dst interface{},
) (keys []*datastore.Key, err error) {
	q = q.KeysOnly()

	keys, err = client.GetAll(ctx, q, nil)

	if err != nil {
		return nil, xerrors.Errorf("failed to get all keys: %+v", err)
	}

	err = client.GetMulti(ctx, keys, dst)

	if err != nil {
		return nil, xerrors.Errorf("failed to get entities for the keys: %+v", err)
	}

	return keys, nil
}
