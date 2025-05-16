// Thanh Vu | 10582614 | Online
// Program summary: Convert Australian dollars to Ethereum using Go
// CSP3341 Programming Languages and Paradigms | Sem 1 2025
// Ali Hur

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

// PriceResult holds the price and any error from a price fetch
// Struct bundles related data (price, error, and source) for organized error handling and result tracking
type PriceResult struct {
	price float64
	err   error
	name  string // Add name to track which API provided the result
}

// Good feature: Interfaces in Go are satisfied implicitly, encouraging decoupling and flexible architecture
// This promotes modular code without needing explicit declarations
type PriceFetcher interface {
	FetchPrice() (float64, error)
	Name() string
}

// API is a concrete implementation of the PriceFetcher interface
// It holds the API's name and URL, demonstrating Go's preference for composition over inheritance
// Composition with timeout field handles API call timeouts gracefully
type API struct {
	name, url string
	timeout   time.Duration // Add timeout for API calls
}

// NewAPI creates a new API instance with default timeout
// Constructor function ensures proper initialization with default values, following Go's idiomatic patterns
func NewAPI(name, url string) API {
	return API{
		name:    name,
		url:     url,
		timeout: 10 * time.Second, // Default 10 second timeout
	}
}

func (a API) Name() string {
	return a.name
}

// FetchPrice performs a HTTP GET request to retrieve ETH/USD price data from the specified API
// Go's error handling model avoids exceptions, errors are returned explicitly and checked after each step
// HTTP client timeout prevents hanging on slow API responses
func (a API) FetchPrice() (float64, error) {
	client := &http.Client{
		Timeout: a.timeout,
	}

	resp, err := client.Get(a.url)
	if err != nil {
		return 0, fmt.Errorf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("non-OK status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("reading body failed: %v", err)
	}

	var price float64
	if err := a.parseResponse(body, &price); err != nil {
		return 0, fmt.Errorf("parsing response failed: %v", err)
	}

	if price <= 0 {
		return 0, fmt.Errorf("invalid price: %f", price)
	}
	return price, nil
}

// parseResponse handles the JSON parsing for each API
// Limitation: Go's lack of inheritance, can't create a base API class with common functionality
// Instead, use composition and switch statements, which can be verbose
// Switch statement handles different API response formats
func (a API) parseResponse(body []byte, price *float64) error {
	switch a.name {
	case "CoinGecko":
		var data map[string]map[string]float64
		if err := json.Unmarshal(body, &data); err != nil {
			return err
		}
		*price = data["ethereum"]["usd"]
	case "Coinbase":
		var data struct {
			Data struct {
				Amount string `json:"amount"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return err
		}
		var err error
		*price, err = strconv.ParseFloat(data.Data.Amount, 64)
		if err != nil {
			return err
		}
	case "Bitstamp":
		var data map[string]string
		if err := json.Unmarshal(body, &data); err != nil {
			return err
		}
		var err error
		*price, err = strconv.ParseFloat(data["last"], 64)
		if err != nil {
			return err
		}
	case "Kraken":
		var data struct {
			Result map[string]struct {
				C []string `json:"c"`
			} `json:"result"`
		}
		if err := json.Unmarshal(body, &data); err != nil {
			return err
		}
		for _, v := range data.Result {
			var err error
			*price, err = strconv.ParseFloat(v.C[0], 64)
			if err != nil {
				return err
			}
			break
		}
	case "Bitfinex":
		var data []float64
		if err := json.Unmarshal(body, &data); err != nil {
			return err
		}
		if len(data) < 7 {
			return fmt.Errorf("invalid data length from Bitfinex")
		}
		*price = data[6]
	default:
		return fmt.Errorf("unknown API: %s", a.name)
	}
	return nil
}

// calculateAverageAndConvertToAUD takes a slice of price results and returns the average in AUD
func calculateAverageAndConvertToAUD(results []PriceResult) (float64, error) {
	var sum float64
	var count int

	// Calculate simple average of valid prices
	for _, result := range results {
		if result.err == nil {
			sum += result.price
			count++
		}
	}

	if count == 0 {
		return 0, fmt.Errorf("no valid prices found")
	}

	averageUSD := sum / float64(count)

	// Get exchange rates with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	resp, err := client.Get("https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd,aud")
	if err != nil {
		return 0, fmt.Errorf("failed to get exchange rates: %v", err)
	}
	defer resp.Body.Close()

	var data map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, fmt.Errorf("failed to decode exchange rates: %v", err)
	}

	ethData := data["ethereum"]
	usdRate := ethData["usd"]
	audRate := ethData["aud"]

	if usdRate == 0 || audRate == 0 {
		return 0, fmt.Errorf("invalid exchange rates")
	}

	conversionRate := audRate / usdRate
	audPrice := averageUSD * conversionRate

	return audPrice, nil
}

// fetchAndCalculatePrice handles all the price fetching and calculation logic using channels and WaitGroup
// Good feature: Go's concurrency model with goroutines and channels makes parallel API calls simple and efficient
// The combination of WaitGroup and channels demonstrates Go's powerful synchronization primitives
// Buffered channel prevents goroutine blocking, ensuring all results can be sent
func fetchAndCalculatePrice() (float64, error) {
	fetchers := []PriceFetcher{
		NewAPI("CoinGecko", "https://api.coingecko.com/api/v3/simple/price?ids=ethereum&vs_currencies=usd"),
		NewAPI("Coinbase", "https://api.coinbase.com/v2/prices/ETH-USD/spot"),
		NewAPI("Bitstamp", "https://www.bitstamp.net/api/v2/ticker/ethusd/"),
		NewAPI("Kraken", "https://api.kraken.com/0/public/Ticker?pair=ETHUSD"),
		NewAPI("Bitfinex", "https://api-pub.bitfinex.com/v2/ticker/tETHUSD"),
	}

	resultsChan := make(chan PriceResult, len(fetchers))
	var wg sync.WaitGroup

	for _, fetcher := range fetchers {
		wg.Add(1)
		go func(f PriceFetcher) {
			defer wg.Done()
			price, err := f.FetchPrice()
			resultsChan <- PriceResult{
				price: price,
				err:   err,
				name:  f.Name(),
			}
			if err != nil {
				fmt.Printf("[%s] Error: %v\n", f.Name(), err)
			} else {
				fmt.Printf("[%s] ETH/USD = $%.2f\n", f.Name(), price)
			}
		}(fetcher)
	}

	// Wait for all goroutines to finish
	wg.Wait()
	close(resultsChan)

	// Collect results from the channel
	var results []PriceResult
	for result := range resultsChan {
		results = append(results, result)
	}

	return calculateAverageAndConvertToAUD(results)
}

// main function demonstrates the program's workflow
// bufio.Scanner for input handling
func main() {
	// Fetch and calculate current ETH price in AUD
	avgAUD, err := fetchAndCalculatePrice()
	if err != nil {
		fmt.Printf("Error calculating average: %v\n", err)
		return
	}
	fmt.Printf("\nCurrent ETH price in AUD: $%.2f\n", avgAUD)

	// CLI Interface for AUD to ETH conversion
	fmt.Println("\n=== ETH Price Converter ===")
	fmt.Println("Enter the amount in AUD (or 'q' to quit):")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("AUD amount: ")
		if !scanner.Scan() {
			break
		}
		input := scanner.Text()

		if input == "q" || input == "Q" {
			fmt.Println("\nGoodbye!\n")
			break
		}

		audAmount, err := strconv.ParseFloat(input, 64)
		if err != nil {
			fmt.Println("Invalid input. Please enter a valid number or 'q' to quit.")
			continue
		}

		if audAmount <= 0 {
			fmt.Println("Please enter a positive amount.")
			continue
		}

		// Calculate ETH amount
		ethAmount := audAmount / avgAUD
		fmt.Printf("You can get %.8f ETH for $%.2f AUD\n", ethAmount, audAmount)
		fmt.Println("\nEnter another amount or 'q' to quit:")
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading input: %v\n", err)
	}
}

// Program summary:
// Go's interface system and goroutines offer simplicity, modularity, and efficient concurrency
// The use of channels demonstrates Go's communication mechanism between goroutines
// However, the language's lack of inheritance and verbose error-handling patterns
// can lead to code repetition and reduced writability in large applications
