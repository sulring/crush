package proto

// Instance represents a running app.App instance with its associated resources
// and state.
type Instance struct {
	ID   string `json:"id"`
	Path string `json:"path"`
	YOLO bool   `json:"yolo"`
}

// Error represents an error response.
type Error struct {
	Message string `json:"message"`
}
