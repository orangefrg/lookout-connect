package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func (r *MonitoringResult) String() string {
	builder := strings.Builder{}
	builder.WriteString(fmt.Sprintf("Result:\nNode config name: %s\n", r.NodeCfgName))
	if r.SSHError != nil {
		builder.WriteString(fmt.Sprintf("SSH Error: %v\n", r.SSHError))
		return builder.String()
	}
	if r.HostNameError != nil {
		builder.WriteString(fmt.Sprintf("Host Name Error: %v\n", r.HostNameError))
	} else {
		builder.WriteString(fmt.Sprintf("Host Name: %s\n", r.NodeName))
	}
	if r.UserNameError != nil {
		builder.WriteString(fmt.Sprintf("User Name Error: %v\n", r.UserNameError))
	} else {
		builder.WriteString(fmt.Sprintf("User Name: %s\n", r.UserName))
	}
	if r.DiskInfoError != nil {
		builder.WriteString(fmt.Sprintf("Disk Info Error: %v\n", r.DiskInfoError))
	} else {
		builder.WriteString(fmt.Sprintf("Free Space: %d\n", r.FreeSpace))
		builder.WriteString(fmt.Sprintf("Total Space: %d\n", r.TotalSpace))
		builder.WriteString(fmt.Sprintf("Disk Usage: %f\n", r.DiskUsage))
	}
	if r.LoginRecordsError != nil {
		builder.WriteString(fmt.Sprintf("Login Records Error: %v\n", r.LoginRecordsError))
	} else {
		builder.WriteString("Login Records:\n")
		for _, record := range r.LoginRecords {
			if record.IsRemote {
				builder.WriteString(fmt.Sprintf("\tUser: %s, Active: %t, IP: %s, Login Time: %s",
					record.UserName, record.Active, record.IP, record.LoginTime.Format(time.RFC3339)))
			} else {
				builder.WriteString(fmt.Sprintf("\tUser: %s, Active: %t, Source: %s, Login Time: %s",
					record.UserName, record.Active, record.Source, record.LoginTime.Format(time.RFC3339)))
			}
			if !record.Active {
				builder.WriteString(fmt.Sprintf(", Logout Time: %s\n", record.LogoutTime.Format(time.RFC3339)))
			} else {
				builder.WriteString("\n")
			}
		}
	}
	if r.ConnectivityICMPError != nil {
		builder.WriteString(fmt.Sprintf("Connectivity ICMP Error: %v\n", r.ConnectivityICMPError))
	} else {
		builder.WriteString("Connectivity ICMP:\n")
		for _, status := range r.ConnectivityICMP {
			builder.WriteString(fmt.Sprintf("\tICMP: %s, Status: %t, Latency: %s\n",
				status.RemoteIP, status.Status, status.Latency))
		}
	}
	if r.ConnectivityTCPError != nil {
		builder.WriteString(fmt.Sprintf("Connectivity TCP Error: %v\n", r.ConnectivityTCPError))
	} else {
		builder.WriteString("Connectivity TCP:\n")
		for _, status := range r.ConnectivityTCP {
			builder.WriteString(fmt.Sprintf("\tTCP: %s, Port: %d, Status: %t\n",
				status.RemoteIP, status.Port, status.Status))
		}
	}
	builder.WriteString(fmt.Sprintf("Check Time: %s - %s (%f seconds)\n",
		r.CheckStartTime.Format(time.RFC3339), r.CheckEndTime.Format(time.RFC3339), r.CheckDuration.Seconds()))
	return builder.String()
}

func createClient(ip string, port int, user string, idFile string) (*ssh.Client, error) {
	log.Printf("Creating client to %s:%d", ip, port)
	key, err := os.ReadFile(idFile)
	if err != nil {
		log.Printf("Unable to read private key: %v", err)
		return nil, fmt.Errorf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Printf("Unable to parse private key: %v", err)
		return nil, fmt.Errorf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", ip, port), config)
	if err != nil {
		log.Printf("Unable to connect: %v", err)
		return nil, fmt.Errorf("unable to connect: %v", err)
	}

	return client, nil
}

func (c *MonitoringConfig) PerformChecks(connConfig ConnectivityConfig) MonitoringResult {
	log.Printf("Performing checks for %s", c.NodeName)
	result := MonitoringResult{
		NodeCfgName:    c.NodeName,
		CheckStartTime: time.Now(),
	}

	client, err := createClient(c.IP, c.Port, c.UserName, c.IDFile)
	if err != nil {
		result.SSHError = err
		return result
	}
	defer client.Close()

	log.Printf("[%s] Getting node name", c.NodeName)
	result.NodeName, result.HostNameError = getNodeName(client)

	log.Printf("[%s] Getting user name", c.NodeName)
	result.UserName, result.UserNameError = getUserName(client)

	log.Printf("[%s] Getting disk info", c.NodeName)
	result.FreeSpace, result.TotalSpace, result.DiskUsage, result.DiskInfoError = getDiskInfo(client)

	log.Printf("[%s] Getting login records", c.NodeName)
	result.LoginRecords, result.LoginRecordsError = getLoginRecords(client)

	log.Printf("[%s] Getting connectivity (ICMP)", c.NodeName)
	result.ConnectivityICMP, result.ConnectivityICMPError = getConnectivityICMP(client, connConfig.ICMP)

	log.Printf("[%s] Getting connectivity (TCP)", c.NodeName)
	result.ConnectivityTCP, result.ConnectivityTCPError = getConnectivityTCP(client, connConfig.TCP)

	result.CheckEndTime = time.Now()
	result.CheckDuration = result.CheckEndTime.Sub(result.CheckStartTime)
	return result
}
