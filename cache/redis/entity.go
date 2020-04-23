package redis

import (
	"strconv"
	"strings"

	"github.com/golang/protobuf/proto"
	"golang.org/x/xerrors"
	"google.golang.org/genproto/googleapis/datastore/v1"
)

func escapeKey(str string) string {
	str = strings.ReplaceAll(str, "\\", "\\\\")
	str = strings.ReplaceAll(str, ":", "\\:")

	return str
}

func encodeEntity(entity *datastore.EntityResult) ([]byte, error) {
	return proto.Marshal(entity)
}

func decodeEntity(data []byte) (*datastore.EntityResult, error) {
	entity := &datastore.EntityResult{}

	if err := proto.Unmarshal(data, entity); err != nil {
		return nil, xerrors.Errorf("failed to unmarshal protobuf", err)
	}

	return entity, nil
}

func calcKeyForEntity(projectID string, key *datastore.Key) string {
	paths := make([]string, 0, len(key.Path))
	for _, path := range key.Path {
		var id string
		if idInt, ok := path.GetIdType().(*datastore.Key_PathElement_Id); ok {
			id = "i:" + strconv.FormatInt(idInt.Id, 10)
		} else if name, ok := path.GetIdType().(*datastore.Key_PathElement_Name); ok {
			id = "n:" + escapeKey(name.Name)
		} else {
			return ""
		}

		paths = append(paths, escapeKey(path.Kind)+":"+id)
	}

	var namespaceID string

	if key.PartitionId != nil {
		if key.PartitionId.ProjectId != "" {
			projectID = key.PartitionId.ProjectId
		}

		namespaceID = key.PartitionId.NamespaceId
	}

	return escapeKey(projectID) +
		":" +
		escapeKey(namespaceID) +
		":" +
		strings.Join(paths, ":")
}

func isReserved(id string) bool {
	return strings.HasPrefix(id, "__") && strings.HasSuffix(id, "__")
}
