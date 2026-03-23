module github.com/alialaee/raf/benchmark

go 1.26.1

require (
	github.com/alialaee/raf v0.2.2
	github.com/fxamacker/cbor/v2 v2.9.0
	github.com/vmihailenco/msgpack/v5 v5.4.1
	go.mongodb.org/mongo-driver/v2 v2.5.0
	google.golang.org/protobuf v1.36.11
)

require (
	github.com/vmihailenco/tagparser/v2 v2.0.0 // indirect
	github.com/x448/float16 v0.8.4 // indirect
)

replace github.com/alialaee/raf => ../
