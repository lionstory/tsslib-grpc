package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/lionstory/tsslib-grpc/pkg/proto"
	"google.golang.org/grpc"
	"strings"
	"time"
)

func main() {
	urls := flag.String("urls", ":8000,:8001,:8002", "party to connect")
	threshold := flag.Int64("t", 1, "threshold for vss")
	project := flag.Int64("project", 1, "project id for vss")
	raw := flag.String("raw", "", "content for encrypt/decrypt")

	setup := flag.Bool("setup", false, "flag for project setup")
	keygen := flag.Bool("keygen", false, "flag for keygen")
	encrypt := flag.Bool("encrypt", false, "flag for encrypt")
	decrypt := flag.Bool("decrypt", false, "flag for decrypt")
	flag.Parse()

	u := strings.Split(strings.TrimSpace(*urls), ",")
	var partners []*pb.Partner
	for k, v := range u {
		partners = append(partners, &pb.Partner{ID: int64(k + 1), Address: v})
	}

	var conn *grpc.ClientConn
	var data *pb.CommonResonse
	var err error
	if conn, err = grpc.Dial(partners[0].Address, grpc.WithInsecure(), grpc.WithBlock(), grpc.WithTimeout(5*time.Second)); err != nil {
		panic(err)
	}
	defer conn.Close()

	client := pb.NewVssServiceClient(conn)
	ctx := context.Background()

	if *setup {
		data, err = client.ProjectSetup(ctx, &pb.ProjectSetupRequest{
			ProjectID: *project,
			PartID:    partners[0].ID,
			Partners:  partners,
		})
	} else if *keygen {
		data, err = client.KeyGenerate(ctx, &pb.KeyGenerateRequest{
			ProjectID: *project,
			Curve:     pb.CurveType_p256,
			Threshold: *threshold,
		})
	} else if *encrypt {
		data, err = client.Encrypt(context.TODO(), &pb.CommonRequest{
			ProjectID: *project,
			Raw:       *raw,
		})
	} else if *decrypt {
		data, err = client.Decrypt(context.TODO(), &pb.CommonRequest{
			ProjectID: *project,
			Raw:       *raw,
		})
	}

	if err != nil {
		panic(err)
	}
	bs, _ := json.Marshal(data)
	fmt.Printf("%s\n", bs)
}
