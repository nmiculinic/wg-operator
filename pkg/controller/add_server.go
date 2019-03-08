package controller

import (
	"github.com/KrakenSystems/wg-operator/pkg/controller/server"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, server.Add)
}
