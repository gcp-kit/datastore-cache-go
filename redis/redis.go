package redis

import (
	"context"

	"github.com/gcp-kit/datastore-cache-go"
	"github.com/gomodule/redigo/redis"
	"golang.org/x/xerrors"
	"google.golang.org/genproto/googleapis/datastore/v1"
)

type Redis struct {
	connPool *redis.Pool
}

func NewRedis(connPool *redis.Pool) *Redis {
	return &Redis{
		connPool: connPool,
	}
}

var _ cache.Cache = &Redis{}

func (r *Redis) runInTransaction(f func(conn redis.Conn) error) ([]interface{}, error) {
	conn := r.connPool.Get()
	defer conn.Close()

	_, err := conn.Do("MULTI")

	if err != nil {
		return nil, xerrors.Errorf("failed to start transaction: %w", err)
	}

	err = f(conn)

	if err != nil {
		// nolint:errcheck
		conn.Do("DISCARD")

		return nil, xerrors.Errorf("failed to exec in-transaction function: %w", err)
	}

	reply, err := redis.Values(conn.Do("EXEC"))

	if err != nil {
		return nil, xerrors.Errorf("failed to parse byte slices: %w", err)
	}

	return reply, nil
}

func (r *Redis) GetMulti(
	_ context.Context,
	projectID string,
	keys []*datastore.Key,
) (items []*datastore.EntityResult, err error) {
	if isReserved(projectID) {
		return nil, nil
	}

	filtered := make([]*datastore.Key, 0, len(keys))

	for i := range keys {
		partitionID := keys[i].PartitionId

		if partitionID != nil &&
			(isReserved(partitionID.ProjectId) || isReserved(partitionID.NamespaceId)) {
			continue
		}

		filtered = append(filtered, keys[i])
	}

	keys = filtered

	slices, err := r.runInTransaction(func(conn redis.Conn) error {
		for i := range keys {
			key := calcKeyForEntity(projectID, keys[i])

			if key == "" {
				continue
			}

			_, err = conn.Do("ZREVRANGE", key, 0, 0)

			if err != nil {
				return xerrors.Errorf("ZREVRANGE failed: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return nil, xerrors.Errorf("GetMulti in transaction failed: %w", err)
	}

	items = make([]*datastore.EntityResult, len(keys))
	for i, buf := range slices {
		if buf == nil {
			continue
		}

		b, err := redis.ByteSlices(buf, nil)

		if err != nil {
			return nil, xerrors.Errorf("failed to convert result to []byte for %dth element: %+v", i, err)
		}

		if len(b) == 0 {
			continue
		}

		entity, err := decodeEntity(b[0])

		if err != nil {
			continue
		}

		items[i] = entity
	}

	return items, nil
}

func (r *Redis) SetMulti(_ context.Context, projectID string, items []*datastore.EntityResult) (err error) {
	if isReserved(projectID) {
		return nil
	}

	_, err = r.runInTransaction(func(conn redis.Conn) error {
		for i := range items {
			partitionID := items[i].Entity.Key.PartitionId
			if isReserved(partitionID.ProjectId) || isReserved(partitionID.NamespaceId) {
				continue
			}

			//nolint:govet
			encoded, err := encodeEntity(items[i])

			if err != nil {
				return xerrors.Errorf("failed to encode entity for Redis: %w", err)
			}

			key := calcKeyForEntity(projectID, items[i].Entity.Key)

			if key == "" {
				continue
			}

			_, err = conn.Do("ZADD", key, items[i].Version, encoded)

			if err != nil {
				return xerrors.Errorf("ZADD failed: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return xerrors.Errorf("SetMulti in transaction failed: %w", err)
	}

	return nil
}

func (r *Redis) DeleteMulti(_ context.Context, projectID string, keys []*datastore.Key) (err error) {
	if isReserved(projectID) {
		return nil
	}

	_, err = r.runInTransaction(func(conn redis.Conn) error {
		for i := range keys {
			partitionID := keys[i].PartitionId

			if partitionID != nil &&
				(isReserved(partitionID.ProjectId) || isReserved(partitionID.NamespaceId)) {
				continue
			}

			key := calcKeyForEntity(projectID, keys[i])

			if key == "" {
				continue
			}

			_, err = conn.Do("DEL", key)

			if err != nil {
				return xerrors.Errorf("DEL failed: %w", err)
			}
		}

		return nil
	})

	if err != nil {
		return xerrors.Errorf("DeleteMulti in transaction failed: %w", err)
	}

	return nil
}
