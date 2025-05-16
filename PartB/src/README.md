# ETH Price Converter
A Go program that converts Australian Dollars (AUD) to Ethereum (ETH) using real-time price data from multiple cryptocurrency exchanges.

## Features
- Fetches ETH prices from multiple APIs concurrently (CoinGecko, Coinbase, Bitstamp, Kraken, Bitfinex)
- Calculates average ETH price in AUD
- Provides a command-line interface for AUD to ETH conversion
- Demonstrates features of Go such as concurrency, interfaces, error handling and lack of inheritance.

## Requirements
- Go 1.21 or later

## How to Run
1. Ensure Go is installed
2. Clone or download this repository
3. Navigate to the project directory
4. Run the program:
   ```bash
   go run main.go
   ```

## Usage
1. The program will fetch current ETH prices and display the average price in AUD
2. Enter an amount in AUD to convert to ETH
3. Type 'q' to quit the program
