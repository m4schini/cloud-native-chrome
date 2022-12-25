package server

import (
	"github.com/m4schini/logger"
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
	Name:    "cdp_action_durations",
	Help:    "histogram of chromedp actions execution duration",
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

func (s *scraperServer) Control(instructions pb.Scraper_ControlServer) error {
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
		start := time.Now()
		inCh <- instruction
		response := <-outCh
		scraperDurations.Observe(time.Since(start).Seconds())
		if response != nil {
			err := instructions.Send(response)
			if err != nil {
				return err
			}
		}
	}
}
