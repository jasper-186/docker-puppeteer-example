package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/sync/errgroup"
)

type Context struct {
	CallbackUrl  string `json:"callbackurl"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	SecretAnswer string `json:"secretanswer"`
}

type BrowserlessParameters struct {
	Code    string  `json:"code"`
	Context Context `json:"context"`
}

type CallbackParameters struct {
	PageContent string `json:"pageContent"`
	SessionId   string `json:"sessionId"`
}

// Verizon URL To Call
//https://myvprepay.verizon.com/prepaid/ui/mobile/index.html#/user/landing
func sendRequest() {
	browserlessHost := os.Getenv("BROWSERLESSHOST")
	browserlessUrl := fmt.Sprintf("https://%s/function", browserlessHost)

	// Verizon Credentials from Secrets
	verizonUserBytes, fileErr := os.ReadFile("/run/secrets/verizon_account")
	if fileErr != nil {
		log.Fatal("Failed to read verizon account from secret (verizon_account)")
	}

	verizonPasswordBytes, fileErr := os.ReadFile("/run/secrets/verizon_password")
	if fileErr != nil {
		log.Fatal("Failed to read verizon password from secret (verizon_password)")
	}

	verizonQuestionAnswerBytes, fileErr := os.ReadFile("/run/secrets/verizon_question_answer")
	if fileErr != nil {
		log.Fatal("Failed to read verizon secret answer from secret (verizon_password)")
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

	// build the context for the functioncall
	context := Context{
		CallbackUrl:  fmt.Sprintf("https://%s:8080/verizoncallback", hostname),
		Username:     string(verizonUserBytes),
		Password:     string(verizonPasswordBytes),
		SecretAnswer: string(verizonQuestionAnswerBytes),
	}

	// build the parameter object
	parameter := BrowserlessParameters{
		Code:    string(browserlessFunction),
		Context: context,
	}

	jsonparameters, _ := json.Marshal(parameter)
	// call the server
	resp, err := http.Post(browserlessUrl, "application/json", bytes.NewBuffer(jsonparameters))
	if err != nil {
		log.Print("Failed to create new request.")
		log.Fatal(err.Error())
	}

	defer resp.Body.Close()
}

func handler(w http.ResponseWriter, r *http.Request) {
	// Handle the POST Request with the <i>data</i>   heheheheheh
	// Currently just write it to a timestamped file in /output

	switch r.Method {
	case "POST":
		var dat CallbackParameters
		// check that its a valid callback
		if r.Body == nil {
			log.Print("Cant process an empty body")
			http.Error(w, "Cant process an empty body", http.StatusBadRequest)
			return
		}

		// get the POSTED body
		body, _ := ioutil.ReadAll(r.Body)
		if err := json.Unmarshal(body, &dat); err != nil {
			log.Print(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		if dat.PageContent == "" {
			log.Print("Page content is empty")
			http.Error(w, "Page content is empty", http.StatusBadRequest)
		}

		filename := fmt.Sprintf("/output/page_%s.html", time.Now().UTC().Format("2006_01_02_03_04"))
		fileErr := os.WriteFile(filename, []byte(dat.PageContent), 0644)
		if fileErr != nil {
			log.Print(fileErr)
			http.Error(w, fileErr.Error(), http.StatusInternalServerError)
		}

	default:
		fmt.Fprintf(w, "Sorry, Only POST methods are supported.")
		http.Error(w, "Sorry, Only POST methods are supported.", http.StatusBadRequest)
	}
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	// setup ticket to make request 1/hour [1/300sec for debug]
	ticker := time.NewTicker(300 * time.Second)
	done := make(chan bool)

	go func() {
		// setup for graceful shutdown
		osStopCall := make(chan os.Signal, 1)
		signal.Notify(osStopCall, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

		<-osStopCall
		// stop the repeat call timer
		ticker.Stop()
		// release the function to complete
		done <- true
		// stop the http server
		cancel()
	}()

	go func() {
		for {
			select {
			case <-done:
				return
			case t := <-ticker.C:
				log.Printf("%s - sending request to Verizon", t)
				sendRequest()
			}
		}
	}()
	//json.NewDecoder(resp.Body).Decode(&serverresult)

	// Listen for Callback
	http.HandleFunc("/verizoncallback", handler)
	httpServer := &http.Server{
		Addr: ":8080",
	}

	g, gCtx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return httpServer.ListenAndServe()
	})

	g.Go(func() error {
		<-gCtx.Done()
		return httpServer.Shutdown(context.Background())
	})

	if err := g.Wait(); err != nil {
		fmt.Printf("exit reason: %s \n", err)
	}
}
