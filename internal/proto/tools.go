package proto

type ToolResponseType string

const (
	ToolResponseTypeText  ToolResponseType = "text"
	ToolResponseTypeImage ToolResponseType = "image"
)

type ToolResponse struct {
	Type     ToolResponseType `json:"type"`
	Content  string           `json:"content"`
	Metadata string           `json:"metadata,omitempty"`
	IsError  bool             `json:"is_error"`
}

const (
	BashToolName = "bash"
)

type BashParams struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type BashPermissionsParams struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
}

type BashResponseMetadata struct {
	StartTime        int64  `json:"start_time"`
	EndTime          int64  `json:"end_time"`
	Output           string `json:"output"`
	WorkingDirectory string `json:"working_directory"`
}

type DiagnosticsParams struct {
	FilePath string `json:"file_path"`
}

const DownloadToolName = "download"

type DownloadParams struct {
	URL      string `json:"url"`
	FilePath string `json:"file_path"`
	Timeout  int    `json:"timeout,omitempty"`
}

type DownloadPermissionsParams struct {
	URL      string `json:"url"`
	FilePath string `json:"file_path"`
	Timeout  int    `json:"timeout,omitempty"`
}

const EditToolName = "edit"

type EditParams struct {
	FilePath   string `json:"file_path"`
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

type EditPermissionsParams struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

type EditResponseMetadata struct {
	Additions  int    `json:"additions"`
	Removals   int    `json:"removals"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

const FetchToolName = "fetch"

type FetchParams struct {
	URL     string `json:"url"`
	Format  string `json:"format"`
	Timeout int    `json:"timeout,omitempty"`
}

type FetchPermissionsParams struct {
	URL     string `json:"url"`
	Format  string `json:"format"`
	Timeout int    `json:"timeout,omitempty"`
}

const GlobToolName = "glob"

type GlobParams struct {
	Pattern string `json:"pattern"`
	Path    string `json:"path"`
}

type GlobResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

const GrepToolName = "grep"

type GrepParams struct {
	Pattern     string `json:"pattern"`
	Path        string `json:"path"`
	Include     string `json:"include"`
	LiteralText bool   `json:"literal_text"`
}
type GrepResponseMetadata struct {
	NumberOfMatches int  `json:"number_of_matches"`
	Truncated       bool `json:"truncated"`
}

const LSToolName = "ls"

type LSParams struct {
	Path   string   `json:"path"`
	Ignore []string `json:"ignore"`
}

type LSPermissionsParams struct {
	Path   string   `json:"path"`
	Ignore []string `json:"ignore"`
}

type TreeNode struct {
	Name     string      `json:"name"`
	Path     string      `json:"path"`
	Type     string      `json:"type"` // "file" or "directory"
	Children []*TreeNode `json:"children,omitempty"`
}

type LSResponseMetadata struct {
	NumberOfFiles int  `json:"number_of_files"`
	Truncated     bool `json:"truncated"`
}

const MultiEditToolName = "multiedit"

type MultiEditOperation struct {
	OldString  string `json:"old_string"`
	NewString  string `json:"new_string"`
	ReplaceAll bool   `json:"replace_all,omitempty"`
}

type MultiEditParams struct {
	FilePath string               `json:"file_path"`
	Edits    []MultiEditOperation `json:"edits"`
}

type MultiEditPermissionsParams struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}

type MultiEditResponseMetadata struct {
	Additions    int    `json:"additions"`
	Removals     int    `json:"removals"`
	OldContent   string `json:"old_content,omitempty"`
	NewContent   string `json:"new_content,omitempty"`
	EditsApplied int    `json:"edits_applied"`
}

const SourcegraphToolName = "sourcegraph"

type SourcegraphParams struct {
	Query         string `json:"query"`
	Count         int    `json:"count,omitempty"`
	ContextWindow int    `json:"context_window,omitempty"`
	Timeout       int    `json:"timeout,omitempty"`
}

type SourcegraphResponseMetadata struct {
	NumberOfMatches int  `json:"number_of_matches"`
	Truncated       bool `json:"truncated"`
}

const ViewToolName = "view"

type ViewParams struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}

type ViewPermissionsParams struct {
	FilePath string `json:"file_path"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}
type ViewResponseMetadata struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

const WriteToolName = "write"

type WriteParams struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content"`
}

type WritePermissionsParams struct {
	FilePath   string `json:"file_path"`
	OldContent string `json:"old_content,omitempty"`
	NewContent string `json:"new_content,omitempty"`
}
type WriteResponseMetadata struct {
	Diff      string `json:"diff"`
	Additions int    `json:"additions"`
	Removals  int    `json:"removals"`
}
