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

// Package precool provides service-level precool status primitives.
package precool

// Status represents the precool status of a service.
type Status int

const (
	// Unknown indicates the service is not registered or its precool status is unknown.
	Unknown Status = iota
	// Success indicates the service precool process has completed successfully.
	Success
	// Failure indicates the service precool process has failed.
	Failure
	// Ongoing indicates the service precool process is still running.
	Ongoing
)

// String returns the string representation of the status.
func (s Status) String() string {
	switch s {
	case Success:
		return "proc_success"
	case Failure:
		return "proc_failure"
	case Ongoing:
		return "proc_ongoing"
	default:
		return "unknown"
	}
}

// Func is the function type for custom precool strategies.
type Func func() Status

// Checker wraps the basic precool check methods.
type Checker interface {
	// Register registers a service with a custom precool strategy.
	Register(name string, fn Func) error
	// Unregister unregisters a service from precool check.
	Unregister(name string)
	// CheckService returns the precool status of a specific service.
	CheckService(name string) Status
	// CheckServer returns the precool status of the entire server.
	CheckServer() Status
}
