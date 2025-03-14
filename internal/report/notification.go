// send notification to UI when contract breach detected
package report

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/grid-stream-org/validator/internal/types"
)

func SendFaultNotification(fault types.FaultNotification) {
	const apiURL = "https://api.gridstream.app/v1/notifications"
	jsonData, err := json.Marshal(fault)
	if err != nil {
		log.Println("Error marshalling JSON:", err)
		return
	}

	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Println("Error sending fault notification:", err)
		return
	}
	defer resp.Body.Close()

	log.Println("Fault notification sent successfully, response:", resp.Status)
}
