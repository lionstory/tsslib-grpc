package server

import (
	"github.com/lionstory/tsslib-grpc/pkg/config"
	"github.com/lionstory/tsslib-grpc/pkg/ecdsa"
	tss_grpc "github.com/lionstory/tsslib-grpc/pkg/grpc"
	"github.com/lionstory/tsslib-grpc/pkg/smt"
)

type Server struct {
	ecdsaServer *ecdsa.EcdsaServer
	grpcServer  *tss_grpc.GrpcServer
	smtServer   *smt.SmtServer
}

func NewServer(conf *config.TssConfig) *Server {
	ecdsa_chan := ecdsa.NewEcdsaChan()
	ecdsaServer := ecdsa.NewEcdsaServer(ecdsa_chan)
	ecdsaServer.KeygenServer.SetConfig(conf)
	ecdsaServer.SignServer.SetConfig(conf)
	ecdsaServer.ReSharingServer.SetConfig(conf)
	smt_chan := smt.NewSmtChan()
	smtServer := smt.NewSmtServer(smt_chan)
	smtServer.KeygenServer.SetConfig(conf)
	smtServer.SignServer.SetConfig(conf)
	smtServer.ReshareServer.SetConfig(conf)
	grpcServer := tss_grpc.NewGrpcServer(ecdsaServer, smtServer, conf.SavePath)
	server := Server{
		ecdsaServer: ecdsaServer,
		smtServer:   smtServer,
		grpcServer:  grpcServer,
	}
	return &server
}

func (server *Server) Start(port int64) {
	go server.ecdsaServer.Start()
	go server.smtServer.Start()
	server.grpcServer.Start(port)
}
