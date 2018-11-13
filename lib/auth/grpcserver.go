package auth

import (
	"io"
	"net/http"
	"strings"

	"github.com/gravitational/teleport"
	"github.com/gravitational/teleport/lib/auth/proto"

	"github.com/gravitational/trace"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// GRPCServer is GPRC Auth Server API
type GRPCServer struct {
	*logrus.Entry
	APIConfig
	// httpHandler is a server serving HTTP API
	httpHandler http.Handler
	// grpcHandler is golang GRPC handler
	grpcHandler *grpc.Server
}

// ConnectHeartbeat connects node or proxy to auth service
// auth service accepts a stream of events,
// nodes, send heartbeats and cluster updates back
func (g *GRPCServer) ConnectHeartbeat(stream proto.AuthService_ConnectHeartbeatServer) error {
	for {
		event, err := stream.Recv()
		if err == io.EOF {
			g.Debugf("Connection closed.")
			return nil
		}
		if err != nil {
			g.Debugf("Failed to receive event: %v", err)
			return err
		}
		g.Debugf("Received event: %v.", event)
	}
}

// NewGRPCServer returns a new instance of GRPC server
func NewGRPCServer(cfg APIConfig) http.Handler {
	authServer := &GRPCServer{
		Entry: logrus.WithFields(logrus.Fields{
			trace.Component: teleport.Component(teleport.ComponentAuth, teleport.ComponentGRPC),
		}),
		httpHandler: NewAPIServer(&cfg),
		grpcHandler: grpc.NewServer(),
	}
	proto.RegisterAuthServiceServer(authServer.grpcHandler, authServer)
	return authServer
}

// ServeHTTP dispatches requests based on the request type
func (g *GRPCServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// magic combo match signifying GRPC request
	// https://grpc.io/blog/coreos
	if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
		g.grpcHandler.ServeHTTP(w, r)
	} else {
		g.httpHandler.ServeHTTP(w, r)
	}
}
