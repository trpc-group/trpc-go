// Tencent is pleased to support the open source community by making tRPC available.
// Copyright (C) 2023 THL A29 Limited, a Tencent company. All rights reserved.
// If you have downloaded a copy of the tRPC source code from Tencent,
// please note that tRPC source code is licensed under the Apache 2.0 License that can be found in the LICENSE file.

package restful_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/emptypb"

	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/restful"
	"trpc.group/trpc-go/trpc-go/server"
	bpb "trpc.group/trpc-go/trpc-go/testdata/restful/bookstore"
	hpb "trpc.group/trpc-go/trpc-go/testdata/restful/helloworld"
	"trpc.group/trpc-go/trpc-go/transport"
)

// helloworld service impl
type greeterServerImpl struct{}

func (s *greeterServerImpl) SayHello(ctx context.Context, req *hpb.HelloRequest) (*hpb.HelloReply, error) {
	rsp := &hpb.HelloReply{}
	if req.Name != "xyz" {
		return nil, errors.New("test error")
	}
	rsp.Message = "test"
	return rsp, nil
}

func TestHelloworldService(t *testing.T) {
	// service registration
	s := &server.Server{}
	service := server.New(
		server.WithAddress("127.0.0.1:6677"),
		server.WithServiceName("trpc.test.helloworld.Service"),
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
	_, err := gBuf.Write([]byte(data))
	require.Nil(t, err)
	gBuf.Close()
	req, err := http.NewRequest("POST", "http://127.0.0.1:6677/v1/foobar", &buf)
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
	req2, err := http.NewRequest("GET", "http://127.0.0.1:6677/v2/bar?name=xyz", nil)
	require.Nil(t, err)
	resp2, err := http.DefaultClient.Do(req2)
	require.Nil(t, err)
	defer resp2.Body.Close()
	require.Equal(t, resp2.StatusCode, http.StatusOK)

	// test response content-type
	require.Equal(t, resp2.Header.Get("Content-Type"), "application/json")
}

func TestHeaderMatcher(t *testing.T) {
	// service registration
	s := &server.Server{}
	service := server.New(server.WithAddress("127.0.0.1:6678"),
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
	req, err := http.NewRequest("POST", "http://127.0.0.1:6678/v1/foobar",
		bytes.NewBuffer([]byte(`{"name": "xyz"}`)))
	require.Nil(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestResponseHandler(t *testing.T) {
	// service registration
	s := &server.Server{}
	service := server.New(
		server.WithAddress("127.0.0.1:6679"),
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
	req, err := http.NewRequest("POST", "http://127.0.0.1:6679/v1/foobar",
		bytes.NewBuffer([]byte(`{"name": "xyz"}`)))
	require.Nil(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusInternalServerError, resp.StatusCode)
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
	// service registration
	s := &server.Server{}
	service := server.New(server.WithAddress("127.0.0.1:6666"),
		server.WithServiceName("trpc.test.bookstore.Bookstore"),
		server.WithProtocol("restful"))
	s.AddService("trpc.test.bookstore.Bookstore", service)
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
			httpRequest: httpNewRequest(t, "GET", "http://127.0.0.1:6666/shelves", nil,
				"application/json"),
			respStatusCode: http.StatusOK,
			desc:           "test listing shelves",
		},
		{
			httpRequest: httpNewRequest(t, "POST", "http://127.0.0.1:6666/shelf", bytes.NewBuffer([]byte(
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
			httpRequest: httpNewRequest(t, "POST", "http://127.0.0.1:6666/shelf", bytes.NewBuffer([]byte(
				`{"shelf":{"id":2,"theme":"shelf_02"}}`)), "application/json"),
			respStatusCode: http.StatusConflict,
			desc:           "test creating dup shelf",
		},
		{
			httpRequest: httpNewRequest(t, "DELETE", "http://127.0.0.1:6666/shelf/2",
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
			httpRequest: httpNewRequest(t, "DELETE", "http://127.0.0.1:6666/shelf/2",
				nil, "application/json"),
			respStatusCode: http.StatusNotFound,
			desc:           "test deleting a shelf non exists",
		},
		{
			httpRequest: httpNewRequest(t, "POST", "http://127.0.0.1:6666/anything",
				nil, "application/json"),
			respStatusCode: http.StatusNotFound,
			desc:           "test invalid url path",
		},
		{
			httpRequest: httpNewRequest(t, "POST",
				"http://127.0.0.1:6666/shelf/theme/shelf_2?shelf.theme=x&shelf.id=2", nil,
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
			httpRequest: httpNewRequest(t, "POST",
				"http://127.0.0.1:6666/shelf/theme/shelf_2?anything=2",
				nil, "application/json"),
			respStatusCode: http.StatusBadRequest,
			desc:           "test creating a shelf with invalid query params",
		},
		{
			httpRequest: httpNewRequest(t, "POST", "http://127.0.0.1:6666/book/shelf/1",
				bytes.NewBuffer([]byte("anything")), "application/json"),
			respStatusCode: http.StatusBadRequest,
			desc:           "test creating a book with invalid body data",
		},
		{
			httpRequest: httpNewRequest(t, "PATCH", "http://127.0.0.1:6666/book/shelfid/1/bookid/1",
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
			httpRequest: httpNewRequest(t, "POST", "http://127.0.0.1:6666/book/shelf/2",
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
			httpRequest: httpNewRequest(t, "POST", "http://127.0.0.1:6666/book/shelf/2",
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
			httpRequest: httpNewRequest(t, "PATCH", "http://127.0.0.1:6666/book/shelfid/2",
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
		require.Equal(t, test.respStatusCode, resp.StatusCode, test.desc)

		if resp.StatusCode > 200 && resp.StatusCode < 300 {
			require.Equal(t, "", cmp.Diff(shelves, test.expectShelves, protocmp.Transform()), test.desc)
			require.Equal(t, "", cmp.Diff(shelf2Books, test.expectShelf2Books, protocmp.Transform()), test.desc)
		}
	}
}

func TestBasedOnFastHTTP(t *testing.T) {
	// replace server transport based on fasthttp
	transport.RegisterServerTransport("restful_based_on_fasthttp",
		thttp.NewRESTServerTransport(true))

	// service registration
	s := &server.Server{}
	service := server.New(server.WithAddress("127.0.0.1:45678"),
		server.WithServiceName("trpc.test.helloworld.FastHTTP"),
		server.WithProtocol("restful_based_on_fasthttp"),
		server.WithRESTOptions(
			restful.WithFastHTTPHeaderMatcher(
				func(ctx context.Context, requestCtx *fasthttp.RequestCtx, serviceName string,
					methodName string) (context.Context, error) {
					return context.Background(), nil
				},
			),
			restful.WithFastHTTPRespHandler(
				func(
					ctx context.Context,
					requestCtx *fasthttp.RequestCtx,
					resp proto.Message,
					body []byte,
				) error {
					if string(requestCtx.Request.Header.Peek("Accept-Encoding")) != "gzip" {
						return errors.New("test error")
					}
					writeCloser, err := (&restful.GZIPCompressor{}).
						Compress(requestCtx.Response.BodyWriter())
					if err != nil {
						return err
					}
					defer writeCloser.Close()
					requestCtx.Response.Header.Set("Content-Encoding", "gzip")
					requestCtx.Response.Header.Set("Content-Type", "application/json")
					writeCloser.Write(body)
					return nil
				},
			),
			restful.WithFastHTTPErrorHandler(
				func(ctx context.Context, requestCtx *fasthttp.RequestCtx, err error) {
					requestCtx.Response.SetStatusCode(http.StatusInternalServerError)
					requestCtx.Response.Header.Set("Content-Type", "application/json")
					requestCtx.Write([]byte(`{"massage":"test error"}`))
				},
			),
		),
	)
	s.AddService("trpc.test.helloworld.FastHTTP", service)
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
	_, err := gBuf.Write([]byte(data))
	require.Nil(t, err)
	gBuf.Close()
	req, err := http.NewRequest("POST", "http://127.0.0.1:45678/v1/foobar", &buf)
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
	req2, err := http.NewRequest("GET", "http://127.0.0.1:45678/v2/bar?name=xyz", nil)
	require.Nil(t, err)
	resp2, err := http.DefaultClient.Do(req2)
	require.Nil(t, err)
	defer resp2.Body.Close()
	require.Equal(t, resp2.StatusCode, http.StatusOK)

	// test response content-type
	require.Equal(t, resp2.Header.Get("Content-Type"), "application/json")

	// test server error
	req3, err := http.NewRequest("GET", "http://127.0.0.1:45678/v2/bar?name=anything", nil)
	require.Nil(t, err)
	resp3, err := http.DefaultClient.Do(req3)
	require.Nil(t, err)
	defer resp3.Body.Close()
	require.Equal(t, resp3.StatusCode, http.StatusInternalServerError)

	// test err handler
	data4 := `{"name": "abc"}`
	buf4 := bytes.Buffer{}
	gBuf4 := gzip.NewWriter(&buf4)
	_, err = gBuf4.Write([]byte(data4))
	require.Nil(t, err)
	gBuf4.Close()
	req4, err := http.NewRequest("POST", "http://127.0.0.1:45678/v1/foobar", &buf4)
	require.Nil(t, err)
	req4.Header.Add("Content-Type", "anything")
	req4.Header.Add("Content-Encoding", "gzip")
	req4.Header.Add("Accept-Encoding", "gzip")
	cli4 := http.Client{}
	resp4, err := cli4.Do(req4)
	require.Nil(t, err)
	defer resp4.Body.Close()
	require.Equal(t, resp4.StatusCode, http.StatusInternalServerError)
	bodyBytes4, err := io.ReadAll(resp4.Body)
	require.Nil(t, err)
	require.Equal(t, bodyBytes4, []byte(`{"massage":"test error"}`))
}

func TestDiscardUnknownParams(t *testing.T) {
	// service registration
	s := &server.Server{}
	service := server.New(
		server.WithAddress("127.0.0.1:6680"),
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
	req, err := http.NewRequest("GET", "http://127.0.0.1:6680/v2/bar?name=xyz&unknown_arg=anything", nil)
	require.Nil(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Nil(t, s.Close(nil))
}

func TestMultipleServiceBinding(t *testing.T) {
	s := &server.Server{}
	serviceName := "trpc.test.helloworld.TestMultipleServiceBinding"
	service := server.New(
		server.WithAddress("127.0.0.1:6681"),
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
	req, err := http.NewRequest("GET", "http://127.0.0.1:6681/v2/bar?name=xyz", nil)
	require.Nil(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.Nil(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Test service 2.
	req2, err := http.NewRequest("GET", "http://127.0.0.1:6681/shelves", nil)
	require.Nil(t, err)
	resp2, err := http.DefaultClient.Do(req2)
	require.Nil(t, err)
	defer resp2.Body.Close()
	require.Equal(t, http.StatusOK, resp2.StatusCode)
	require.Nil(t, s.Close(nil))
}
