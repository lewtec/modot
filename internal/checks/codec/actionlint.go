package codec

import (
	"github.com/owenrumney/go-sarif/v2/sarif"
)

const actionlintInfoURI = "https://github.com/rhysd/actionlint"

type actionlintIssue struct {
	Message   string `json:"message"`
	Filepath  string `json:"filepath"`
	Line      int    `json:"line"`
	Column    int    `json:"column"`
	Kind      string `json:"kind"`
	EndColumn int    `json:"end_column,omitempty"`
}

func decodeActionlint(toolName string, data []byte) (*sarif.Run, error) {
	issues, err := decodeJSONArray[actionlintIssue](data, "actionlint")
	if err != nil || len(issues) == 0 {
		return nil, err
	}
	run := newNamedRun(toolName, "actionlint", actionlintInfoURI)
	for _, issue := range issues {
		run.AddResult(
			sarif.NewRuleResult(issue.Kind).
				WithLevel("error").
				WithMessage(sarif.NewTextMessage(issue.Message)).
				WithLocations([]*sarif.Location{
					resultLocation(issue.Filepath, issue.Line, issue.Column, 0, issue.EndColumn),
				}),
		)
	}
	return run, nil
}
