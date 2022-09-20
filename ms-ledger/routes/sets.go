// Copyright © 2022 Intel Corporation. All rights reserved.
// SPDX-License-Identifier: BSD-3-Clause

package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"

	utilities "github.com/intel-iot-devkit/automated-checkout-utilities"
)

// SetPaymentStatus sets the `isPaid` field for a transaction to true/false
func (c *Controller) SetPaymentStatus(writer http.ResponseWriter, req *http.Request) {
	utilities.ProcessCORS(writer, req, func(writer http.ResponseWriter, req *http.Request) {

		// Read request body
		body := make([]byte, req.ContentLength)
		_, err := io.ReadFull(req.Body, body)
		if err != nil {
			utilities.WriteStringHTTPResponse(writer, req, http.StatusBadRequest, "Failed to parse request body", true)
			c.lc.Errorf("Failed to parse request body %s", err.Error())
			return
		}

		// Unmarshal the string contents of request into a proper structure
		var paymentStatus paymentInfo
		if err := json.Unmarshal(body, &paymentStatus); err != nil {
			utilities.WriteStringHTTPResponse(writer, req, http.StatusBadRequest, "Failed to unmarshal body", true)
			c.lc.Errorf("Failed to unmarshal body %s", err.Error())
			return
		}

		//Get all ledgers for all accounts
		accountLedgers, err := c.GetAllLedgers()
		if err != nil {
			utilities.WriteStringHTTPResponse(writer, req, http.StatusInternalServerError, "Failed to retrieve all ledgers for accounts "+err.Error(), true)
			c.lc.Errorf("Failed to retrieve all ledgers for accounts %s", err.Error())
			return
		}

		for accountIndex, account := range accountLedgers.Data {
			if paymentStatus.AccountID == account.AccountID {
				for transactionIndex, transaction := range account.Ledgers {
					if paymentStatus.TransactionID == transaction.TransactionID {
						accountLedgers.Data[accountIndex].Ledgers[transactionIndex].IsPaid = paymentStatus.IsPaid

						err := utilities.WriteToJSONFile(LedgerFileName, &accountLedgers, 0644)
						if err != nil {
							utilities.WriteStringHTTPResponse(writer, req, http.StatusInternalServerError, "Failed to update ledger", true)
							c.lc.Errorf("Failed to update ledger %s", err.Error())
							return
						}

						utilities.WriteStringHTTPResponse(writer, req, http.StatusOK, "Updated Payment Status for transaction "+strconv.FormatInt(paymentStatus.TransactionID, 10), false)
						c.lc.Infof("Updated Payment Status for transaction %s ", strconv.FormatInt(paymentStatus.TransactionID, 10))
						return
					}
				}
				utilities.WriteStringHTTPResponse(writer, req, http.StatusBadRequest, "Could not find Transaction "+strconv.FormatInt(paymentStatus.TransactionID, 10), true)
				c.lc.Errorf("Could not find Transaction %s", strconv.FormatInt(paymentStatus.TransactionID, 10))
				return
			}
		}
		utilities.WriteStringHTTPResponse(writer, req, http.StatusBadRequest, "Could not find account "+strconv.Itoa(paymentStatus.AccountID), true)
		c.lc.Errorf("Could not find account %s", strconv.Itoa(paymentStatus.AccountID))
	})
}

// LedgerAddTransaction adds a new transaction to the Account Ledger
func (c *Controller) LedgerAddTransaction(writer http.ResponseWriter, req *http.Request) {
	utilities.ProcessCORS(writer, req, func(writer http.ResponseWriter, req *http.Request) {

		response := utilities.GetHTTPResponseTemplate()

		// Read request body (this is the inference data)
		body := make([]byte, req.ContentLength)
		_, err := io.ReadFull(req.Body, body)
		if err != nil {
			utilities.WriteStringHTTPResponse(writer, req, http.StatusBadRequest, "Failed to parse request body", true)
			c.lc.Errorf("Failed to parse request body %s", err.Error())
			return
		}

		// Unmarshal the string contents of request for inference data into a proper structure
		// deltaLedger is accountID and list of Sku:delta
		var updateLedger deltaLedger
		if err := json.Unmarshal(body, &updateLedger); err != nil {
			utilities.WriteStringHTTPResponse(writer, req, http.StatusBadRequest, "Failed to unmarshal request body", true)
			c.lc.Errorf("Failed to unmarshal request body %s", err.Error())
			return
		}

		//Get all ledgers for all accounts
		accountLedgers, err := c.GetAllLedgers()
		if err != nil {
			utilities.WriteStringHTTPResponse(writer, req, http.StatusInternalServerError, "Failed to retrieve all ledgers for accounts "+err.Error(), true)
			c.lc.Errorf("Failed to retrieve all ledgers for accounts %s", err.Error())
			return
		}

		ledgerChanged := false
		var newLedger Ledger

		for accountIndex, account := range accountLedgers.Data {
			if updateLedger.AccountID == account.AccountID {
				newLedger = Ledger{
					TransactionID: time.Now().UnixNano(),
					TxTimeStamp:   time.Now().UnixNano(),
					LineTotal:     0,
					CreatedAt:     time.Now().UnixNano(),
					UpdatedAt:     time.Now().UnixNano(),
					IsPaid:        false,
					LineItems:     []LineItem{},
				}

				for _, deltaSKU := range updateLedger.DeltaSKUs {
					itemInfo, err := c.getInventoryItemInfo(c.inventoryEndpoint, deltaSKU.SKU)
					if err != nil {
						utilities.WriteStringHTTPResponse(writer, req, http.StatusBadRequest, "Could not find product Info for "+deltaSKU.SKU+" "+err.Error(), true)
						c.lc.Errorf("Could not find product Info for %s errir: %s", deltaSKU.SKU, err.Error())
						return
					}
					newLineItem := LineItem{
						SKU:         deltaSKU.SKU,
						ProductName: itemInfo.ProductName,
						ItemPrice:   itemInfo.ItemPrice,
						ItemCount:   int(math.Abs(float64(deltaSKU.Delta))),
					}
					newLedger.LineItems = append(newLedger.LineItems, newLineItem)
					newLedger.LineTotal = newLedger.LineTotal + (newLineItem.ItemPrice * float64(newLineItem.ItemCount))
				}

				// Add new Ledger to array of Ledgers for that account
				accountLedgers.Data[accountIndex].Ledgers = append(accountLedgers.Data[accountIndex].Ledgers, newLedger)
				ledgerChanged = true
			}
		}

		if !ledgerChanged {
			utilities.WriteStringHTTPResponse(writer, req, http.StatusBadRequest, "Account not found", true)
			c.lc.Error("No ledger change in any account")
			return
		}

		err = utilities.WriteToJSONFile(LedgerFileName, &accountLedgers, 0644)
		if err != nil {
			utilities.WriteStringHTTPResponse(writer, req, http.StatusInternalServerError, "Failed to update ledger", true)
			c.lc.Errorf("Failed to update ledger %s", err.Error())
			return
		}

		// return the new ledger as JSON, or if for some reason it cannot be processed back into
		// JSON for returning to the user, fallback to a simple string
		newLedgerJSON, err := utilities.GetAsJSON(newLedger)
		if err != nil {
			response.SetStringHTTPResponseFields(http.StatusOK, "Updated ledger successfully", false)
			c.lc.Warnf("Updated ledger successfully with error %s", err.Error())
		} else {
			response.SetJSONHTTPResponseFields(http.StatusOK, newLedgerJSON, false)
			c.lc.Infof("Updated ledger %s successfully", newLedgerJSON)
		}
		response.WriteHTTPResponse(writer, req)
	})
}

// getInventoryItemInfo is a helper function that will take the inference data (SKU)
// and return product details for a transaction to be recorded in the ledger
func (c *Controller) getInventoryItemInfo(inventoryEndpoint string, SKU string) (Product, error) {

	resp, err := c.sendCommand("GET", inventoryEndpoint+"/"+SKU, []byte(""))
	if err != nil {
		return Product{}, fmt.Errorf("Could not hit inventoryEndpoint, SKU may not exist")
	}

	defer resp.Body.Close()

	// Read the HTTP Response Body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Product{}, fmt.Errorf("Could not read response body from InventoryEndpoint")
	}

	// Prepare to store the http response in this variable
	var HTTPResponse utilities.HTTPResponse

	// Unmarshal the http response
	err = json.Unmarshal(body, &HTTPResponse)
	if err != nil {
		return Product{}, fmt.Errorf("Received an invalid data structure from InventoryEndpoint")
	}
	// Check the HTTP response error condition
	if HTTPResponse.Error {
		return Product{}, fmt.Errorf("Received an error response from the inventory service: " + HTTPResponse.Content.(string))
	}

	// Prepare to unmarshal the desired inventory item from the HTTP response's body (json)
	var inventoryItem Product
	err = json.Unmarshal([]byte(HTTPResponse.Content.(string)), &inventoryItem)
	if err != nil {
		return Product{}, fmt.Errorf("Received an invalid data structure from InventoryEndpoint")
	}

	return inventoryItem, nil
}
