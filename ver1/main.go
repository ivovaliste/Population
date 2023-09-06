package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
	"ver1/models"

	"github.com/Pallinder/go-randomdata"
)

// üks töögrupp kõik teevad sama asja. loob kasutaja saab id ja teeb 3 kaarti
func main() {
	apiUrl := "http://localhost:8080/user"
	userCount := 10000 // Total number of users to create( user = 1 user ja 3 kaarti)
	workerCount := 4   // Number of worker goroutines

	var wg sync.WaitGroup
	userChannel := make(chan int)
	startTime := time.Now()

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker(apiUrl, userChannel, &wg)
	}

	for i := 0; i < userCount; i++ {
		userChannel <- i
	}

	close(userChannel)
	wg.Wait()

	elapsedTime := time.Since(startTime)
	elapsedHours := int(elapsedTime.Hours())
	elapsedMinutes := int(elapsedTime.Minutes()) % 60
	elapsedSeconds := int(elapsedTime.Seconds()) % 60

	fmt.Printf("\nFinished in %02d:%02d:%02d\n", elapsedHours, elapsedMinutes, elapsedSeconds)
}

func worker(apiUrl string, userChannel <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	for userCount := range userChannel {
		newUser := generateRandomUser()
		jsonData, err := json.Marshal(newUser)
		if err != nil {
			fmt.Println("Error marshaling JSON:", err)
			return
		}

		resp, err := http.Post(apiUrl, "application/json", bytes.NewBuffer(jsonData))
		if err != nil {
			fmt.Println("Error sending HTTP request:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Failed to create user #%d, status code: %d\n", userCount+1, resp.StatusCode)
		} else {
			var responseMap map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&responseMap); err != nil {
				fmt.Println("Error decoding response JSON:", err)
				return
			}

			message, ok := responseMap["message"].(string)
			if !ok {
				fmt.Println("Error extracting message from response JSON")
				return
			}
			userID := extractUserID(message)
			makeAddCardRequests(userID)
			fmt.Printf("User #%d created successfully with ID: %d\n", userCount+1, userID)
		}
	}
}

func generateRandomUser() models.User {
	return models.User{
		FirstName: randomdata.FirstName(randomdata.RandomGender),
		LastName:  randomdata.LastName(),
		UserPhone: randomdata.PhoneNumber(),
		Email:     randomdata.Email(),
	}
}

func extractUserID(message string) int {
	var userID int
	_, err := fmt.Sscanf(message, "New user created with ID: %d", &userID)
	if err != nil {
		fmt.Println("Error extracting user ID from message:", err)
	}
	return userID
}

func makeAddCardRequests(userID int) {
	apiUrl := fmt.Sprintf("http://localhost:8080/card/%d", userID)

	for i := 0; i < 3; i++ {
		resp, err := http.Post(apiUrl, "", nil) // No request body needed
		if err != nil {
			fmt.Println("Error sending HTTP request:", err)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			fmt.Printf("Failed to create card #%d for user #%d, status code: %d\n", i+1, userID, resp.StatusCode)
		}
	}
}
