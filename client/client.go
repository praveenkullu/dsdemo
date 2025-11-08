package client

import (
	"log"
	"net/rpc"
	"time"

	"github.com/praveenkullu/dsdemo/kvserver"
	"github.com/praveenkullu/dsdemo/viewservice"
)

// Client is a client for the KV service
type Client struct {
	vsAddress      string
	vsClient       *rpc.Client
	currentPrimary string
	primaryClient  *rpc.Client
}

// MakeClient creates a new client
func MakeClient(vsAddress string) *Client {
	ck := &Client{
		vsAddress:      vsAddress,
		currentPrimary: "",
	}

	// Connect to view service
	for {
		client, err := rpc.Dial("tcp", vsAddress)
		if err == nil {
			ck.vsClient = client
			log.Printf("Client connected to view service at %s\n", vsAddress)
			break
		}
		log.Printf("Failed to connect to view service, retrying...\n")
		time.Sleep(1 * time.Second)
	}

	return ck
}

// Get retrieves the value for a key
func (ck *Client) Get(key string) string {
	args := &kvserver.GetArgs{Key: key}

	for {
		// Get current primary
		if ck.currentPrimary == "" {
			ck.updatePrimary()
			if ck.currentPrimary == "" {
				time.Sleep(500 * time.Millisecond)
				continue
			}
		}

		// Try to call Get on primary
		reply := &kvserver.GetReply{}
		err := ck.call("KVServer.Get", args, reply)

		if err == nil && reply.Err == kvserver.OK {
			return reply.Value
		} else if err == nil && reply.Err == kvserver.ErrNoKey {
			return ""
		} else if err != nil || reply.Err == kvserver.ErrNotPrimary {
			// Primary changed or failed, update and retry
			log.Printf("Get failed, updating primary and retrying...\n")
			ck.currentPrimary = ""
			if ck.primaryClient != nil {
				ck.primaryClient.Close()
				ck.primaryClient = nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// Put stores a key-value pair
func (ck *Client) Put(key string, value string) {
	args := &kvserver.PutArgs{Key: key, Value: value}

	for {
		// Get current primary
		if ck.currentPrimary == "" {
			ck.updatePrimary()
			if ck.currentPrimary == "" {
				time.Sleep(500 * time.Millisecond)
				continue
			}
		}

		// Try to call Put on primary
		reply := &kvserver.PutReply{}
		err := ck.call("KVServer.Put", args, reply)

		if err == nil && reply.Err == kvserver.OK {
			return
		} else if err != nil || reply.Err == kvserver.ErrNotPrimary {
			// Primary changed or failed, update and retry
			log.Printf("Put failed, updating primary and retrying...\n")
			ck.currentPrimary = ""
			if ck.primaryClient != nil {
				ck.primaryClient.Close()
				ck.primaryClient = nil
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// updatePrimary queries the view service for the current primary
func (ck *Client) updatePrimary() {
	args := &viewservice.GetViewArgs{}
	reply := &viewservice.GetViewReply{}

	err := ck.vsClient.Call("ViewServer.GetView", args, reply)
	if err != nil {
		log.Printf("GetView failed: %v\n", err)
		return
	}

	if reply.View.Primary != "" && reply.View.Primary != ck.currentPrimary {
		ck.currentPrimary = reply.View.Primary
		if ck.primaryClient != nil {
			ck.primaryClient.Close()
		}

		// Connect to new primary
		client, err := rpc.Dial("tcp", ck.currentPrimary)
		if err != nil {
			log.Printf("Failed to connect to primary %s: %v\n", ck.currentPrimary, err)
			ck.currentPrimary = ""
			return
		}
		ck.primaryClient = client
		log.Printf("Client connected to primary %s\n", ck.currentPrimary)
	}
}

// call makes an RPC call to the primary
func (ck *Client) call(method string, args interface{}, reply interface{}) error {
	if ck.primaryClient == nil {
		return rpc.ErrShutdown
	}
	return ck.primaryClient.Call(method, args, reply)
}

// Close closes the client connections
func (ck *Client) Close() {
	if ck.vsClient != nil {
		ck.vsClient.Close()
	}
	if ck.primaryClient != nil {
		ck.primaryClient.Close()
	}
}
