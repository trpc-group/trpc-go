## Features

This folder lists usage examples of various features, please ensure that any newly added examples adhere to the following standards:

* The subdirectory naming of features should reflect the features themselves, and be concise and expressive.
* The subdirectories of features need to follow a specific structure that includes a `README.md` file. The client and server implementations should each be in their own folder and implemented in a single `main.go` file (ensuring that complete example code for both the client and server is contained in a single file, so users only need to read one file to obtain all the information about the client or server implementation and avoid jumping around). If other shared components are needed, they can be provided in a new folder. The example directory structure is as follows:
```shell
$ tree somefeature/
somefeature/
├── README.md
├── client/
│   ├── main.go
|   └── trpc_go.yaml
│── server/
│   ├── main.go
|   └── trpc_go.yaml
└── shared/ # optional
    └── utility.go
```
* The README.md file in each subdirectory for a feature should also follow a specific format. The template is as follows:
````markdown
# Feature Name

Brief introduction of the feature.

## Usage

Steps to use the feature. Typically:

* Start server.

```shell
$ go run server/main.go -conf server/trpc_go.yaml
```

* Start client.

```shell
$ go run client/main.go -conf client/trpc_go.yaml
```

Then explain the expected result.
````
