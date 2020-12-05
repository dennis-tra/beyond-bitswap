package utils

import (
	"bytes"
	"context"
	"fmt"
	"math"

	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"golang.org/x/sync/errgroup"
)

func AddrInfosFromChan(peerCh chan *peer.AddrInfo, count int) ([]peer.AddrInfo, error) {
	var ais []peer.AddrInfo
	for i := 1; i <= count; i++ {
		ai, ok := <-peerCh
		if !ok {
			return ais, fmt.Errorf("subscription closed")
		}
		ais = append(ais, *ai)
	}
	return ais, nil
}

// DialOtherPeers connects to a set of peers in the experiment.
func DialOtherPeers(ctx context.Context, self core.Host, ais []peer.AddrInfo, maxConnectionRate int) ([]peer.AddrInfo, error) {
	// Grab list of other peers that are available for this Run
	var toDial []peer.AddrInfo
	for _, ai := range ais {
		id1, _ := ai.ID.MarshalBinary()
		id2, _ := self.ID().MarshalBinary()

		// skip over dialing ourselves, and prevent TCP simultaneous
		// connect (known to fail) by only dialing peers whose peer ID
		// is smaller than ours.
		if bytes.Compare(id1, id2) < 0 {
			toDial = append(toDial, ai)
		}
	}

	// Limit max number of connections for the peer according to rate.
	rate := float64(maxConnectionRate) / 100
	toDial = toDial[:int(math.Ceil(float64(len(toDial))*rate))]

	// Dial to all the other peers
	g, ctx := errgroup.WithContext(ctx)
	for _, ai := range toDial {
		ai := ai
		g.Go(func() error {
			if err := self.Connect(ctx, ai); err != nil {
				return fmt.Errorf("Error while dialing peer %v: %w", ai.Addrs, err)
			}
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}

	return toDial, nil
}