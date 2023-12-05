package client

import (
	"context"
	"fmt"
	"time"

	counterpb "github.com/dinoallo/sealos-networkmanager-synchronizer/client/proto/agent"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const defaultNMAgentPort = "50051"

type Client struct {
	nodeIP  string
	conn    *grpc.ClientConn
	closing bool
	// After calling NewClient, this logger will have the value of nodeIP
	logger logr.Logger
}

func (c *Client) Close() {
	if !c.closing {
		c.closing = true
		if c.conn != nil {
			c.conn.Close()
		}
	}
}

func NewClient(nodeIP string) (*Client, error) {
	c := &Client{
		nodeIP: nodeIP,
		logger: log.Log.WithName("NMAgentClient").WithValues("nodeIP", nodeIP),
	}
	address := fmt.Sprintf("%s:%s", nodeIP, defaultNMAgentPort)
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithKeepaliveParams(
		keepalive.ClientParameters{
			Time:    time.Minute,
			Timeout: 5 * time.Second,
		}),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("connect on %s failed: %s", c.nodeIP, err)
	}
	c.conn = conn
	c.closing = false
	return c, nil
}

func (c *Client) DumpTraffic(ctx context.Context, addr string, tagToSync string, reset_ bool) (*counterpb.DumpTrafficResponse, error) {
	csc := counterpb.NewCountingServiceClient(c.conn)
	req := &counterpb.DumpTrafficRequest{
		Address: addr,
		Tag:     tagToSync,
		Reset_:  reset_,
	}

	if resp, err := csc.DumpTraffic(ctx, req); err != nil {
		return nil, err
	} else {
		return resp, nil
	}
}

func (c *Client) Subscribe(ctx context.Context, addr string, port uint32) error {
	csc := counterpb.NewCountingServiceClient(c.conn)
	req := &counterpb.SubscribeRequest{
		Address: addr,
		Port:    port,
	}
	if _, err := csc.Subscribe(ctx, req); err != nil {
		return err
	}
	return nil
}
