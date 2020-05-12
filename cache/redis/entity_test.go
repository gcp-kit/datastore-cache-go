package redis

import (
	"testing"

	"google.golang.org/genproto/googleapis/datastore/v1"
)

func testEscapeKey(t *testing.T, query, expected string) {
	t.Helper()
	actual := escapeKey(query)

	if expected != actual {
		t.Errorf("the escaped key differed:\nactual  : %s\nexpected: %s", actual, expected)
	}
}

func TestEscapeKey(t *testing.T) {
	testEscapeKey(t, "abc:abc", "abc\\:abc")
	testEscapeKey(t, "abc\\abc", "abc\\\\abc")
	testEscapeKey(t, "abc:\\:abc", "abc\\:\\\\\\:abc")
	testEscapeKey(t, "abc:\\\\abc", "abc\\:\\\\\\\\abc")
}

func testCalcKeyForEntity(t *testing.T, expected string, query *datastore.Key) {
	t.Helper()

	actual := calcKeyForEntity("project", query)

	if expected != actual {
		t.Errorf("the calced key differed:\nactual  : %s\nexpected: %s", actual, expected)
	}
}

func TestCalcKeyForEntity(t *testing.T) {
	testCalcKeyForEntity(
		t,
		"pj:ns:kind1:i:10",
		&datastore.Key{
			PartitionId: &datastore.PartitionId{
				ProjectId:   "pj",
				NamespaceId: "ns",
			},
			Path: []*datastore.Key_PathElement{
				{
					Kind:   "kind1",
					IdType: &datastore.Key_PathElement_Id{Id: 10},
				},
			},
		},
	)

	testCalcKeyForEntity(
		t,
		"pj:ns:kind1:i:10:kind2:n:abc:kind3:i:11",
		&datastore.Key{
			PartitionId: &datastore.PartitionId{
				ProjectId:   "pj",
				NamespaceId: "ns",
			},
			Path: []*datastore.Key_PathElement{
				{
					Kind:   "kind1",
					IdType: &datastore.Key_PathElement_Id{Id: 10},
				},
				{
					Kind:   "kind2",
					IdType: &datastore.Key_PathElement_Name{Name: "abc"},
				},
				{
					Kind:   "kind3",
					IdType: &datastore.Key_PathElement_Id{Id: 11},
				},
			},
		},
	)

	testCalcKeyForEntity(
		t,
		"project::kind1:i:10:kind2:n:abc:kind3:i:11",
		&datastore.Key{
			PartitionId: nil,
			Path: []*datastore.Key_PathElement{
				{
					Kind:   "kind1",
					IdType: &datastore.Key_PathElement_Id{Id: 10},
				},
				{
					Kind:   "kind2",
					IdType: &datastore.Key_PathElement_Name{Name: "abc"},
				},
				{
					Kind:   "kind3",
					IdType: &datastore.Key_PathElement_Id{Id: 11},
				},
			},
		},
	)
}
