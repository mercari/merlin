package rules

import "strings"

type Rule struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
}

type Issues []string

type EvaluationResult struct {
	Err    error
	Issues Issues
}

func (i Issues) String() string {
	return strings.Join(i, ",")
}

//
//func (e EvaluationResult) Error() string {
//	errString := fmt.Sprintf("error: %s\n", e.Err.Error())
//	if len(e.Issues) > 0 {
//		errString += fmt.Sprintf("issues:\n")
//	}
//	for i := range e.Issues {
//		errString += fmt.Sprintf("\t%v: %s", i+1, e.Issues[i])
//	}
//	return errString
//}
