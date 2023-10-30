# tRPC-Go Testing
[api-testing](https://github.com/LinuxSuRen/api-testing) is a tool that helps developers and testers to explore the APIs without writing any codes. It's open-source, and provides two ways to use: CLI, and Web UI. It supports multiple protocols including `HTTP`, `gRPC`, and `tRPC`.

This guide will walk you through the `tRPC` usage with this tool.

## Install
It provides a variety  of ways to install, for instance: Docker, Helm chart, Kubernetes Operator .etc. Considering Docker might be the easiest way, you will see this way below:

```shell
# tRPC feature support from v0.0.14
docker run -p 8080:8080 linuxsuren/api-testing:master
```

## Usage
Once the container is ready, you can visit the web page with the following address:

`http://localhost:8080`

![image](https://github.com/trpc-group/trpc-go/assets/1450685/fa9c66fc-ec5b-4a70-9466-f6d923aac229)
![image](https://github.com/trpc-group/trpc-go/assets/1450685/03347a59-51a4-43ed-aa44-7b0dec340b90)
![image](https://github.com/trpc-group/trpc-go/assets/1450685/12d1c7b0-bffd-4a48-adc4-5ed4a156afef)
![image](https://github.com/trpc-group/trpc-go/assets/1450685/0c264a5e-dec0-400a-a420-55af79dbedbd)

## Read More
[api-testing](https://github.com/linuxsuren/api-testing) has several backend storage, it will save the data as a regular file if you just want to have a try. But it also can save the data to an ORM database or git repository. Please feel free to read more detail from the official document.
