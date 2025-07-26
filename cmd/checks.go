package main

import (
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

type MonitoringResult struct {
	NodeCfgName           string                        `json:"node_cfg_name"`
	NodeName              string                        `json:"hostname"`
	UserName              string                        `json:"user"`
	FreeSpace             int64                         `json:"free_space"`
	TotalSpace            int64                         `json:"total_space"`
	DiskUsage             float64                       `json:"disk_usage"`
	LoginRecords          []UserLoginRecord             `json:"login_records"`
	ConnectivityICMP      []ConnectivityStatusICMP      `json:"-"`
	ConnectivityTCP       []ConnectivityStatusTCP       `json:"-"`
	ConnectivityHTTP      []ConnectivityStatusHTTP      `json:"-"`
	Connectivity          map[string]ConnectivityStatus `json:"connectivity"`
	CheckStartTime        time.Time                     `json:"check_start_time"`
	CheckEndTime          time.Time                     `json:"check_end_time"`
	CheckDuration         float64                       `json:"check_duration"`
	SSHError              error                         `json:"ssh_error,omitempty"`
	HostNameError         error                         `json:"host_name_error,omitempty"`
	UserNameError         error                         `json:"user_name_error,omitempty"`
	DiskInfoError         error                         `json:"disk_info_error,omitempty"`
	LoginRecordsError     error                         `json:"login_records_error,omitempty"`
	ConnectivityICMPError error                         `json:"connectivity_icmp_error,omitempty"`
	ConnectivityTCPError  error                         `json:"connectivity_tcp_error,omitempty"`
	ConnectivityHTTPError error                         `json:"connectivity_http_error,omitempty"`
	ConnectivityError     error                         `json:"connectivity_error,omitempty"`
}

type ConnectivityStatusICMP struct {
	Name     string
	RemoteIP string        `json:"remote_ip"`
	Status   bool          `json:"status"`
	Error    string        `json:"error,omitempty"`
	Latency  time.Duration `json:"latency"`
}

type ConnectivityStatusTCP struct {
	Name     string
	RemoteIP string `json:"remote_ip"`
	Port     int    `json:"port"`
	Status   bool   `json:"status"`
}

type ConnectivityStatusHTTP struct {
	Name   string
	Host   string `json:"host"`
	Status bool   `json:"status"`
	Code   int    `json:"code"`
	Error  string `json:"error,omitempty"`
}

type ConnectivityStatus struct {
	TCP  []ConnectivityStatusTCP  `json:"tcp,omitempty"`
	ICMP []ConnectivityStatusICMP `json:"icmp,omitempty"`
	HTTP []ConnectivityStatusHTTP `json:"http,omitempty"`
}

type UserLoginRecord struct {
	UserName   string    `json:"username"`
	Active     bool      `json:"is_active"`
	IsRemote   bool      `json:"is_remote"`
	IP         string    `json:"ip"`
	Source     string    `json:"source"`
	LoginTime  time.Time `json:"login_time"`
	LogoutTime time.Time `json:"logout_time"`
}

func (m *MonitoringConfig) getNodeName(client *ssh.Client) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("hostname")
	if err != nil {
		return "", fmt.Errorf("failed to execute hostname command: %v", err)
	}
	hostname := strings.TrimSpace(string(output))
	if hostname == "" {
		return "", fmt.Errorf("hostname command returned empty result")
	}
	return hostname, nil
}

func (m *MonitoringConfig) getUserName(client *ssh.Client) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("whoami")
	if err != nil {
		return "", fmt.Errorf("failed to execute command: %v", err)
	}
	username := strings.TrimSpace(string(output))
	if username == "" {
		return "", fmt.Errorf("whoami command returned empty result")
	}
	return username, nil
}

func (m *MonitoringConfig) getDiskInfo(client *ssh.Client) (int64, int64, float64, error) {
	session, err := client.NewSession()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("df -h / | awk 'NR==2 {print $2 \" \" $4 \" \" $5}'")
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to execute command: %v", err)
	}

	fields := strings.Fields(string(output))
	if len(fields) != 3 {
		return 0, 0, 0, fmt.Errorf("unexpected output format: %s", string(output))
	}

	totalSpace, err := parseHumanReadableSize(fields[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse total space: %v", err)
	}

	freeSpace, err := parseHumanReadableSize(fields[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse free space: %v", err)
	}

	diskUsageStr := strings.TrimSuffix(fields[2], "%")
	diskUsage, err := strconv.ParseFloat(diskUsageStr, 64)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to parse disk usage: %v", err)
	}

	return totalSpace, freeSpace, diskUsage, nil
}

func parseHumanReadableSize(sizeStr string) (int64, error) {
	multiplier := 1
	if strings.HasSuffix(sizeStr, "K") {
		multiplier = 1024
	} else if strings.HasSuffix(sizeStr, "M") {
		multiplier = 1024 * 1024
	} else if strings.HasSuffix(sizeStr, "G") {
		multiplier = 1024 * 1024 * 1024
	} else if strings.HasSuffix(sizeStr, "T") {
		multiplier = 1024 * 1024 * 1024 * 1024
	} else if strings.HasSuffix(sizeStr, "P") {
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024
	} else if strings.HasSuffix(sizeStr, "E") {
		multiplier = 1024 * 1024 * 1024 * 1024 * 1024 * 1024
	} else if !strings.HasSuffix(sizeStr, "B") {
		return 0, fmt.Errorf("unsupported size format: %s", sizeStr)
	}

	size, err := strconv.ParseInt(strings.TrimRight(sizeStr, "BKMGTPE"), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size: %v", err)
	}

	return size * int64(multiplier), nil
}

func (m *MonitoringConfig) getLoginRecords(client *ssh.Client) ([]UserLoginRecord, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %v", err)
	}
	defer session.Close()

	output, err := session.CombinedOutput("last --time-format=iso")
	if err != nil {
		return nil, fmt.Errorf("failed to execute command: %v", err)
	}

	lines := strings.Split(string(output), "\n")
	var records []UserLoginRecord

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "wtmp begins") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}

		userName := fields[0]
		terminal := fields[1]
		src := fields[2]
		isRemote := true
		ip := ""
		if parsedIP := net.ParseIP(src); parsedIP != nil {
			ip = parsedIP.String()
		} else if strings.HasPrefix(src, "[") && strings.HasSuffix(src, "]") {
			if parsedIP := net.ParseIP(strings.Trim(src, "[]")); parsedIP != nil {
				ip = parsedIP.String()
			}
		}
		if ip == "" {
			isRemote = false
		}

		if userName == "reboot" || strings.Contains(terminal, "system") {
			continue
		}

		loginTime, _ := time.Parse("2006-01-02T15:04:05-07:00", fields[3])
		logoutTime, err := time.Parse("2006-01-02T15:04:05-07:00", fields[5])
		if err != nil {
			if len(fields) >= 7 && fields[5] == "down" {
				timeDiff := fields[6] //  (00:04) or (94+18:58)
				timeDiffParts := strings.Split(timeDiff, "+")
				days := 0
				hoursMinutes := timeDiffParts[len(timeDiffParts)-1]
				timeParts := strings.Split(hoursMinutes, ":")
				hours, _ := strconv.Atoi(timeParts[0])
				minutes, _ := strconv.Atoi(timeParts[1])
				if len(timeDiffParts) > 1 {
					days, _ = strconv.Atoi(timeDiffParts[0])
				}
				logoutTime = loginTime.Add(
					time.Duration(days)*24*time.Hour +
						time.Duration(hours)*time.Hour +
						time.Duration(minutes)*time.Minute,
				)
			}
		}

		record := UserLoginRecord{
			UserName:   userName,
			IP:         ip,
			IsRemote:   isRemote,
			Source:     src,
			LoginTime:  loginTime,
			LogoutTime: logoutTime,
			Active:     false,
		}

		record.Active = len(fields) >= 5 && (fields[4] == "still" || fields[4] == "gone")

		records = append(records, record)
	}

	return records, nil
}

func (m *MonitoringConfig) getConnectivityICMP(client *ssh.Client, endpoints []ICMPEndpoint) ([]ConnectivityStatusICMP, error) {
	statuses := []ConnectivityStatusICMP{}

	for _, endpoint := range endpoints {
		log.Printf("[%s] Getting ICMP connectivity for %s", m.NodeName, endpoint.Name)
		session, err := client.NewSession()
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %v", err)
		}
		defer session.Close()

		rawPingData, err := session.CombinedOutput(fmt.Sprintf("ping -c 5 -W 1 %s", endpoint.Address))
		if err != nil {
			log.Printf("Failed to execute command: %v", err)
			statuses = append(statuses, ConnectivityStatusICMP{
				Name:     endpoint.Name,
				RemoteIP: endpoint.Address,
				Status:   false,
				Error:    err.Error(),
			})
			continue
		}

		lines := strings.Split(string(rawPingData), "\n")

		count := 0
		totalTime := 0.0
		minTime := 0.0
		maxTime := 0.0

		for _, line := range lines {
			if strings.Contains(line, "bytes from") {
				fields := strings.Fields(line)
				for _, field := range fields {
					if strings.HasPrefix(field, "time=") {
						timeStr := field[5:]
						timeMs, err := strconv.ParseFloat(timeStr, 64)
						if err != nil {
							continue
						}
						totalTime += timeMs
						count++
						if minTime == 0 || timeMs < minTime {
							minTime = timeMs
						}
						if timeMs > maxTime {
							maxTime = timeMs
						}
					}
				}
			}
		}

		if count > 0 {
			avgTime := totalTime / float64(count)
			statuses = append(statuses, ConnectivityStatusICMP{
				Name:     endpoint.Name,
				RemoteIP: endpoint.Address,
				Status:   true,
				Latency:  time.Duration(avgTime) * time.Millisecond,
			})
		} else {
			statuses = append(statuses, ConnectivityStatusICMP{
				Name:     endpoint.Name,
				RemoteIP: endpoint.Address,
				Status:   false,
				Latency:  0,
			})
		}
	}

	return statuses, nil
}

func (m *MonitoringConfig) getConnectivityTCP(client *ssh.Client, endpoints []TCPEndpoint) ([]ConnectivityStatusTCP, error) {
	statuses := []ConnectivityStatusTCP{}

	for _, endpoint := range endpoints {
		log.Printf("[%s] Getting ICMP connectivity for %s", m.NodeName, endpoint.Name)
		session, err := client.NewSession()
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %v", err)
		}
		defer session.Close()

		cmd := fmt.Sprintf("timeout 3 bash -c '</dev/tcp/%s/%d' && echo 'true' || echo 'false'", endpoint.Address, endpoint.Port)
		output, err := session.CombinedOutput(cmd)

		if err == nil && strings.TrimSpace(string(output)) == "true" {
			statuses = append(statuses, ConnectivityStatusTCP{
				Name:     endpoint.Name,
				RemoteIP: endpoint.Address,
				Port:     endpoint.Port,
				Status:   true,
			})
		} else {
			statuses = append(statuses, ConnectivityStatusTCP{
				Name:     endpoint.Name,
				RemoteIP: endpoint.Address,
				Port:     endpoint.Port,
				Status:   false,
			})
		}
	}

	return statuses, nil
}

func (m *MonitoringConfig) getConnectivityHTTP(client *ssh.Client, endpoints []HTTPEndpoint) ([]ConnectivityStatusHTTP, error) {
	statuses := []ConnectivityStatusHTTP{}

	for _, endpoint := range endpoints {
		log.Printf("[%s] Getting TCP connectivity for %s", m.NodeName, endpoint.Name)
		session, err := client.NewSession()
		if err != nil {
			return nil, fmt.Errorf("failed to create session: %v", err)
		}
		defer session.Close()
		currentStatus := ConnectivityStatusHTTP{
			Name:   endpoint.Name,
			Host:   endpoint.Address,
			Status: false,
			Code:   0,
			Error:  "",
		}
		cmd := fmt.Sprintf("curl -s -o /dev/null  --connect-timeout 5 --max-time 10 -w \"%%{http_code}\" %s", endpoint.Address)
		output, err := session.CombinedOutput(cmd)
		if err != nil {
			currentStatus.Error = err.Error()
			statuses = append(statuses, currentStatus)
			continue
		}

		code, err := strconv.Atoi(strings.TrimSpace(string(output)))
		if err != nil {
			currentStatus.Error = err.Error()
			statuses = append(statuses, currentStatus)
			continue
		}
		currentStatus.Code = code
		currentStatus.Status = true
		statuses = append(statuses, currentStatus)
	}
	return statuses, nil
}

func (m *MonitoringConfig) getConnectivity(client *ssh.Client, tcpEndpoints []TCPEndpoint, icmpEndpoints []ICMPEndpoint, httpEndpoints []HTTPEndpoint) (map[string]ConnectivityStatus, error) {
	log.Printf("[%s] Getting TCP connectivity", m.NodeName)
	tcpStatuses, err := m.getConnectivityTCP(client, tcpEndpoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get TCP connectivity: %v", err)
	}
	log.Printf("[%s] Getting ICMP connectivity", m.NodeName)
	icmpStatuses, err := m.getConnectivityICMP(client, icmpEndpoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get ICMP connectivity: %v", err)
	}
	log.Printf("[%s] Getting HTTP connectivity", m.NodeName)
	httpStatuses, err := m.getConnectivityHTTP(client, httpEndpoints)
	if err != nil {
		return nil, fmt.Errorf("failed to get HTTP connectivity: %v", err)
	}

	connectivity := make(map[string]ConnectivityStatus)
	for _, tcpStatus := range tcpStatuses {
		currentConn, ok := connectivity[tcpStatus.Name]
		if !ok {
			currentConn = ConnectivityStatus{
				TCP:  []ConnectivityStatusTCP{},
				ICMP: []ConnectivityStatusICMP{},
				HTTP: []ConnectivityStatusHTTP{},
			}
		}
		currentConn.TCP = append(currentConn.TCP, tcpStatus)
		connectivity[tcpStatus.Name] = currentConn
	}

	for _, icmpStatus := range icmpStatuses {
		currentConn, ok := connectivity[icmpStatus.Name]
		if !ok {
			currentConn = ConnectivityStatus{
				TCP:  []ConnectivityStatusTCP{},
				ICMP: []ConnectivityStatusICMP{},
				HTTP: []ConnectivityStatusHTTP{},
			}
		}
		currentConn.ICMP = append(currentConn.ICMP, icmpStatus)
		connectivity[icmpStatus.Name] = currentConn
	}

	for _, httpStatus := range httpStatuses {
		currentConn, ok := connectivity[httpStatus.Name]
		if !ok {
			currentConn = ConnectivityStatus{
				TCP:  []ConnectivityStatusTCP{},
				ICMP: []ConnectivityStatusICMP{},
				HTTP: []ConnectivityStatusHTTP{},
			}
		}
		currentConn.HTTP = append(currentConn.HTTP, httpStatus)
		connectivity[httpStatus.Name] = currentConn
	}

	return connectivity, nil
}
