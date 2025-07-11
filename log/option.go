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

package log

// Option modifies the options of optionLogger.
type Option func(*options)

type options struct {
	skip int
}

// WithAdditionalCallerSkip adds additional caller skip.
func WithAdditionalCallerSkip(skip int) Option {
	return func(o *options) {
		o.skip = skip
	}
}
