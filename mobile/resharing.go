package mobile

import (
	"context"
	"fmt"
	"github.com/lionstory/tsslib-grpc/pkg/proto"
	"google.golang.org/grpc"
	"strings"
)

func TssReshare(urls string, nt, ot int, isSm2Enabled bool) {
	u := strings.Split(urls, ",")

	var conn *grpc.ClientConn
	var data *pb.CommonResonse
	var err error

	if conn, err = grpc.Dial(u[0], grpc.WithInsecure(), grpc.WithBlock()); err != nil {
		fmt.Println("conn failed")
		return
	}
	defer func() {
		if conn != nil {
			conn.Close()
		}
	}()

	client := pb.NewTssServerClient(conn)
	ctx := context.Background()

	if isSm2Enabled {
		data, err = client.SmtResharePrepare(ctx, &pb.SmtResharePrepareRequest{Urls: u, NewThreshold: int32(nt), OldThreshold: int32(ot)})
	} else {
		data, err = client.ReSharingPrepare(ctx, &pb.ReSharingPrepareRequest{Urls: u, Threshold: int32(nt), OldThreshold: int32(ot)})
	}
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("=========", data)
}
