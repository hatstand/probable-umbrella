package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/shopspring/decimal"
)

var token = flag.String("token", "", "Personal access token for Starling API")

const (
	apiURL           = "https://api.starlingbank.com/"
	transactionsPath = "api/v1/transactions"
)

type DetailMessage struct {
	HREF      string
	Templated bool
}

type LinksMessage struct {
	Detail DetailMessage `json:"detail"`
}

type Transaction struct {
	ID        string
	Currency  string
	Amount    decimal.Decimal
	Direction string
	Narrative string
	Source    string
	Balance   decimal.Decimal
	Links     LinksMessage `json:"_links"`
}

type TransactionsMessage struct {
	Transactions []Transaction `json:"transactions"`
}

type Message struct {
	Embedded TransactionsMessage `json:"_embedded"`
}

type TransactionDetailMessage struct {
	Amount           decimal.Decimal
	Currency         string
	Direction        string
	Narrative        string
	Source           string
	SpendingCategory string
}

func getCategory(t *TransactionDetailMessage) string {
	if t.SpendingCategory == "" {
		return "UNKNOWN"
	} else {
		return t.SpendingCategory
	}
}

func collateSpending(transactions []*TransactionDetailMessage) map[string]decimal.Decimal {
	ret := make(map[string]decimal.Decimal)
	for _, t := range transactions {
		i, ok := ret[getCategory(t)]
		if ok {
			ret[getCategory(t)] = i.Add(t.Amount)
		} else {
			ret[getCategory(t)] = t.Amount
		}
	}
	return ret
}

func fetchTransaction(HREF string) (*TransactionDetailMessage, error) {
	req, _ := http.NewRequest("GET", apiURL+HREF, nil)
	req.Header.Add("Authorization", "Bearer "+*token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Failed to fetch transaction: %v", err)
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)
	log.Printf("Data: %s\nCode: %d Status: %s\n", data, resp.StatusCode, resp.Status)
	if resp.StatusCode != 200 {
		log.Printf("Failed to fetch transaction details for: %s\n", HREF)
		return nil, fmt.Errorf("Failed to fetch transaction details for: %s", HREF)
	}

	var m TransactionDetailMessage
	err = json.Unmarshal(data, &m)
	if err != nil {
		log.Fatalf("Failed to decode JSON: %v", err)
	}
	log.Printf("JSON: %+v", m)
	return &m, nil
}

func main() {
	flag.Parse()

	req, _ := http.NewRequest("GET", apiURL+transactionsPath, nil)
	req.Header.Add("Authorization", "Bearer "+*token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalf("Failed to fetch transactions: %v", err)
	}
	defer resp.Body.Close()
	data, _ := ioutil.ReadAll(resp.Body)

	var m Message
	err = json.Unmarshal(data, &m)
	if err != nil {
		log.Fatalf("Failed to parse JSON: %v", err)
	}
	log.Printf("JSON: %+v\n", m.Embedded)

	var details []*TransactionDetailMessage
	for _, t := range m.Embedded.Transactions {
		d, _ := fetchTransaction(t.Links.Detail.HREF)
		if d != nil {
			details = append(details, d)
		}
	}

	log.Printf("Collating...\n")
	categorised := collateSpending(details)
	log.Printf("Spending: %+v", categorised)
}
