package server

import (
	"context"
	"github.com/m4schini/logger"
	"github.com/m4schini/util"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/time/rate"
	"io"
	"time"
	pb "web-scraper-module/proto"
	"web-scraper-module/scraper"
)

var log = logger.Named("server").Sugar()
var scraperDurations = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "rpc_handler_get_durations",
	Help:    "histogram of scraper requests",
	Buckets: []float64{2, 4, 5, 6, 7, 8, 10, 20, 30, 40, 50, 60},
})

type scraperServer struct {
	pb.UnimplementedScraperServer
	limiter *rate.Limiter
}

func NewScraperServer() *scraperServer {
	s := new(scraperServer)
	s.limiter = rate.NewLimiter(rate.Limit(1), 4)
	return s
}

func (s *scraperServer) Get(ctx context.Context, in *pb.GetRequest) (*pb.GetResponse, error) {
	requestStart := time.Now()
	requestId := util.GetRequestId(ctx)

	log := log.With("requestId", requestId, "url", in.GetUrl())
	log.Debugw("received request")
	start := time.Now()
	log.Debugf("waited for %v", time.Since(start).String())

	htmlStr, imgbuf, err := scraper.Get(ctx, in.GetUrl(), time.Duration(in.GetTimeoutInMs()*1000000), in.Screenshot)
	scraperDurations.Observe(time.Since(requestStart).Seconds())
	if err != nil {
		log.Error(err)
		return nil, err
	}

	log.Infow("processed request", "duration", time.Since(requestStart).String())
	return &pb.GetResponse{
		Html:       htmlStr,
		Screenshot: imgbuf,
	}, nil
}

func (s *scraperServer) Control(instructions pb.Scraper_ControlServer) error {
	requestId := util.GetRequestId(instructions.Context())
	log := log.With("requestId", requestId)

	inCh := make(chan *pb.ControlRequest)
	defer close(inCh)
	outCh, cancel := scraper.Control(instructions.Context(), inCh)
	defer func() {
		cancel()
		log.Debug("scraper (chromedp) context closed")
	}()
	log.Debug("scraper controller initilized")

	for {
		log.Debug("waiting for instruction")
		instruction, err := instructions.Recv()
		if err == io.EOF {
			log.Info("instructions finished")
			return nil
		}
		if err != nil {
			log.Error(err)
			return err
		}

		log.Debug("forwarding instructions to scraper controller")
		inCh <- instruction

		response := <-outCh
		if response != nil {
			err := instructions.Send(response)
			if err != nil {
				return err
			}
		}
	}
}
