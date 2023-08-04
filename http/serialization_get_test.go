package http_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"trpc.group/trpc-go/trpc-go/codec"
	"trpc.group/trpc-go/trpc-go/http"
)

func TestGetSerializerRegister(t *testing.T) {
	defer func() {
		e := recover()
		require.Nil(t, e)
	}()

	s := codec.GetSerializer(codec.SerializationTypeGet)
	defer func() {
		codec.RegisterSerializer(codec.SerializationTypeGet, s)
	}()
	codec.RegisterSerializer(codec.SerializationTypeGet, http.NewGetSerialization("json"))
	serializer := codec.GetSerializer(codec.SerializationTypeGet)
	require.NotNil(t, serializer)
}

func TestGetSerializer(t *testing.T) {
	require := require.New(t)
	s := codec.GetSerializer(codec.SerializationTypeGet)

	type Data struct {
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

	var expects = []*Data{
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
		"{\"x\":1,\"y\":\"nice\",\"z\":[\"3\"]}",
		"{\"x\":1,\"y\":\"2\",\"z\":[\"\"]}",
		"{\"x\":1,\"y\":\"2\",\"z\":null}",
		"{\"x\":1,\"y\":\"2\",\"z\":[\"z1\",\"z2\"]}",
	}

	for i, query := range queries {
		data := &Data{}
		s.Unmarshal([]byte(query), &data)
		require.Equal(data.X, expects[i].X, "x should be equal")
		require.Equal(data.Y, expects[i].Y, "y should be equal")
		require.Equal(data.Z, expects[i].Z, "z should be equal")
	}

	for i, query := range expects {
		buf, _ := s.Marshal(&query)
		require.Equal(string(buf), expectedQueries[i], "x should be equal")
	}

	old := codec.GetSerializer(codec.SerializationTypeJSON)
	defer func() {
		codec.RegisterSerializer(codec.SerializationTypeJSON, old)
	}()
	codec.RegisterSerializer(codec.SerializationTypeJSON, &codec.NoopSerialization{})
	_, err := s.Marshal(queries)
	require.NotNil(t, err, "json codec empty")

}

func TestGetUnmarshal(t *testing.T) {
	require := require.New(t)
	s := codec.GetSerializer(codec.SerializationTypeGet)

	type formStruct struct{}
	form := &formStruct{}

	require.NotNil(s.Unmarshal([]byte("%gh&%ij"), &form))
	require.NotNil(s.Unmarshal([]byte("x=1&y=2"), (map[string]interface{})(nil)))

	type queryStruct struct {
		Query []byte `json:"query"`
	}
	query := &queryStruct{}
	require.NotNil(s.Unmarshal([]byte("%gh&%ij"), &query))
	require.Nil(s.Unmarshal([]byte("x=1&y=2"), &query))

	// Test Chinese.
	query = &queryStruct{}
	require.Nil(s.Unmarshal([]byte("query=中文&y=2"), &query))
	require.Equal([]byte("中文"), query.Query)
}

func TestGetMarshal(t *testing.T) {
	require := require.New(t)
	s := codec.GetSerializer(codec.SerializationTypeGet)
	old := codec.GetSerializer(codec.SerializationTypeJSON)
	defer func() {
		codec.RegisterSerializer(codec.SerializationTypeJSON, old)
	}()
	codec.RegisterSerializer(codec.SerializationTypeJSON, nil)
	_, err := s.Marshal(require)
	require.NotNil(err)
}
