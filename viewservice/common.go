package viewservice

// View represents the current system configuration
type View struct {
	ViewNumber uint64 // Increments every time the view changes
	Primary    string // Address of the primary server
	Backup     string // Address of the backup server (can be empty)
}

// PingArgs is the argument for Ping RPC
type PingArgs struct {
	ServerName string // Name/address of the server sending ping
	ViewNumber uint64 // The view number the server currently knows
}

// PingReply is the reply for Ping RPC
type PingReply struct {
	View View
}

// GetViewArgs is the argument for GetView RPC
type GetViewArgs struct{}

// GetViewReply is the reply for GetView RPC
type GetViewReply struct {
	View View
}
