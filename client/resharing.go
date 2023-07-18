package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/lionstory/tsslib-grpc/pkg/proto"
	"google.golang.org/grpc"
	"strings"
	"time"
	//"github.com/golang/protobuf/ptypes/empty"
)

func main() {
	urls := flag.String("urls", ":8000,:8001,:8002", "tss servers for key generate")
	nt := flag.Int("nt", 2, "new threshold for tss")
	ot := flag.Int("ot", 1, "old threshold for tss")
	isSm2Enabled := flag.Bool("sm2", false, "is sm2 enabled")
	flag.Parse()

	u := strings.Split(strings.TrimSpace(*urls), ",")

	var conn *grpc.ClientConn
	var data *pb.CommonResonse
	var err error
	if conn, err = grpc.Dial(u[0], grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second)); err != nil {
		panic(err)
	}
	defer conn.Close()

	client := pb.NewTssServerClient(conn)
	ctx := context.Background()

	if *isSm2Enabled {
		fmt.Println("===sm2 reshare======")
		data, err = client.SmtResharePrepare(ctx, &pb.SmtResharePrepareRequest{Urls: u, NewThreshold: int32(*nt), OldThreshold: int32(*ot)})
	} else {
		data, err = client.ReSharingPrepare(ctx, &pb.ReSharingPrepareRequest{Urls: u, Threshold: int32(*nt), OldThreshold: int32(*ot)})
	}
	if err != nil {
		panic(err)
	}
	fmt.Println("=========", data)
}
