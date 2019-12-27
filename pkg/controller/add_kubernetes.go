package controller

import (
	"github.com/opsboost/kube-operator/pkg/controller/kubernetes"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, kubernetes.Add)
}
