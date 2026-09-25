package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/patriceckhart/zot/packages/agent/ext"
)

const extensionName = "zot-autoresearch"

var version = "0.1.0"

func main() {
	e := ext.New(extensionName, version)
	a := newApp(e)

	e.OnHello(func(host ext.HostInfo) {
		a.cwd = host.CWD
		a.dataDir = host.DataDir
		if a.dataDir == "" {
			a.dataDir = host.ExtensionDir
		}
		if err := a.loadState(); err != nil {
			a.logf("load state: %v", err)
		}
	})
	e.Command("autoresearch", "run autonomous benchmark-driven research (/autoresearch help)", a.command)
	e.Tool("autoresearch_experiment", "Run the configured benchmark, compare its score, and accept or reject the current experiment.", experimentSchema(), a.experimentTool)
	e.OnPanelKey(panelID, a.panelKey, func() { a.setPanelOpen(false) })

	if err := e.Run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func experimentSchema() json.RawMessage {
	return json.RawMessage(`{
  "type": "object",
  "properties": {
    "hypothesis": {"type": "string", "description": "The idea tested by this iteration."},
    "summary": {"type": "string", "description": "A short imperative description used in accepted commit messages."}
  },
  "required": ["hypothesis", "summary"],
  "additionalProperties": false
}`)
}
