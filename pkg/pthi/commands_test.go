/*********************************************************************
 * Copyright (c) Intel Corporation 2021
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/
package pthi

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockHECICommands struct{}

var message []byte
var numBytes uint32 = GET_REQUEST_SIZE

func (c *MockHECICommands) Init() error           { return nil }
func (c *MockHECICommands) GetBufferSize() uint32 { return 5120 } // MaxMessageLength

func (c *MockHECICommands) SendMessage(buffer []byte, done *uint32) (bytesWritten uint32, err error) {
	return numBytes, nil
}
func (c *MockHECICommands) ReceiveMessage(buffer []byte, done *uint32) (bytesRead uint32, err error) {
	for i := 0; i < len(message) && i < len(buffer); i++ {
		buffer[i] = message[i]
	}
	return 12, nil
}
func (c *MockHECICommands) Close() {}

var pthi Command

func init() {
	pthi = Command{}
	pthi.heci = &MockHECICommands{}
}

func TestGetGUID(t *testing.T) {

	// Call function will check that numBytes equals the command size
	numBytes = GET_REQUEST_SIZE

	// Load byte array of response into message
	prepareMessage := GetUUIDResponse{
		Header: ResponseMessageHeader{},
		UUID:   [16]uint8{1, 2, 3, 4},
	}
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.LittleEndian, prepareMessage)
	message = bin_buf.Bytes()

	// Run function and test cases
	result, err := pthi.GetUUID()
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Equal(t, "\x01\x02\x03\x04\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", result)
}
func TestGetControlMode(t *testing.T) {
	numBytes = GET_REQUEST_SIZE
	prepareMessage := GetControlModeResponse{
		Header: ResponseMessageHeader{},
		State:  3,
	}
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.LittleEndian, prepareMessage)
	message = bin_buf.Bytes()

	result, err := pthi.GetControlMode()
	assert.NoError(t, err)
	assert.Equal(t, 3, result)
}
func TestGetCodeVersions(t *testing.T) {

	numBytes = GET_REQUEST_SIZE
	prepareMessage := GetCodeVersionsResponse{
		Header: ResponseMessageHeader{},
		CodeVersion: CodeVersions{
			BiosVersion:   [BIOS_VERSION_LEN]uint8{1, 2, 3},
			VersionsCount: 1,
		},
	}
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.LittleEndian, prepareMessage)
	message = bin_buf.Bytes()

	result, err := pthi.GetCodeVersions()
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Equal(t, int(result.CodeVersion.BiosVersion[1]), 2)
	assert.Equal(t, int(result.CodeVersion.VersionsCount), 1)
}

func TestGetDNSSuffix(t *testing.T) {
	numBytes = GET_REQUEST_SIZE
	prepareMessage := GetPKIFQDNSuffixResponse{
		Header: ResponseMessageHeader{},
		Suffix: AMTANSIString{
			Length: 4,
			Buffer: [1000]uint8{1, 2, 3, 4},
		},
	}
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.LittleEndian, prepareMessage)
	message = bin_buf.Bytes()

	result, err := pthi.GetDNSSuffix()
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Equal(t, "\x01\x02\x03\x04", result)
}

func TestEnumerateHashHandles(t *testing.T) {
	numBytes = GET_REQUEST_SIZE
	prepareMessage := GetHashHandlesResponse{
		Header: ResponseMessageHeader{},
		HashHandles: AMTHashHandles{
			Length:  1,
			Handles: [CERT_HASH_MAX_NUMBER]uint32{0},
		},
	}
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.LittleEndian, prepareMessage)
	message = bin_buf.Bytes()
	result, err := pthi.enumerateHashHandles()
	assert.NoError(t, err)
	assert.Equal(t, uint32(1), result.Length)
}

func TestGetCertificateHashes(t *testing.T) { // Needs more work
	numBytes = 16
	prepareMessage2 := GetCertHashEntryResponse{
		Header: ResponseMessageHeader{Status: 1},
		Hash: CertHashEntry{
			IsDefault:       1,
			IsActive:        1,
			HashAlgorithm:   4,
			CertificateHash: [CERT_HASH_MAX_LENGTH]uint8{9, 9, 9},
			Name: AMTANSIString{
				Length: 4,
				Buffer: [1000]uint8{1, 2, 3, 4},
			},
		},
	}
	var bin_buf2 bytes.Buffer
	binary.Write(&bin_buf2, binary.LittleEndian, prepareMessage2)
	message = bin_buf2.Bytes()

	result, err := pthi.GetCertificateHashes(AMTHashHandles{Length: 1})
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Equal(t, int(result[0].IsDefault), 1)
	assert.Equal(t, int(result[0].IsActive), 1)
	assert.Equal(t, int(result[0].CertificateHash[0]), 9)
	assert.Equal(t, int(result[0].HashAlgorithm), 4)
	assert.Equal(t, int(result[0].Name.Length), 4)
}

func TestGetRemoteAccessConnectionStatus(t *testing.T) {
	numBytes = GET_REQUEST_SIZE
	prepareMessage := GetRemoteAccessConnectionStatusResponse{
		Header:        ResponseMessageHeader{},
		NetworkStatus: 1,
		RemoteStatus:  2,
		RemoteTrigger: 3,
		MPSHostname: AMTANSIString{
			Length: 4,
			Buffer: [1000]uint8{1, 2, 3, 4},
		},
	}
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.LittleEndian, prepareMessage)
	message = bin_buf.Bytes()

	result, err := pthi.GetRemoteAccessConnectionStatus()
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Equal(t, int(result.NetworkStatus), 1)
	assert.Equal(t, int(result.RemoteStatus), 2)
	assert.Equal(t, int(result.RemoteTrigger), 3)
	assert.Equal(t, int(result.MPSHostname.Length), 4)
}

func TestGetLANInterfaceSettings(t *testing.T) {
	numBytes = 16
	prepareMessage := GetLANInterfaceSettingsResponse{
		Header:      ResponseMessageHeader{},
		Enabled:     1,
		Ipv4Address: 1020,
		DhcpEnabled: 0,
		DhcpIpMode:  0,
		LinkStatus:  1,
		MacAddress:  [6]uint8{1, 2, 3, 4, 5, 6},
	}
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.LittleEndian, prepareMessage)
	message = bin_buf.Bytes()

	result, err := pthi.GetLANInterfaceSettings(true)
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Equal(t, int(result.Enabled), 1)
	assert.Equal(t, int(result.Ipv4Address), 1020)
	assert.Equal(t, int(result.DhcpEnabled), 0)
	assert.Equal(t, int(result.DhcpIpMode), 0)
	assert.Equal(t, int(result.LinkStatus), 1)
	assert.Equal(t, result.MacAddress, [6]uint8{1, 2, 3, 4, 5, 6})
}

func TestGetLocalSystemAccount(t *testing.T) {
	numBytes = 52
	prepareMessage := GetLocalSystemAccountResponse{
		Header: ResponseMessageHeader{},
		Account: LocalSystemAccount{
			Username: [CFG_MAX_ACL_USER_LENGTH]uint8{1, 2, 3, 4},
			Password: [CFG_MAX_ACL_USER_LENGTH]uint8{8, 7, 6, 5},
		},
	}
	var bin_buf bytes.Buffer
	binary.Write(&bin_buf, binary.LittleEndian, prepareMessage)
	message = bin_buf.Bytes()

	result, err := pthi.GetLocalSystemAccount()
	assert.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.Equal(t, result.Account.Username, [CFG_MAX_ACL_USER_LENGTH]uint8{1, 2, 3, 4})
	assert.Equal(t, result.Account.Password, [CFG_MAX_ACL_USER_LENGTH]uint8{8, 7, 6, 5})

}
