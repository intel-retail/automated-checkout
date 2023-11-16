// Copyright © 2023 Intel Corporation. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package routes

import (
	"encoding/json"
	"fmt"
	"net/http"

	"as-controller-board-status/functions"

	"github.com/edgexfoundry/app-functions-sdk-go/v3/pkg/interfaces"
	"github.com/edgexfoundry/go-mod-core-contracts/v3/clients/logger"
)

type Controller struct {
	lc          logger.LoggingClient
	service     interfaces.ApplicationService
	boardStatus *functions.CheckBoardStatus
}

func NewController(lc logger.LoggingClient, service interfaces.ApplicationService, boardStatus *functions.CheckBoardStatus) Controller {
	return Controller{
		lc:          lc,
		service:     service,
		boardStatus: boardStatus,
	}
}

func (c *Controller) AddAllRoutes() error {
	// Add the "status" REST API route
	err := c.service.AddRoute("/status", c.GetStatus, http.MethodGet, http.MethodOptions)
	if err != nil {
		return fmt.Errorf("error adding route: %s", err.Error())
	}
	return nil
}

// GetStatus is a REST API endpoint that enables a web UI or some other downstream
// service to inquire about the status of the upstream Automated Vending hardware interface(s).
func (c *Controller) GetStatus(writer http.ResponseWriter, req *http.Request) {
	controllerBoardStatus, err := json.Marshal(c.boardStatus.ControllerBoardStatus)
	if err != nil {
		errMsg := fmt.Sprintf("Failed to serialize the controller board's current state: %s", err.Error())
		c.lc.Error(errMsg)

		writer.WriteHeader(http.StatusInternalServerError)
		writer.Write([]byte(errMsg))
		return
	}
	c.lc.Info("GetStatus successfully!")
	writer.Write(controllerBoardStatus)
}
