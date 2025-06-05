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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/emptypb"

	"trpc.group/trpc-go/trpc-go/filter"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	bpb "trpc.group/trpc-go/trpc-go/testdata/restful/bookstore"
	hpb "trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
)

// helloworld service impl
type greeterServerImpl struct {
	sleepTime time.Duration
}

func (s *greeterServerImpl) SayHello(ctx context.Context, req *hpb.HelloRequest) (*hpb.HelloReply, error) {
	time.Sleep(s.sleepTime)
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	rsp := &hpb.HelloReply{}
	if req.Name != "xyz" {
		return nil, errors.New("test error")
	}
	rsp.Message = "test"
	return rsp, nil
}

func TestHelloworldService(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := fmt.Sprintf("http://%s", ln.Addr())
	defer ln.Close()
	// service registration
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("trpc.test.helloworld.Service"+t.Name()),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithRESTOptions(
			restful.WithHeaderMatcher(
				func(ctx context.Context, w http.ResponseWriter, r *http.Request, serviceName string,
					methodName string) (context.Context, error) {
					return context.Background(), nil
				},
			),
			restful.WithResponseHandler(
				func(
					ctx context.Context,
					w http.ResponseWriter,
					r *http.Request,
					resp proto.Message,
					body []byte,
				) error {
					if r.Header.Get("Accept-Encoding") != "gzip" {
						return errors.New("test error")
					}
					writeCloser, err := (&restful.GZIPCompressor{}).Compress(w)
					if err != nil {
						return err
					}
					defer writeCloser.Close()
					w.Header().Set("Content-Encoding", "gzip")
					w.Header().Set("Content-Type", "application/json")
					writeCloser.Write(body)
					return nil
				},
			),
			restful.WithErrorHandler(
				func(ctx context.Context, w http.ResponseWriter, r *http.Request, err error) {
					w.WriteHeader(500)
					w.Header().Add("Content-Type", "application/json")
					w.Write([]byte(`{"massage":"test error"}`))
				},
			),
		),
	)
	s.AddService("trpc.test.helloworld.Service", service)
	hpb.RegisterGreeterService(s, &greeterServerImpl{})

	// start server
	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	// create restful request
	data := `{"name": "xyz"}`
	buf := bytes.Buffer{}
	gBuf := gzip.NewWriter(&buf)
	_, err = gBuf.Write([]byte(data))
	require.Nil(t, err)
	gBuf.Close()
	req, err := http.NewRequest(http.MethodPost, addr+"/v1/foobar", &buf)
	require.Nil(t, err)
	req.Header.Add("Content-Type", "anything")
	req.Header.Add("Content-Encoding", "gzip")
	req.Header.Add("Accept-Encoding", "gzip")

	// send restful request
	cli := http.Client{}
	resp, err := cli.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, resp.StatusCode, http.StatusOK)
	reader, err := gzip.NewReader(resp.Body)
	require.Nil(t, err)
	bodyBytes, err := io.ReadAll(reader)
	require.Nil(t, err)
	type responseBody struct {
		Message string `json:"message"`
	}
	respBody := &responseBody{}
	json.Unmarshal(bodyBytes, respBody)
	require.Equal(t, respBody.Message, "test")

	// test matching all by query params
	req2, err := http.NewRequest(http.MethodGet, addr+"/v2/bar?name=xyz", nil)
	require.Nil(t, err)
	resp2, err := http.DefaultClient.Do(req2)
	require.Nil(t, err)
	defer resp2.Body.Close()
	require.Equal(t, resp2.StatusCode, http.StatusOK)

	// test response content-type
	require.Equal(t, resp2.Header.Get("Content-Type"), "application/json")
}

func TestHeaderMatcher(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := fmt.Sprintf("http://%s", ln.Addr())
	defer ln.Close()
	// service registration
	s := &server.Server{}
	service := server.New(server.WithListener(ln),
		server.WithServiceName("test"),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithRESTOptions(restful.WithHeaderMatcher(func(ctx context.Context, w http.ResponseWriter,
			r *http.Request, serviceName string, methodName string) (context.Context, error) {
			return nil, errors.New("test error")
		})),
	)
	s.AddService("test", service)
	hpb.RegisterGreeterService(s, &greeterServerImpl{})

	// start server
	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	// test header matcher error
	req, err := http.NewRequest(http.MethodPost, addr+"/v1/foobar",
		bytes.NewBuffer([]byte(`{"name": "xyz"}`)))
	require.Nil(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestResponseHandler(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := fmt.Sprintf("http://%s", ln.Addr())
	defer ln.Close()
	// service registration
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("test.ResponseHandler"),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithRESTOptions(restful.WithResponseHandler(
			func(
				ctx context.Context,
				w http.ResponseWriter,
				r *http.Request,
				resp proto.Message,
				body []byte,
			) error {
				return errors.New("test error")
			},
		)),
	)
	s.AddService("test.ResponseHandler", service)
	hpb.RegisterGreeterService(s, &greeterServerImpl{})

	// start server
	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	// test response handler error
	req, err := http.NewRequest(http.MethodPost, addr+"/v1/foobar",
		bytes.NewBuffer([]byte(`{"name": "xyz"}`)))
	require.Nil(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestRestfulRequestTimeout(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := fmt.Sprintf("http://%s", ln.Addr())
	defer ln.Close()
	s := &server.Server{}
	serviceName := "trpc.test.helloworld.Service_" + t.Name()
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("trpc.test.helloworld.Service"+t.Name()),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		// Only with this line, the rsp.StatusCode will be http.StatusOK.
		server.WithDisableRequestTimeout(true),
	)
	s.AddService(serviceName, service)
	const requestTimeout = time.Millisecond
	hpb.RegisterGreeterService(s, &greeterServerImpl{
		sleepTime: requestTimeout * 10,
	})

	go func() {
		s.Serve()
	}()

	time.Sleep(100 * time.Millisecond)

	data := []byte(`{"name": "xyz"}`)
	require.Nil(t, err)
	req, err := http.NewRequest(http.MethodPost, addr+"/v1/foobar", bytes.NewBuffer(data))
	require.Nil(t, err)
	req.Header.Add(textproto.CanonicalMIMEHeaderKey(thttp.TrpcTimeout), "1")

	cli := http.Client{}
	rsp, err := cli.Do(req)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}

// bookstore service impl
type bookstoreServiceImpl struct{}

var shelves = map[int64]*bpb.Shelf{
	0: {
		Id:    0,
		Theme: "shelf_0",
	},
	1: {
		Id:    1,
		Theme: "shelf_1",
	},
}

var shelf2Books = map[int64]map[int64]*bpb.Book{
	0: {
		0: {
			Id:     0,
			Author: "author_0",
			Title:  "title_0",
		},
	},
	1: {
		1: {
			Id:     1,
			Author: "author_1",
			Title:  "title_1",
		},
	},
}

// ListShelves lists all shelves.
func (s *bookstoreServiceImpl) ListShelves(ctx context.Context, req *emptypb.Empty) (rsp *bpb.ListShelvesResponse, err error) {
	rsp = &bpb.ListShelvesResponse{}
	for _, each := range shelves {
		rsp.Shelves = append(rsp.Shelves, each)
	}

	return rsp, nil
}

// CreateShelf creates a shelf.
func (s *bookstoreServiceImpl) CreateShelf(ctx context.Context, req *bpb.CreateShelfRequest) (rsp *bpb.Shelf, err error) {
	rsp = &bpb.Shelf{}
	id := req.GetShelf().GetId()
	if _, ok := shelves[id]; ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusConflict,
			Err:        errors.New("shelf already exists"),
		}
	}

	shelves[id] = req.GetShelf()
	rsp.Id = req.GetShelf().Id
	rsp.Theme = req.GetShelf().Theme
	shelf2Books[id] = make(map[int64]*bpb.Book)
	restful.SetStatusCodeOnSucceed(ctx, http.StatusCreated)
	return rsp, nil
}

// GetShelf returns a shelf.
func (s *bookstoreServiceImpl) GetShelf(ctx context.Context, req *bpb.GetShelfRequest) (rsp *bpb.Shelf, err error) {
	rsp = &bpb.Shelf{}
	id := req.GetShelf()
	if shelf, ok := shelves[id]; !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("shelf not found"),
		}
	} else {
		rsp.Id = shelf.Id
		rsp.Theme = shelf.Theme
		restful.SetStatusCodeOnSucceed(ctx, http.StatusAccepted)
	}
	return rsp, nil
}

// DeleteShelf deletes a shelf.
func (s *bookstoreServiceImpl) DeleteShelf(ctx context.Context, req *bpb.DeleteShelfRequest) (rsp *emptypb.Empty, err error) {
	rsp = &emptypb.Empty{}
	id := req.GetShelf()
	if _, ok := shelves[id]; !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("shelf not found"),
		}
	}
	delete(shelves, id)
	delete(shelf2Books, id)
	restful.SetStatusCodeOnSucceed(ctx, http.StatusNoContent)
	return rsp, nil
}

// ListBooks lists all books.
func (s *bookstoreServiceImpl) ListBooks(ctx context.Context, req *bpb.ListBooksRequest) (rsp *bpb.ListBooksResponse, err error) {
	rsp = &bpb.ListBooksResponse{}
	shelfID := req.GetShelf()
	books, ok := shelf2Books[shelfID]
	if !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("shelf not found"),
		}
	}

	for _, book := range books {
		rsp.Books = append(rsp.Books, book)
	}
	restful.SetStatusCodeOnSucceed(ctx, http.StatusAccepted)
	return rsp, nil
}

// CreateBook creates a book.
func (s *bookstoreServiceImpl) CreateBook(ctx context.Context, req *bpb.CreateBookRequest) (rsp *bpb.Book, err error) {
	rsp = &bpb.Book{}
	shelfID := req.GetShelf()
	books, ok := shelf2Books[shelfID]
	if !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("shelf not found"),
		}
	}

	bookID := req.GetBook().GetId()
	if _, ok := books[bookID]; ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusConflict,
			Err:        errors.New("book already exists"),
		}
	}

	shelf2Books[shelfID][bookID] = req.GetBook()
	rsp.Id = req.GetBook().Id
	rsp.Title = req.GetBook().Title
	rsp.Author = req.GetBook().Author
	rsp.Content = req.GetBook().Content
	restful.SetStatusCodeOnSucceed(ctx, http.StatusCreated)
	return rsp, nil
}

// GetBook returns a book.
func (s *bookstoreServiceImpl) GetBook(ctx context.Context, req *bpb.GetBookRequest) (rsp *bpb.Book, err error) {
	rsp = &bpb.Book{}
	shelfID := req.GetShelf()
	books, ok := shelf2Books[shelfID]
	if !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("shelf not found"),
		}
	}

	bookID := req.GetBook()
	if book, ok := books[bookID]; !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("book not found"),
		}
	} else {
		rsp.Id = book.Id
		rsp.Author = book.Author
		rsp.Title = book.Title
		rsp.Content = book.Content
		restful.SetStatusCodeOnSucceed(ctx, http.StatusAccepted)
	}
	return rsp, nil
}

// DeleteBook deletes a book.
func (s *bookstoreServiceImpl) DeleteBook(ctx context.Context, req *bpb.DeleteBookRequest) (rsp *emptypb.Empty, err error) {
	rsp = &emptypb.Empty{}
	shelfID := req.GetShelf()
	books, ok := shelf2Books[shelfID]
	if !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("shelf not found"),
		}
	}

	bookID := req.GetBook()
	if _, ok := books[bookID]; !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("book not found"),
		}
	}

	delete(shelf2Books[shelfID], bookID)
	restful.SetStatusCodeOnSucceed(ctx, http.StatusAccepted)
	return rsp, nil
}

// UpdateBook updates a book.
func (s *bookstoreServiceImpl) UpdateBook(ctx context.Context, req *bpb.UpdateBookRequest) (rsp *bpb.Book, err error) {
	rsp = &bpb.Book{}
	shelfID := req.GetShelf()
	books, ok := shelf2Books[shelfID]
	if !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("shelf not found"),
		}
	}

	bookID := req.GetBook().Id
	book, ok := books[bookID]
	if !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("book not found"),
		}
	}

	for _, path := range req.GetUpdateMask().Paths {
		switch path {
		case "author":
			book.Author = req.Book.Author
		case "title":
			book.Title = req.Book.Title
		case "content.summary":
			book.Content = req.Book.Content
		default:
		}
	}

	rsp.Id = book.Id
	rsp.Author = book.Author
	rsp.Title = book.Title
	rsp.Content = book.Content
	restful.SetStatusCodeOnSucceed(ctx, http.StatusAccepted)
	return rsp, nil
}

func (s *bookstoreServiceImpl) UpdateBooks(ctx context.Context, req *bpb.UpdateBooksRequest) (rsp *bpb.ListBooksResponse, err error) {
	books, ok := shelf2Books[req.Shelf]
	if !ok {
		return nil, &restful.WithStatusCode{
			StatusCode: http.StatusNotFound,
			Err:        errors.New("shelf not found"),
		}
	}

	for _, b := range req.Books {
		_, ok := books[b.Id]
		if !ok {
			return nil, &restful.WithStatusCode{
				StatusCode: http.StatusNotFound,
				Err:        fmt.Errorf("book %d not found", b.Id),
			}
		}
		books[b.Id] = b
	}

	restful.SetStatusCodeOnSucceed(ctx, http.StatusAccepted)
	return &bpb.ListBooksResponse{Books: req.Books}, nil
}

func httpNewRequest(t *testing.T, method, url string, body io.Reader, contentType string) *http.Request {
	req, err := http.NewRequest(method, url, body)
	require.Nil(t, err)
	req.Header.Add("Content-Type", contentType)
	return req
}

func TestBookstoreService(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := fmt.Sprintf("http://%s", ln.Addr())
	defer ln.Close()
	// service registration
	s := &server.Server{}
	service := server.New(server.WithListener(ln),
		server.WithServiceName("trpc.test.bookstore.Bookstore"+t.Name()),
		server.WithProtocol("restful"))
	s.AddService("trpc.test.bookstore.Bookstore"+t.Name(), service)
	bpb.RegisterBookstoreService(s, &bookstoreServiceImpl{})

	// start server
	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	for _, test := range []struct {
		httpRequest       *http.Request
		respStatusCode    int
		expectShelves     map[int64]*bpb.Shelf
		expectShelf2Books map[int64]map[int64]*bpb.Book
		desc              string
	}{
		{
			httpRequest: httpNewRequest(t, http.MethodGet, addr+"/shelves", nil,
				"application/json"),
			respStatusCode: http.StatusOK,
			desc:           "test listing shelves",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodPost, addr+"/shelf", bytes.NewBuffer([]byte(
				`{"shelf":{"id":2,"theme":"shelf_2"}}`)), "application/json"),
			respStatusCode: http.StatusCreated,
			expectShelves: map[int64]*bpb.Shelf{
				0: {Id: 0, Theme: "shelf_0"},
				1: {Id: 1, Theme: "shelf_1"},
				2: {Id: 2, Theme: "shelf_2"},
			},
			expectShelf2Books: map[int64]map[int64]*bpb.Book{
				0: {0: {Id: 0, Author: "author_0", Title: "title_0"}},
				1: {1: {Id: 1, Author: "author_1", Title: "title_1"}},
				2: {},
			},
			desc: "test creating a shelf",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodPost, addr+"/shelf", bytes.NewBuffer([]byte(
				`{"shelf":{"id":2,"theme":"shelf_02"}}`)), "application/json"),
			respStatusCode: http.StatusConflict,
			desc:           "test creating dup shelf",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodDelete, addr+"/shelf/2",
				nil, "application/json"),
			respStatusCode: http.StatusNoContent,
			expectShelves: map[int64]*bpb.Shelf{
				0: {Id: 0, Theme: "shelf_0"},
				1: {Id: 1, Theme: "shelf_1"},
			},
			expectShelf2Books: map[int64]map[int64]*bpb.Book{
				0: {0: {Id: 0, Author: "author_0", Title: "title_0"}},
				1: {1: {Id: 1, Author: "author_1", Title: "title_1"}},
			},
			desc: "test deleting a shelf",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodDelete, addr+"/shelf/2",
				nil, "application/json"),
			respStatusCode: http.StatusNotFound,
			desc:           "test deleting a shelf non exists",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodPost, addr+"/anything",
				nil, "application/json"),
			respStatusCode: http.StatusNotFound,
			desc:           "test invalid url path",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodPost,
				addr+"/shelf/theme/shelf_2?shelf.theme=x&shelf.id=2", nil,
				"application/json"),
			respStatusCode: http.StatusCreated,
			expectShelves: map[int64]*bpb.Shelf{
				0: {Id: 0, Theme: "shelf_0"},
				1: {Id: 1, Theme: "shelf_1"},
				2: {Id: 2, Theme: "shelf_2"},
			},
			expectShelf2Books: map[int64]map[int64]*bpb.Book{
				0: {0: {Id: 0, Author: "author_0", Title: "title_0"}},
				1: {1: {Id: 1, Author: "author_1", Title: "title_1"}},
				2: {},
			},
			desc: "test creating a shelf with query params",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodPost,
				addr+"/shelf/theme/shelf_2?anything=2",
				nil, "application/json"),
			respStatusCode: http.StatusBadRequest,
			desc:           "test creating a shelf with invalid query params",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodPost, addr+"/book/shelf/1",
				bytes.NewBuffer([]byte("anything")), "application/json"),
			respStatusCode: http.StatusBadRequest,
			desc:           "test creating a book with invalid body data",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodPatch, addr+"/book/shelfid/1/bookid/1",
				bytes.NewBuffer([]byte(`{"author":"anonymous","content":{"summary":"life of a hero"}}`)),
				"application/json"),
			respStatusCode: http.StatusAccepted,
			expectShelves: map[int64]*bpb.Shelf{
				0: {Id: 0, Theme: "shelf_0"},
				1: {Id: 1, Theme: "shelf_1"},
				2: {Id: 2, Theme: "shelf_2"},
			},
			expectShelf2Books: map[int64]map[int64]*bpb.Book{
				0: {0: {Id: 0, Author: "author_0", Title: "title_0"}},
				1: {1: {Id: 1, Author: "anonymous", Title: "title_1",
					Content: &bpb.Content{Summary: "life of a hero"}}},
				2: {},
			},
			desc: "test updating a book",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodPost, addr+"/book/shelf/2",
				strings.NewReader("id=2&author=author_2&title=title_2&content.summary=whatever"),
				"application/x-www-form-urlencoded"),
			respStatusCode: http.StatusCreated,
			expectShelves: map[int64]*bpb.Shelf{
				0: {Id: 0, Theme: "shelf_0"},
				1: {Id: 1, Theme: "shelf_1"},
				2: {Id: 2, Theme: "shelf_2"},
			},
			expectShelf2Books: map[int64]map[int64]*bpb.Book{
				0: {0: {Id: 0, Author: "author_0", Title: "title_0"}},
				1: {1: {Id: 1, Author: "anonymous", Title: "title_1",
					Content: &bpb.Content{Summary: "life of a hero"}}},
				2: {2: {Id: 2, Author: "author_2", Title: "title_2",
					Content: &bpb.Content{Summary: "whatever"}}},
			},
			desc: "test posting form to create book",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodPost, addr+"/book/shelf/2",
				strings.NewReader("id=3&author=author_3&title=title_3&content.summary=whatever"),
				"application/x-www-form-urlencoded; charset=UTF-8"),
			respStatusCode: http.StatusCreated,
			expectShelves: map[int64]*bpb.Shelf{
				0: {Id: 0, Theme: "shelf_0"},
				1: {Id: 1, Theme: "shelf_1"},
				2: {Id: 2, Theme: "shelf_2"},
			},
			expectShelf2Books: map[int64]map[int64]*bpb.Book{
				0: {0: {Id: 0, Author: "author_0", Title: "title_0"}},
				1: {1: {Id: 1, Author: "anonymous", Title: "title_1",
					Content: &bpb.Content{Summary: "life of a hero"}}},
				2: {
					2: {Id: 2, Author: "author_2", Title: "title_2",
						Content: &bpb.Content{Summary: "whatever"}},
					3: {Id: 3, Author: "author_3", Title: "title_3",
						Content: &bpb.Content{Summary: "whatever"}},
				},
			},
			desc: "test posting form to create book",
		},
		{
			httpRequest: httpNewRequest(t, http.MethodPatch, addr+"/book/shelfid/2",
				bytes.NewBuffer([]byte(`[{"id":"2", "author":"author_2"},{"id":"3", "author":"author_3"}]`)),
				"application/json"),
			respStatusCode: http.StatusAccepted,
			expectShelves: map[int64]*bpb.Shelf{
				0: {Id: 0, Theme: "shelf_0"},
				1: {Id: 1, Theme: "shelf_1"},
				2: {Id: 2, Theme: "shelf_2"},
			},
			expectShelf2Books: map[int64]map[int64]*bpb.Book{
				0: {0: {Id: 0, Author: "author_0", Title: "title_0"}},
				1: {1: {Id: 1, Author: "anonymous", Title: "title_1",
					Content: &bpb.Content{Summary: "life of a hero"}}},
				2: {
					2: {Id: 2, Author: "author_2"},
					3: {Id: 3, Author: "author_3"},
				},
			},
			desc: "test updating books of a shelf",
		},
	} {
		req := test.httpRequest
		cli := http.Client{}
		resp, err := cli.Do(req)
		require.Nil(t, err, test.desc)
		log.Printf("%+v\n", resp)
		bs, _ := io.ReadAll(resp.Body)
		log.Println(string(bs))
		require.Equal(t, test.respStatusCode, resp.StatusCode, test.desc)
		if resp.StatusCode > 200 && resp.StatusCode < 300 {
			require.Equal(t, "", cmp.Diff(shelves, test.expectShelves, protocmp.Transform()), test.desc)
			require.Equal(t, "", cmp.Diff(shelf2Books, test.expectShelf2Books, protocmp.Transform()), test.desc)
		}
		log.Println("--------------------")
	}
}

func TestDiscardUnknownParams(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := fmt.Sprintf("http://%s", ln.Addr())
	defer ln.Close()
	// service registration
	s := &server.Server{}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName("trpc.test.helloworld.GreeterDiscardUnknownParams"),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithRESTOptions(
			restful.WithDiscardUnknownParams(true),
		),
	)
	s.AddService("trpc.test.helloworld.GreeterDiscardUnknownParams", service)
	hpb.RegisterGreeterService(s, &greeterServerImpl{})

	// start server
	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	// unknown query params
	req, err := http.NewRequest(http.MethodGet, addr+"/v2/bar?name=xyz&unknown_arg=anything", nil)
	require.Nil(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Nil(t, s.Close(nil))
}

func TestMultipleServiceBinding(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := fmt.Sprintf("http://%s", ln.Addr())
	defer ln.Close()
	s := &server.Server{}
	serviceName := "trpc.test.helloworld.TestMultipleServiceBinding"
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName(serviceName),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithRESTOptions(
			restful.WithDiscardUnknownParams(true),
		),
	)
	s.AddService(serviceName, service)
	// Register multiple services from different pb.
	hpb.RegisterGreeterService(s, &greeterServerImpl{})
	bpb.RegisterBookstoreService(s, &bookstoreServiceImpl{})

	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()

	time.Sleep(100 * time.Millisecond)

	// Test service 1.
	req, err := http.NewRequest(http.MethodGet, addr+"/v2/bar?name=xyz", nil)
	require.Nil(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Test service 2.
	req2, err := http.NewRequest(http.MethodGet, addr+"/shelves", nil)
	require.Nil(t, err)
	resp2, err := http.DefaultClient.Do(req2)
	require.Nil(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	require.Nil(t, s.Close(nil))
}

func TestRESTfulRspTypeAssertion(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.Nil(t, err)
	addr := fmt.Sprintf("http://%s", ln.Addr())
	defer ln.Close()
	s := &server.Server{}
	serviceName := "trpc.test.helloworld." + t.Name()
	type someCustomType struct {
		SomeField string
	}
	service := server.New(
		server.WithListener(ln),
		server.WithServiceName(serviceName),
		server.WithNetwork("tcp"),
		server.WithProtocol("restful"),
		server.WithNamedFilter("custom_rsp_type",
			func(ctx context.Context, req interface{}, next filter.ServerHandleFunc) (rsp interface{}, err error) {
				_, _ = next(ctx, req)
				return &someCustomType{"hello"}, nil
			}),
	)
	s.AddService(serviceName, service)
	hpb.RegisterGreeterService(s, &greeterServerImpl{})
	go func() {
		err := s.Serve()
		require.Nil(t, err)
	}()
	time.Sleep(100 * time.Millisecond)
	req, err := http.NewRequest(http.MethodGet, addr+"/v2/bar?name=xyz", nil)
	require.Nil(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	bs, err := io.ReadAll(resp.Body)
	require.Nil(t, err)
	t.Logf("response: %q\n", bs)
}
