package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
	"ver1/models"

	"gitlab.com/avarf/getenvs"

	"github.com/Pallinder/go-randomdata"
)

type UserPopulation struct {
	APIEndpoint string
	UserCount   int
	WorkerCount int
	UserChannel chan int
	WaitGroup   sync.WaitGroup
	StartTime   time.Time
}

func NewUserPopulation(apiURL string, userCount, workerCount int) *UserPopulation {
	return &UserPopulation{
		APIEndpoint: apiURL,
		UserCount:   userCount,
		WorkerCount: workerCount,
		UserChannel: make(chan int),
	}
}

func (up *UserPopulation) Start() {
	up.StartTime = time.Now()
	for i := 0; i < up.WorkerCount; i++ {
		up.WaitGroup.Add(1)
		go up.worker()
	}

	for i := 0; i < up.UserCount; i++ {
		up.UserChannel <- i
	}

	close(up.UserChannel)

	up.WaitGroup.Wait()

	elapsedTime := time.Since(up.StartTime)
	elapsedHours := int(elapsedTime.Hours())
	elapsedMinutes := int(elapsedTime.Minutes()) % 60
	elapsedSeconds := int(elapsedTime.Seconds()) % 60

	fmt.Printf("\nFinished in %02d:%02d:%02d\n", elapsedHours, elapsedMinutes, elapsedSeconds)
}

func (up *UserPopulation) worker() {
	apiURL := fmt.Sprintf("%s/user", up.APIEndpoint)
	defer up.WaitGroup.Done()

	for userCount := range up.UserChannel {
		newUser := generateRandomUser()
		jsonData, err := json.Marshal(newUser)
		if err != nil {
			fmt.Println("Error marshaling JSON:", err)
			return
		}

		resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
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
			up.makeAddCardRequests(userID)
			fmt.Printf("User #%d created successfully with ID: %d\n", userCount+1, userID)
		}
	}
}

func (up *UserPopulation) makeAddCardRequests(userID int) {
	apiURL := fmt.Sprintf("%s/card/%d", up.APIEndpoint, userID)

	for i := 0; i < 3; i++ {
		resp, err := http.Post(apiURL, "", nil) // No request body needed
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

func main() {
	apiURL := getenvs.GetEnvString("APIURL", "http://localhost:8080")
	userCount, _ := getenvs.GetEnvInt("USERCOUNT", 10000)
	workerCount, _ := getenvs.GetEnvInt("WORKERCOUNT", 4)

	userPopulation := NewUserPopulation(apiURL, userCount, workerCount)
	userPopulation.Start()
}
