/*********************************************************************
 * Copyright (c) Intel Corporation 2021
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/
package rps

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"rpc/internal/amt"
	"rpc/internal/rpc"
	"rpc/pkg/utils"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Mock the AMT Hardware
type MockAMT struct{}

var mebxDNSSuffix string
var controlMode int = 0

func (c MockAMT) Initialize() (bool, error) {
	return true, nil
}
func (c MockAMT) GetVersionDataFromME(key string) (string, error) { return "Version", nil }
func (c MockAMT) GetUUID() (string, error)                        { return "123-456-789", nil }
func (c MockAMT) GetUUIDV2() (string, error)                      { return "", nil }
func (c MockAMT) GetControlMode() (int, error)                    { return controlMode, nil }
func (c MockAMT) GetControlModeV2() (int, error)                  { return controlMode, nil }
func (c MockAMT) GetOSDNSSuffix() (string, error)                 { return "osdns", nil }
func (c MockAMT) GetDNSSuffix() (string, error)                   { return mebxDNSSuffix, nil }
func (c MockAMT) GetCertificateHashes() ([]amt.CertHashEntry, error) {
	return []amt.CertHashEntry{}, nil
}
func (c MockAMT) GetRemoteAccessConnectionStatus() (amt.RemoteAccessStatus, error) {
	return amt.RemoteAccessStatus{}, nil
}
func (c MockAMT) GetLANInterfaceSettings(useWireless bool) (amt.InterfaceSettings, error) {
	return amt.InterfaceSettings{}, nil
}
func (c MockAMT) GetLocalSystemAccount() (amt.LocalSystemAccount, error) {
	return amt.LocalSystemAccount{Username: "Username", Password: "Password"}, nil
}

var p Payload

func (c MockAMT) InitiateLMS() {}

func init() {
	p = Payload{}
	p.AMT = MockAMT{}

}
func TestCreatePayload(t *testing.T) {
	mebxDNSSuffix = "mebxdns"
	result, err := p.createPayload("", "")
	assert.Equal(t, "Version", result.Version)
	assert.Equal(t, "Version", result.Build)
	assert.Equal(t, "Version", result.SKU)
	assert.Equal(t, "123-456-789", result.UUID)
	assert.Equal(t, "Username", result.Username)
	assert.Equal(t, "Password", result.Password)
	assert.Equal(t, 0, result.CurrentMode)
	assert.NotEmpty(t, result.Hostname)
	assert.Equal(t, "mebxdns", result.FQDN)
	assert.Equal(t, utils.ClientName, result.Client)
	assert.Len(t, result.CertificateHashes, 0)
	assert.NoError(t, err)
}
func TestCreatePayloadWithOSDNSSuffix(t *testing.T) {
	mebxDNSSuffix = ""
	result, err := p.createPayload("", "")
	assert.NoError(t, err)
	assert.Equal(t, "osdns", result.FQDN)
}
func TestCreatePayloadWithDNSSuffix(t *testing.T) {

	result, err := p.createPayload("vprodemo.com", "")
	assert.NoError(t, err)
	assert.Equal(t, "vprodemo.com", result.FQDN)
}
func TestCreateActivationRequestNoDNSSuffix(t *testing.T) {
	flags := rpc.Flags{
		Command: "method",
	}
	result, err := p.CreateMessageRequest(flags)
	assert.NoError(t, err)
	assert.Equal(t, "method", result.Method)
	assert.Equal(t, "key", result.APIKey)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, "ok", result.Message)
	assert.NotEmpty(t, result.Payload)
	assert.Equal(t, utils.ProtocolVersion, result.ProtocolVersion)
	assert.Equal(t, utils.ProjectVersion, result.AppVersion)
}
func TestCreateActivationRequestNoPasswordShouldPrompt(t *testing.T) {
	controlMode = 1
	flags := rpc.Flags{
		Command: "method",
	}
	input := []byte("password")
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	_, err = w.Write(input)
	if err != nil {
		t.Error(err)
	}
	w.Close()

	stdin := os.Stdin
	// Restore stdin right after the test.
	defer func() {
		os.Stdin = stdin
		controlMode = 0
	}()
	os.Stdin = r
	result, err := p.CreateMessageRequest(flags)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.Payload)
}
func TestCreateActivationRequestWithPasswordShouldNotPrompt(t *testing.T) {
	controlMode = 1
	flags := rpc.Flags{
		Command:  "method",
		Password: "password",
	}
	// Restore stdin right after the test.
	defer func() {
		controlMode = 0
	}()
	result, err := p.CreateMessageRequest(flags)
	msgPayload, decodeErr := base64.StdEncoding.DecodeString(result.Payload)
	payload := MessagePayload{}
	jsonErr := json.Unmarshal(msgPayload, &payload)
	assert.NoError(t, err)
	assert.NoError(t, decodeErr)
	assert.NoError(t, jsonErr)
	assert.NotEmpty(t, result.Payload)
	assert.Equal(t, "password", payload.Password)
}

func TestCreateActivationRequestWithDNSSuffix(t *testing.T) {
	flags := rpc.Flags{
		Command: "method",
		DNS:     "vprodemo.com",
	}
	result, err := p.CreateMessageRequest(flags)
	assert.NoError(t, err)
	assert.Equal(t, "method", result.Method)
	assert.Equal(t, "key", result.APIKey)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, "ok", result.Message)
	assert.Equal(t, utils.ProtocolVersion, result.ProtocolVersion)
	assert.Equal(t, utils.ProjectVersion, result.AppVersion)
}

func TestCreateActivationResponse(t *testing.T) {

	result, err := p.CreateMessageResponse([]byte("123"))
	assert.NoError(t, err)
	assert.Equal(t, "response", result.Method)
	assert.Equal(t, "key", result.APIKey)
	assert.Equal(t, "ok", result.Status)
	assert.Equal(t, "ok", result.Message)
	assert.NotEmpty(t, result.Payload)
	assert.Equal(t, utils.ProtocolVersion, result.ProtocolVersion)
	assert.Equal(t, utils.ProjectVersion, result.AppVersion)

}
