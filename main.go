package main

import (
	"bytes"
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

//go:embed instruments.json
var rhInstruments []byte

//go:embed order.json
var orders []byte

type instrumentMap map[string]string

type result struct {
	InstrumentID string  `json:"instrument_id"`
	Quantity     float64 `json:"quantity,string"`
	Side         string
	OrderType    string `json:"type"`
	Trigger      string
	AveragePrice float64
	Price        float64 `json:"price,string"`
	Executions   []execution
}

type Ticker struct {
	Symbol string
	Id     string
}

func GetInstrument(instrumentID string) (*Ticker, error) {
	url := fmt.Sprintf("https://api.robinhood.com/instruments/%s/", instrumentID)
	request, err := http.NewRequest("GET", url, nil)
	client := http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		fmt.Println("Unable to query robinhood", instrumentID)
		return nil, err
	}

	time.Sleep(5 * time.Second)

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, resp.Body)
	defer resp.Body.Close()

	var t Ticker
	err = json.Unmarshal(buf.Bytes(), &t)
	if err != nil {
		fmt.Println("Unable to marshall response body")
		return nil, err
	}
	return &t, nil
}

type execution struct {
	Price                  string      `json:"price"`
	Quantity               string      `json:"quantity"`
	RoundedNotional        interface{} `json:"rounded_notional"`
	SettlementDate         string      `json:"settlement_date"`
	Timestamp              time.Time   `json:"timestamp"`
	Id                     string      `json:"id"`
	IpoAccessExecutionRank interface{} `json:"ipo_access_execution_rank"`
}

type order struct {
	Next     *string
	Previous *string
	Results  []result
}

func main() {

	var allInstruments instrumentMap
	err := json.Unmarshal(rhInstruments, &allInstruments)
	if err != nil {
		return
	}

	instrumentsToSymbols := map[string]string{}
	for k, v := range allInstruments {
		if len(v) == 0 {
			continue
		}
		if _, ok := instrumentsToSymbols[v]; ok {
			fmt.Printf("instrumentMap %s already present \n", v)
		}
		instrumentsToSymbols[v] = k
	}

	var allOrders order
	err = json.Unmarshal(orders, &allOrders)
	if err != nil {
		return
	}

	defer func() {
		f, err := os.Open("instruments.json")
		if err != nil {
			fmt.Println("unable to open instruments.json")
			return
		}
		defer f.Close()
		encoder := json.NewEncoder(f)
		encoder.Encode(instrumentsToSymbols)

	}()

	fmt.Println("hello world")

	transactionsFile, err := os.Create("result.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer transactionsFile.Close()

	writer := csv.NewWriter(transactionsFile)
	defer writer.Flush()
	writer.Write([]string{"Date", "Action", "Quantity", "Price", "Symbol"})

	for _, o := range allOrders.Results {
		for _, e := range o.Executions {
			i, ok := instrumentsToSymbols[o.InstrumentID]
			if !ok {
				t, err := GetInstrument(o.InstrumentID)
				if err != nil {
					fmt.Println(err)
					panic(err)
				}
				instrumentsToSymbols[t.Id] = t.Symbol
				i = t.Symbol
			}
			record := []string{e.SettlementDate, o.Side, e.Quantity, e.Price, i}
			fmt.Println(record)
			writer.Write(record)
		}
	}
}
