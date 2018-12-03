package controller

import (
	"github.com/fstab/promagent-operator/pkg/controller/promagent"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, promagent.Add)
}
