package controller

import (
	"github.com/KrakenSystems/wg-operator/pkg/controller/client"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, client.Add)
}
