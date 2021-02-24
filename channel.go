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
	Interval      int     //minutes
	SigIncrease   float64 //increase (proportion) over interval that is considered significant
	SigNumIncs    int     //Number of consecutive recent significant increases that is considered significant
	MinRunLen     int     //minimum number of intervals within corridor i.e. how long should it have "wobbled about" before breaking upwards
	MaxExceptions int     //in the preceding, we are allowed this number of exceptions
	Corridor      float64 //corridor size as proportion of the price sigNumIncs intervals back
	Coinlist      string  //file name
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

	tick := config.Interval //minutes

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
			if np >= config.SigNumIncs+config.MinRunLen+1 {
				interesting := true
				for counter := np - 1; counter > np-config.SigNumIncs-1; counter-- { //gone up recently?
					interesting = interesting && (data[counter]/data[counter-1]-1 > config.SigIncrease)
				}
				if !interesting {
					continue //next coin
				}
				prevMax := data[np-config.SigNumIncs] //look at a corridor relative to this
				lenOfRun := 0
				numExcepts := 0
				for counter := np - config.SigNumIncs - 1; counter >= 0; counter-- {
					if data[counter] > (1+config.Corridor)*prevMax || data[counter] < (1-config.Corridor)*prevMax { //check for outside corridor
						numExcepts++
						if numExcepts > config.MaxExceptions {
							interesting = false
							break
						}
					} else {
						lenOfRun++
					}
				}
				interesting = interesting && (lenOfRun >= config.MinRunLen)
				if interesting {
					fmt.Println(k, lenOfRun)
				}
			}
		}
		resp.Body.Close()
		time.Sleep(time.Duration(tick) * time.Minute)
	}
}
