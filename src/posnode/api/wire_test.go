package api

import (
	"context"
	"math"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"

	"github.com/Fantom-foundation/go-lachesis/src/crypto"
	"github.com/Fantom-foundation/go-lachesis/src/hash"
	"github.com/Fantom-foundation/go-lachesis/src/inter/wire"
	"github.com/Fantom-foundation/go-lachesis/src/network"
)

func TestGRPC(t *testing.T) {

	t.Run("over TCP", func(t *testing.T) {
		testGRPC(t, "", "::1", network.TCPListener)
		testGRPCWithoutAuthServer(t, "", "::1", network.TCPListener)
	})

	t.Run("over Fake", func(t *testing.T) {
		from := "client.fake"
		dialer := network.FakeDialer(from)
		testGRPC(t, "server.fake:0", from, network.FakeListener, grpc.WithContextDialer(dialer))
		testGRPCWithoutAuthServer(t, "server.fake:0", from, network.FakeListener, grpc.WithContextDialer(dialer))
	})
}

func testGRPC(t *testing.T, bind, from string, listen network.ListenFunc, opts ...grpc.DialOption) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// keys
	serverKey := crypto.GenerateKey()
	serverID := hash.PeerOfPubkey(serverKey.Public())
	clientKey := crypto.GenerateKey()
	clientID := hash.PeerOfPubkey(clientKey.Public())

	// service
	svc := NewMockNodeServer(ctrl)
	svc.EXPECT().
		SyncEvents(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *KnownEvents) (*KnownEvents, error) {
			assert.Equal(t, from, GrpcPeerHost(ctx))
			assert.Equal(t, clientID, GrpcPeerID(ctx))
			return &KnownEvents{}, nil
		}).
		AnyTimes()
	svc.EXPECT().
		GetEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *EventRequest) (*wire.Event, error) {
			assert.Equal(t, from, GrpcPeerHost(ctx))
			assert.Equal(t, clientID, GrpcPeerID(ctx))
			return &wire.Event{}, nil
		}).
		AnyTimes()
	svc.EXPECT().
		GetPeerInfo(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *PeerRequest) (*PeerInfo, error) {
			assert.Equal(t, from, GrpcPeerHost(ctx))
			assert.Equal(t, clientID, GrpcPeerID(ctx))
			return &PeerInfo{}, nil
		}).
		AnyTimes()

	// server
	server, addr := StartService(bind, serverKey, svc, t.Logf, listen)
	defer server.Stop()

	t.Run("authorized", func(t *testing.T) {
		assert := assert.New(t)

		opts := append(opts,
			grpc.WithInsecure(),
			grpc.WithUnaryInterceptor(ClientAuth(clientKey)),
		)
		conn, err := grpc.DialContext(context.Background(), addr, opts...)
		if err != nil {
			t.Fatal(err)
		}
		client := NewNodeClient(conn)

		// SyncEvents() rpc
		id1, ctx := ServerPeerID(nil)
		_, err = client.SyncEvents(ctx, &KnownEvents{})
		if !assert.NoError(err) {
			return
		}
		if !assert.Equal(serverID, *id1) {
			return
		}

		// GetEvent() rpc
		id2, ctx := ServerPeerID(nil)
		_, err = client.GetEvent(ctx, &EventRequest{})
		if !assert.NoError(err) {
			return
		}
		if !assert.Equal(serverID, *id2) {
			return
		}

		// GetPeerInfo() rpc
		id3, ctx := ServerPeerID(nil)
		_, err = client.GetPeerInfo(ctx, &PeerRequest{})
		if !assert.NoError(err) {
			return
		}
		if !assert.Equal(serverID, *id3) {
			return
		}
	})

	t.Run("unauthorized client", func(t *testing.T) {
		assert := assert.New(t)

		opts := append(opts,
			grpc.WithInsecure(),
		)
		conn, err := grpc.DialContext(context.Background(), addr, opts...)
		if err != nil {
			t.Fatal(err)
		}
		client := NewNodeClient(conn)

		// SyncEvents() rpc
		id1, ctx := ServerPeerID(nil)
		_, err = client.SyncEvents(ctx, &KnownEvents{})
		if !assert.Error(err) {
			return
		}
		if !assert.Equal(hash.EmptyPeer, *id1) {
			return
		}

		// GetEvent() rpc
		id2, ctx := ServerPeerID(nil)
		_, err = client.GetEvent(ctx, &EventRequest{})
		if !assert.Error(err) {
			return
		}
		if !assert.Equal(hash.EmptyPeer, *id2) {
			return
		}

		// GetPeerInfo() rpc
		id3, ctx := ServerPeerID(nil)
		_, err = client.GetPeerInfo(ctx, &PeerRequest{})
		if !assert.Error(err) {
			return
		}
		if !assert.Equal(hash.EmptyPeer, *id3) {
			return
		}
	})
}

func testGRPCWithoutAuthServer(t *testing.T, bind, from string, listen network.ListenFunc, opts ...grpc.DialOption) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// keys
	clientKey := crypto.GenerateKey()

	panicMsg := "GrpcPeerID should be called from gRPC handler only"

	// service with panic handler
	svc := NewMockNodeServer(ctrl)
	svc.EXPECT().
		SyncEvents(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *KnownEvents) (*KnownEvents, error) {
			assert.Equal(t, from, GrpcPeerHost(ctx))
			assert.PanicsWithValue(t, panicMsg, func() { GrpcPeerID(ctx) })
			return &KnownEvents{}, nil
		}).
		AnyTimes()
	svc.EXPECT().
		GetEvent(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *EventRequest) (*wire.Event, error) {
			assert.Equal(t, from, GrpcPeerHost(ctx))
			assert.PanicsWithValue(t, panicMsg, func() { GrpcPeerID(ctx) })
			return &wire.Event{}, nil
		}).
		AnyTimes()
	svc.EXPECT().
		GetPeerInfo(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, req *PeerRequest) (*PeerInfo, error) {
			assert.Equal(t, from, GrpcPeerHost(ctx))
			assert.PanicsWithValue(t, panicMsg, func() { GrpcPeerID(ctx) })
			return &PeerInfo{}, nil
		}).
		AnyTimes()

	t.Run("unauthorized server", func(t *testing.T) {
		assert := assert.New(t)

		// Server
		server := grpc.NewServer(
			grpc.MaxRecvMsgSize(math.MaxInt32),
			grpc.MaxSendMsgSize(math.MaxInt32))
		RegisterNodeServer(server, svc)

		listener := listen(bind)

		go func() {
			if err := server.Serve(listener); err != nil {
				t.Fatal(err)
			}
		}()

		addr := listener.Addr().String()

		defer server.Stop()

		// Client
		opts := append(opts,
			grpc.WithInsecure(),
			grpc.WithUnaryInterceptor(ClientAuth(clientKey)),
		)
		conn, err := grpc.DialContext(context.Background(), addr, opts...)
		if err != nil {
			t.Fatal(err)
		}
		client := NewNodeClient(conn)

		// SyncEvents() rpc
		id1, ctx := ServerPeerID(nil)
		_, err = client.SyncEvents(ctx, &KnownEvents{})
		if !assert.Error(err) {
			return
		}
		if !assert.Equal(hash.EmptyPeer, *id1) {
			return
		}

		// GetEvent() rpc
		id2, ctx := ServerPeerID(nil)
		_, err = client.GetEvent(ctx, &EventRequest{})
		if !assert.Error(err) {
			return
		}
		if !assert.Equal(hash.EmptyPeer, *id2) {
			return
		}

		// GetPeerInfo() rpc
		id3, ctx := ServerPeerID(nil)
		_, err = client.GetPeerInfo(ctx, &PeerRequest{})
		if !assert.Error(err) {
			return
		}
		if !assert.Equal(hash.EmptyPeer, *id3) {
			return
		}
	})
}
