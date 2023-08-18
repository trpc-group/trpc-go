// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package rpcz

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_newSpanArray(t *testing.T) {
	t.Run("capacity equal zero", func(t *testing.T) {
		require.PanicsWithValue(t, "capacity should be greater than 0", func() { newSpanArray(0) })
	})
	t.Run("capacity greater zero", func(t *testing.T) {
		const capacity = 1
		require.Equal(t, &spanArray{capacity: capacity, data: make([]*span, capacity)}, newSpanArray(capacity))
	})
}

func Test_spanArray_dequeue(t *testing.T) {
	t.Run("dequeue empty span array", func(t *testing.T) {
		sa := newSpanArray(10)
		sa.dequeue()
		require.Equal(t, uint32(math.MaxUint32), sa.length)
		require.Equal(t, uint32(1), sa.head)
	})
	t.Run("#num dequeue more than length of span array", func(t *testing.T) {
		sa := newSpanArray(10)
		sa.enqueue(&span{})
		sa.dequeue()
		sa.dequeue()
		require.Equal(t, uint32(math.MaxUint32), sa.length)
		require.Equal(t, uint32(2), sa.head)
	})
	t.Run("#num dequeue equal #num enqueue", func(t *testing.T) {
		const capacity = 10
		sa := newSpanArray(capacity)
		for i := 0; i < capacity; i++ {
			sa.enqueue(&span{id: SpanID(i)})
		}
		for i := 0; i < capacity; i++ {
			require.Equal(t, &span{id: SpanID(i)}, sa.front())
			sa.dequeue()
		}
		require.Equal(t, sa.head, sa.tail)
		require.Equal(t, uint32(0), sa.length)
	})
	t.Run("dequeue and enqueue in pairs", func(t *testing.T) {
		const capacity = 10
		sa := newSpanArray(capacity)
		for i := 0; i < capacity; i++ {
			sa.enqueue(&span{id: SpanID(i)})
			require.Equal(t, &span{id: SpanID(i)}, sa.front())
			sa.dequeue()
		}
		require.Equal(t, sa.head, sa.tail)
		require.Equal(t, uint32(0), sa.length)
	})
}

func Test_spanArray_doBackward(t *testing.T) {
	const (
		capacity = 10
		num      = 5
	)

	sa := newSpanArray(capacity)
	for i := 0; i < capacity; i++ {
		sa.enqueue(&span{id: SpanID(i)})
	}

	var got []*span
	count := num
	sa.doBackward(func(s *span) bool {
		if count <= 0 {
			return false
		}
		count--
		got = append(got, s)
		return true
	})

	var want []*span
	for i := capacity - 1; i >= num; i-- {
		want = append(want, &span{id: SpanID(i)})
	}

	require.Equal(t, want, got)
}

func Test_spanArray_enqueue(t *testing.T) {
	t.Run("len less than capacity", func(t *testing.T) {
		a := &spanArray{
			length:   1,
			capacity: 10,
			data:     make([]*span, 10),
			head:     0,
			tail:     1,
		}
		want := *a
		want.length++
		want.tail++

		a.enqueue(&span{})
		require.Equal(t, &want, a)
	})
	t.Run("tail equals capacity-1", func(t *testing.T) {
		a := &spanArray{
			length:   9,
			capacity: 10,
			data:     make([]*span, 10),
			head:     0,
			tail:     9,
		}
		want := *a
		want.length++
		want.tail = 0

		a.enqueue(&span{})
		require.Equal(t, &want, a)
	})
	t.Run("len equals capacity", func(t *testing.T) {
		a := &spanArray{
			length:   10,
			capacity: 10,
			data:     make([]*span, 10),
			head:     0,
			tail:     0,
		}
		want := *a
		want.tail = 1
		want.head = 1

		a.enqueue(&span{})
		require.Equal(t, &want, a)
	})
	t.Run("array is full", func(t *testing.T) {
		a := newSpanArray(10)
		for i := 0; i < 1000; i++ {
			a.enqueue(&span{})
			if i >= 10 {
				require.Equal(t, uint32(10), a.length)
				require.True(t, a.tail == a.head)
			}
		}
	})
}

func Test_spanArray_full(t *testing.T) {
	type fields struct {
		capacity uint32
		length   uint32
		data     []*span
		front    uint32
		rear     uint32
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{
			name: "length equals capacity",
			fields: fields{
				capacity: 10,
				length:   10,
				data:     make([]*span, 10),
				front:    0,
				rear:     0,
			},
			want: true,
		},
		{
			name: "length greater than capacity",
			fields: fields{
				capacity: 10,
				length:   11,
				// other fields is invalid, this case shouldn't happen.
			},
			want: true,
		},
		{
			name: "length less than capacity",
			fields: fields{
				capacity: 10,
				length:   9,
				data:     make([]*span, 10),
				front:    0,
				rear:     9,
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &spanArray{
				capacity: tt.fields.capacity,
				length:   tt.fields.length,
				data:     tt.fields.data,
				head:     tt.fields.front,
				tail:     tt.fields.rear,
			}
			if got := a.full(); got != tt.want {
				t.Errorf("full() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_spanArray_nextIndex(t *testing.T) {
	type fields struct {
		capacity uint32
		length   uint32
		data     []*span
		front    uint32
		rear     uint32
	}
	type args struct {
		index uint32
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uint32
	}{
		{
			name:   "index less than spanArray.capacity-1, and no less than zero",
			fields: fields{capacity: 10, length: 0, front: 0, rear: 0},
			args:   args{index: 0},
			want:   1,
		},
		{
			name:   "index equals spanArray.capacity-1",
			fields: fields{capacity: 10, length: 0, front: 0, rear: 0},
			args:   args{index: 9},
			want:   0,
		},
		{
			name:   "index greater than spanArray.capacity-1",
			fields: fields{capacity: 10, length: 0, front: 0, rear: 0},
			args:   args{index: 11},
			want:   2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &spanArray{
				capacity: tt.fields.capacity,
				length:   tt.fields.length,
				data:     tt.fields.data,
				head:     tt.fields.front,
				tail:     tt.fields.rear,
			}
			if got := a.nextIndex(tt.args.index); got != tt.want {
				t.Errorf("nextIndex() = %v, want %v", got, tt.want)
			}
		})
	}
	t.Run("input and output in valid range", func(t *testing.T) {
		const capacity = 10
		sa := newSpanArray(capacity)
		var (
			got  []uint32
			want []uint32
		)
		for i := uint32(0); i < capacity; i++ {
			want = append(want, i)
			got = append(got, sa.nextIndex(i))
		}
		require.ElementsMatch(t, want, got)
	})
}

func Test_spanArray_previousIndex(t *testing.T) {
	type fields struct {
		capacity uint32
		length   uint32
		data     []*span
		front    uint32
		rear     uint32
	}
	type args struct {
		index uint32
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   uint32
	}{
		{
			name:   "index greater than zero  and no less than  capacity",
			fields: fields{capacity: 10, length: 0, front: 0, rear: 0},
			args:   args{index: 10},
			want:   9,
		},
		{
			name:   "index greater than zero  and less than capacity",
			fields: fields{capacity: 10, length: 0, front: 0, rear: 0},
			args:   args{index: 1},
			want:   0,
		},
		{
			name:   "index equals zero",
			fields: fields{capacity: 10, length: 0, front: 0, rear: 0},
			args:   args{index: 0},
			want:   9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &spanArray{
				capacity: tt.fields.capacity,
				length:   tt.fields.length,
				data:     tt.fields.data,
				head:     tt.fields.front,
				tail:     tt.fields.rear,
			}
			if got := a.previousIndex(tt.args.index); got != tt.want {
				t.Errorf("previousIndex() = %v, want %v", got, tt.want)
			}
		})
	}
	t.Run("input and output in valid range", func(t *testing.T) {
		const capacity = 10
		sa := newSpanArray(capacity)
		var (
			got  []uint32
			want []uint32
		)
		for i := uint32(0); i < capacity; i++ {
			want = append(want, i)
			got = append(got, sa.previousIndex(i))
		}
		require.ElementsMatch(t, want, got)
	})
}

func Test_spanArray_front(t *testing.T) {
	a := newSpanArray(2)
	a.enqueue(&span{id: 1})
	s := a.front()
	require.Equal(t, SpanID(1), s.id)

	a.dequeue()
	require.Empty(t, a.length)
	require.Equal(t, SpanID(1), s.id)

	a.enqueue(&span{id: 2})
	a.enqueue(&span{id: 3})
	a.enqueue(&span{id: 4})
	require.Equal(t, SpanID(1), s.id)
}
