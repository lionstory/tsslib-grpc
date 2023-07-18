package mobile

import (
	"context"
	"fmt"
	"github.com/lionstory/tsslib-grpc/pkg/proto"
	"google.golang.org/grpc"
	"strings"
)

func SssSetup(projectId int, urls string) {
	u := strings.Split(urls, ",")

	var partners []*pb.Partner
	for k, v := range u {
		partners = append(partners, &pb.Partner{ID:int64(k+1), Address: v})
	}

	conn, err := grpc.Dial(partners[0].Address, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		fmt.Println("conn failed")
	}
	defer conn.Close()

	client := pb.NewVssServiceClient(conn)

	data, err := client.ProjectSetup(context.TODO(), &pb.ProjectSetupRequest{
			ProjectID: int64(projectId),
			Partners:  partners,
		})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("=========", data)
}

func SssKeyGen(projectId int, threshold int, urls string) {
	u := strings.Split(urls, ",")

	conn, err := grpc.Dial(u[0], grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		fmt.Println("conn failed")
	}
	defer conn.Close()

	client := pb.NewVssServiceClient(conn)

	data, err := client.KeyGenerate(context.TODO(), &pb.KeyGenerateRequest{
			ProjectID: int64(projectId),
			Curve:     pb.CurveType_p256,
			Threshold: int64(threshold),
		})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("=========", data)
}

func SssEncrypt(projectId int, raw, urls string) {
	u := strings.Split(urls, ",")

	conn, err := grpc.Dial(u[0], grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		fmt.Println("conn failed")
	}
	defer conn.Close()

	client := pb.NewVssServiceClient(conn)

	data, err := client.Encrypt(context.TODO(), &pb.CommonRequest{
			ProjectID: int64(projectId),
			Raw:       raw,
		})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("=========", data)
}

func SssDecrypt(projectId int, raw, urls string) {
	u := strings.Split(urls, ",")

	conn, err := grpc.Dial(u[0], grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		fmt.Println("conn failed")
	}
	defer conn.Close()

	client := pb.NewVssServiceClient(conn)

	data, err := client.Decrypt(context.TODO(), &pb.CommonRequest{
			ProjectID: int64(projectId),
			Raw:       raw,
		})
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println("=========", data)
}
