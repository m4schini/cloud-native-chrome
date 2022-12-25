package scraper

import (
	"context"
	"fmt"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
	"github.com/m4schini/logger"
	"github.com/m4schini/util"
	"os"
	"time"
	pb "web-scraper-module/proto"
)

var log = logger.Named("scraper").Sugar()
var ctxFactory = NewAllocatorContextFactory(context.Background(), os.Getenv("CDP_PROXY_ADDR"))

func Get(ctx context.Context, url string, pageLoadDuration time.Duration, screenshot bool) (string, []byte, error) {
	requestId := util.GetRequestId(ctx)
	log := log.With("requestId", requestId)

	cdp, cancel := ctxFactory.NewContext()
	defer func() {
		cancel()
		log.Debug("chromedp context closed")
	}()
	log.Debugw("created chromedp context", "proxy", os.Getenv("CDP_PROXY_ADDR"))

	actions := []chromedp.Action{
		chromedp.EmulateViewport(1920, 1080),
		chromedp.Navigate(url),
		chromedp.Sleep(pageLoadDuration),
	}

	var htmlBuf = new(string)
	actions = append(actions, CaptureHtml(htmlBuf))
	log.Debug("added action: capture html")

	var imgBuf = make([]byte, 0, 4096)
	if screenshot {
		actions = append(actions, chromedp.CaptureScreenshot(&imgBuf))
		log.Debug("added action: capture screenshot")
	}

	log.Debug("executing chromedp actions")
	err := chromedp.Run(cdp, actions...)
	log.Debug("completed chromedp actions")
	if err != nil {
		return "", imgBuf, err
	}

	return *htmlBuf, imgBuf, nil
}

func Control(ctx context.Context, instructions <-chan *pb.ControlRequest) (<-chan *pb.ControlResponse, context.CancelFunc) {
	requestId := util.GetRequestId(ctx)
	log := log.With("requestId", requestId)

	cdp, cancel := ctxFactory.NewContext()
	log.Debugw("created chromedp context", "proxy", os.Getenv("CDP_PROXY_ADDR"))

	responseCh := make(chan *pb.ControlResponse)

	go func() {
		for instruction := range instructions {
			var cdpAction chromedp.Action
			var returnVal = &pb.ControlResponse{}

			run := func() {
				log.Debug("executing chromedp action")
				err := chromedp.Run(cdp, cdpAction)
				if err != nil {
					returnVal.Payload = &pb.ControlResponse_Error{Error: &pb.CA_Error_Response{Error: err.Error()}}
				}
				log.Debug("executed chromedp action")
			}

			switch x := instruction.Action.(type) {
			case *pb.ControlRequest_EmulateViewport:
				log.Debug("Action: Emulate Viewport")
				cdpAction = chromedp.EmulateViewport(x.EmulateViewport.Width, x.EmulateViewport.Height)
				run()
				break
			case *pb.ControlRequest_Navigate:
				log.Debug("Action: Navigate")
				cdpAction = chromedp.Navigate(x.Navigate.Url)
				run()
				break
			case *pb.ControlRequest_Sleep:
				log.Debug("Action: Sleep")
				cdpAction = chromedp.Sleep(time.Duration(x.Sleep.Duration) * time.Millisecond)
				run()
				break
			case *pb.ControlRequest_Click:
				log.Debug("Action: Click")
				cdpAction = chromedp.Click(x.Click.Selector)
				run()
				break
			case *pb.ControlRequest_SendKeys:
				log.Debug("Action: Send keys")
				cdpAction = chromedp.SendKeys(x.SendKeys.Selector, x.SendKeys.Input)
				run()
				break
			case *pb.ControlRequest_CaptureHtml:
				log.Debug("Action: Capture html")
				var htmlOut = new(string)
				cdpAction = CaptureHtml(htmlOut)
				run()
				returnVal.Payload = &pb.ControlResponse_Html{Html: &pb.CA_CaptureHtml_Response{Html: *htmlOut}}
				break
			case *pb.ControlRequest_CaptureScreenshot:
				log.Debug("Action: Capture screenshot")
				var bufOut = make([]byte, 0)
				cdpAction = chromedp.CaptureScreenshot(&bufOut)
				run()
				returnVal.Payload = &pb.ControlResponse_Screenshot{Screenshot: &pb.CA_CaptureScreenshot_Response{Img: bufOut}}
				break
			case *pb.ControlRequest_WaitVisible:
				log.Debug("Action: Wait visible")
				cdpAction = chromedp.WaitReady(x.WaitVisible.Selector)
				run()
				break
			case *pb.ControlRequest_ScrollBy:
				log.Debug("Action: scroll by")
				cdpAction = chromedp.ActionFunc(func(ctx context.Context) error {
					_, exp, err := runtime.Evaluate(fmt.Sprintf(`window.scrollTo(0,%v);`, x.ScrollBy.ScrollBy)).Do(ctx)
					if err != nil {
						return err
					}
					if exp != nil {
						return exp
					}
					return nil
				})
				run()
				break
			}

			if returnVal.Payload != nil {
				responseCh <- returnVal
			} else {
				responseCh <- nil
			}
		}
		close(responseCh)
	}()

	return responseCh, cancel
}

func CaptureHtml(out *string) chromedp.ActionFunc {
	return func(ctx context.Context) error {
		if out == nil {
			return fmt.Errorf("out cannot be nil")
		}

		node, err := dom.GetDocument().Do(ctx)
		if err != nil {
			return err
		}

		str, err := dom.GetOuterHTML().WithNodeID(node.NodeID).Do(ctx)
		if err != nil {
			return err
		}

		*out = str
		return nil
	}
}
