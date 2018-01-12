package mudp_test

import (
	"context"
	"time"

	"github.com/influx6/mnet"
	"github.com/influx6/mnet/mudp"
)

func createBenchmarkNetwork(ctx context.Context, addr string) (*mudp.Network, error) {
	var netw mudp.Network
	netw.Addr = addr
	netw.Metrics = events
	netw.MaxWriteDeadline = 3 * time.Second

	netw.Handler = func(client mnet.Client) error {
		for {
			if _, err := client.Read(); err != nil {
				if err == mnet.ErrNoDataYet {
					continue
				}

				return err
			}
		}
	}

	return &netw, netw.Start(ctx)
}
