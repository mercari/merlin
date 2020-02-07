package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

type Rule interface {
	Evaluate(ctx context.Context, cli client.Client, log logr.Logger, resource interface{}) *merlinv1.EvaluationResult
}
