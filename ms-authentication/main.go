// Copyright © 2023 Intel Corporation. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package main

import (
	"ms-authentication/routes"
	"os"

	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg"
)

const (
	serviceKey = "ms-authentication"
)

func main() {
	var ok bool
	service, ok := pkg.NewAppService(serviceKey)
	if !ok {
		os.Exit(1)
	}
	lc := service.LoggingClient()

	controller := routes.NewController(service)
	err := controller.AddAllRoutes()
	if err != nil {
		lc.Errorf("failed to add all Routes: %s", err.Error())
		os.Exit(1)
	}
	if err := service.Run(); err != nil {
		lc.Errorf("Run returned error: %s", err.Error())
		os.Exit(1)
	}

	os.Exit(0)
}
