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

package restful_test

import (
	"reflect"
	"testing"

	"trpc.group/trpc-go/trpc-go/restful"
	"github.com/stretchr/testify/require"
)

// TestJSONSerializer_Marshal tests the Marshal function of JSONSerializer.
func TestJSONSerializer_Marshal(t *testing.T) {
	serializer := restful.JSONSerializer{}

	// Define a sample struct to marshal.
	type sampleStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	// Create an instance of the struct with some data.
	sampleData := sampleStruct{Name: "John Doe", Age: 30}

	// Marshal the data.
	marshaledData, err := serializer.Marshal(sampleData)

	// Check for errors.
	require.NoError(t, err, "Marshaling should not produce an error")

	// Check if the marshaled data is a valid JSON representation of sampleData.
	expectedJSON := `{"name":"John Doe","age":30}`
	require.JSONEq(t, expectedJSON, string(marshaledData), "Marshaled data should match expected JSON")
}

// TestJSONSerializer_Unmarshal tests the Unmarshal function of JSONSerializer.
func TestJSONSerializer_Unmarshal(t *testing.T) {
	serializer := restful.JSONSerializer{}

	// Define a sample JSON string to unmarshal.
	jsonData := `{"name":"Jane Doe","age":25}`

	// Define the struct that matches the expected JSON structure.
	type sampleStruct struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	// Create an instance of the struct where the data will be unmarshaled.
	var resultData sampleStruct

	// Unmarshal the data.
	err := serializer.Unmarshal([]byte(jsonData), &resultData)

	// Check for errors.
	require.NoError(t, err, "Unmarshaling should not produce an error")

	// Check if the unmarshaled data matches the expected struct data.
	expectedData := sampleStruct{Name: "Jane Doe", Age: 25}
	require.True(t,
		reflect.DeepEqual(resultData, expectedData), "Unmarshaled data should match the expected struct data")
}

// TestJSONSerializer_Name tests the Name function of JSONSerializer.
func TestJSONSerializer_Name(t *testing.T) {
	serializer := restful.JSONSerializer{}

	// Call the Name function.
	name := serializer.Name()

	// Check if the name matches the expected content type.
	require.Equal(t, "application/json", name, "Serializer name should be 'application/json'")
}

// TestJSONSerializer_ContentType tests the ContentType function of JSONSerializer.
func TestJSONSerializer_ContentType(t *testing.T) {
	serializer := restful.JSONSerializer{}

	// Call the ContentType function.
	contentType := serializer.ContentType()

	// Check if the content type matches the expected content type.
	require.Equal(t, "application/json", contentType, "Serializer content type should be 'application/json'")
}
