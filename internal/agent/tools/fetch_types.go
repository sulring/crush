package tools

// FetchToolName is the name of the fetch tool.
const FetchToolName = "fetch"

// WebFetchToolName is the name of the web_fetch tool.
const WebFetchToolName = "web_fetch"

// LargeContentThreshold is the size threshold for saving content to a file.
const LargeContentThreshold = 50000 // 50KB

// FetchParams defines the parameters for the fetch tool.
type FetchParams struct {
	URL    string `json:"url" description:"The URL to fetch content from"`
	Prompt string `json:"prompt" description:"The prompt to run on the fetched content"`
}

// FetchPermissionsParams defines the permission parameters for the fetch tool.
type FetchPermissionsParams struct {
	URL    string `json:"url"`
	Prompt string `json:"prompt"`
}

// WebFetchParams defines the parameters for the web_fetch tool.
type WebFetchParams struct {
	URL string `json:"url" description:"The URL to fetch content from"`
}
