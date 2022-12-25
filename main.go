package main

import (
	"context"
	"fmt"
	"github.com/m4schini/logger"
	metrics "github.com/m4schini/logger/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
	"net"
	"os"
	"os/signal"
	"syscall"
	"web-scraper-module/config"
	pb "web-scraper-module/proto"
	"web-scraper-module/server"
)

var log = logger.Named("scraper").Sugar()

func main() {
	defer logger.Sync()
	defer log.Info("graceful shutdown succeeded")
	log.Infof("starting...")
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Debugf("listening on port: %v", config.Port())
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", config.Port()))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	defer grpcServer.GracefulStop()
	log.Debug("initialized grpc server")

	sSrv := server.NewScraperServer()
	log.Debug("initialized scraper server")

	pb.RegisterScraperServer(grpcServer, sSrv)
	reflection.Register(grpcServer)
	log.Debug("registered reflection")
	log.Infof("serving grpc on: %v", lis.Addr())
	go func() {

		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalln(err)
		}
	}()

	go func() {
		if config.MetricsPort() == 0 {
			log.Warn("$METRICS_PORT is undefined. Prometheus is disabled")
			return
		}
		if err := metrics.ServeMetrics(); err != nil {
			log.Error(err)
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")
}
