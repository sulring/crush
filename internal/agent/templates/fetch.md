Fetches content from a specified URL and processes it using an AI model.

<usage>
- Takes a URL and a prompt as input
- Fetches the URL content, converts HTML to markdown
- Processes the content with the prompt using a small, fast model
- Returns the model's response about the content
- Use this tool when you need to retrieve and analyze web content
</usage>

<usage_notes>

- IMPORTANT: If an MCP-provided web fetch tool is available, prefer using that tool instead of this one, as it may have fewer restrictions. All MCP-provided tools start with "mcp_".
- The URL must be a fully-formed valid URL
- HTTP URLs will be automatically upgraded to HTTPS
- The prompt should describe what information you want to extract from the page
- This tool is read-only and does not modify any files
- Results may be summarized if the content is very large
- For very large pages, the content will be saved to a temporary file and the agent will have access to grep/view tools to analyze it
- When a URL redirects to a different host, the tool will inform you and provide the redirect URL. You should then make a new fetch request with the redirect URL to fetch the content.
  </usage_notes>

<limitations>
- Max response size: 5MB
- Only supports HTTP and HTTPS protocols
- Cannot handle authentication or cookies
- Some websites may block automated requests
</limitations>

<tips>
- Be specific in your prompt about what information you want to extract
- For complex pages, ask the agent to focus on specific sections
- The agent has access to grep and view tools when analyzing large pages
</tips>
