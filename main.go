package main

import (
	"bytes"
	"log"
	"net/http"
	"os"
)

// Verizon URL To Call
//https://myvprepay.verizon.com/prepaid/ui/mobile/index.html#/user/landing

func handler(w http.ResponseWriter, r *http.Request) {
	// write to a database?
}

func main() {
	browserlessUrl := os.Getenv("BROWSERLESSURL")
	browserlessUrl += "/function"

	// Verizon Credentials from Secrets
	verizonUserBytes, fileErr := os.ReadFile("/run/secrets/verizon_account")
	if fileErr != nil {
		log.Fatal("Failed to read verizon account from secret (verizon_account)")
	}

	verizonPasswordBytes, fileErr := os.ReadFile("/run/secrets/verizon_password")
	if fileErr != nil {
		log.Fatal("Failed to read verizon password from secret (verizon_password)")
	}

	// Read the browserless code into mem
	browserlessFunction, fileErr := os.ReadFile("verizon-login.json")
	if fileErr != nil {
		log.Fatal("Failed to read browserlesscode Bailing")
	}

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	// replace username/pass in the function with the values from the secrets
	browserlessFunction = bytes.ReplaceAll(browserlessFunction, []byte("##USERNAME##"), verizonUserBytes)
	browserlessFunction = bytes.ReplaceAll(browserlessFunction, []byte("##PASSWORD##"), verizonPasswordBytes)
	browserlessFunction = bytes.ReplaceAll(browserlessFunction, []byte("##HOSTNAME##"), []byte(hostname))

	// call the server
	resp, err := http.Post(browserlessUrl, "application/json", bytes.NewBuffer(browserlessFunction))
	if err != nil {
		log.Print("Failed to create new request.")
		log.Fatal(err.Error())
	}

	defer resp.Body.Close()

	//json.NewDecoder(resp.Body).Decode(&serverresult)

	// Listen for Callback
	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
