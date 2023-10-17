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
	"bytes"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	thttp "trpc.group/trpc-go/trpc-go/http"
	"trpc.group/trpc-go/trpc-go/server"
)

func TestRegisterDefaultService(t *testing.T) {
	defer func() {
		err := recover()
		require.New(t).Contains(err, "duplicate method name")
		thttp.DefaultServerCodec.AutoReadBody = true
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	s := server.New()
	thttp.HandleFunc("/test/path", func(w http.ResponseWriter, r *http.Request) error { return nil })
	thttp.HandleFunc("/test/path", func(w http.ResponseWriter, r *http.Request) error { return nil })
	thttp.RegisterDefaultService(s)
}

func TestRegisterServiceMux(t *testing.T) {
	defer func() {
		err := recover()
		require.New(t).Contains(err, "duplicate method name")
		thttp.DefaultServerCodec.AutoReadBody = true
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	s := server.New()
	thttp.RegisterServiceMux(s, nil)
	thttp.RegisterServiceMux(s, nil)
}

func TestMultipartTmpFileCleaning(t *testing.T) {
	// Setup server.
	ln, err := net.Listen("tcp", "localhost:0")
	require.Nil(t, err)
	defer ln.Close()
	serviceName := "trpc.http.server.MultipartTmpFileCleaningTest"
	service := server.New(
		server.WithServiceName(serviceName),
		server.WithNetwork("tcp"),
		server.WithProtocol("http_no_protocol"),
		server.WithListener(ln),
	)
	var tmpFiles []string
	defer func() {
		// Ensure that the temporary files are removed despite the test failure.
		for i := range tmpFiles {
			os.Remove(tmpFiles[i])
		}
	}()
	thttp.HandleFunc("/test/multipart", func(_ http.ResponseWriter, r *http.Request) error {
		const verySmallMaximumMemory = 4
		if err := r.ParseMultipartForm(verySmallMaximumMemory); err != nil {
			log.Println("err: ", err)
			return err
		}
		for _, files := range r.MultipartForm.File {
			f, _ := files[0].Open()
			if osFile, ok := f.(*os.File); ok {
				tmpFiles = append(tmpFiles, osFile.Name())
			}
			f.Close()
		}
		return nil
	})
	defer func() {
		// Remove the registered handle func to ensure the independency of each test. 😅
		thttp.ServiceDesc.Methods = thttp.ServiceDesc.Methods[:0]
	}()
	s := &server.Server{}
	s.AddService(serviceName, service)
	thttp.RegisterNoProtocolService(s.Service(serviceName))
	go func() {
		require.Nil(t, s.Serve())
	}()
	defer s.Close(nil)
	time.Sleep(100 * time.Millisecond)

	// Setup multipart form data.
	const fileSize = 33554432 // 32MB
	largeFileContent := make([]byte, fileSize)
	rd := bytes.NewReader(largeFileContent)
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("data", "largefile.test")
	require.Nil(t, err)
	_, err = io.Copy(fw, rd)
	require.Nil(t, err)
	require.Nil(t, w.Close())

	// Setup client.
	req, err := http.NewRequest("POST", "http://"+ln.Addr().String()+"/test/multipart", &b)
	require.Nil(t, err)
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := http.DefaultClient
	res, err := client.Do(req)
	require.Nil(t, err)
	require.Equal(t, res.StatusCode, http.StatusOK)

	// Check whether temporary file is removed.
	require.Eventually(t, func() bool {
		for i := range tmpFiles {
			if _, err := os.Stat(tmpFiles[i]); !errors.Is(err, os.ErrNotExist) {
				t.Logf("tmp file %s may still exist, err: %+v", tmpFiles[i], err)
				return false
			}
		}
		return true
	}, time.Second, 10*time.Millisecond)
}
