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

package testdata

const (
	MultipartFormDataBody = `----------------------------487682300036072392114180
Content-Disposition: form-data; name="competition"

NBA
----------------------------487682300036072392114180
Content-Disposition: form-data; name="teams"

湖人
----------------------------487682300036072392114180
Content-Disposition: form-data; name="teams"

勇士
----------------------------487682300036072392114180
Content-Disposition: form-data; name="season"

2021
----------------------------487682300036072392114180
Content-Disposition: form-data; name="file1"; filename="1.txt"
Content-Type: text/plain

1
----------------------------487682300036072392114180
Content-Disposition: form-data; name="file2"; filename="1px.png"
Content-Type: image/png

�PNG

IHDR%�V�PLTE�����
IDA�c�!�3IEND�B�
----------------------------487682300036072392114180
Content-Disposition: form-data; name="file3"; filename="json.json"
Content-Type: application/json

{
    "name":"1"
}
----------------------------487682300036072392114180--
`
	MultipartFormDataBoundary       = "multipart/form-data; boundary=--------------------------487682300036072392114180"
	MultipartFormDataFirstPartNames = "competition=NBA&season=2021&teams=%E6%B9%96%E4%BA%BA&teams=%E5%8B%87%E5%A3%AB"

	HTTPSServerAddress = "127.0.0.1:1443"
)
