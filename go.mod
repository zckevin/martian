module github.com/google/martian/v3

go 1.11

replace github.com/zckevin/reverse-proxy-cdn => /home/zc/PROJECTS/reverse-proxy-cdn

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/golang/snappy v0.0.3
	github.com/zckevin/reverse-proxy-cdn v0.0.0-00010101000000-000000000000
	golang.org/x/net v0.0.0-20190628185345-da137c7871d7
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.26.0
)
