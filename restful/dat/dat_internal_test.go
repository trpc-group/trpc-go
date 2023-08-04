package dat

import (
	"reflect"
	"testing"
)

func Test_newFieldDict(t *testing.T) {
	tests := []struct {
		name string
		fps  fieldPaths
		want fieldDict
	}{
		{"nil", nil, fieldDict{}},
		{"empty paths", [][]string{}, fieldDict{}},
		{"one path with unique fields", [][]string{{"bb", "cc", "aa"}}, fieldDict{"aa": 0, "bb": 1, "cc": 2}},
		{"one path with duplicated fields", [][]string{{"aa", "aa", "aa"}}, fieldDict{"aa": 0}},
		{
			"multiple paths with unique fields",
			[][]string{
				{"acb", "cab"},
				{"abc", "bac"},
				{"bca", "cba"},
			},
			fieldDict{
				"abc": 0,
				"acb": 1,
				"bac": 2,
				"bca": 3,
				"cab": 4,
				"cba": 5,
			},
		},
		{
			"multiple paths with duplicated fields: case 1",
			[][]string{
				{"acb", "bca"},
				{"abc", "bac"},
				{"bca", "cba"},
			},
			fieldDict{
				"abc": 0,
				"acb": 1,
				"bac": 2,
				"bca": 3,
				"cba": 4,
			},
		},
		{
			"multiple paths with duplicated fields: case 2",
			[][]string{
				{"baz"},
				{"foobar", "foo"},
				{"foobar", "bar"},
				{"foobar", "baz", "baz"},
				{"foo", "bar", "baz", "qux"},
			},
			fieldDict{
				"bar":    0,
				"baz":    1,
				"foo":    2,
				"foobar": 3,
				"qux":    4,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := newFieldDict(tt.fps); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newFieldDict() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDoubleArrayTrie_fetch(t *testing.T) {
	var fps = [][]string{
		{"baz"},
		{"foobar", "foo"},
		{"foobar", "bar"},
		{"foobar", "baz", "baz"},
		{"foo", "bar", "baz", "qux"},
	}
	// trie:
	//     (*root)
	//    /   ｜    \
	//  baz  foo    foobar
	//        |    /   |  \
	//       bar  bar baz foo
	//        |        |
	//       baz 	  baz
	//        |
	//       qux
	// ------------------------------
	//       (*0)
	//    /   ｜   \
	//   2    3     4
	//        |   /  |  \
	//        1  1   2   3
	//        |      |
	//        2 	 2
	//        |
	//        5
	dat := mustBuild(t, fps)
	tests := []struct {
		name    string
		parent  *node
		want    []*node
		wantErr bool
	}{
		{
			"root",
			&node{left: 0, right: len(dat.fps), depth: 0},
			[]*node{
				{code: 2, left: 0, right: 1, depth: 1},
				{code: 3, left: 1, right: 2, depth: 1},
				{code: 4, left: 2, right: 5, depth: 1}},
			false,
		},
		{
			"internal node-baz",
			&node{left: 1, right: 2, depth: 2},
			[]*node{
				{code: 2, left: 1, right: 2, depth: 3},
			},
			false,
		},
		{
			"leaf-qux",
			&node{left: 1, right: 2, depth: 3},
			[]*node{
				{code: 5, left: 1, right: 2, depth: 4},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dat.fetch(tt.parent)
			if (err != nil) != tt.wantErr {
				t.Errorf("fetch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for _, item := range got {
				t.Log(item.code, item.left, item.right, item.depth)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("fetch() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustBuild(t *testing.T, fps [][]string) *DoubleArrayTrie {
	t.Helper()
	trie, err := Build(fps)
	if err != nil {
		t.Fatalf("could not build DoubleArrayTrie under test: %v", err)
	}
	return trie
}
