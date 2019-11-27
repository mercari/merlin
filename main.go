package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kouzoh/merlin/controller"
	"github.com/kouzoh/merlin/logger"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // for kubeclient GCP auth
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	exitError = 1
)

var kubeConfigPath string
var interval string

func init() {
	flag.StringVar(&kubeConfigPath, "kubeConfig", "", "Path to a kube config. Only required if out-of-cluster.")
	flag.StringVar(&interval, "interval", "10", "Interval to check the system resources, in seconds")
}

func main() {
	flag.Parse()
	checkInterval, err := strconv.Atoi(interval)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to parse interval, is it int? error: %s\n", err)
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "INFO"
	}

	log, err := logger.New(strings.ToUpper(logLevel))
	if err != nil {
		fmt.Fprintf(os.Stderr, "[ERROR] Failed to create logger: %s\n", err)
		os.Exit(exitError)
	}

	var config *rest.Config

	if kubeConfigPath == "" {
		config, err = rest.InClusterConfig()
	} else {
		kubeConfigPath, err = filepath.Abs(kubeConfigPath)
		if err != nil {
			log.Fatal("cannot get absolute path of: ",
				zap.String("kubeConfig path", kubeConfigPath),
				zap.Error(err),
			)
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	}
	if err != nil {
		log.Fatal("failed to read kubeconfig",
			zap.Error(err),
		)
	}

	ctx := context.Background()

	// create the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal("failed to create kubernetes clientset",
			zap.Error(err),
		)
	}
	c := controller.Controller{
		Clientset: clientset,
		Interval:  time.Duration(checkInterval) * time.Second,
		Logger:    log,
	}
	c.Run(ctx)
}
