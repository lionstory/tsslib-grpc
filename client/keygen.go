package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/lionstory/tsslib-grpc/pkg/proto"
	"google.golang.org/grpc"
	"strings"
	"time"
)

func main() {
	urls := flag.String("urls", ":8000,:8001,:8002", "tss servers for key generate")
	threshold := flag.Int("t", 1, "threshold for tss")
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
		data, err = client.SmtKeyGenPrepare(ctx, &pb.KeyGenPrepareRequest{Urls: u, Threshold: int32(*threshold), PartyNum: int32(len(u))})
	} else {
		data, err = client.KeyGenPrepare(ctx, &pb.KeyGenPrepareRequest{Urls: u, Threshold: int32(*threshold), PartyNum: int32(len(u))})
	}

	if err != nil {
		panic(err)
	}
	fmt.Println("=========", data)
}
