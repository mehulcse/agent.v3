### Data format

GRPC is used for calls between agent and integrations. Endpoints and parameters are defined using .proto files.

Integrations are responsible for getting the data and converting it to pinpoint format. Integrations will use datamodel directly. Agent itself does not need to check the datamodel. Agent will use the metadata to correctly forward that data to backend, but does not have to touch the data itself.

### Export code flow

When agent export command is called, agent loads all available/configured plugins and then inits them using the Init call to allow them to call back to the agent.

After that agent calls Export methods on integrations in parallel. Integration marks the export state for each model type using ExportStarted and ExportDone. It sends the data using SendExported call.

### Agent RPC interface

[See in code](https://github.com/pinpt/agent.next/blob/master/rpcdef/agent.go)

```golang
type Agent interface {

// ExportStarted should be called when starting export for each modelType.
// It returns session id to be used later when sending objects.
ExportStarted(modelType string) 
	(sessionID string,
	lastProcessed interface{})

// ExportDone should be called when export of a certain modelType is complete.
ExportDone(sessionID string, lastProcessed interface{})

// SendExported forwards the exported objects from intergration to agent,
// which then uploads the data (or queues for uploading).
SendExported(
		sessionID string,
		objs []ExportObj)

// Integration can ask agent to download and process git repo using ripsrc.
ExportGitRepo(fetch GitRepoFetch)

}

```


### Integration RPC interface

[See in code](https://github.com/pinpt/agent.next/blob/master/rpcdef/integration.go)

```golang
type Integration interface {

// Init provides the connection details for connecting back to agent.
Init(connectionDetails)

// Export starts export of all data types for this integration.
// Config contains typed config common for all integrations and map[string]interface{} for custom fields.
Export(ctx context.Context,
	agentConfig ExportAgentConfig,
	Export(context.Context, ExportConfig) (ExportResult, error)
}

type ExportConfig struct {
	Pinpoint    ExportConfigPinpoint
	Integration map[string]interface{}
}

type ExportConfigPinpoint struct {
	CustomerID string
}

type ExportResult struct {
	// NewConfig can be returned from Export to update the integration config. Return nil to keep the curren config.
	NewConfig map[string]interface{}
}
```

