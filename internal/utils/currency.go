package utils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

type Currency struct {
	Code   string `json:"code"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

var SupportedCurrencies = map[string]Currency{
	"USD": {Code: "USD", Symbol: "$", Name: "US Dollar"},
	"EUR": {Code: "EUR", Symbol: "€", Name: "Euro"},
	"GBP": {Code: "GBP", Symbol: "£", Name: "British Pound"},
	"CAD": {Code: "CAD", Symbol: "C$", Name: "Canadian Dollar"},
	"AUD": {Code: "AUD", Symbol: "A$", Name: "Australian Dollar"},
	"JPY": {Code: "JPY", Symbol: "¥", Name: "Japanese Yen"},
	"CNY": {Code: "CNY", Symbol: "¥", Name: "Chinese Yuan"},
	"INR": {Code: "INR", Symbol: "₹", Name: "Indian Rupee"},
	"BRL": {Code: "BRL", Symbol: "R$", Name: "Brazilian Real"},
	"MXN": {Code: "MXN", Symbol: "$", Name: "Mexican Peso"},
}

func FormatCurrency(amount float64, currencyCode string) string {
	currency, exists := SupportedCurrencies[currencyCode]
	if !exists {
		currency = SupportedCurrencies["USD"]
	}

	// Round to 2 decimal places
	amount = math.Round(amount*100) / 100

	// Format based on currency
	switch currencyCode {
	case "JPY", "KRW": // Currencies without decimal places
		return fmt.Sprintf("%s%.0f", currency.Symbol, amount)
	default:
		return fmt.Sprintf("%s%.2f", currency.Symbol, amount)
	}
}

func ParseCurrencyAmount(amountStr string) (float64, error) {
	// Remove currency symbols and spaces
	cleaned := strings.TrimSpace(amountStr)
	cleaned = strings.ReplaceAll(cleaned, "$", "")
	cleaned = strings.ReplaceAll(cleaned, "€", "")
	cleaned = strings.ReplaceAll(cleaned, "£", "")
	cleaned = strings.ReplaceAll(cleaned, "¥", "")
	cleaned = strings.ReplaceAll(cleaned, "₹", "")
	cleaned = strings.ReplaceAll(cleaned, "R$", "")
	cleaned = strings.ReplaceAll(cleaned, "C$", "")
	cleaned = strings.ReplaceAll(cleaned, "A$", "")
	cleaned = strings.ReplaceAll(cleaned, ",", "")
	cleaned = strings.TrimSpace(cleaned)

	return strconv.ParseFloat(cleaned, 64)
}

func ConvertCurrency(amount float64, fromCurrency, toCurrency string) float64 {
	// Simplified conversion - in production, use real exchange rates from an API
	// This is just for demonstration
	exchangeRates := map[string]float64{
		"USD": 1.0,
		"EUR": 0.85,
		"GBP": 0.73,
		"CAD": 1.25,
		"AUD": 1.35,
		"JPY": 110.0,
		"CNY": 6.45,
		"INR": 74.5,
		"BRL": 5.2,
		"MXN": 20.1,
	}

	fromRate, fromExists := exchangeRates[fromCurrency]
	toRate, toExists := exchangeRates[toCurrency]

	if !fromExists || !toExists {
		return amount // Return original amount if currency not supported
	}

	// Convert to USD first, then to target currency
	usdAmount := amount / fromRate
	return usdAmount * toRate
}

func GetCurrencySymbol(currencyCode string) string {
	currency, exists := SupportedCurrencies[currencyCode]
	if !exists {
		return "$"
	}
	return currency.Symbol
}

func ValidateCurrencyCode(code string) bool {
	_, exists := SupportedCurrencies[code]
	return exists
}

func RoundCurrency(amount float64, currencyCode string) float64 {
	switch currencyCode {
	case "JPY", "KRW": // Currencies without decimal places
		return math.Round(amount)
	default:
		return math.Round(amount*100) / 100
	}
}

func CalculateTip(amount float64, tipPercentage float64) float64 {
	tip := amount * (tipPercentage / 100)
	return math.Round(tip*100) / 100
}

func CalculateTax(amount float64, taxRate float64) float64 {
	tax := amount * (taxRate / 100)
	return math.Round(tax*100) / 100
}

func CalculateDiscount(amount float64, discountPercentage float64, maxDiscount float64) float64 {
	discount := amount * (discountPercentage / 100)
	if maxDiscount > 0 && discount > maxDiscount {
		discount = maxDiscount
	}
	return math.Round(discount*100) / 100
}