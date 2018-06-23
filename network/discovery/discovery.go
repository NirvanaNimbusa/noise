package discovery

import (
	"strings"

	"github.com/golang/glog"
	"github.com/perlin-network/noise/dht"
	"github.com/perlin-network/noise/network"
	"github.com/perlin-network/noise/network/builders"
	"github.com/perlin-network/noise/peer"
	"github.com/perlin-network/noise/protobuf"
)

type HandshakeRequestProcessor struct{}

func (HandshakeRequestProcessor) Handle(client *network.PeerClient, message *network.IncomingMessage) error {
	// Update routing table w/ peer's ID.
	client.Network().Routes.Update(*client.Id)

	// Send handshake response to peer.
	err := client.Tell(&protobuf.HandshakeResponse{})

	if err != nil {
		return err
	}
	return nil
}

type HandshakeResponseProcessor struct{}

func (HandshakeResponseProcessor) Handle(client *network.PeerClient, raw *network.IncomingMessage) error {
	// Update routing table w/ peer's ID.
	client.Network().Routes.Update(*client.Id)

	addresses, publicKeys := bootstrapPeers(client.Network(), *client.Id, dht.BucketSize)

	// Update routing table w/ bootstrapped peers.
	for i := 0; i < len(addresses); i++ {
		client.Network().Routes.Update(peer.CreateID(addresses[i], publicKeys[i]))
	}

	glog.Infof("[handshake] bootstrapped w/ peer(s): %s.", strings.Join(getConnectedPeers(client), ", "))

	return nil
}

type LookupNodeRequestProcessor struct{}

func (LookupNodeRequestProcessor) Handle(c *network.PeerClient, raw *network.IncomingMessage) error {
	// Deserialize received request.
	msg := raw.Message.(*protobuf.LookupNodeRequest)

	// Prepare response.
	response := &protobuf.LookupNodeResponse{Peers: []*protobuf.ID{}}

	// Update routing table w/ peer's ID.
	c.Network().Routes.Update(*c.Id)

	// Respond back with closest peers to a provided target.
	for _, id := range c.Network().Routes.FindClosestPeers(peer.ID(*msg.Target), dht.BucketSize) {
		id := protobuf.ID(id)
		response.Peers = append(response.Peers, &id)
	}

	err := c.Reply(raw.Nonce, response)
	if err != nil {
		// TODO: Handle error responding to client.
	}

	glog.Infof("[lookup] connected peers: %s.", strings.Join(c.Network().Routes.GetPeerAddresses(), ", "))

	return nil
}

// Registers necessary message processors for peer discovery.
func BootstrapPeerDiscovery(builder *builders.NetworkBuilder) {
	builder.AddProcessor((*protobuf.HandshakeRequest)(nil), new(HandshakeRequestProcessor))
	builder.AddProcessor((*protobuf.HandshakeResponse)(nil), new(HandshakeResponseProcessor))
	builder.AddProcessor((*protobuf.LookupNodeRequest)(nil), new(LookupNodeRequestProcessor))
}

func getConnectedPeers(c *network.PeerClient) []string {
	peers := []string{}
	c.Network().Peers.Range(func(k, v interface{}) bool {
		peers = append(peers, k.(string))
		return true
	})
	return peers
}
