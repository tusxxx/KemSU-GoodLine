package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
)

type UserLocation struct {
	Bbox  [4]float64 `json:"bbox"`
	Photo string     `json:"photo"` // Expecting base64 encoded image
	User  string     `json:"user"`  // Identifier for the user
}

type Building struct {
	StartDate   string `json:"start_date"`
	Architect   string `json:"architect"`
	Description string `json:"description"`
	Image       string `json:"image"`
	Name        string `json:"name"`
	// Add other relevant fields as necessary
}

var userLocations = make(map[string]UserLocation)

func main() {
	http.HandleFunc("/submit", submitLocation)
	http.HandleFunc("/check", checkLocation)
	http.ListenAndServe(":8080", nil)
}

func submitLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var location UserLocation
	err := json.NewDecoder(r.Body).Decode(&location)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	userLocations[location.User] = location

	buildingData, err := fetchBuildingData(location.Bbox)
	if err != nil {
		http.Error(w, "Error fetching building data", http.StatusInternalServerError)
		return
	}

	buildingImageBase64, err := ImageURLToBase64(buildingData[0].Image)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting image from OSM: %s", err), http.StatusInternalServerError)
		return
	}

	cmd := exec.Command("python", "main.py", location.Photo, buildingImageBase64) // Adjust the command as needed
	output, err := cmd.Output()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error executing Python script: %s", err), http.StatusInternalServerError)
		return
	}

	// Convert output to float64
	result, err := strconv.ParseFloat(string(output), 64)
	if err != nil {
		http.Error(w, "Error parsing Python script output", http.StatusInternalServerError)
		return
	}

	// You can now use 'result' as needed
	fmt.Printf("Result from Python script: %f\n", result)

	// Here you could add code to compare images and find the most similar building
	// using the received 'location.Photo'

	// Sending back the building data as response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(buildingData)
}

func ImageURLToBase64(url string) (string, error) {
	response, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	// Read the image data
	imageData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	// Encode the image data to base64
	base64String := base64.StdEncoding.EncodeToString(imageData)
	return base64String, nil
}

func checkLocation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	user := r.URL.Query().Get("user")
	location, exists := userLocations[user]
	if !exists {
		http.Error(w, "No data found for user", http.StatusNotFound)
		return
	}

	response, err := json.Marshal(location)
	if err != nil {
		http.Error(w, "Error in processing request", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

func fetchBuildingData(bbox [4]float64) ([]Building, error) {
	// Example API URL, replace with the actual URL for fetching data from OpenStreetMap
	apiURL := fmt.Sprintf("https://www.openstreetmap.org/api/0.6/map?bbox=%f,%f,%f,%f", bbox[0], bbox[1], bbox[2], bbox[3])

	// Create a new request
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	// Set the Accept header
	req.Header.Set("Accept", "application/json")

	// Perform the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get data from OpenStreetMap")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Define a structure to hold the response
	var response struct {
		Elements []struct {
			Tags map[string]string `json:"tags"`
		} `json:"elements"`
	}

	// Unmarshal the body into the response structure
	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	// Filter elements based on required tags
	var buildings []Building
	for _, element := range response.Elements {
		if _, hasDescription := element.Tags["description"]; hasDescription {
			if _, hasArchitect := element.Tags["architect"]; hasArchitect {
				if _, hasImage := element.Tags["image"]; hasImage {
					if _, hasStartDate := element.Tags["start_date"]; hasStartDate {
						// Create a Building instance and populate it
						building := Building{
							Description: element.Tags["description"],
							Architect:   element.Tags["architect"],
							Image:       element.Tags["image"],
							StartDate:   element.Tags["start_date"],
							Name:        element.Tags["name"],
						}
						buildings = append(buildings, building)
					}
				}
			}
		}
	}

	return buildings, nil
}
