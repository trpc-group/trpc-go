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

package http

import (
	"bytes"
	"fmt"
	"io"
	"strconv"

	"github.com/r3labs/sse/v2"
)

// WriteSSE encodes an event to the sse format, and writes it to the writer.
func WriteSSE(writer io.Writer, event sse.Event) error {
	var buf bytes.Buffer

	if err := writeID(&buf, event.ID); err != nil {
		return fmt.Errorf("write id: %w", err)
	}
	if err := writeEvent(&buf, event.Event); err != nil {
		return fmt.Errorf("write event: %w", err)
	}
	if err := writeRetry(&buf, event.Retry); err != nil {
		return fmt.Errorf("write retry: %w", err)
	}
	if err := writeData(&buf, event.Data); err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	// Write the empty line to indicate the end of the event.
	buf.WriteString("\n")
	_, err := writer.Write(buf.Bytes())
	return err
}

func writeID(w io.Writer, id []byte) error {
	if len(id) == 0 {
		return nil
	}
	if _, err := w.Write([]byte("id:")); err != nil {
		return err
	}
	if _, err := w.Write(id); err != nil {
		return err
	}
	_, err := w.Write([]byte("\n"))
	return err
}

func writeEvent(w io.Writer, event []byte) error {
	if len(event) == 0 {
		return nil
	}
	if _, err := w.Write([]byte("event:")); err != nil {
		return err
	}
	if _, err := w.Write(event); err != nil {
		return err
	}
	_, err := w.Write([]byte("\n"))
	return err
}

func writeRetry(w io.Writer, retry []byte) error {
	retryUint, err := strconv.ParseUint(string(retry), 10, 64)
	if err != nil {
		return nil
	}
	if retryUint == 0 {
		return nil
	}
	if _, err := w.Write([]byte("retry:")); err != nil {
		return err
	}
	if _, err := w.Write(retry); err != nil {
		return err
	}
	_, err = w.Write([]byte("\n"))
	return err
}

func writeData(w io.Writer, data []byte) error {
	if _, err := w.Write([]byte("data:")); err != nil {
		return err
	}
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err := w.Write([]byte("\n"))
	return err
}
