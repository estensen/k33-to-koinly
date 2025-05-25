package converter

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"time"
)

type Converter struct {
	trades map[string]*TradePair
}

type K33Record struct {
	TypeStatus      string
	TradeID         string
	Side            string
	Amount          string
	TradeStatus     string
	Asset           string
	Timestamp       string
	DepositTxhash   string
	WithdrawalTxhash string
}

type TradePair struct {
	TradeID   string
	Timestamp string
	BuyLeg    *K33Record
	SellLeg   *K33Record
}

type KoinlyRecord struct {
	Date             string
	SentAmount       string
	SentCurrency     string
	ReceivedAmount   string
	ReceivedCurrency string
	FeeAmount        string
	FeeCurrency      string
	NetWorthAmount   string
	NetWorthCurrency string
	Label            string
	Description      string
	TxHash           string
}

func New() *Converter {
	return &Converter{
		trades: make(map[string]*TradePair),
	}
}

func (c *Converter) Process(in io.Reader, out io.Writer) error {
	reader := csv.NewReader(in)
	writer := csv.NewWriter(out)
	defer writer.Flush()

	// Write Koinly header
	koinlyHeader := []string{
		"Date", "Sent Amount", "Sent Currency", "Received Amount", "Received Currency",
		"Fee Amount", "Fee Currency", "Net Worth Amount", "Net Worth Currency", 
		"Label", "Description", "TxHash",
	}
	if err := writer.Write(koinlyHeader); err != nil {
		return fmt.Errorf("writing header: %w", err)
	}

	// Read K33 header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading header: %w", err)
	}

	var koinlyRecords []KoinlyRecord

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading record: %w", err)
		}

		k33Record := parseK33Record(header, record)
		
		// Skip rejected trades
		if k33Record.TradeStatus == "Reject" {
			continue
		}

		records := c.processK33Record(k33Record)
		koinlyRecords = append(koinlyRecords, records...)
	}

	// Process any remaining unpaired trades
	for _, trade := range c.trades {
		if trade.BuyLeg != nil || trade.SellLeg != nil {
			log.Printf("Warning: Unpaired trade %s", trade.TradeID)
		}
	}

	// Write all Koinly records
	for _, record := range koinlyRecords {
		row := []string{
			record.Date, record.SentAmount, record.SentCurrency,
			record.ReceivedAmount, record.ReceivedCurrency,
			record.FeeAmount, record.FeeCurrency,
			record.NetWorthAmount, record.NetWorthCurrency,
			record.Label, record.Description, record.TxHash,
		}
		if err := writer.Write(row); err != nil {
			return fmt.Errorf("writing record: %w", err)
		}
	}

	return nil
}

func (c *Converter) ProcessDryRun(in io.Reader) error {
	reader := csv.NewReader(in)

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("reading header: %w", err)
	}

	fmt.Println("K33 to Koinly Conversion (Dry Run)")
	fmt.Println("==================================")

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading record: %w", err)
		}

		k33Record := parseK33Record(header, record)
		
		if k33Record.TradeStatus == "Reject" {
			fmt.Printf("SKIPPED (Rejected): %s\n", k33Record.TypeStatus)
			continue
		}

		records := c.processK33Record(k33Record)
		for _, koinlyRecord := range records {
			fmt.Printf("%s | %s %s -> %s %s | %s\n",
				koinlyRecord.Date,
				koinlyRecord.SentAmount, koinlyRecord.SentCurrency,
				koinlyRecord.ReceivedAmount, koinlyRecord.ReceivedCurrency,
				koinlyRecord.Description)
		}
	}

	return nil
}

func parseK33Record(header []string, record []string) K33Record {
	k33 := K33Record{}
	
	for i, col := range header {
		if i >= len(record) {
			continue
		}
		
		// Clean BOM and whitespace from column names
		col = strings.TrimSpace(strings.TrimPrefix(col, "\ufeff"))
		
		switch col {
		case "Type/Status":
			k33.TypeStatus = record[i]
		case "TradeID":
			k33.TradeID = formatTradeID(record[i])
		case "Side":
			k33.Side = record[i]
		case "Amount":
			k33.Amount = record[i]
		case "Trade Status":
			k33.TradeStatus = record[i]
		case "Asset":
			k33.Asset = record[i]
		case "Timestamp (UTC)":
			k33.Timestamp = record[i]
		case "DepositTxhash":
			k33.DepositTxhash = record[i]
		case "WithdrawalTxhash":
			k33.WithdrawalTxhash = record[i]
		}
	}
	
	return k33
}

func formatTradeID(tradeID string) string {
	if tradeID == "" {
		return ""
	}
	
	// Handle scientific notation
	if f, err := strconv.ParseFloat(tradeID, 64); err == nil {
		return fmt.Sprintf("%.0f", f)
	}
	
	return tradeID
}

func (c *Converter) processK33Record(k33 K33Record) []KoinlyRecord {
	// Skip records with empty required fields
	if k33.TypeStatus == "" || k33.Timestamp == "" {
		return nil
	}
	
	timestamp := convertTimestamp(k33.Timestamp)
	
	switch {
	case strings.Contains(k33.TypeStatus, "Deposit"):
		return []KoinlyRecord{c.createDepositRecord(k33, timestamp)}
		
	case strings.Contains(k33.TypeStatus, "Withdrawal"):
		return []KoinlyRecord{c.createWithdrawalRecord(k33, timestamp)}
		
	case k33.TypeStatus == "Trade":
		return c.processTrade(k33, timestamp)
	}
	
	return nil
}

func (c *Converter) createDepositRecord(k33 K33Record, timestamp string) KoinlyRecord {
	amount := strings.TrimPrefix(k33.Amount, "-")
	
	return KoinlyRecord{
		Date:             timestamp,
		ReceivedAmount:   amount,
		ReceivedCurrency: k33.Asset,
		Description:      "Deposit (K33)",
		TxHash:          k33.DepositTxhash,
	}
}

func (c *Converter) createWithdrawalRecord(k33 K33Record, timestamp string) KoinlyRecord {
	amount := strings.TrimPrefix(k33.Amount, "-")
	
	return KoinlyRecord{
		Date:         timestamp,
		SentAmount:   amount,
		SentCurrency: k33.Asset,
		Description:  "Withdrawal (K33)",
		TxHash:      k33.WithdrawalTxhash,
	}
}

func (c *Converter) processTrade(k33 K33Record, timestamp string) []KoinlyRecord {
	if k33.TradeID == "" {
		return nil
	}
	
	trade, exists := c.trades[k33.TradeID]
	if !exists {
		trade = &TradePair{
			TradeID:   k33.TradeID,
			Timestamp: timestamp,
		}
		c.trades[k33.TradeID] = trade
	}
	
	// Store the trade leg
	if k33.Side == "Buy" {
		trade.BuyLeg = &k33
	} else if k33.Side == "Sell" {
		trade.SellLeg = &k33
	}
	
	// If we have both legs, create the Koinly record
	if trade.BuyLeg != nil && trade.SellLeg != nil {
		record := c.createTradeRecord(trade)
		delete(c.trades, k33.TradeID) // Remove completed trade
		return []KoinlyRecord{record}
	}
	
	return nil
}

func (c *Converter) createTradeRecord(trade *TradePair) KoinlyRecord {
	buyAmount := strings.TrimPrefix(trade.BuyLeg.Amount, "-")
	sellAmount := strings.TrimPrefix(trade.SellLeg.Amount, "-")
	
	return KoinlyRecord{
		Date:             trade.Timestamp,
		SentAmount:       sellAmount,
		SentCurrency:     trade.SellLeg.Asset,
		ReceivedAmount:   buyAmount,
		ReceivedCurrency: trade.BuyLeg.Asset,
		Description:      fmt.Sprintf("Trade (K33) - %s", trade.TradeID),
	}
}

func convertTimestamp(timestamp string) string {
	// Parse: "2025/02/26 11:11:13"
	t, err := time.Parse("2006/01/02 15:04:05", timestamp)
	if err != nil {
		log.Printf("Warning: Could not parse timestamp %s: %v", timestamp, err)
		return timestamp
	}
	
	// Format: "2006-01-02 15:04:05"
	return t.Format("2006-01-02 15:04:05")
}