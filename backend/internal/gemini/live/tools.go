package live

import (
	"google.golang.org/genai"
)

// ShowWorkingFunctionDeclaration lets the model show a whole board's worth of
// typed working - prose, math, and/or code lines - in sync with what it's
// saying. Called once per board (potentially several times across a longer
// spoken explanation), not once per line.
var ShowWorkingFunctionDeclaration = &genai.FunctionDeclaration{
	Name:        "show_working",
	Description: "Show a board's worth of typed working - prose, math, and/or code lines - in sync with what you're saying.",
	Parameters: &genai.Schema{
		Type:     genai.TypeObject,
		Required: []string{"lines"},
		Properties: map[string]*genai.Schema{
			"title": {Type: genai.TypeString},
			"lines": {Type: genai.TypeArray, Items: &genai.Schema{Type: genai.TypeString}},
		},
	},
}

// DrawDiagramFunctionDeclaration lets the model draw a Mermaid diagram - only
// for cycles, branches, or sequences, never a numeric graph/plot (Mermaid
// can't render axes or plotted data meaningfully).
var DrawDiagramFunctionDeclaration = &genai.FunctionDeclaration{
	Name:        "draw_diagram",
	Description: "Draw a Mermaid diagram for a cycle, branch, or sequence - never for a numeric graph/plot.",
	Parameters: &genai.Schema{
		Type:     genai.TypeObject,
		Required: []string{"mermaid"},
		Properties: map[string]*genai.Schema{
			"title":   {Type: genai.TypeString},
			"mermaid": {Type: genai.TypeString},
		},
	},
}

var Tools = []*genai.Tool{
	{FunctionDeclarations: []*genai.FunctionDeclaration{
		ShowWorkingFunctionDeclaration,
		DrawDiagramFunctionDeclaration,
	}},
}
