//load data (hardcoded to be coinmarketcap FTB)
//at a variety of frequencies (OK, 1 FTB) and check all coins for breakouts from channels, inform the user of a potentially lucrative event (dream on, mate!)
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
	Intervals      []int     //minutes
	SigIncreases   []float64 //increase (proportion) over interval that is considered significant
	SigNumIncss    []int     //Number of consecutive recent significant increases that is considered significant
	MinRunLens     []int     //minimum number of intervals within corridor i.e. how long should it have "wobbled about" before breaking upwards
	MaxExceptionss []int     //in the preceding, we are allowed this number of exceptions
	Corridors      []float64 //corridor size as proportion of the price sigNumIncs intervals back
	Coinlist       string    //file name
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
	fmt.Println("Channel Breakout V1.2")
	//read configuration (yaml)
	var config configData
	data, err := os.ReadFile("config.yaml")
	if err != nil {
		fmt.Println("Cannot open config file")
		return
	}
	yaml.Unmarshal(data, &config)

	//get list of coins on our exchange for use in api call
	coinList := "BTC" //hardcoding this avoids mucking around with the number of commas
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

	//get quotes, forever, check for interesting events
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
		first := true
		for xk, xcurr := range inter4 {
			curr := xcurr.(map[string]interface{})
			xquote := curr["quote"]
			quote := xquote.(map[string]interface{})
			xstuff := quote["USD"]
			stuff := xstuff.(map[string]interface{})
			price := stuff["price"].(float64)
			k := string(xk) //key
			data := prices[k].Prices
			data[prices[k].NumPoints] = price
			prices[k].NumPoints++
			np := prices[k].NumPoints
			if first {
				first = false
				fmt.Println(time.Now().String(), len(inter4), "coins")
			}
			for cond := 0; cond < len(config.Intervals); cond++ { //loop through tests
				step := config.Intervals[cond] / config.Intervals[0]
				if np >= step*(config.SigNumIncss[cond]+config.MinRunLens[cond])+1 {
					interesting := true
					for counter := np - 1; counter > np-config.SigNumIncss[cond]*step-1; counter -= step { //gone up recently?
						interesting = interesting && (data[counter]/data[counter-step]-1 > config.SigIncreases[cond])
					}
					if !interesting {
						continue //next test
					}
					prevMax := data[np-config.SigNumIncss[cond]*step-1] //look at a corridor relative to this
					lenOfRun := 0
					numExcepts := 0
					for counter := np - (config.SigNumIncss[cond]+1)*step - 1; counter >= 0; counter -= step {
						if data[counter] > (1+config.Corridors[cond])*prevMax || data[counter] < (1-config.Corridors[cond])*prevMax { //check for outside corridor
							numExcepts++
							if numExcepts > config.MaxExceptionss[cond] {
								interesting = false
								break
							}
						} else {
							lenOfRun++
						}
					}
					interesting = interesting && (lenOfRun >= config.MinRunLens[cond])
					if interesting {
						fmt.Println("Coin", k, "Test Condition", cond+1, "Run Length", lenOfRun)
					}
				}
			}
		}
		resp.Body.Close()
		time.Sleep(time.Duration(tick) * time.Minute)
	}
}
