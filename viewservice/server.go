package viewservice

import (
	"log"
	"net"
	"net/rpc"
	"sync"
	"time"
)

const (
	PingInterval    = 500 * time.Millisecond // Servers ping every 0.5 seconds
	DeadInterval    = 1500 * time.Millisecond // Servers are declared dead after 1.5 seconds
	TickerInterval  = 500 * time.Millisecond // Ticker runs every 0.5 seconds
)

// ServerInfo tracks information about each server
type ServerInfo struct {
	Name         string
	LastPingTime time.Time
	Alive        bool
}

// ViewServer is the View Service implementation
type ViewServer struct {
	mu       sync.Mutex
	l        net.Listener
	dead     bool
	rpcCount int // for testing

	currentView View
	servers     map[string]*ServerInfo // tracks all servers that have pinged
	idleServers []string               // servers that are not primary or backup

	primaryAcked bool // primary has acknowledged the current view
}

// StartServer creates and starts a new ViewServer
func StartServer(address string) *ViewServer {
	vs := &ViewServer{
		currentView: View{
			ViewNumber: 0,
			Primary:    "",
			Backup:     "",
		},
		servers:      make(map[string]*ServerInfo),
		idleServers:  make([]string, 0),
		primaryAcked: true, // no primary initially, so considered acked
	}

	// Register RPC service
	rpcs := rpc.NewServer()
	rpcs.Register(vs)

	// Start listening
	l, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("ViewServer listen error:", err)
	}
	vs.l = l

	// Start RPC server
	go func() {
		for !vs.dead {
			conn, err := vs.l.Accept()
			if err == nil && !vs.dead {
				go rpcs.ServeConn(conn)
			} else if err != nil && !vs.dead {
				log.Printf("ViewServer accept error: %v\n", err)
			}
		}
	}()

	// Start ticker for failure detection and promotions
	go vs.ticker()

	log.Printf("ViewServer started on %s\n", address)
	return vs
}

// Ping RPC handler - called by KV servers every 0.5 seconds
func (vs *ViewServer) Ping(args *PingArgs, reply *PingReply) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.rpcCount++

	// Update server's last ping time
	if server, exists := vs.servers[args.ServerName]; exists {
		server.LastPingTime = time.Now()
		server.Alive = true
	} else {
		// New server
		vs.servers[args.ServerName] = &ServerInfo{
			Name:         args.ServerName,
			LastPingTime: time.Now(),
			Alive:        true,
		}
		// Add to idle servers if not already primary or backup
		if args.ServerName != vs.currentView.Primary && args.ServerName != vs.currentView.Backup {
			vs.idleServers = append(vs.idleServers, args.ServerName)
		}
	}

	// Check if primary has acked the current view
	if args.ServerName == vs.currentView.Primary && args.ViewNumber == vs.currentView.ViewNumber {
		vs.primaryAcked = true
	}

	// Return current view
	reply.View = vs.currentView
	return nil
}

// GetView RPC handler - called by clients to find the current primary
func (vs *ViewServer) GetView(args *GetViewArgs, reply *GetViewReply) error {
	vs.mu.Lock()
	defer vs.mu.Unlock()

	vs.rpcCount++
	reply.View = vs.currentView
	return nil
}

// ticker runs periodically to detect failures and manage promotions
func (vs *ViewServer) ticker() {
	ticker := time.NewTicker(TickerInterval)
	defer ticker.Stop()

	for !vs.dead {
		<-ticker.C
		vs.mu.Lock()
		vs.checkFailuresAndPromote()
		vs.mu.Unlock()
	}
}

// checkFailuresAndPromote detects dead servers and handles promotions
func (vs *ViewServer) checkFailuresAndPromote() {
	now := time.Now()
	viewChanged := false

	// Mark dead servers
	for name, server := range vs.servers {
		if now.Sub(server.LastPingTime) > DeadInterval {
			if server.Alive {
				server.Alive = false
				log.Printf("Server %s declared dead\n", name)
			}
		}
	}

	// Check if primary is dead
	if vs.currentView.Primary != "" {
		if server, exists := vs.servers[vs.currentView.Primary]; exists && !server.Alive {
			log.Printf("Primary %s is dead\n", vs.currentView.Primary)

			// Can only promote if primary has acked the current view
			if vs.primaryAcked && vs.currentView.Backup != "" {
				// Promote backup to primary
				backupServer, backupExists := vs.servers[vs.currentView.Backup]
				if backupExists && backupServer.Alive {
					log.Printf("Promoting backup %s to primary\n", vs.currentView.Backup)
					vs.currentView.Primary = vs.currentView.Backup
					vs.currentView.Backup = ""
					vs.currentView.ViewNumber++
					vs.primaryAcked = false
					viewChanged = true
				}
			} else if vs.primaryAcked {
				// No backup, just remove dead primary
				vs.currentView.Primary = ""
				vs.currentView.ViewNumber++
				vs.primaryAcked = true
				viewChanged = true
			}
		}
	}

	// Check if backup is dead
	if vs.currentView.Backup != "" {
		if server, exists := vs.servers[vs.currentView.Backup]; exists && !server.Alive {
			log.Printf("Backup %s is dead\n", vs.currentView.Backup)
			vs.currentView.Backup = ""
			vs.currentView.ViewNumber++
			viewChanged = true
		}
	}

	// Assign new primary if none exists
	if vs.currentView.Primary == "" && vs.primaryAcked {
		for name, server := range vs.servers {
			if server.Alive && name != vs.currentView.Backup {
				log.Printf("Assigning %s as new primary\n", name)
				vs.currentView.Primary = name
				vs.currentView.ViewNumber++
				vs.primaryAcked = false
				viewChanged = true
				vs.removeFromIdle(name)
				break
			}
		}
	}

	// Assign new backup if none exists and we have a primary
	if vs.currentView.Backup == "" && vs.currentView.Primary != "" && vs.primaryAcked {
		for name, server := range vs.servers {
			if server.Alive && name != vs.currentView.Primary {
				log.Printf("Assigning %s as new backup\n", name)
				vs.currentView.Backup = name
				vs.currentView.ViewNumber++
				viewChanged = true
				vs.removeFromIdle(name)
				break
			}
		}
	}

	if viewChanged {
		log.Printf("View changed: ViewNumber=%d, Primary=%s, Backup=%s\n",
			vs.currentView.ViewNumber, vs.currentView.Primary, vs.currentView.Backup)
	}
}

// removeFromIdle removes a server from the idle list
func (vs *ViewServer) removeFromIdle(serverName string) {
	newIdle := make([]string, 0)
	for _, name := range vs.idleServers {
		if name != serverName {
			newIdle = append(newIdle, name)
		}
	}
	vs.idleServers = newIdle
}

// Kill shuts down the server
func (vs *ViewServer) Kill() {
	vs.dead = true
	vs.l.Close()
}

// GetRPCCount returns the number of RPC calls (for testing)
func (vs *ViewServer) GetRPCCount() int {
	vs.mu.Lock()
	defer vs.mu.Unlock()
	return vs.rpcCount
}
