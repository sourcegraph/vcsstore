package vcsclient

//go:generate protoc -I../../../../github.com/gogo/protobuf/protobuf -I../../../../github.com/gogo/protobuf -I../../../../sourcegraph.com/sqs/pbtypes -I. --gogo_out=. vcsclient.proto
//go:generate sed -i "s#timestamp\\.pb#sourcegraph.com/sqs/pbtypes#g" vcsclient.pb.go
//go:generate sed -i "s#\tTreeEntryType_#\t#g" vcsclient.pb.go
