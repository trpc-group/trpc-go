package roundrobin

import (
	"sync"
	"testing"
	"time"

	"trpc.group/trpc-go/trpc-go/naming/registry"

	"github.com/stretchr/testify/assert"
)

func TestRoundRobinGetOne(t *testing.T) {
	rr := NewRoundRobin(0)
	length := len(list1)
	for i := 0; i < length*2; i++ {
		n, err := rr.Select("test1", list1)
		assert.Nil(t, err)
		assert.Equal(t, list1[i%length], n)
	}
}

func TestRoundRobinListLengthChange(t *testing.T) {
	rr := NewRoundRobin(time.Second * 10)
	n1, err := rr.Select("test1", list1)
	assert.Nil(t, err)
	assert.Equal(t, n1, list1[0])

	length := len(list2)
	assert.Equal(t, 3, length)
	for i := 0; i < length*2; i++ {
		n, err := rr.Select("test1", list2)
		assert.Nil(t, err)
		assert.Equal(t, list2[i%length], n)
	}
}

func TestRoundRobinInterval(t *testing.T) {
	rr := NewRoundRobin(time.Second * 1)
	n1, err := rr.Select("test1", list1)
	assert.Nil(t, err)
	assert.Equal(t, n1, list1[0])

	n2, err := rr.Select("test1", list3)
	assert.Nil(t, err)
	assert.Equal(t, n2, list1[1])

	time.Sleep(time.Second)
	n3, err := rr.Select("test1", list3)
	assert.Nil(t, err)
	assert.Equal(t, n3, list3[0])
}

func TestRoundRobinConCurrentSelect(t *testing.T) {
	rr := NewRoundRobin(time.Second * 1)

	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := rr.Select("test1", list1)
			assert.Nil(t, err)
			wg.Done()
		}()
	}

	wg.Wait()
}

func TestRoundRobinSelectDifferentService(t *testing.T) {
	rr := NewRoundRobin(time.Second * 1)
	for i := 0; i < 10; i++ {
		n1, err := rr.Select("test1", list1)
		assert.Nil(t, err)
		assert.Equal(t, n1, list1[i%len(list1)])

		n2, err := rr.Select("test2", list2)
		assert.Nil(t, err)
		assert.Equal(t, n2, list2[i%len(list2)])
	}
}

var list1 = []*registry.Node{
	{
		Address: "list1.ip.1:8080",
	},
	{
		Address: "list1.ip.2:8080",
	},
	{
		Address: "list1.ip.3:8080",
	},
	{
		Address: "list1.ip.4:8080",
	},
}

var list2 = []*registry.Node{
	{
		Address: "list2.ip.2:8080",
	},
	{
		Address: "list2.ip.3:8080",
	},
	{
		Address: "list2.ip.4:8080",
	},
}

var list3 = []*registry.Node{
	{
		Address: "list3.ip.5:8080",
	},
	{
		Address: "list3.ip.6:8080",
	},
	{
		Address: "list3.ip.7:8080",
	},
	{
		Address: "list3.ip.8:8080",
	},
}
