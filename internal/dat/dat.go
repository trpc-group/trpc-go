//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 THL A29 Limited, a Tencent company.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

// Package dat provides a double array trie.
// A DAT is used to filter protobuf fields specified by HttpRule.
// These fields will be ignored if they also present in http request query parameters
// to prevent repeated reference.
package dat

import (
	"errors"
	"math"
	"sort"
)

var (
	errByDictOrder = errors.New("not by dict order")
	errEncoded     = errors.New("field name not encoded")
)

const (
	defaultArraySize         = 64   // default array size of dat
	minExpansionRate         = 1.05 // minimal expansion rate, based on experience
	nextCheckPosStrategyRate = 0.95 // next check pos strategy rate, based on experience
)

// DoubleArrayTrie is a double array trie.
// It's based on https://github.com/komiya-atsushi/darts-java.
// State Transition Equation：
//
//	base[0] = 1
//	base[s] + c = t
//	check[t] = base[s]
type DoubleArrayTrie struct {
	base         []int      // base array
	check        []int      // check array
	used         []bool     // used array
	size         int        // size of base/check/used arrays
	allocSize    int        // allocated size of base/check/used arrays
	fps          fieldPaths // fieldPaths
	dict         fieldDict  // fieldDict
	progress     int        // number of processed fieldPaths
	nextCheckPos int        // record next index of begin to prevent start over from 0
}

// node is node of DAT.
type node struct {
	code  int // code = dictCodeOfFieldName + 1, dictCodeOfFieldName: [0, 1, 2, ..., n-1]
	depth int // depth of node
	left  int // left boundary
	right int // right boundary
}

// Build performs static construction of a DAT.
func Build(fps [][]string) (*DoubleArrayTrie, error) {
	// sort
	sort.Sort(fieldPaths(fps))

	// init dat
	dat := &DoubleArrayTrie{
		fps:  fps,
		dict: newFieldDict(fps),
	}
	dat.resize(defaultArraySize)
	dat.base[0] = 1

	// root node handling
	root := &node{
		right: len(dat.fps),
	}
	children, err := dat.fetch(root)
	if err != nil {
		return nil, err
	}
	if _, err := dat.insert(children); err != nil {
		return nil, err
	}

	// shrink
	dat.resize(dat.size)

	return dat, nil
}

// CommonPrefixSearch check if input fieldPath has common prefix with fps in DAT.
func (dat *DoubleArrayTrie) CommonPrefixSearch(fieldPath []string) bool {
	var pos int
	baseValue := dat.base[0]

	for _, name := range fieldPath {
		// get dict code
		v, ok := dat.dict[name]
		if !ok {
			break
		}
		code := v + 1 // code = dictCodeOfFieldName + 1

		// check if leaf node has been reached, that is, check if next node is NULL according to
		// the State Transition Equation.
		if baseValue == dat.check[baseValue] && dat.base[baseValue] < 0 {
			// has reached leaf node，it's the common prefix.
			return true
		}

		// state transition
		pos = baseValue + code
		if pos >= len(dat.check) || baseValue != dat.check[pos] { // mismatch
			return false
		}
		baseValue = dat.base[pos]
	}

	// check again if leaf node has been reached for last state transition
	if baseValue == dat.check[baseValue] && dat.base[baseValue] < 0 {
		// has reached leaf node，it's the common prefix.
		return true
	}

	return false
}

// fetch returns children nodes given parent node.
// If the fps in DAT is like：
//
//	["foobar", "foo", "bar"]
//	["foobar", "baz"]
//	["foo", "qux"]
//
// children, _ := dat.fetch(root)，children should be ["foobar", "foo"]，
// and their depths should all be 1.
func (dat *DoubleArrayTrie) fetch(parent *node) ([]*node, error) {
	var (
		children []*node // children nodes would be returned
		prev     int     // code of prev child node
	)

	// search range [parent.left, parent.right)
	// for root node，search range [0, len(dat.fps))
	for i := parent.left; i < parent.right; i++ {
		if len(dat.fps[i]) < parent.depth { // all fp of fps[i] have been fetched
			continue
		}

		var curr int // code of curr child node
		if len(dat.fps[i]) > parent.depth {
			v, ok := dat.dict[dat.fps[i][parent.depth]]
			if !ok { // not encoded
				return nil, errEncoded
			}
			curr = v + 1 // code = dictCodeOfFieldName + 1
		}

		// not by dict order
		if prev > curr {
			return nil, errByDictOrder
		}

		// Normally, if curr == prev, skip this.
		// But curr == prev && len(children) == 0 makes an exception,
		// it means fetching fp from fps[i] comes to an end and an empty node should be added
		// like an EOF.
		if curr != prev || len(children) == 0 {
			// update right boundary of prev child node
			if len(children) != 0 {
				children[len(children)-1].right = i
			}
			// curr child node
			// no need to update right boundary,
			// let next child node update this node's right boundary
			children = append(children, &node{
				code:  curr,
				depth: parent.depth + 1, // depth +1
				left:  i,
			})
		}

		prev = curr
	}

	// update right boundary of the last child node
	if len(children) > 0 {
		children[len(children)-1].right = parent.right // same right boundary as parent node
	}

	return children, nil
}

// max returns the bigger int value.
func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

// loopForBegin loops for begin value that meets the condition.
func (dat *DoubleArrayTrie) loopForBegin(children []*node) (int, error) {
	var (
		begin        int                                         // begin to loop for
		numOfNonZero int                                         // number of non zero
		pos          = max(children[0].code, dat.nextCheckPos-1) // prevent start over from 0 to loop for begin value
	)

	for first := true; ; { // whether first time to meet a non zero
		pos++
		if dat.allocSize <= pos { // expand
			dat.resize(pos + 1)
		}
		if dat.check[pos] != 0 { // occupied
			numOfNonZero++
			continue
		} else {
			if first {
				dat.nextCheckPos = pos
				first = false
			}
		}

		// try this begin value
		begin = pos - children[0].code

		// compare with lastChildPos to check if expansion is needed
		if lastChildPos := begin + children[len(children)-1].code; dat.allocSize <= lastChildPos {
			// rate = {total number of fieldPaths} / ({number of processed fieldPaths} + 1), but not less than 1.05
			rate := math.Max(minExpansionRate, float64(1.0*len(dat.fps)/(dat.progress+1)))
			dat.resize(int(float64(dat.allocSize) * rate))
		}

		if dat.used[begin] { // check dup
			continue
		}

		// check if remaining children nodes could be inserted
		conflict := func() bool {
			for i := 1; i < len(children); i++ {
				if dat.check[begin+children[i].code] != 0 {
					return true
				}
			}
			return false
		}
		// if conflicting, next pos
		if conflict() {
			continue
		}
		// no conflicting, found the begin value
		break
	}

	// if nodes from nextCheckPos to pos are all occupied, set nextCheckPos to pos
	if float64((1.0*numOfNonZero)/(pos-dat.nextCheckPos+1)) >= nextCheckPosStrategyRate {
		dat.nextCheckPos = pos
	}

	return begin, nil
}

// insert inserts children nodes into DAT, returns begin value that is looking for.
func (dat *DoubleArrayTrie) insert(children []*node) (int, error) {
	// loop for begin value
	begin, err := dat.loopForBegin(children)
	if err != nil {
		return 0, err
	}

	dat.used[begin] = true
	dat.size = max(dat.size, begin+children[len(children)-1].code+1)

	// check arrays assignment
	for i := range children {
		dat.check[begin+children[i].code] = begin
	}

	// dfs
	for _, child := range children {
		grandchildren, err := dat.fetch(child)
		if err != nil {
			return 0, err
		}
		if len(grandchildren) == 0 { // no children nodes
			dat.base[begin+child.code] = -child.left - 1
			dat.progress++
			continue
		}
		t, err := dat.insert(grandchildren)
		if err != nil {
			return 0, err
		}
		// base arrays assignment
		dat.base[begin+child.code] = t
	}

	return begin, nil
}

// resize changes the size of the arrays.
func (dat *DoubleArrayTrie) resize(newSize int) {
	newBase := make([]int, newSize, newSize)
	newCheck := make([]int, newSize, newSize)
	newUsed := make([]bool, newSize, newSize)

	if dat.allocSize > 0 {
		copy(newBase, dat.base)
		copy(newCheck, dat.check)
		copy(newUsed, dat.used)
	}

	dat.base = newBase
	dat.check = newCheck
	dat.used = newUsed

	dat.allocSize = newSize
}

type fieldPaths [][]string

// Len implements sort.Interface
func (fps fieldPaths) Len() int { return len(fps) }

// Swap implements sort.Interface
func (fps fieldPaths) Swap(i, j int) { fps[i], fps[j] = fps[j], fps[i] }

// Less implements sort.Interface
func (fps fieldPaths) Less(i, j int) bool {
	var k int
	for k = 0; k < len(fps[i]) && k < len(fps[j]); k++ {
		if fps[i][k] < fps[j][k] {
			return true
		}
		if fps[i][k] > fps[j][k] {
			return false
		}
	}
	return k < len(fps[j])
}

type fieldDict map[string]int // FieldName -> DictCodeOfFieldName

func newFieldDict(fps fieldPaths) fieldDict {
	dict := make(map[string]int)
	// rm dup
	for _, fieldPath := range fps {
		for _, name := range fieldPath {
			dict[name] = 0
		}
	}

	// sort
	fields := make([]string, 0, len(dict))
	for name := range dict {
		fields = append(fields, name)
	}
	sort.Sort(sort.StringSlice(fields))

	// dict assignment

	for code, name := range fields {
		dict[name] = code
	}
	return dict
}
