package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"tcp/azurelogger"
	"time"
)

func connectInitial(desiredConnectionCount int, logger azurelogger.AzureLogAnalytics, connections *[]net.TCPConn, fn func(connections *[]net.TCPConn, logger azurelogger.AzureLogAnalytics, connectionErrors *int, successfulConnections *int, index int)) {

	var wg sync.WaitGroup
	wg.Add(desiredConnectionCount)

	var connectionErrors = 0
	var successfulConnections = 0

	for i := 0; i < desiredConnectionCount; i++ {
		go func() {
			fn(connections, logger, &connectionErrors, &successfulConnections, i)
			wg.Done()
		}()
	}
	wg.Wait()
	fmt.Printf("All connections created for Host: %v - Successful Connections: %v - Connection Errors: %v\n", hostname(), successfulConnections, connectionErrors)
	logger.PostData(fmt.Sprintf("[{ 'hostname': '%v', 'ConnectionSuccess': '%v', 'ConnectionErrors': '%v' }]\n", hostname(), successfulConnections, connectionErrors))
}

func repair(repairIndices []int, logger azurelogger.AzureLogAnalytics, connections *[]net.TCPConn, fn func(connections *[]net.TCPConn, logger azurelogger.AzureLogAnalytics, connectionErrors *int, successfulConnections *int, index int)) {
	var wg sync.WaitGroup
	var newConnections = len(repairIndices)
	var connectionErrors = 0
	var successfulConnections = 0
	wg.Add(newConnections)
	for i := range repairIndices {
		index := repairIndices[i]
		go func() {
			fn(connections, logger, &connectionErrors, &successfulConnections, index)
			wg.Done()
		}()
	}
	wg.Wait()
	logger.PostData(fmt.Sprintf("[{ 'Operation': 'Restored broken connections', 'hostname': '%v', 'ConnectionSuccess': '%v', 'ConnectionErrors': '%v' }]\n", hostname(), successfulConnections, connectionErrors))
}

func hostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return hostname
}

func connect(connections *[]net.TCPConn, logger azurelogger.AzureLogAnalytics, connectionErrors *int, successfulConnections *int, index int) {

	tcpAddr, _ := net.ResolveTCPAddr("tcp4", "10.1.0.4:80")

	c, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		fmt.Println(fmt.Errorf("Error connecting to host: %w", err))
		*connectionErrors++
	} else {
		c.SetKeepAlive(true)
		c.SetKeepAlivePeriod(20 * time.Second)
		*successfulConnections++
		(*connections)[index] = *c
	}
}

func main() {

	desiredConnections := 8000
	connections := make([]net.TCPConn, desiredConnections)
	logger := azurelogger.AzureLogAnalytics{
		CustomerId:     "<id>",
		SharedKey:      "<key>",
		LogType:        "TcpClient",
		TimeStampField: "DateValue",
	}

	connectInitial(desiredConnections-1, logger, &connections, connect)

	for {
		var successfulPings = 0
		var brokenConnectionIndices = make([]int, 0)
		for i := range connections {
			_, err := connections[i].Write([]byte(fmt.Sprintf("%v", hostname())))
			if err == nil {
				successfulPings++
			} else {
				log.Print(fmt.Sprintln("Error while writing to server: ", err))
				brokenConnectionIndices = append(brokenConnectionIndices, i)
			}
		}
		logger.PostData(fmt.Sprintf("[{ 'hostname': '%v', 'SuccessfulPings': '%v' }]\n", hostname(), successfulPings))
		if successfulPings < desiredConnections {
			logger.PostData(fmt.Sprintf("[{ 'hostname': '%v', 'Restoring Connections': '%v' }]\n", hostname(), desiredConnections-successfulPings))
			repair(brokenConnectionIndices, logger, &connections, connect)
		}
		time.Sleep(10 * time.Second)
	}
}
