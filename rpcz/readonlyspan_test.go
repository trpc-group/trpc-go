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

package rpcz

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	// currentTime use to replace time.Now() for reproducibility in unit test.
	currentTime, _ = time.Parse(time.StampMicro, "Nov 24 15:34:08.151326")
	clientSpan     = &ReadOnlySpan{
		Name:      "client",
		StartTime: currentTime,
		EndTime:   currentTime.Add(30 * time.Second),
		Attributes: []Attribute{
			{
				Name:  "RPCName",
				Value: "/trpc.testing.end2end.TestTRPC/EmptyCall",
			},
			{
				Name:  "Error",
				Value: fmt.Errorf(""),
			},
			{
				Name: TRPCAttributeFilterNames,
				Value: []string{
					"1",
					"2",
					"3",
				},
			},
		},
		ChildSpans: []*ReadOnlySpan{
			{
				Name:      "1",
				StartTime: currentTime.Add(1 * time.Second),
				EndTime:   currentTime.Add(29 * time.Second),
				ChildSpans: []*ReadOnlySpan{
					{
						Name:      "2",
						StartTime: currentTime.Add(2 * time.Second),
						EndTime:   currentTime.Add(28 * time.Second),
						ChildSpans: []*ReadOnlySpan{
							{
								Name:      "3",
								StartTime: currentTime.Add(3 * time.Second),
								EndTime:   currentTime.Add(27 * time.Second),
								ChildSpans: []*ReadOnlySpan{
									{
										Name:      "CallFunc",
										StartTime: currentTime.Add(4 * time.Second),
										EndTime:   currentTime.Add(26 * time.Second),
										ChildSpans: []*ReadOnlySpan{
											{
												Name:      "Marshal",
												StartTime: currentTime.Add(5 * time.Second),
												EndTime:   currentTime.Add(6 * time.Second),
											},
											{
												Name:      "Compress",
												StartTime: currentTime.Add(7 * time.Second),
												EndTime:   currentTime.Add(8 * time.Second),
											},
											{
												Name:      "EncodeProtocolHead",
												StartTime: currentTime.Add(9 * time.Second),
												EndTime:   currentTime.Add(10 * time.Second),
											},
											{
												Name:      "SendMessage",
												StartTime: currentTime.Add(11 * time.Second),
												EndTime:   currentTime.Add(12 * time.Second),
											},
											{
												Name:      "ReceiveMessage",
												StartTime: currentTime.Add(13 * time.Second),
												EndTime:   currentTime.Add(14 * time.Second),
											},
											{
												Name:      "DecodeProtocolHead",
												StartTime: currentTime.Add(15 * time.Second),
												EndTime:   currentTime.Add(16 * time.Second),
											},
											{
												Name:      "Decompress",
												StartTime: currentTime.Add(17 * time.Second),
												EndTime:   currentTime.Add(18 * time.Second),
											},
											{
												Name:      "Unmarshal",
												StartTime: currentTime.Add(19 * time.Second),
												EndTime:   currentTime.Add(20 * time.Second),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
	serverSpan = &ReadOnlySpan{
		Name:      "server",
		StartTime: currentTime,
		EndTime:   currentTime.Add(30 * time.Second),
		Attributes: []Attribute{
			{
				Name:  "RPCName",
				Value: "/trpc.testing.end2end.TestTRPC/EmptyCall",
			},
			{
				Name:  "Error",
				Value: fmt.Errorf(""),
			},
			{
				Name:  "RequestSize",
				Value: 125,
			},
			{
				Name:  "ResponseSize",
				Value: 18,
			},
		},
		ChildSpans: []*ReadOnlySpan{
			{
				Name:      "DecodeProtocolHead",
				StartTime: currentTime.Add(1 * time.Second),
				EndTime:   currentTime.Add(2 * time.Second),
			},
			{
				Name:      "Decompress",
				StartTime: currentTime.Add(3 * time.Second),
				EndTime:   currentTime.Add(4 * time.Second),
			},
			{
				Name:      "Unmarshal",
				StartTime: currentTime.Add(5 * time.Second),
				EndTime:   currentTime.Add(6 * time.Second),
			},
			{
				Name:      "filter1",
				StartTime: currentTime.Add(7 * time.Second),
				EndTime:   currentTime.Add(20 * time.Second),
				ChildSpans: []*ReadOnlySpan{
					{
						Name:      "filter2",
						StartTime: currentTime.Add(8 * time.Second),
						EndTime:   currentTime.Add(19 * time.Second),
						ChildSpans: []*ReadOnlySpan{
							{
								Name:      "filter3",
								StartTime: currentTime.Add(9 * time.Second),
								EndTime:   currentTime.Add(18 * time.Second),
								ChildSpans: []*ReadOnlySpan{
									{
										Name:      "HandleFunc",
										StartTime: currentTime.Add(10 * time.Second),
										EndTime:   currentTime.Add(17 * time.Second),
									},
								},
							},
						},
					},
				},
			},
			{
				Name:      "Marshal",
				StartTime: currentTime.Add(21 * time.Second),
				EndTime:   currentTime.Add(22 * time.Second),
			},
			{
				Name:      "Compress",
				StartTime: currentTime.Add(23 * time.Second),
				EndTime:   currentTime.Add(24 * time.Second),
			},
			{
				Name:      "EncodeProtocolHead",
				StartTime: currentTime.Add(25 * time.Second),
				EndTime:   currentTime.Add(26 * time.Second),
			},
			{
				Name:      "SendMessage",
				StartTime: currentTime.Add(27 * time.Second),
				EndTime:   currentTime.Add(28 * time.Second),
			},
		},
	}
	serverSpanHasClientChildSpan = &ReadOnlySpan{
		Name:      "server",
		StartTime: currentTime,
		EndTime:   currentTime.Add(30 * time.Second),
		ChildSpans: []*ReadOnlySpan{
			{
				Name:      "DecodeProtocolHead",
				StartTime: currentTime.Add(1 * time.Second),
				EndTime:   currentTime.Add(2 * time.Second),
			},
			{
				Name:      "Decompress",
				StartTime: currentTime.Add(3 * time.Second),
				EndTime:   currentTime.Add(4 * time.Second),
			},
			{
				Name:      "Unmarshal",
				StartTime: currentTime.Add(5 * time.Second),
				EndTime:   currentTime.Add(6 * time.Second),
			},
			{
				Name:      "",
				StartTime: currentTime.Add(7 * time.Second),
				EndTime:   currentTime.Add(20 * time.Second),
				ChildSpans: []*ReadOnlySpan{
					{
						Name:      "",
						StartTime: currentTime.Add(8 * time.Second),
						EndTime:   currentTime.Add(19 * time.Second),
						ChildSpans: []*ReadOnlySpan{
							{
								Name:      "",
								StartTime: currentTime.Add(9 * time.Second),
								EndTime:   currentTime.Add(18 * time.Second),
								ChildSpans: []*ReadOnlySpan{
									{
										Name:      "HandleFunc",
										StartTime: currentTime.Add(10 * time.Second),
										EndTime:   currentTime.Add(17 * time.Second),
										ChildSpans: []*ReadOnlySpan{
											clientSpan, clientSpan,
										},
									},
								},
							},
						},
					},
				},
			},
			{
				Name:      "Marshal",
				StartTime: currentTime.Add(21 * time.Second),
				EndTime:   currentTime.Add(22 * time.Second),
			},
			{
				Name:      "Compress",
				StartTime: currentTime.Add(23 * time.Second),
				EndTime:   currentTime.Add(24 * time.Second),
			},
			{
				Name:      "EncodeProtocolHead",
				StartTime: currentTime.Add(25 * time.Second),
				EndTime:   currentTime.Add(26 * time.Second),
			},
			{
				Name:      "SendMessage",
				StartTime: currentTime.Add(27 * time.Second),
				EndTime:   currentTime.Add(28 * time.Second),
			},
		},
	}
	serverSpanHasEvents = &ReadOnlySpan{
		Name:      "server",
		StartTime: currentTime,
		EndTime:   currentTime.Add(30 * time.Second),
		Attributes: []Attribute{
			{
				Name:  "RPCName",
				Value: "/trpc.testing.end2end.TestTRPC/EmptyCall",
			},
			{
				Name:  "Error",
				Value: fmt.Errorf(""),
			},
			{
				Name:  "RequestSize",
				Value: 125,
			},
			{
				Name:  "ResponseSize",
				Value: 18,
			},
		},
		Events: []Event{
			{
				Name: "enter DecodeProtocolHead",
				Time: currentTime.Add(500 * time.Millisecond),
			},
			{
				Name: "handle DecodeProtocolHead",
				Time: currentTime.Add(1500 * time.Millisecond),
			},
			{
				Name: "leave DecodeProtocolHead, enter Decompress",
				Time: currentTime.Add(2500 * time.Millisecond),
			},
			{
				Name: "handle Decompress",
				Time: currentTime.Add(3500 * time.Millisecond),
			},
			{
				Name: "leave Decompress, enter Unmarshal",
				Time: currentTime.Add(4500 * time.Millisecond),
			},
		},
		ChildSpans: []*ReadOnlySpan{
			{
				Name:      "DecodeProtocolHead",
				StartTime: currentTime.Add(1 * time.Second),
				EndTime:   currentTime.Add(2 * time.Second),
			},
			{
				Name:      "Decompress",
				StartTime: currentTime.Add(3 * time.Second),
				EndTime:   currentTime.Add(4 * time.Second),
			},
			{
				Name:      "Unmarshal",
				StartTime: currentTime.Add(5 * time.Second),
				EndTime:   currentTime.Add(6 * time.Second),
			},
			{
				Name:      "filter1",
				StartTime: currentTime.Add(7 * time.Second),
				EndTime:   currentTime.Add(20 * time.Second),
				ChildSpans: []*ReadOnlySpan{
					{
						Name:      "HandleFunc",
						StartTime: currentTime.Add(10 * time.Second),
						EndTime:   currentTime.Add(17 * time.Second),
					},
				},
			},
			{
				Name:      "Marshal",
				StartTime: currentTime.Add(21 * time.Second),
				EndTime:   currentTime.Add(22 * time.Second),
			},
			{
				Name:      "Compress",
				StartTime: currentTime.Add(23 * time.Second),
				EndTime:   currentTime.Add(24 * time.Second),
			},
			{
				Name:      "EncodeProtocolHead",
				StartTime: currentTime.Add(25 * time.Second),
				EndTime:   currentTime.Add(26 * time.Second),
			},
			{
				Name:      "SendMessage",
				StartTime: currentTime.Add(27 * time.Second),
				EndTime:   currentTime.Add(28 * time.Second),
			},
		},
	}
)

func TestReadOnlySpan_PrintDetail(t *testing.T) {
	t.Run("client span", func(t *testing.T) {
		require.Equal(
			t,
			`span: (client, 0)
  time: (Nov 24 15:34:08.151326, Nov 24 15:34:38.151326)
  duration: (0, 30s, 0)
  attributes: (RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, )
  span: (1, 0)
    time: (Nov 24 15:34:09.151326, Nov 24 15:34:37.151326)
    duration: (1s, 28s, 1s)
    span: (2, 0)
      time: (Nov 24 15:34:10.151326, Nov 24 15:34:36.151326)
      duration: (1s, 26s, 1s)
      span: (3, 0)
        time: (Nov 24 15:34:11.151326, Nov 24 15:34:35.151326)
        duration: (1s, 24s, 1s)
        span: (CallFunc, 0)
          time: (Nov 24 15:34:12.151326, Nov 24 15:34:34.151326)
          duration: (1s, 22s, 1s)
          span: (Marshal, 0)
            time: (Nov 24 15:34:13.151326, Nov 24 15:34:14.151326)
            duration: (1s, 1s, 20s)
          span: (Compress, 0)
            time: (Nov 24 15:34:15.151326, Nov 24 15:34:16.151326)
            duration: (3s, 1s, 18s)
          span: (EncodeProtocolHead, 0)
            time: (Nov 24 15:34:17.151326, Nov 24 15:34:18.151326)
            duration: (5s, 1s, 16s)
          span: (SendMessage, 0)
            time: (Nov 24 15:34:19.151326, Nov 24 15:34:20.151326)
            duration: (7s, 1s, 14s)
          span: (ReceiveMessage, 0)
            time: (Nov 24 15:34:21.151326, Nov 24 15:34:22.151326)
            duration: (9s, 1s, 12s)
          span: (DecodeProtocolHead, 0)
            time: (Nov 24 15:34:23.151326, Nov 24 15:34:24.151326)
            duration: (11s, 1s, 10s)
          span: (Decompress, 0)
            time: (Nov 24 15:34:25.151326, Nov 24 15:34:26.151326)
            duration: (13s, 1s, 8s)
          span: (Unmarshal, 0)
            time: (Nov 24 15:34:27.151326, Nov 24 15:34:28.151326)
            duration: (15s, 1s, 6s)
`,
			clientSpan.PrintDetail(""),
		)

	})
	t.Run("server span without proxy at service impl function", func(t *testing.T) {
		require.Equal(
			t,
			`span: (server, 0)
  time: (Nov 24 15:34:08.151326, Nov 24 15:34:38.151326)
  duration: (0, 30s, 0)
  attributes: (RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, ),(RequestSize, 125),(ResponseSize, 18)
  span: (DecodeProtocolHead, 0)
    time: (Nov 24 15:34:09.151326, Nov 24 15:34:10.151326)
    duration: (1s, 1s, 28s)
  span: (Decompress, 0)
    time: (Nov 24 15:34:11.151326, Nov 24 15:34:12.151326)
    duration: (3s, 1s, 26s)
  span: (Unmarshal, 0)
    time: (Nov 24 15:34:13.151326, Nov 24 15:34:14.151326)
    duration: (5s, 1s, 24s)
  span: (filter1, 0)
    time: (Nov 24 15:34:15.151326, Nov 24 15:34:28.151326)
    duration: (7s, 13s, 10s)
    span: (filter2, 0)
      time: (Nov 24 15:34:16.151326, Nov 24 15:34:27.151326)
      duration: (1s, 11s, 1s)
      span: (filter3, 0)
        time: (Nov 24 15:34:17.151326, Nov 24 15:34:26.151326)
        duration: (1s, 9s, 1s)
        span: (HandleFunc, 0)
          time: (Nov 24 15:34:18.151326, Nov 24 15:34:25.151326)
          duration: (1s, 7s, 1s)
  span: (Marshal, 0)
    time: (Nov 24 15:34:29.151326, Nov 24 15:34:30.151326)
    duration: (21s, 1s, 8s)
  span: (Compress, 0)
    time: (Nov 24 15:34:31.151326, Nov 24 15:34:32.151326)
    duration: (23s, 1s, 6s)
  span: (EncodeProtocolHead, 0)
    time: (Nov 24 15:34:33.151326, Nov 24 15:34:34.151326)
    duration: (25s, 1s, 4s)
  span: (SendMessage, 0)
    time: (Nov 24 15:34:35.151326, Nov 24 15:34:36.151326)
    duration: (27s, 1s, 2s)
`,
			serverSpan.PrintDetail(""),
		)
	})
}

func TestReadOnlySpan_PrintDetailServerWithProxy(t *testing.T) {
	require.Equal(
		t,
		`span: (server, 0)
  time: (Nov 24 15:34:08.151326, Nov 24 15:34:38.151326)
  duration: (0, 30s, 0)
  span: (DecodeProtocolHead, 0)
    time: (Nov 24 15:34:09.151326, Nov 24 15:34:10.151326)
    duration: (1s, 1s, 28s)
  span: (Decompress, 0)
    time: (Nov 24 15:34:11.151326, Nov 24 15:34:12.151326)
    duration: (3s, 1s, 26s)
  span: (Unmarshal, 0)
    time: (Nov 24 15:34:13.151326, Nov 24 15:34:14.151326)
    duration: (5s, 1s, 24s)
  span: (, 0)
    time: (Nov 24 15:34:15.151326, Nov 24 15:34:28.151326)
    duration: (7s, 13s, 10s)
    span: (, 0)
      time: (Nov 24 15:34:16.151326, Nov 24 15:34:27.151326)
      duration: (1s, 11s, 1s)
      span: (, 0)
        time: (Nov 24 15:34:17.151326, Nov 24 15:34:26.151326)
        duration: (1s, 9s, 1s)
        span: (HandleFunc, 0)
          time: (Nov 24 15:34:18.151326, Nov 24 15:34:25.151326)
          duration: (1s, 7s, 1s)
          span: (client, 0)
            time: (Nov 24 15:34:08.151326, Nov 24 15:34:38.151326)
            duration: (-10s, 30s, -13s)
            attributes: (RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, )
            span: (1, 0)
              time: (Nov 24 15:34:09.151326, Nov 24 15:34:37.151326)
              duration: (1s, 28s, 1s)
              span: (2, 0)
                time: (Nov 24 15:34:10.151326, Nov 24 15:34:36.151326)
                duration: (1s, 26s, 1s)
                span: (3, 0)
                  time: (Nov 24 15:34:11.151326, Nov 24 15:34:35.151326)
                  duration: (1s, 24s, 1s)
                  span: (CallFunc, 0)
                    time: (Nov 24 15:34:12.151326, Nov 24 15:34:34.151326)
                    duration: (1s, 22s, 1s)
                    span: (Marshal, 0)
                      time: (Nov 24 15:34:13.151326, Nov 24 15:34:14.151326)
                      duration: (1s, 1s, 20s)
                    span: (Compress, 0)
                      time: (Nov 24 15:34:15.151326, Nov 24 15:34:16.151326)
                      duration: (3s, 1s, 18s)
                    span: (EncodeProtocolHead, 0)
                      time: (Nov 24 15:34:17.151326, Nov 24 15:34:18.151326)
                      duration: (5s, 1s, 16s)
                    span: (SendMessage, 0)
                      time: (Nov 24 15:34:19.151326, Nov 24 15:34:20.151326)
                      duration: (7s, 1s, 14s)
                    span: (ReceiveMessage, 0)
                      time: (Nov 24 15:34:21.151326, Nov 24 15:34:22.151326)
                      duration: (9s, 1s, 12s)
                    span: (DecodeProtocolHead, 0)
                      time: (Nov 24 15:34:23.151326, Nov 24 15:34:24.151326)
                      duration: (11s, 1s, 10s)
                    span: (Decompress, 0)
                      time: (Nov 24 15:34:25.151326, Nov 24 15:34:26.151326)
                      duration: (13s, 1s, 8s)
                    span: (Unmarshal, 0)
                      time: (Nov 24 15:34:27.151326, Nov 24 15:34:28.151326)
                      duration: (15s, 1s, 6s)
          span: (client, 0)
            time: (Nov 24 15:34:08.151326, Nov 24 15:34:38.151326)
            duration: (-10s, 30s, -13s)
            attributes: (RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, )
            span: (1, 0)
              time: (Nov 24 15:34:09.151326, Nov 24 15:34:37.151326)
              duration: (1s, 28s, 1s)
              span: (2, 0)
                time: (Nov 24 15:34:10.151326, Nov 24 15:34:36.151326)
                duration: (1s, 26s, 1s)
                span: (3, 0)
                  time: (Nov 24 15:34:11.151326, Nov 24 15:34:35.151326)
                  duration: (1s, 24s, 1s)
                  span: (CallFunc, 0)
                    time: (Nov 24 15:34:12.151326, Nov 24 15:34:34.151326)
                    duration: (1s, 22s, 1s)
                    span: (Marshal, 0)
                      time: (Nov 24 15:34:13.151326, Nov 24 15:34:14.151326)
                      duration: (1s, 1s, 20s)
                    span: (Compress, 0)
                      time: (Nov 24 15:34:15.151326, Nov 24 15:34:16.151326)
                      duration: (3s, 1s, 18s)
                    span: (EncodeProtocolHead, 0)
                      time: (Nov 24 15:34:17.151326, Nov 24 15:34:18.151326)
                      duration: (5s, 1s, 16s)
                    span: (SendMessage, 0)
                      time: (Nov 24 15:34:19.151326, Nov 24 15:34:20.151326)
                      duration: (7s, 1s, 14s)
                    span: (ReceiveMessage, 0)
                      time: (Nov 24 15:34:21.151326, Nov 24 15:34:22.151326)
                      duration: (9s, 1s, 12s)
                    span: (DecodeProtocolHead, 0)
                      time: (Nov 24 15:34:23.151326, Nov 24 15:34:24.151326)
                      duration: (11s, 1s, 10s)
                    span: (Decompress, 0)
                      time: (Nov 24 15:34:25.151326, Nov 24 15:34:26.151326)
                      duration: (13s, 1s, 8s)
                    span: (Unmarshal, 0)
                      time: (Nov 24 15:34:27.151326, Nov 24 15:34:28.151326)
                      duration: (15s, 1s, 6s)
  span: (Marshal, 0)
    time: (Nov 24 15:34:29.151326, Nov 24 15:34:30.151326)
    duration: (21s, 1s, 8s)
  span: (Compress, 0)
    time: (Nov 24 15:34:31.151326, Nov 24 15:34:32.151326)
    duration: (23s, 1s, 6s)
  span: (EncodeProtocolHead, 0)
    time: (Nov 24 15:34:33.151326, Nov 24 15:34:34.151326)
    duration: (25s, 1s, 4s)
  span: (SendMessage, 0)
    time: (Nov 24 15:34:35.151326, Nov 24 15:34:36.151326)
    duration: (27s, 1s, 2s)
`,
		serverSpanHasClientChildSpan.PrintDetail(""),
	)
}

func TestReadOnlySpan_PrintSketch(t *testing.T) {
	t.Run("client span", func(t *testing.T) {
		require.Equal(
			t,
			`span: (client, 0)
  time: (Nov 24 15:34:08.151326, Nov 24 15:34:38.151326)
  duration: (0, 30s, 0)
  attributes: (RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, )
`,
			clientSpan.PrintSketch(""),
		)
	})
	t.Run("server span", func(t *testing.T) {
		require.Equal(
			t,
			`span: (server, 0)
  time: (Nov 24 15:34:08.151326, Nov 24 15:34:38.151326)
  duration: (0, 30s, 0)
  attributes: (RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, ),(RequestSize, 125),(ResponseSize, 18)
`,
			serverSpan.PrintSketch(""),
		)
	})
}

func TestParentSpanContainUnfinishedChildSpan(t *testing.T) {
	t.Run("record root span to rpcz", func(t *testing.T) {
		rpcz := NewRPCZ(&Config{Capacity: 10})
		const (
			spanID   = SpanID(1)
			spanName = "server"
		)
		ps := newSpan(spanName, spanID, rpcz)
		require.False(t, ps.startTime.IsZero())
		ps.startTime = currentTime

		cs, end := ps.NewChild("client")
		require.False(t, cs.(*span).startTime.IsZero())
		cs.(*span).startTime = currentTime.Add(time.Second)

		ps.End()
		require.False(t, ps.endTime.IsZero())
		ps.endTime = currentTime.Add(3 * time.Second)

		readOnlySpan, ok := rpcz.Query(spanID)
		require.True(t, ok)
		require.Contains(t, readOnlySpan.PrintDetail(""), "duration: (1s, unknown, unknown)")

		end.End()
		require.False(t, cs.(*span).endTime.IsZero())
		cs.(*span).endTime = currentTime.Add(2 * time.Second)
		readOnlySpan, ok = rpcz.Query(spanID)
		require.True(t, ok)
		require.NotContains(t, readOnlySpan.PrintDetail(""), "unknown")
	})
}

func TestSpanHasEvents(t *testing.T) {
	require.Equal(
		t,
		`span: (server, 0)
  time: (Nov 24 15:34:08.151326, Nov 24 15:34:38.151326)
  duration: (0, 30s, 0)
  attributes: (RPCName, /trpc.testing.end2end.TestTRPC/EmptyCall),(Error, ),(RequestSize, 125),(ResponseSize, 18)
  event: (enter DecodeProtocolHead, Nov 24 15:34:08.651326)
  span: (DecodeProtocolHead, 0)
    time: (Nov 24 15:34:09.151326, Nov 24 15:34:10.151326)
    duration: (1s, 1s, 28s)
  event: (handle DecodeProtocolHead, Nov 24 15:34:09.651326)
  event: (leave DecodeProtocolHead, enter Decompress, Nov 24 15:34:10.651326)
  span: (Decompress, 0)
    time: (Nov 24 15:34:11.151326, Nov 24 15:34:12.151326)
    duration: (3s, 1s, 26s)
  event: (handle Decompress, Nov 24 15:34:11.651326)
  event: (leave Decompress, enter Unmarshal, Nov 24 15:34:12.651326)
  span: (Unmarshal, 0)
    time: (Nov 24 15:34:13.151326, Nov 24 15:34:14.151326)
    duration: (5s, 1s, 24s)
  span: (filter1, 0)
    time: (Nov 24 15:34:15.151326, Nov 24 15:34:28.151326)
    duration: (7s, 13s, 10s)
    span: (HandleFunc, 0)
      time: (Nov 24 15:34:18.151326, Nov 24 15:34:25.151326)
      duration: (3s, 7s, 3s)
  span: (Marshal, 0)
    time: (Nov 24 15:34:29.151326, Nov 24 15:34:30.151326)
    duration: (21s, 1s, 8s)
  span: (Compress, 0)
    time: (Nov 24 15:34:31.151326, Nov 24 15:34:32.151326)
    duration: (23s, 1s, 6s)
  span: (EncodeProtocolHead, 0)
    time: (Nov 24 15:34:33.151326, Nov 24 15:34:34.151326)
    duration: (25s, 1s, 4s)
  span: (SendMessage, 0)
    time: (Nov 24 15:34:35.151326, Nov 24 15:34:36.151326)
    duration: (27s, 1s, 2s)
`,
		serverSpanHasEvents.PrintDetail(""),
	)

	s := newSpan("client", 1, nil)
	s.AddEvent("send request")
	s.AddEvent("receive response")
	content := s.convertedToReadOnlySpan().PrintDetail("")
	require.Contains(t, content, "event: (send request,")
	require.Contains(t, content, "event: (receive response,")
}
