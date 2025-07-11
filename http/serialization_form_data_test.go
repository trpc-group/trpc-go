//
//
// Tencent is pleased to support the open source community by making tRPC available.
//
// Copyright (C) 2023 Tencent.
// All rights reserved.
//
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the  Apache 2.0 License,
// A copy of the Apache 2.0 License is included in this file.
//
//

package http

import (
	"reflect"
	"testing"

	"trpc.group/trpc-go/trpc-go/codec"
)

func Test_getFormDataContentType(t *testing.T) {
	tests := []struct {
		name        string
		marshalType int
		want        string
	}{
		{
			name:        "err",
			marshalType: -1,
			want:        "",
		},
		{
			name:        "normal",
			marshalType: codec.SerializationTypeJSON,
			want:        "application/json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			FormDataMarshalType = tt.marshalType
			if got := getFormDataContentType(); got != tt.want {
				t.Errorf("getFormDataContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewFormDataSerialization(t *testing.T) {
	type args struct {
		tag string
	}
	tests := []struct {
		name string
		args args
		want codec.Serializer
	}{
		{
			name: "",
			args: args{
				tag: "json",
			},
			want: &FormDataSerialization{
				tagName: "json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewFormDataSerialization(tt.args.tag); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewFormDataSerialization() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormDataSerialization_Unmarshal(t *testing.T) {
	type fields struct {
		tagName string
	}
	type args struct {
		in   []byte
		body interface{}
	}

	type request struct {
		Competition string   `protobuf:"bytes,1,opt,name=competition,proto3" json:"competition,omitempty"`
		Season      int32    `protobuf:"varint,2,opt,name=season,proto3" json:"season,omitempty"`
		Teams       []string `protobuf:"bytes,3,rep,name=teams,proto3" json:"teams,omitempty"`
	}

	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  bool
		wantBody interface{}
	}{
		{
			name: "normal",
			fields: fields{
				tagName: "json",
			},
			args: args{
				in:   []byte("competition=NBA&season=2021&teams=%E6%B9%96%E4%BA%BA&teams=%E5%8B%87%E5%A3%AB"),
				body: &request{},
			},
			wantErr: false,
			wantBody: &request{
				Competition: "NBA",
				Season:      2021,
				Teams:       []string{"湖人", "勇士"},
			},
		},
		{
			name: "err",
			fields: fields{
				tagName: "json",
			},
			args: args{
				in:   []byte("competition=NBA&season=2021&teams=%E6%B9%96%E4%BA%BA&teams=%E5%8B%87%E5%A3%AB&competition=OPTA"),
				body: &request{},
			},
			wantErr: true,
			wantBody: &request{
				Competition: "",
				Season:      2021,
				Teams:       []string{"湖人", "勇士"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &FormDataSerialization{
				tagName: tt.fields.tagName,
			}
			if err := j.Unmarshal(tt.args.in, tt.args.body); (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(tt.args.body, tt.wantBody) {
				t.Errorf("Unmarshal() body = %#v, wantBody %#v", tt.args.body, tt.wantBody)
			}
		})
	}
}

func TestFormDataSerialization_Marshal(t *testing.T) {
	type fields struct {
		tagName string
	}
	type args struct {
		serializationType int
		body              interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "err",
			fields: fields{
				tagName: "json",
			},
			args: args{
				serializationType: -1,
				body: &struct {
					DataOrigin  string  `json:"dataOrigin,omitempty"`
					OriginalID  string  `json:"originalID,omitempty"`
					NewPlayerID int64   `json:"newPlayerID,omitempty"`
					PlayerIDs   []int64 `json:"playerIDs,omitempty"`
				}{
					DataOrigin:  "opta",
					OriginalID:  "40669",
					NewPlayerID: 148681,
					PlayerIDs:   []int64{13, 14},
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "normal",
			fields: fields{
				tagName: "json",
			},
			args: args{
				serializationType: codec.SerializationTypeJSON,
				body: &struct {
					DataOrigin  string  `json:"dataOrigin,omitempty"`
					OriginalID  string  `json:"originalID,omitempty"`
					NewPlayerID int64   `json:"newPlayerID,omitempty"`
					PlayerIDs   []int64 `json:"playerIDs,omitempty"`
				}{
					DataOrigin:  "opta",
					OriginalID:  "40669",
					NewPlayerID: 148681,
					PlayerIDs:   []int64{13, 14},
				},
			},
			want:    `{"dataOrigin":"opta","originalID":"40669","newPlayerID":148681,"playerIDs":[13,14]}`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			j := &FormDataSerialization{
				tagName: tt.fields.tagName,
			}
			FormDataMarshalType = tt.args.serializationType
			got, err := j.Marshal(tt.args.body)
			if (err != nil) != tt.wantErr {
				t.Errorf("Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(string(got), tt.want) {
				t.Errorf("Marshal() got = %v, want %v", string(got), tt.want)
			}
		})
	}
}
