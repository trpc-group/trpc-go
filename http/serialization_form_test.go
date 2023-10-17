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

package http_test

import (
	"fmt"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/http"
)

// go test -v -coverprofile=cover.out
// go tool cover -func=cover.out

func TestFormSerializerRegister(t *testing.T) {
	defer func() {
		e := recover()
		require.Nil(t, e)
	}()

	s := codec.GetSerializer(codec.SerializationTypeForm)
	defer func() {
		codec.RegisterSerializer(codec.SerializationTypeForm, s)
	}()
	codec.RegisterSerializer(codec.SerializationTypeForm, http.NewFormSerialization("json"))
	formSerializer := codec.GetSerializer(codec.SerializationTypeForm)
	require.NotNil(t, formSerializer)
}

func TestFormSerializer(t *testing.T) {
	require := require.New(t)
	formSerializer := codec.GetSerializer(codec.SerializationTypeForm)

	type FormStruct struct {
		X int      `json:"x"`
		Y string   `json:"y"`
		Z []string `json:"z"`
	}

	var queries = []string{
		"x=1&y=nice&z=3",
		"x=1&y=2&z",
		"x=1&y=2",
		"x=1&y=2&z=z1&z=z2",
	}

	var expects = []*FormStruct{
		{
			X: 1,
			Y: "nice",
			Z: []string{"3"},
		},
		{
			X: 1,
			Y: "2",
			Z: []string{""},
		},
		{
			X: 1,
			Y: "2",
			Z: nil,
		},
		{
			X: 1,
			Y: "2",
			Z: []string{"z1", "z2"},
		},
	}

	var expectedQueries = []string{
		"x=1&y=nice&z=3",
		"x=1&y=2&z=",
		"x=1&y=2",
		"x=1&y=2&z=z1&z=z2",
	}

	for i, query := range queries {
		form := &FormStruct{}
		formSerializer.Unmarshal([]byte(query), &form)
		require.Equal(form.X, expects[i].X, "x should be equal")
		require.Equal(form.Y, expects[i].Y, "y should be equal")
		require.Equal(form.Z, expects[i].Z, "z should be equal")

		m := make(map[string]interface{})
		formSerializer.Unmarshal([]byte(query), &m)
		require.Equal(m["y"], expects[i].Y, "y should be equal")
	}

	for i, query := range expects {
		buf, _ := formSerializer.Marshal(&query)
		require.Equal(string(buf), expectedQueries[i], "x should be equal")
	}

}

func TestUnmarshal(t *testing.T) {
	require := require.New(t)
	s := codec.GetSerializer(codec.SerializationTypeForm)

	type formStruct struct{}
	form := &formStruct{}

	require.NotNil(s.Unmarshal([]byte("%gh&%ij"), &form))
	require.NotNil(s.Unmarshal([]byte("x=1&y=2"), (map[string]interface{})(nil)))
}

func TestMarshal(t *testing.T) {
	require := require.New(t)
	s := codec.GetSerializer(codec.SerializationTypeForm)

	v := make(url.Values)
	buf, err := s.Marshal(v)
	require.NotNil(buf)
	require.Nil(err)

	type testError struct {
		Time      time.Time
		BadMapKey map[time.Time]string
		Iface     map[interface{}]string
		Struct    map[struct{}]string
	}

	test := testError{
		Iface:  map[interface{}]string{nil: "time"},
		Struct: map[struct{}]string{{}: "str"},
	}
	_, err = s.Marshal(&test)
	require.NotNil(err)

	nestedMap := map[string]interface{}{
		"id": "123",
		"attr": map[string]interface{}{
			"name": "haha",
		},
	}
	_, err = s.Marshal(nestedMap)
	require.Nil(err)
}

type queryRequest struct {
	Ints  []int  `json:"ints"`
	Query []byte `json:"query"`
}

func TestUnmarshalBytes(t *testing.T) {
	query := &queryRequest{}
	s := codec.GetSerializer(codec.SerializationTypeForm)
	require.NotNil(t, s.Unmarshal([]byte("%gh&%ij"), &query))
	require.Nil(t, s.Unmarshal([]byte("x=1&y=2"), &query))
}

func TestUnmarshalChinese(t *testing.T) {
	query := &queryRequest{}
	s := codec.GetSerializer(codec.SerializationTypeForm)
	err := s.Unmarshal([]byte("ints=1&ints=2&query=中文"), &query)
	require.Nil(t, err, fmt.Sprintf("err: %+v", err))
	require.Equal(t, []byte("中文"), query.Query)
	require.Equal(t, []int{1, 2}, query.Ints)
}

func TestUnmarshalNested(t *testing.T) {
	type Nested struct {
		Msg string `json:"msg"`
	}
	type nested struct {
		Nest Nested `json:"nest"`
	}
	q := &nested{}
	s := codec.GetSerializer(codec.SerializationTypeForm)
	err := s.Unmarshal([]byte("nest.msg=hhh"), &q)
	require.Nil(t, err, fmt.Sprintf("err: %+v", err))
	require.Equal(t, "hhh", q.Nest.Msg)
}
