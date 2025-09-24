package proto

import (
	"encoding/json"
)

type CreatePermissionRequest struct {
	SessionID   string `json:"session_id"`
	ToolCallID  string `json:"tool_call_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

type PermissionNotification struct {
	ToolCallID string `json:"tool_call_id"`
	Granted    bool   `json:"granted"`
	Denied     bool   `json:"denied"`
}

type PermissionRequest struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	ToolCallID  string `json:"tool_call_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

// UnmarshalJSON implements the json.Unmarshaler interface. This is needed
// because the Params field is of type any, so we need to unmarshal it into
// it's appropriate type based on the [PermissionRequest.ToolName].
func (p *PermissionRequest) UnmarshalJSON(data []byte) error {
	type Alias PermissionRequest
	aux := &struct {
		Params json.RawMessage `json:"params"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch p.ToolName {
	case BashToolName:
		var params BashPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case DownloadToolName:
		var params DownloadPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case EditToolName:
		var params EditPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case WriteToolName:
		var params WritePermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case MultiEditToolName:
		var params MultiEditPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case FetchToolName:
		var params FetchPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case ViewToolName:
		var params ViewPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case LSToolName:
		var params LSPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	default:
		panic("unknown tool name: " + p.ToolName)
	}
	return nil
}

// UnmarshalJSON implements the json.Unmarshaler interface. This is needed
// because the Params field is of type any, so we need to unmarshal it into
// it's appropriate type based on the [CreatePermissionRequest.ToolName].
func (p *CreatePermissionRequest) UnmarshalJSON(data []byte) error {
	type Alias CreatePermissionRequest
	aux := &struct {
		Params json.RawMessage `json:"params"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	switch p.ToolName {
	case BashToolName:
		var params BashPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case DownloadToolName:
		var params DownloadPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case EditToolName:
		var params EditPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case WriteToolName:
		var params WritePermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case MultiEditToolName:
		var params MultiEditPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case FetchToolName:
		var params FetchPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case ViewToolName:
		var params ViewPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	case LSToolName:
		var params LSPermissionsParams
		if err := json.Unmarshal(aux.Params, &params); err != nil {
			return err
		}
		p.Params = params
	default:
		panic("unknown tool name: " + p.ToolName)
	}
	return nil
}
