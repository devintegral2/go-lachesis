package posnode

import (
	"context"
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/Fantom-foundation/go-lachesis/src/posnode/api"
)

type (
	// client of node service.
	// TODO: make reusable connections pool
	client struct {
		opts []grpc.DialOption
	}
)

// ConnectTo connects to other node service.
func (n *Node) ConnectTo(peer *Peer) (api.NodeClient, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), n.conf.ConnectTimeout)
	defer cancel()

	addr := n.NetAddrOf(peer.Host)
	n.log.Debugf("connect to %s", addr)

	cp := x509.NewCertPool()
	cp.AppendCertsFromPEM(peer.Cert)

	creds := credentials.NewTLS(&tls.Config{ServerName: "", RootCAs: cp})

	conn, err := grpc.DialContext(ctx, addr, append(n.client.opts, grpc.WithTransportCredentials(creds), grpc.WithBlock())...)
	if err != nil {
		n.log.Warn(errors.Wrapf(err, "connect to: %s", addr))
		return nil, nil, err
	}

	free := func() {
		conn.Close()
	}

	return api.NewNodeClient(conn), free, nil
}
