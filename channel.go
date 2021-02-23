//load data (hardcoded to be coinmarketcap or a test file FTB)
//at a variety of frequencies and check all coins for breakouts from channels, inform the user of a potentially lucrative event (yeah, right!)
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	yaml "gopkg.in/yaml.v2"
)

type configData struct {
	Source                     string
	Testdatafile               string
	Intervals                  []int
	Daysofdatainmemory         []int
	Minnummaximaforchannel     []int
	Minnumproponoutsidechannel []float64
	Minnumpointsoutsidechannel []int
	Maxnumpointsoutsidechannel []int
	Coinlist                   string
}

type coin struct {
	Symbol    string
	Prices    []float64
	NumPoints int
}

const maxNumPrices int = 1000

func newCoin(name string) *coin {
	return &coin{Symbol: name, Prices: make([]float64, maxNumPrices), NumPoints: 0}
}

func main() {
	//read configuration (yaml)
	var config configData
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("Cannot open config file")
		return
	}
	yaml.Unmarshal(data, &config)

	//get list of coins on our exchange
	coinList := "BTC"
	prices := make(map[string]*coin)
	prices["BTC"] = newCoin("BTC")
	if config.Coinlist != "none" {
		file, err := os.Open(config.Coinlist)
		if err != nil {
			fmt.Println("Cannot open coin list file")
			return
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			coinList += "," + scanner.Text()
			prices[scanner.Text()] = newCoin(scanner.Text())
		}
	}

	tick := config.Intervals[0] //minutes

	//set up request
	client := &http.Client{}
	url := "https://pro-api.coinmarketcap.com/v1/cryptocurrency/quotes/latest?symbol=" + coinList + "&skip_invalid=true"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Accepts", "application/json")
	req.Header.Add("X-CMC_PRO_API_KEY", "c0023f87-09da-40a7-815d-5e76ea5750b2")

	//get quotes, forever
	for {
		resp, err := client.Do(req)
		if err != nil {
			fmt.Println(err)
			return
		}
		var inter1 interface{}
		body, _ := ioutil.ReadAll(resp.Body)
		json.Unmarshal(body, &inter1)
		inter2 := inter1.(map[string]interface{})
		inter3 := inter2["data"]
		inter4 := inter3.(map[string]interface{})
		for xk, xcurr := range inter4 {
			curr := xcurr.(map[string]interface{})
			xquote := curr["quote"]
			quote := xquote.(map[string]interface{})
			xstuff := quote["USD"]
			stuff := xstuff.(map[string]interface{})
			price := stuff["price"].(float64)
			k := string(xk)
			prices[k].Prices[prices[k].NumPoints] = price
			prices[k].NumPoints++
		}
		resp.Body.Close()
		fmt.Println(prices["BTC"].Prices[prices["BTC"].NumPoints-1])
		time.Sleep(time.Duration(tick) * time.Minute)
	}
}
