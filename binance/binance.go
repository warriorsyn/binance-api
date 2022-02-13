package binance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/ReneKroon/ttlcache/v2"
	"github.com/adshao/go-binance/v2"
	"github.com/leekchan/accounting"
)

var cache ttlcache.SimpleCache = ttlcache.NewCache()

func init() {
	cache.SetTTL(time.Duration(24 * time.Hour))
}

func ListenCoins() {
	times := 0
	for {

		if times != 0 {
			fmt.Println("--------WS will reprocess in 30 seconds-------")
			time.Sleep(10 * time.Second)
		}

		times += 1

		wsDepthHandler := func(event *binance.WsMarketStatEvent) {
			currentPrice, err := strconv.ParseFloat(event.LastPrice, 64)

			if err != nil {
				fmt.Println("something went wrong to convert str to int")
				panic(err)
			}

			defer handlePricing(event, currentPrice)
		}

		errHandler := func(err error) {
			fmt.Println(err)
		}

		doneC, stopC, err := binance.WsCombinedMarketStatServe([]string{"BTCBRL", "BNBBRL", "ETHBRL"}, wsDepthHandler, errHandler)

		if err != nil {
			fmt.Println(err)
			break
		}

		go func() {
			time.Sleep(7 * time.Second)
			stopC <- struct{}{}
		}()

		<-doneC
	}

	defer cache.Close()
}

func calculatePercentageBetweenTwoNumbers(num1 float64, num2 float64) float64 {
	result := ((num1 - num2) / ((num1 + num2) / 2) * 100)

	return result
}

func formatToMoney(value float64, symbol string) string {
	ac := accounting.Accounting{Symbol: symbol, Precision: 2, Decimal: ","}
	return ac.FormatMoney(value)
}

func handleNotification(symbol string, currentPrice float64, convertedLastPrice float64) {

	var message = fmt.Sprintf("A moeda de código %s caiu seu valor para %s, antes era %s, houve uma queda de %s", symbol,
		formatToMoney(currentPrice, "R$"),
		formatToMoney(convertedLastPrice, "R$"),
		formatToMoney(convertedLastPrice-currentPrice, "R$"))
	postBody, _ := json.Marshal(map[string][]map[string]string{
		"embeds": {
			{
				"title":       symbol,
				"description": message,
			},
		},
	})

	discordWebhook := os.Getenv("DISCORD_WEBHOOK")

	req, err := http.NewRequest("POST", discordWebhook, bytes.NewBuffer(postBody))
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		fmt.Println("Something went wrong to notificate")
	}

	client := &http.Client{}
	resp, err := client.Do(req)

	if err != nil {
		fmt.Println("Something went wrong to notificate")
	}

	fmt.Println(resp.Body)

	defer resp.Body.Close()
}

func handlePricing(event *binance.WsMarketStatEvent, currentPrice float64) {
	symbol := event.Symbol
	fullCacheKey := fmt.Sprintf("%s-lastPrice", symbol)
	setCacheDynamic(fullCacheKey)

	lastPrice, _ := cache.Get(fullCacheKey)

	if lastPrice == 0.0 {
		cache.Set(fullCacheKey, currentPrice)
	}

	fmt.Println(fullCacheKey)

	convertedLastPrice, _ := strconv.ParseFloat(fmt.Sprintf("%v", lastPrice), 64)

	if calculatePercentageBetweenTwoNumbers(convertedLastPrice, currentPrice) > 0.0 && currentPrice < convertedLastPrice {
		handleNotification(symbol, currentPrice, convertedLastPrice)
	}

	fmt.Println("-------------------------------------")
	fmt.Println("Coin: ", symbol)
	fmt.Println("last: ", convertedLastPrice)
	fmt.Println("current: ", currentPrice)
}

func setCacheDynamic(cacheName string) {
	_, err := cache.Get(cacheName)

	if err != nil {
		cache.Set(cacheName, 0.0)
	}
}
