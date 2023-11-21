package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
	"ver2/models"

	"gitlab.com/avarf/getenvs"

	"github.com/Pallinder/go-randomdata"
)

// user annab id edasi teisele töörühmale ja teeb ise uue kasutaja. Teine töögrupp teeb kaarte
func main() {
	apiEp := getenvs.GetEnvString("APIURL", "http://localhost:8080")
	userCount, _ := getenvs.GetEnvInt("USERCOUNT", 10000)         // Total number of users to create( user = 1 user ja 3 kaarti)
	workerCount, _ := getenvs.GetEnvInt("WORKERCOUNT", 3)         // Number of worker goroutines
	cardWorkerCount, _ := getenvs.GetEnvInt("CARDWORKERCOUNT", 6) // Number of worker goroutines

	var wg sync.WaitGroup
	userChannel := make(chan int, userCount)
	cardChannel := make(chan int, userCount*3) // Each user creates 3 cards

	startTime := time.Now()

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go userCreationWorker(apiEp, userChannel, cardChannel, &wg)
	}

	var cardWg sync.WaitGroup
	for i := 0; i < cardWorkerCount; i++ {
		cardWg.Add(1)
		go cardCreationWorker(apiEp, cardChannel, &cardWg)
	}

	for i := 0; i < userCount; i++ {
		userChannel <- i
	}
	close(userChannel)

	wg.Wait()

	close(cardChannel)
	cardWg.Wait()

	elapsedTime := time.Since(startTime)
	elapsedHours := int(elapsedTime.Hours())

	elapsedMinutes := int(elapsedTime.Minutes()) % 60
	elapsedSeconds := int(elapsedTime.Seconds()) % 60

	fmt.Printf("\nFinished in %02d:%02d:%02d\n", elapsedHours, elapsedMinutes, elapsedSeconds)
}

func userCreationWorker(apiEp string, userChannel <-chan int, cardChannel chan<- int, wg *sync.WaitGroup) {
	apiUrl := fmt.Sprintf("%s/user", apiEp)
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
			cardChannel <- userID
			fmt.Printf("User #%d created successfully with ID: %d\n", userCount+1, userID)
		}
	}
}

func cardCreationWorker(apiEp string, cardChannel <-chan int, wg *sync.WaitGroup) {
	defer wg.Done()

	for userID := range cardChannel {
		apiUrl := fmt.Sprintf("%s/card/%d", apiEp, userID)

		for i := 0; i < 3; i++ {
			resp, err := http.Post(apiUrl, "", nil)
			if err != nil {
				fmt.Println("Error sending HTTP request:", err)
				return
			}

			if resp.StatusCode != http.StatusOK {
				fmt.Printf("Failed to create card #%d for user #%d, status code: %d\n", i+1, userID, resp.StatusCode)
			}

			resp.Body.Close()
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
