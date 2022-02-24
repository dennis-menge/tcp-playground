package azurelogger

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type AzureLogAnalytics struct {
	// Replace with your Workspace ID
	CustomerId string
	// Replace with your Primary or Secondary ID
	SharedKey string
	//Specify the name of the record type that you'll be creating
	LogType string
	//Specify a field with the created time for the records
	TimeStampField string
}

func buildSignature(message, secret string) (string, error) {

	keyBytes, err := base64.StdEncoding.DecodeString(secret)
	if err != nil {
		return "", err
	}

	mac := hmac.New(sha256.New, keyBytes)
	mac.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

func (m AzureLogAnalytics) PostData(data string) error {

	println(data)

	dateString := time.Now().UTC().Format(time.RFC1123)
	dateString = strings.Replace(dateString, "UTC", "GMT", -1)

	stringToHash := "POST\n" + strconv.Itoa(len(data)) + "\napplication/json\n" + "x-ms-date:" + dateString + "\n/api/logs"
	hashedString, err := buildSignature(stringToHash, m.SharedKey)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	signature := "SharedKey " + m.CustomerId + ":" + hashedString
	url := "https://" + m.CustomerId + ".ods.opinsights.azure.com/api/logs?api-version=2016-04-01"

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, bytes.NewReader([]byte(data)))
	if err != nil {
		return err
	}

	req.Header.Add("Log-Type", m.LogType)
	req.Header.Add("Authorization", signature)
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("x-ms-date", dateString)
	req.Header.Add("time-generated-field", m.TimeStampField)

	resp, err := client.Do(req)
	if err == nil {
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()
		return nil
	}
	return err
}
