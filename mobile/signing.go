package mobile

import (
	"context"
	"fmt"
	"github.com/lionstory/tsslib-grpc/pkg/proto"
	"google.golang.org/grpc"
	"strings"
)

func TssSign(urls string, msg string, isSm2Enabled bool) {
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
		data, err = client.SmtSignPrepare(ctx, &pb.SignPrepareRequest{Message: msg, Urls: u})
	} else {
		data, err = client.SignPrepare(ctx, &pb.SignPrepareRequest{Message: msg, Urls: u})
	}
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("=========", data)
}
