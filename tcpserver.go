package main

import (
	"fmt"
	"net"
	"os"
	"tcp/azurelogger"
	"time"
)

const (
	HOST = "0.0.0.0"
	PORT = "80"
	TYPE = "tcp"
)

type connectionStats struct {
	connectionCountByHost map[string]int
	lastSent              time.Time
	totalConnections      int64
}

func (stats *connectionStats) updateTimestamp() {
	stats.lastSent = time.Now()
	stats.connectionCountByHost = make(map[string]int)
}

func (stats *connectionStats) increaseTotalConnections() {
	stats.totalConnections++
}

func (stats *connectionStats) decreaseTotalConnections() {
	if stats.totalConnections > 0 {
		stats.totalConnections--
	}
}

func serverHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return hostname
}

func sendStats(connStats *connectionStats, logger azurelogger.AzureLogAnalytics) {
	for range time.Tick(time.Second * 60) {
		for host, count := range connStats.connectionCountByHost {
			logger.PostData(fmt.Sprintf("[{ 'server_host': '%v', 'remote_host': '%v', 'count_of_messages': '%v' }]\n", serverHostname(), host, count))
		}
		logger.PostData(fmt.Sprintf("[{ 'server_host': '%v', 'total_connections': '%v' }]\n", serverHostname(), connStats.totalConnections))
		connStats.updateTimestamp()
	}
}

func collectStats(connectionStatsChannel <-chan string, connStats *connectionStats, logger azurelogger.AzureLogAnalytics) {
	go sendStats(connStats, logger)
	for {
		host := <-connectionStatsChannel
		if host != "error" {
			connStats.connectionCountByHost[host]++
		}
	}
}

func main() {
	l, err := net.Listen(TYPE, HOST+":"+PORT)

	connectionStats := connectionStats{
		connectionCountByHost: make(map[string]int),
		lastSent:              time.Now(),
		totalConnections:      0,
	}

	logger := azurelogger.AzureLogAnalytics{
		CustomerId:     "<id>",
		SharedKey:      "<key>",
		LogType:        "TcpServer",
		TimeStampField: "DateValue",
	}

	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}
	fmt.Println("Listening on " + HOST + ":" + PORT)
	statsChannel := make(chan string)
	go collectStats(statsChannel, &connectionStats, logger)
	for {
		conn, err := l.Accept()
		conn.SetDeadline(time.Now().Add(time.Hour))
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			connectionStats.decreaseTotalConnections()
			logger.PostData(fmt.Sprintf("[{ 'Operation': 'Error accepting connection: %v', 'server_host': '%v' }]\n", err.Error(), serverHostname()))
		} else {
			connectionStats.increaseTotalConnections()
			go handleRequest(&connectionStats, statsChannel, logger, conn)
		}
	}
}

func handleRequest(stats *connectionStats, statsChannel chan<- string, logger azurelogger.AzureLogAnalytics, conn net.Conn) {
	buf := make([]byte, 256)
	var consecutiveCallsForConnection = 0
	for {
		length, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Error reading:", err.Error())
			stats.decreaseTotalConnections()
			logger.PostData(fmt.Sprintf("[{ 'Operation': 'Error reading from client, closing connection: %v', 'server_host': '%v', 'remote_host': '%v' }]\n", err.Error(), serverHostname(), conn.RemoteAddr().String()))
			conn.Close()
			return
		} else {
			consecutiveCallsForConnection++
			logger.PostData(fmt.Sprintf("[{ 'Operation': 'Connection Read', 'consecutive_call': '%v', 'remote_addr': '%v', 'local_addr': '%v', 'server_host': '%v' }]", consecutiveCallsForConnection, conn.RemoteAddr().String(), conn.LocalAddr().String(), serverHostname()))
			received := string(buf[:length])
			statsChannel <- received
		}
	}
}
