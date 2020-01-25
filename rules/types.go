package rules

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type ResourceRules interface {
	EvaluateAll(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult
}

type Rule interface {
	Evaluate(ctx context.Context, req ctrl.Request, cli client.Client, log logr.Logger, resource interface{}) *EvaluationResult
}

type Issues []string

func (i Issues) String() string {
	return strings.Join(i, ",")
}

type EvaluationResult struct {
	Err    error
	Issues Issues
}

func (e *EvaluationResult) Combine(a *EvaluationResult) *EvaluationResult {
	if e.Err != nil && a.Err != nil {
		e.Err = fmt.Errorf("%s, %s", e.Err.Error(), a.Err.Error())
	}
	for _, i := range a.Issues {
		e.Issues = append(e.Issues, i)
	}
	return e
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
