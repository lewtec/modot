package codec

import (
	"fmt"

	"github.com/owenrumney/go-sarif/v2/sarif"
)

const shellcheckInfoURI = "https://github.com/koalaman/shellcheck"

type shellcheckIssue struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	EndLine   int    `json:"endLine"`
	Column    int    `json:"column"`
	EndColumn int    `json:"endColumn"`
	Level     string `json:"level"`
	Code      int    `json:"code"`
	Message   string `json:"message"`
}

func decodeShellcheck(toolName string, data []byte) (*sarif.Run, error) {
	issues, err := decodeJSONArray[shellcheckIssue](data, "shellcheck")
	if err != nil || len(issues) == 0 {
		return nil, err
	}
	run := newNamedRun(toolName, "shellcheck", shellcheckInfoURI)
	for _, issue := range issues {
		level := "warning"
		switch issue.Level {
		case "error":
			level = "error"
		case "info", "style":
			level = "note"
		}
		run.AddResult(
			sarif.NewRuleResult(fmt.Sprintf("SC%d", issue.Code)).
				WithLevel(level).
				WithMessage(sarif.NewTextMessage(issue.Message)).
				WithLocations([]*sarif.Location{
					resultLocation(issue.File, issue.Line, issue.Column, issue.EndLine, issue.EndColumn),
				}),
		)
	}
	return run, nil
}
