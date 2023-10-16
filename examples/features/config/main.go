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

// Package main is the main package.
package main

import (
	"fmt"

	"trpc.group/trpc-go/trpc-go/config"
)

func main() {
	// Parse configuration files in yaml format.
	c, err := config.Load("custom.yaml", config.WithCodec("yaml"), config.WithProvider("file"))
	if err != nil {
		fmt.Println(err)
		return
	}

	// The format of the configuration file corresponds to custom struct.
	var custom struct {
		Custom struct {
			Test    string `yaml:"test"`
			TestObj struct {
				Key1 string `yaml:"key1"`
				Key2 bool   `yaml:"key2"`
				Key3 int32  `yaml:"key3"`
			} `yaml:"test_obj"`
		} `yaml:"custom"`
	}

	err = c.Unmarshal(&custom)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("custom : %v \n", custom)

	fmt.Printf("test : %s \n", c.GetString("custom.test", ""))
	fmt.Printf("key1 : %s \n", c.GetString("custom.test_obj.key1", ""))
	fmt.Printf("key2 : %t \n", c.GetBool("custom.test_obj.key2", false))
	fmt.Printf("key2 : %d \n", c.GetInt32("custom.test_obj.key3", 0))

}
