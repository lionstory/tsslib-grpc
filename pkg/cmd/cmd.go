package cmd

import (
	"github.com/lionstory/tsslib-grpc/pkg/config"
	"github.com/lionstory/tsslib-grpc/pkg/server"
	"github.com/lionstory/tsslib-grpc/pkg/utils"
	"github.com/ipfs/go-log"
	"github.com/spf13/cobra"
)

type ServerCmd struct {
	command    *cobra.Command
	configFile string
	port       int64
	dataDir    string
}

func NewServerCmd() *ServerCmd {
	command := ServerCmd{}
	command.command = &cobra.Command{
		Use:          "TSS GRPC",
		Short:        "TSS GRPC",
		Long:         "TSS party by grpc",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := log.SetLogLevel("tss-lib", "debug"); err != nil {
				panic(err)
			}

			conf, err := config.LoadConfig(command.configFile)
			if err != nil {
				panic(err)
			}

			if command.dataDir != "" {
				conf.SavePath = command.dataDir
			}

			if err := utils.CheckDirectory(conf.SavePath); err != nil {
				panic(err)
			}

			server := server.NewServer(conf)
			server.Start(command.port)
			return nil
		},
	}
	command.command.Flags().StringVarP(&command.configFile, "cfg", "c", "./conf/config.yaml", "config file for license server (for example: aios.yaml)")
	command.command.Flags().StringVarP(&command.dataDir, "dataDir", "d", "./data", "directory to save data (for example: ./data/party-1)")
	command.command.Flags().Int64VarP(&command.port, "port", "p", 8000, "grpc port")
	return &command
}

func (cmd *ServerCmd) GetCommand() *cobra.Command {
	return cmd.command
}
