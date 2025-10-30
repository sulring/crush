# Hooks Guide

⚠️ **Security Warning**: Hooks run automatically with your user's permissions and have full access to your filesystem and environment. Only configure hooks from trusted sources and review all commands before adding them.

Hooks are user-defined shell commands that execute at various points in Crush's lifecycle. They provide deterministic control over Crush's behavior, ensuring certain actions always occur rather than relying on the LLM to choose to run them.

## Hook Events

Crush provides several lifecycle events where hooks can run:

### Tool Events
- **`PreToolUse`**: Runs before tool calls. If a hook fails (non-zero exit code), the tool execution is blocked.
- **`PostToolUse`**: Runs after tool calls complete, can be used to process results or trigger actions.

### Session Events
- **`UserPromptSubmit`**: Runs when the user submits a prompt, before processing
- **`Stop`**: Runs when Crush finishes responding to a prompt
- **`SubagentStop`**: Runs when subagent tasks complete (e.g., fetch tool, agent tool)
- **`SessionStart`**: Runs when a session starts or resumes
- **`SessionEnd`**: Runs when a session ends

### Other Events
- **`Notification`**: Runs when Crush sends notifications
- **`PreCompact`**: Runs before running a compact operation

## Configuration Format

Hooks are configured in your Crush configuration file (e.g., `crush.json` or `~/.crush/crush.json`):

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "bash",
        "hooks": [
          {
            "type": "command",
            "command": "jq -r '.tool_name + \": \" + .tool_input.command' >> ~/crush-commands.log",
            "timeout": 5
          }
        ]
      }
    ],
    "PostToolUse": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "echo \"Tool $(jq -r .tool_name) completed\" | notify-send \"Crush Hook\""
          }
        ]
      }
    ],
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "echo \"Prompt completed. Tokens used: $(jq -r .tokens_used)\""
          }
        ]
      }
    ]
  }
}
```

## Hook Context

Each hook receives a JSON context object via stdin containing information about the event:

```json
{
  "event_type": "PreToolUse",
  "session_id": "abc123",
  "tool_name": "bash",
  "tool_input": {
    "command": "echo hello",
    "description": "Print hello"
  },
  "tool_result": "",
  "tool_error": false,
  "user_prompt": "",
  "timestamp": "2025-10-30T12:00:00Z",
  "working_dir": "/path/to/project",
  "message_id": "msg123",
  "provider": "anthropic",
  "model": "claude-3-5-sonnet-20241022",
  "tokens_used": 1000,
  "tokens_input": 500
}
```

### Context Fields by Event Type

Different events include different fields:

- **PreToolUse**: `event_type`, `session_id`, `tool_name`, `tool_input`, `message_id`, `provider`, `model`
- **PostToolUse**: `event_type`, `session_id`, `tool_name`, `tool_input`, `tool_result`, `tool_error`, `message_id`, `provider`, `model`
- **UserPromptSubmit**: `event_type`, `session_id`, `user_prompt`, `provider`, `model`
- **Stop**: `event_type`, `session_id`, `message_id`, `provider`, `model`, `tokens_used`, `tokens_input`

All events include: `event_type`, `timestamp`, `working_dir`

## Environment Variables

Hooks also receive environment variables:

- `CRUSH_HOOK_CONTEXT`: Full JSON context as a string
- `CRUSH_HOOK_EVENT`: The event type (e.g., "PreToolUse")
- `CRUSH_SESSION_ID`: The session ID (if applicable)
- `CRUSH_TOOL_NAME`: The tool name (for tool events)

## Hook Configuration

### Matchers

For tool events (`PreToolUse`, `PostToolUse`), you can specify matchers to target specific tools:

- `"bash"` - Only matches the bash tool
- `"edit"` - Only matches the edit tool
- `"*"` or `""` - Matches all tools

For non-tool events, leave the matcher empty or use `"*"`.

### Hook Properties

- `type`: Currently only `"command"` is supported
- `command`: The shell command to execute
- `timeout`: (optional) Maximum execution time in seconds (default: 30, max: 300)

## Examples

### Log All Bash Commands

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "bash",
        "hooks": [
          {
            "type": "command",
            "command": "jq -r '.timestamp + \" - \" + .tool_input.command' >> ~/.crush/bash-log.txt"
          }
        ]
      }
    ]
  }
}
```

### Auto-format Files After Editing

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "edit",
        "hooks": [
          {
            "type": "command",
            "command": "jq -r .tool_input.file_path | xargs prettier --write"
          }
        ]
      }
    ]
  }
}
```

### Notify on Completion

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "osascript -e 'display notification \"Crush completed\" with title \"Crush\"'"
          }
        ]
      }
    ]
  }
}
```

### Track Token Usage

```json
{
  "hooks": {
    "Stop": [
      {
        "hooks": [
          {
            "type": "command",
            "command": "jq -r '\"\\(.timestamp): \\(.tokens_used) tokens\"' >> ~/.crush/token-usage.log"
          }
        ]
      }
    ]
  }
}
```

### Validate Tool Usage

```json
{
  "hooks": {
    "PreToolUse": [
      {
        "matcher": "bash",
        "hooks": [
          {
            "type": "command",
            "command": "if jq -e '.tool_input.command | contains(\"rm -rf\")' > /dev/null; then echo \"Dangerous command detected\" >&2; exit 1; fi"
          }
        ]
      }
    ]
  }
}
```

### Multiple Hooks

You can execute multiple hooks for the same event:

```json
{
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "*",
        "hooks": [
          {
            "type": "command",
            "command": "jq -r .tool_name >> ~/.crush/tool-usage.log"
          },
          {
            "type": "command",
            "command": "if jq -e .tool_error > /dev/null; then echo 'Error in tool' | pbcopy; fi"
          }
        ]
      }
    ]
  }
}
```

## Best Practices

1. **Keep hooks fast**: Hooks run synchronously and can slow down Crush if they take too long.
2. **Set appropriate timeouts**: Use shorter timeouts (1-5 seconds) for quick operations.
3. **Handle errors gracefully**: Hooks should not crash or hang.
4. **Use jq for JSON processing**: The context is piped to stdin as JSON.
5. **Test hooks independently**: Run your shell commands manually with test data before configuring them.
6. **Use absolute paths**: Hooks run in the project directory, but absolute paths are more reliable.
7. **Consider privacy**: Don't log sensitive information like API keys or passwords.

## Debugging Hooks

Hooks log errors and warnings to Crush's log output. To see hook execution:

1. Run Crush with debug logging enabled: `crush --debug`
2. Check the logs for hook-related messages
3. Test hook shell commands manually:
   ```bash
   echo '{"event_type":"PreToolUse","tool_name":"bash"}' | jq -r '.tool_name'
   ```

## Limitations

- Hooks must complete within their timeout (default 30 seconds)
- Hooks run in a shell environment and require shell utilities (bash, jq, etc.)
- Hooks cannot modify Crush's internal state
- Hook errors are logged but don't stop Crush execution (except for PreToolUse)
- Interactive hooks are not supported
