## Keep Order

## Usage

Start server (you can switch between different `-keep-order` options to check the difference):

```shell
cd examples/features/keeporder
cd server
## Run the server using pre-decode mode of keep-order feature.
go run . -keep-order=pre-decode
## Or run the server using pre-unmarshal mode of keep-order feature.
# go run . -keep-order=pre-unmarshal
## Or run the server without any keep-order feature to show the differences.
# go run . -keep-order=none
```

Start client:

```shell
cd examples/features/keeporder
cd client
go run .

# Expect output for keep-order feature enabled (-keep-order=pre-decode or -keep-order=pre-unmarshal):
2024-10-09 21:05:26.064 INFO    client/main.go:71       [SUCCESS] key key1: expect 1 2 3 4 5 6 7 8 9 10, got 1 2 3 4 5 6 7 8 9 10
2024-10-09 21:05:26.064 INFO    client/main.go:71       [SUCCESS] key key2: expect 1 2 3 4 5 6 7 8 9 10, got 1 2 3 4 5 6 7 8 9 10
2024-10-09 21:05:26.064 INFO    client/main.go:71       [SUCCESS] key key3: expect 1 2 3 4 5 6 7 8 9 10, got 1 2 3 4 5 6 7 8 9 10
2024-10-09 21:05:26.064 INFO    client/main.go:71       [SUCCESS] key key4: expect 1 2 3 4 5 6 7 8 9 10, got 1 2 3 4 5 6 7 8 9 10
2024-10-09 21:05:26.064 INFO    client/main.go:71       [SUCCESS] key key5: expect 1 2 3 4 5 6 7 8 9 10, got 1 2 3 4 5 6 7 8 9 10

# Expect output for keep-order feature disabled (-keep-order=none)
2024-10-09 21:05:40.242 ERROR   client/main.go:69       [FAIL] key key1: expect 1 2 3 4 5 6 7 8 9 10, but got 6 10 4 2 8 9 3 1 5 7
2024-10-09 21:05:40.242 ERROR   client/main.go:69       [FAIL] key key2: expect 1 2 3 4 5 6 7 8 9 10, but got 9 6 2 8 7 10 3 4 5 1
2024-10-09 21:05:40.242 ERROR   client/main.go:69       [FAIL] key key3: expect 1 2 3 4 5 6 7 8 9 10, but got 8 9 2 6 10 4 5 3 7 1
2024-10-09 21:05:40.242 ERROR   client/main.go:69       [FAIL] key key4: expect 1 2 3 4 5 6 7 8 9 10, but got 6 9 10 4 7 2 3 5 1 8
2024-10-09 21:05:40.242 ERROR   client/main.go:69       [FAIL] key key5: expect 1 2 3 4 5 6 7 8 9 10, but got 2 8 4 10 3 5 7 6 1 9
```
