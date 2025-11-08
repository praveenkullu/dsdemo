package kvserver

// GetArgs is the argument for Get RPC
type GetArgs struct {
	Key string
}

// GetReply is the reply for Get RPC
type GetReply struct {
	Value string
	Err   string // "" for success, "ErrNoKey" if key doesn't exist, "ErrNotPrimary" if server is not primary
}

// PutArgs is the argument for Put RPC
type PutArgs struct {
	Key   string
	Value string
}

// PutReply is the reply for Put RPC
type PutReply struct {
	Err string // "" for success, "ErrNotPrimary" if server is not primary
}

// ForwardUpdateArgs is the argument for ForwardUpdate RPC (Primary -> Backup)
type ForwardUpdateArgs struct {
	Key   string
	Value string
}

// ForwardUpdateReply is the reply for ForwardUpdate RPC
type ForwardUpdateReply struct {
	Err string
}

// SyncStateArgs is the argument for SyncState RPC (state transfer)
type SyncStateArgs struct {
	Data       map[string]string
	ViewNumber uint64
}

// SyncStateReply is the reply for SyncState RPC
type SyncStateReply struct {
	Err string
}

// Error constants
const (
	OK            = ""
	ErrNoKey      = "ErrNoKey"
	ErrNotPrimary = "ErrNotPrimary"
)
