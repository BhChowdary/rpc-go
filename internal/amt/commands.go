/*********************************************************************
 * Copyright (c) Intel Corporation 2021
 * SPDX-License-Identifier: Apache-2.0
 **********************************************************************/
package amt

// #cgo linux CFLAGS: -g -Wno-error -Wformat -Wformat-security -D_POSIX -DBUILD_LIBRARY -D_FORTIFY_SOURCE=2 -fstack-protector-strong
// #cgo windows CFLAGS: -g -w -DMICROSTACK_NO_STDAFX -DWIN32 -DWIN64 -DNDEBUG -D_CONSOLE -DMICROSTACK_NO_STDAFX -DWINSOCK2 -DMICROSTACK_NOTLS -D_UNICODE -D_WINDOWS -D_WIN32_WINNT=0x0A00 -DBUILD_LIBRARY
// #cgo windows LDFLAGS: -lDbgHelp -lIphlpapi -lSetupapi -lws2_32 -lPsapi -lCrypt32 -lWintrust -lVersion -lWtsapi32 -lGdiplus -lUserenv -lgdi32 -lucrtbase
// #include "../../microlms/MicroLMS/main.c"
// #include "../../microlms/core/utils.c"
// #include "../../microlms/heci/HECIWin.c"
// #include "../../microlms/heci/HECILinux.c"
// #include "../../microlms/heci/LMEConnection.c"
// #include "../../microlms/heci/PTHICommand.c"
// #include "../../microlms/microstack/ILibAsyncServerSocket.c"
// #include "../../microlms/microstack/ILibAsyncSocket.c"
// #include "../../microlms/microstack/ILibLMS.c"
// #include "../../microlms/microstack/ILibParsers.c"
import "C"
import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"os"
	"rpc/pkg/pthi"
	"rpc/pkg/utils"
	"strconv"
	"strings"
	"unsafe"
)

//TODO: Ensure pointers are freed properly throughout this file

// AMTUnicodeString ...
type AMTUnicodeString struct {
	Length uint16
	String [20]uint8 //[UNICODE_STRING_LEN]
}

// AMTVersionType ...
type AMTVersionType struct {
	Description AMTUnicodeString
	Version     AMTUnicodeString
}

// CodeVersions ...
type CodeVersions struct {
	BiosVersion   [65]uint8 //[BIOS_VERSION_LEN]
	VersionsCount uint32
	Versions      [50]AMTVersionType //[VERSIONS_NUMBER]
}

// AMTANSIString ...
type AMTANSIString struct {
	Length uint16
	Buffer [1000]C.char
}

// InterfaceSettings ...
type InterfaceSettings struct {
	IsEnabled   bool
	LinkStatus  string
	DHCPEnabled bool
	DHCPMode    string
	IPAddress   string //net.IP
	MACAddress  string
}

// RemoteAccessStatus holds connect status information
type RemoteAccessStatus struct {
	NetworkStatus string
	RemoteStatus  string
	RemoteTrigger string
	MPSHostname   string
}

// CCertHashEntry is used for reading data from the C call for Cert Hashes
type CCertHashEntry struct {
	CertificateHash [64]uint8
	HashAlgorithm   uint8
	IsActive        uint32
	IsDefault       uint32
	Name            AMTANSIString
}

// CertHashEntry is the GO struct for holding Cert Hash Entries
type CertHashEntry struct {
	Hash      string
	Name      string
	Algorithm string
	IsActive  bool
	IsDefault bool
}

// LocalSystemAccount holds username and password
type LocalSystemAccount struct {
	Username string
	Password string
}

type AMT interface {
	Initialize() (bool, error)
	GetVersionDataFromME(key string) (string, error)
	GetUUID() (string, error)
	GetUUIDV2() (string, error)
	GetControlMode() (int, error)
	GetControlModeV2() (int, error)
	GetOSDNSSuffix() (string, error)
	GetDNSSuffix() (string, error)
	GetDNSSuffixV2() (string, error)
	GetCertificateHashes() ([]CertHashEntry, error)
	GetCertificateHashesV2() ([]CertHashEntry, error)
	GetRemoteAccessConnectionStatus() (RemoteAccessStatus, error)
	GetRemoteAccessConnectionStatusV2() (RemoteAccessStatus, error)
	GetLANInterfaceSettings(useWireless bool) (InterfaceSettings, error)
	GetLANInterfaceSettingsV2(useWireless bool) (InterfaceSettings, error)
	GetLocalSystemAccount() (LocalSystemAccount, error)
	GetLocalSystemAccountV2() (LocalSystemAccount, error)
	InitiateLMS()
}

func ANSI2String(ansi pthi.AMTANSIString) string {
	output := ""
	for i := 0; i < int(ansi.Length); i++ {
		output = output + string(ansi.Buffer[i])
	}

	return output
}

type Command struct {
}

// Initialize determines if rpc is able to initialize the heci driver
func (amt Command) Initialize() (bool, error) {
	// initialize HECI interface
	result := C.heci_Init(nil, 0)
	if !*((*bool)(unsafe.Pointer(&result))) {
		return false, errors.New("unable to initialize")
	}

	return true, nil
}

// GetVersionDataFromME ...
func (amt Command) GetVersionDataFromME(key string) (string, error) {

	_, err := amt.Initialize()
	if err != nil {
		return "", err
	}
	//get code version
	codeVersion := CodeVersions{}
	packedCodeVersion := C.struct__CODE_VERSIONS{}
	status := C.pthi_GetCodeVersions(&packedCodeVersion)

	// additional versions
	if status == 0 {
		cdata := C.GoBytes(unsafe.Pointer(&packedCodeVersion), C.sizeof_struct__CODE_VERSIONS)
		buf := bytes.NewBuffer(cdata)
		binary.Read(buf, binary.LittleEndian, &codeVersion.BiosVersion)
		binary.Read(buf, binary.LittleEndian, &codeVersion.VersionsCount)
		binary.Read(buf, binary.LittleEndian, &codeVersion.Versions)

		for i := 0; i < int(codeVersion.VersionsCount); i++ {
			if string(codeVersion.Versions[i].Description.String[:codeVersion.Versions[i].Description.Length]) == key {
				return strings.Replace(string(codeVersion.Versions[i].Version.String[:]), "\u0000", "", -1), nil
			}
		}
	}

	return "", errors.New(key + " Not Found")
}

// GetUUID ...
func (amt Command) GetUUID() (string, error) {
	_, err := amt.Initialize()
	if err != nil {
		return "", err
	}
	//get code version
	//codeVersion := CodeVersions{}
	packedUUID := C.AMT_UUID{}
	status := C.pthi_GetUUID(&packedUUID)

	var hexValues [16]string

	if status == 0 {
		for i := 0; i < 16; i++ {
			hexValues[i] = fmt.Sprintf("%02x", int(packedUUID[i]))
		}

		uuidStr := hexValues[3] + hexValues[2] + hexValues[1] + hexValues[0] + "-" +
			hexValues[5] + hexValues[4] + "-" +
			hexValues[7] + hexValues[6] + "-" +
			hexValues[8] + hexValues[9] + "-" +
			hexValues[10] + hexValues[11] + hexValues[12] + hexValues[13] + hexValues[14] + hexValues[15]
		return uuidStr, nil

	}
	return "", errors.New("UUID not found")
}

// GetUUID ...
func (amt Command) GetUUIDV2() (string, error) {
	pthi := pthi.NewPTHICommand()
	defer pthi.Close()
	result, err := pthi.GetUUID()
	if err != nil {
		return "", err
	}

	var hexValues [16]string

	for i := 0; i < 16; i++ {
		hexValues[i] = fmt.Sprintf("%02x", int(result[i]))
	}

	uuidStr := hexValues[3] + hexValues[2] + hexValues[1] + hexValues[0] + "-" +
		hexValues[5] + hexValues[4] + "-" +
		hexValues[7] + hexValues[6] + "-" +
		hexValues[8] + hexValues[9] + "-" +
		hexValues[10] + hexValues[11] + hexValues[12] + hexValues[13] + hexValues[14] + hexValues[15]
	return uuidStr, nil

}

// GetControlMode ...
func (amt Command) GetControlMode() (int, error) {
	_, err := amt.Initialize()
	if err != nil {
		return -1, err
	}

	var controlMode C.int = 0
	status := C.pthi_GetControlMode(&controlMode)
	if status == 0 {
		return int(controlMode), nil
	}
	return -1, errors.New("unable to determine control mode")
}

// GetControlMode ...
func (amt Command) GetControlModeV2() (int, error) {
	pthi := pthi.NewPTHICommand()
	defer pthi.Close()
	result, err := pthi.GetControlMode()
	if err != nil {
		return -1, err
	}

	return result, nil

}

// GetDNSSuffix ...
func (amt Command) GetOSDNSSuffix() (string, error) {
	lanResult, _ := amt.GetLANInterfaceSettings(false)
	ifaces, _ := net.Interfaces()
	for _, v := range ifaces {
		if v.HardwareAddr.String() == lanResult.MACAddress {
			addrs, _ := v.Addrs()
			for _, a := range addrs {
				networkIp, ok := a.(*net.IPNet)
				if ok && !networkIp.IP.IsLoopback() && networkIp.IP.To4() != nil {
					ip := networkIp.IP.String()
					suffix, _ := net.LookupAddr(ip)
					if len(suffix) > 0 {
						hostname, _ := os.Hostname()
						dnsSuffix := strings.Trim(suffix[0], hostname)
						dnsSuffix = strings.TrimLeft(dnsSuffix, ".")
						dnsSuffix = strings.TrimRight(dnsSuffix, ".")
						return dnsSuffix, nil
					}
					return "", nil
				}
			}
		}
	}
	return "", nil
}

// GetDNSSuffix ...
func (amt Command) GetDNSSuffix() (string, error) {
	_, err := amt.Initialize()
	if err != nil {
		return "", err
	}

	dnsSuffix := AMTANSIString{}
	packedDNSSuffix := C.struct__AMT_ANSI_STRING{}
	status := C.pthi_GetDnsSuffix(&packedDNSSuffix)
	if status == 0 {

		cdata := C.GoBytes(unsafe.Pointer(&packedDNSSuffix), C.sizeof_struct__AMT_ANSI_STRING*253) //253 is maximum FQDN Length
		buf := bytes.NewBuffer(cdata)

		binary.Read(buf, binary.LittleEndian, &dnsSuffix.Length)
		binary.Read(buf, binary.LittleEndian, &dnsSuffix.Buffer)
		cStrings := (*[1 << 28]*C.char)(unsafe.Pointer(&dnsSuffix.Buffer))[:int(dnsSuffix.Length):int(dnsSuffix.Length)]
		if len(cStrings) > 0 {
			return C.GoString(cStrings[0])[:int(dnsSuffix.Length)], nil
		}
		return "", nil
	}
	return "", errors.New("unable to retrieve DNS suffix")
}

func (amt Command) GetDNSSuffixV2() (string, error) {
	pthi := pthi.NewPTHICommand()
	defer pthi.Close()
	result, err := pthi.GetDNSSuffix()
	if err != nil {
		return "", err
	}

	return result, nil
}

// GetCertificateHashes ...
func (amt Command) GetCertificateHashes() ([]CertHashEntry, error) {
	hashEntries := []CertHashEntry{}

	_, err := amt.Initialize()
	if err != nil {
		return hashEntries, err
	}
	hashedHandles := C.struct__AMT_HASH_HANDLES{}
	packedCertHashEntry := C.struct__CERTHASH_ENTRY{}
	status := C.pthi_EnumerateHashHandles(&hashedHandles)
	if status == 0 {
		for i := 0; i < int(hashedHandles.Length); i++ {
			status2 := C.pthi_GetCertificateHashEntry(hashedHandles.Handles[i], &packedCertHashEntry)
			tmp := CertHashEntry{}
			ccerthash := CCertHashEntry{}
			hashSize := 0
			if status2 == 0 {
				cdata := C.GoBytes(unsafe.Pointer(&packedCertHashEntry), C.sizeof_struct__CERTHASH_ENTRY+(1024))
				buf := bytes.NewBuffer(cdata)

				binary.Read(buf, binary.LittleEndian, &ccerthash.IsDefault)
				binary.Read(buf, binary.LittleEndian, &ccerthash.IsActive)
				binary.Read(buf, binary.LittleEndian, &ccerthash.CertificateHash)
				binary.Read(buf, binary.LittleEndian, &ccerthash.HashAlgorithm)
				binary.Read(buf, binary.LittleEndian, &ccerthash.Name)

				hashSize, tmp.Algorithm = utils.InterpretHashAlgorithm(int(ccerthash.HashAlgorithm))
				if ccerthash.IsActive == 1 {
					cStrings := (*[1 << 28]*C.char)(unsafe.Pointer(&ccerthash.Name.Buffer))[:int(ccerthash.Name.Length):int(ccerthash.Name.Length)]
					if len(cStrings) > 0 {

						tmp.Name = strings.Trim(C.GoString(cStrings[0])[:int(ccerthash.Name.Length)], "\xab")
					}
					tmp.IsDefault = ccerthash.IsDefault == 1
					tmp.IsActive = ccerthash.IsActive == 1

					hashString := ""
					for i := 0; i < hashSize; i++ {
						hashString = hashString + fmt.Sprintf("%02x", int(ccerthash.CertificateHash[i]))
					}

					tmp.Hash = hashString
					hashEntries = append(hashEntries, tmp)
				}

			} else {
				//todo: log error
			}
		}
		return hashEntries, nil
	}
	return hashEntries, errors.New("unable to retrieve certificate hashes")
}

func (amt Command) GetCertificateHashesV2() ([]CertHashEntry, error) {
	pthi := pthi.NewPTHICommand()
	defer pthi.Close()

	pthiEntryList, err := pthi.GetCertificateHashes()
	amtEntryList := []CertHashEntry{}
	if err != nil {
		return amtEntryList, err
	}

	// Convert pthi results to amt results
	for _, pthiEntry := range pthiEntryList {
		hashString := ""
		for i := 0; i < len(pthiEntry.CertificateHash); i++ {
			hashString = hashString + fmt.Sprintf("%02x", int(pthiEntry.CertificateHash[i]))
		}

		_, algo := utils.InterpretHashAlgorithm(int(pthiEntry.HashAlgorithm))

		amtEntry := CertHashEntry{
			Hash:      hashString,
			Name:      ANSI2String(pthiEntry.Name),
			Algorithm: algo,
			IsActive:  pthiEntry.IsActive > 0,
			IsDefault: pthiEntry.IsDefault > 0,
		}

		amtEntryList = append(amtEntryList, amtEntry)
	}

	return amtEntryList, nil
}

// GetRemoteAccessConnectionStatus ...
func (amt Command) GetRemoteAccessConnectionStatus() (RemoteAccessStatus, error) {
	remoteAccessStatus := RemoteAccessStatus{}

	_, err := amt.Initialize()
	if err != nil {
		return remoteAccessStatus, err
	}

	mpsHostname := AMTANSIString{}
	packedRAS := C.struct__REMOTE_ACCESS_STATUS{}
	status := C.pthi_GetRemoteAccessConnectionStatus(&packedRAS)
	if status == 0 {
		remoteAccessStatus.NetworkStatus = utils.InterpretAMTNetworkConnectionStatus(int(packedRAS.AmtNetworkConnectionStatus))
		remoteAccessStatus.RemoteStatus = utils.InterpretRemoteAccessConnectionStatus(int(packedRAS.RemoteAccessConnectionStatus))
		remoteAccessStatus.RemoteTrigger = utils.InterpretRemoteAccessTrigger(int(packedRAS.RemoteAccessConnectionTrigger))

		cdata := C.GoBytes(unsafe.Pointer(&packedRAS.MpsHostname), C.sizeof_struct__AMT_ANSI_STRING*utils.MPSServerMaxLength)
		buf := bytes.NewBuffer(cdata)

		binary.Read(buf, binary.LittleEndian, &mpsHostname.Length)
		binary.Read(buf, binary.LittleEndian, &mpsHostname.Buffer)
		cStrings := (*[1 << 28]*C.char)(unsafe.Pointer(&mpsHostname.Buffer))[:int(mpsHostname.Length):int(mpsHostname.Length)]
		if len(cStrings) > 0 {
			remoteAccessStatus.MPSHostname = strings.Trim(C.GoString(cStrings[0]), "\xab")
		}
	} else {
		return remoteAccessStatus, nil
	}
	return remoteAccessStatus, errors.New("unable to retrieve remote access connection status")
}

func (amt Command) GetRemoteAccessConnectionStatusV2() (RemoteAccessStatus, error) {
	pthi := pthi.NewPTHICommand()
	defer pthi.Close()
	result, err := pthi.GetRemoteAccessConnectionStatus()
	if err != nil {
		emptyRAStatus := RemoteAccessStatus{}
		return emptyRAStatus, err
	}

	RAStatus := RemoteAccessStatus{
		NetworkStatus: utils.InterpretAMTNetworkConnectionStatus(int(result.NetworkStatus)),
		RemoteStatus:  utils.InterpretRemoteAccessConnectionStatus(int(result.RemoteStatus)),
		RemoteTrigger: utils.InterpretRemoteAccessTrigger(int(result.RemoteTrigger)),
		MPSHostname:   ANSI2String(result.MPSHostname),
	}

	return RAStatus, nil
}

// GetLANInterfaceSettings ...
func (amt Command) GetLANInterfaceSettings(useWireless bool) (InterfaceSettings, error) {
	interfaceSettings := InterfaceSettings{}

	_, err := amt.Initialize()
	if err != nil {
		return interfaceSettings, err
	}

	LANSettings := C.struct__LAN_SETTINGS{}
	var status C.uint
	if useWireless {
		status = C.pthi_GetLanInterfaceSettings(1, &LANSettings)
	} else {
		status = C.pthi_GetLanInterfaceSettings(0, &LANSettings)

	}
	if status == 0 {
		interfaceSettings.IsEnabled = LANSettings.Enabled == 1
		interfaceSettings.DHCPEnabled = LANSettings.DhcpEnabled == 1

		if LANSettings.DhcpIpMode == 1 {
			interfaceSettings.DHCPMode = "active"
		} else {
			interfaceSettings.DHCPMode = "passive"
		}

		if LANSettings.LinkStatus == 1 {
			interfaceSettings.LinkStatus = "up"
		} else {
			interfaceSettings.LinkStatus = "down"
		}

		part1 := LANSettings.Ipv4Address >> 24 & 0xff
		part2 := LANSettings.Ipv4Address >> 16 & 0xff
		part3 := LANSettings.Ipv4Address >> 8 & 0xff
		part4 := LANSettings.Ipv4Address & 0xff

		interfaceSettings.IPAddress = strconv.Itoa(int(part1)) + "." + strconv.Itoa(int(part2)) + "." + strconv.Itoa(int(part3)) + "." + strconv.Itoa(int(part4))

		macPart0 := fmt.Sprintf("%02x", int(LANSettings.MacAddress[0]))
		macPart1 := fmt.Sprintf("%02x", int(LANSettings.MacAddress[1]))
		macPart2 := fmt.Sprintf("%02x", int(LANSettings.MacAddress[2]))
		macPart3 := fmt.Sprintf("%02x", int(LANSettings.MacAddress[3]))
		macPart4 := fmt.Sprintf("%02x", int(LANSettings.MacAddress[4]))
		macPart5 := fmt.Sprintf("%02x", int(LANSettings.MacAddress[5]))
		interfaceSettings.MACAddress = macPart0 + ":" + macPart1 + ":" + macPart2 + ":" + macPart3 + ":" + macPart4 + ":" + macPart5
	} else {
		return interfaceSettings, errors.New("unable to retrieve interface settings")
	}
	return interfaceSettings, nil
}

func (amt Command) GetLANInterfaceSettingsV2(useWireless bool) (InterfaceSettings, error) {
	pthi := pthi.NewPTHICommand()
	defer pthi.Close()
	result, err := pthi.GetLANInterfaceSettings(useWireless)
	if err != nil {
		emptySettings := InterfaceSettings{}
		return emptySettings, err
	}

	settings := InterfaceSettings{
		IsEnabled:   result.Enabled > 0,
		LinkStatus:  "placeholder",
		DHCPEnabled: result.DhcpEnabled > 0,
		DHCPMode:    "placeholder",
		IPAddress:   "placeholder",
		MACAddress:  "placeholder",
	}

	if result.LinkStatus == 1 {
		settings.LinkStatus = "up"
	} else {
		settings.LinkStatus = "down"
	}

	if result.DhcpIpMode == 1 {
		settings.DHCPMode = "active"
	} else {
		settings.DHCPMode = "passive"
	}

	str := string(result.Ipv4Address)
	settings.IPAddress = str[:2] + "." + str[2:5] + "." + str[5:8] + "." + str[8:]

	settings.MACAddress = (string(result.MacAddress[0]) + ":" +
		string(result.MacAddress[1]) + ":" +
		string(result.MacAddress[2]) + ":" +
		string(result.MacAddress[3]) + ":" +
		string(result.MacAddress[4]) + ":" +
		string(result.MacAddress[5]))

	return settings, nil
}

// GetLocalSystemAccount ...
func (amt Command) GetLocalSystemAccount() (LocalSystemAccount, error) {
	lsa := LocalSystemAccount{}

	_, err := amt.Initialize()
	if err != nil {
		return lsa, err
	}

	localSystemAccount := C.struct__LOCAL_SYSTEM_ACCOUNT{}
	status := C.pthi_GetLocalSystemAccount(&localSystemAccount)
	println(status)
	//todo: should these be trimmed?
	if status == 0 {
		lsa.Username = strings.Replace(C.GoStringN((*C.char)(unsafe.Pointer(&localSystemAccount.username)), 33), "\u0000", "", -1) //33 from CFG_MAX_ACL_USER_LENGTH
		lsa.Password = strings.Replace(C.GoStringN((*C.char)(unsafe.Pointer(&localSystemAccount.password)), 33), "\u0000", "", -1)
	} else {
		return lsa, errors.New("unable to retrieve local system account info")
	}
	return lsa, nil

}

func (amt Command) GetLocalSystemAccountV2() (LocalSystemAccount, error) {
	// pthi := pthi.NewPTHICommand()
	// defer pthi.Close()
	// result, err := pthi.GetLocalSystemAccount()
	// if err != nil {
	// 	//return nil, err
	// }

	return LocalSystemAccount{}, nil
}

// InitiateLMS ...
func (amt Command) InitiateLMS() {
	C.main_micro_lms()
}
