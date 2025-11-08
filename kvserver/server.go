package kvserver

import (
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"

	"github.com/praveenkullu/dsdemo/viewservice"
)

const (
	PingInterval = 500 * time.Millisecond // Ping viewservice every 0.5 seconds
)

// KVServer is a key-value server that can act as Primary or Backup
type KVServer struct {
	mu   sync.Mutex
	l    net.Listener
	dead bool
	me   string // my server name/address

	vsAddress string // view service address
	vsClient  *rpc.Client

	currentView  viewservice.View
	data         map[string]string
	role         string // "primary", "backup", or "idle"
	lastBackup   string // last known backup address
	syncing      bool   // true when state transfer is in progress
	pendingQueue []PutArgs // queue for puts during state transfer
}

// StartServer creates and starts a new KV server
func StartServer(serverName string, vsAddress string) *KVServer {
	kv := &KVServer{
		me:           serverName,
		vsAddress:    vsAddress,
		data:         make(map[string]string),
		role:         "idle",
		lastBackup:   "",
		syncing:      false,
		pendingQueue: make([]PutArgs, 0),
	}

	// Register RPC service
	rpcs := rpc.NewServer()
	rpcs.Register(kv)

	// Start listening
	l, err := net.Listen("tcp", serverName)
	if err != nil {
		log.Fatal("KVServer listen error:", err)
	}
	kv.l = l

	// Start RPC server
	go func() {
		for !kv.dead {
			conn, err := kv.l.Accept()
			if err == nil && !kv.dead {
				go rpcs.ServeConn(conn)
			} else if err != nil && !kv.dead {
				log.Printf("KVServer accept error: %v\n", err)
			}
		}
	}()

	// Connect to view service
	go kv.connectToViewService()

	// Start pinging view service
	go kv.pingLoop()

	log.Printf("KVServer %s started\n", serverName)
	return kv
}

// connectToViewService establishes connection to view service
func (kv *KVServer) connectToViewService() {
	for !kv.dead {
		client, err := rpc.Dial("tcp", kv.vsAddress)
		if err == nil {
			kv.mu.Lock()
			kv.vsClient = client
			kv.mu.Unlock()
			log.Printf("Connected to view service at %s\n", kv.vsAddress)
			return
		}
		time.Sleep(1 * time.Second)
	}
}

// pingLoop periodically pings the view service
func (kv *KVServer) pingLoop() {
	ticker := time.NewTicker(PingInterval)
	defer ticker.Stop()

	for !kv.dead {
		<-ticker.C
		kv.ping()
	}
}

// ping sends a ping to the view service and updates the view
func (kv *KVServer) ping() {
	kv.mu.Lock()
	if kv.vsClient == nil {
		kv.mu.Unlock()
		return
	}

	args := &viewservice.PingArgs{
		ServerName: kv.me,
		ViewNumber: kv.currentView.ViewNumber,
	}
	reply := &viewservice.PingReply{}
	client := kv.vsClient
	kv.mu.Unlock()

	err := client.Call("ViewServer.Ping", args, reply)
	if err != nil {
		log.Printf("Ping error: %v\n", err)
		return
	}

	kv.mu.Lock()
	defer kv.mu.Unlock()

	oldView := kv.currentView
	kv.currentView = reply.View

	// Check if view has changed
	if oldView.ViewNumber != kv.currentView.ViewNumber {
		kv.handleViewChange(oldView)
	}
}

// handleViewChange handles changes in the view
func (kv *KVServer) handleViewChange(oldView viewservice.View) {
	log.Printf("View changed from %d to %d (Primary: %s, Backup: %s)\n",
		oldView.ViewNumber, kv.currentView.ViewNumber,
		kv.currentView.Primary, kv.currentView.Backup)

	oldRole := kv.role

	// Determine new role
	if kv.currentView.Primary == kv.me {
		kv.role = "primary"
	} else if kv.currentView.Backup == kv.me {
		kv.role = "backup"
	} else {
		kv.role = "idle"
	}

	if oldRole != kv.role {
		log.Printf("Role changed from %s to %s\n", oldRole, kv.role)
	}

	// If I became primary or if backup changed, handle state transfer
	if kv.role == "primary" {
		// Check if backup changed
		if kv.currentView.Backup != "" && kv.currentView.Backup != kv.lastBackup {
			log.Printf("New backup detected: %s, initiating state transfer\n", kv.currentView.Backup)
			kv.lastBackup = kv.currentView.Backup
			go kv.transferState(kv.currentView.Backup, kv.currentView.ViewNumber)
		} else if kv.currentView.Backup == "" {
			kv.lastBackup = ""
		}
	}
}

// transferState transfers the entire state to the new backup
func (kv *KVServer) transferState(backup string, viewNumber uint64) {
	kv.mu.Lock()
	kv.syncing = true
	dataCopy := make(map[string]string)
	for k, v := range kv.data {
		dataCopy[k] = v
	}
	kv.mu.Unlock()

	log.Printf("Transferring state to backup %s (view %d)\n", backup, viewNumber)

	// Connect to backup
	client, err := rpc.Dial("tcp", backup)
	if err != nil {
		log.Printf("Failed to connect to backup %s: %v\n", backup, err)
		kv.mu.Lock()
		kv.syncing = false
		kv.mu.Unlock()
		return
	}
	defer client.Close()

	// Send state
	args := &SyncStateArgs{
		Data:       dataCopy,
		ViewNumber: viewNumber,
	}
	reply := &SyncStateReply{}

	err = client.Call("KVServer.SyncState", args, reply)
	if err != nil {
		log.Printf("SyncState RPC failed: %v\n", err)
		kv.mu.Lock()
		kv.syncing = false
		kv.mu.Unlock()
		return
	}

	log.Printf("State transfer completed successfully\n")

	kv.mu.Lock()
	kv.syncing = false

	// Process pending puts
	if len(kv.pendingQueue) > 0 {
		log.Printf("Processing %d pending puts\n", len(kv.pendingQueue))
		pending := kv.pendingQueue
		kv.pendingQueue = make([]PutArgs, 0)
		kv.mu.Unlock()

		for _, putArgs := range pending {
			reply := &PutReply{}
			kv.Put(&putArgs, reply)
		}
	} else {
		kv.mu.Unlock()
	}
}

// Get RPC handler
func (kv *KVServer) Get(args *GetArgs, reply *GetReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if kv.role != "primary" {
		reply.Err = ErrNotPrimary
		return nil
	}

	value, ok := kv.data[args.Key]
	if ok {
		reply.Value = value
		reply.Err = OK
	} else {
		reply.Err = ErrNoKey
	}

	return nil
}

// Put RPC handler
func (kv *KVServer) Put(args *PutArgs, reply *PutReply) error {
	kv.mu.Lock()

	if kv.role != "primary" {
		kv.mu.Unlock()
		reply.Err = ErrNotPrimary
		return nil
	}

	// If state transfer is in progress, queue the request
	if kv.syncing {
		kv.pendingQueue = append(kv.pendingQueue, *args)
		kv.mu.Unlock()
		reply.Err = OK
		return nil
	}

	backup := kv.currentView.Backup
	kv.mu.Unlock()

	// If there's a backup, forward the update
	if backup != "" {
		client, err := rpc.Dial("tcp", backup)
		if err != nil {
			log.Printf("Failed to connect to backup %s: %v\n", backup, err)
			// Continue anyway, update local state
		} else {
			defer client.Close()

			forwardArgs := &ForwardUpdateArgs{
				Key:   args.Key,
				Value: args.Value,
			}
			forwardReply := &ForwardUpdateReply{}

			err = client.Call("KVServer.ForwardUpdate", forwardArgs, forwardReply)
			if err != nil {
				log.Printf("ForwardUpdate RPC failed: %v\n", err)
				// Continue anyway, update local state
			}
		}
	}

	// Update local state
	kv.mu.Lock()
	kv.data[args.Key] = args.Value
	kv.mu.Unlock()

	reply.Err = OK
	return nil
}

// ForwardUpdate RPC handler (called by Primary on Backup)
func (kv *KVServer) ForwardUpdate(args *ForwardUpdateArgs, reply *ForwardUpdateReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	if kv.role != "backup" {
		reply.Err = ErrNotPrimary
		return nil
	}

	kv.data[args.Key] = args.Value
	reply.Err = OK
	return nil
}

// SyncState RPC handler (called by Primary on new Backup for state transfer)
func (kv *KVServer) SyncState(args *SyncStateArgs, reply *SyncStateReply) error {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	log.Printf("Receiving state transfer: %d keys\n", len(args.Data))

	// Overwrite local state
	kv.data = make(map[string]string)
	for k, v := range args.Data {
		kv.data[k] = v
	}

	reply.Err = OK
	return nil
}

// Kill shuts down the server
func (kv *KVServer) Kill() {
	kv.dead = true
	if kv.l != nil {
		kv.l.Close()
	}
	if kv.vsClient != nil {
		kv.vsClient.Close()
	}
}
