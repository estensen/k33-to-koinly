package converter

import (
	"strings"
	"testing"
)

func TestConvertTimestamp(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"2023/01/15 10:30:45", "2023-01-15 10:30:45"},
		{"2023/12/25 23:59:59", "2023-12-25 23:59:59"},
	}

	for _, test := range tests {
		result := convertTimestamp(test.input)
		if result != test.expected {
			t.Errorf("convertTimestamp(%s) = %s, want %s", test.input, result, test.expected)
		}
	}
}

func TestFormatTradeID(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1000000012345", "1000000012345"},
		{"1.000000012345e+12", "1000000012345"},
		{"", ""},
	}

	for _, test := range tests {
		result := formatTradeID(test.input)
		if result != test.expected {
			t.Errorf("formatTradeID(%s) = %s, want %s", test.input, result, test.expected)
		}
	}
}

func TestProcessK33Record(t *testing.T) {
	conv := New()

	// Test deposit
	deposit := K33Record{
		TypeStatus: "Deposit Complete",
		Amount:     "1000",
		Asset:      "USD",
		Timestamp:  "2023/01/15 10:30:45",
		DepositTxhash: "0xabc123",
	}

	records := conv.processK33Record(deposit)
	if len(records) != 1 {
		t.Fatalf("Expected 1 record for deposit, got %d", len(records))
	}

	record := records[0]
	if record.ReceivedAmount != "1000" || record.ReceivedCurrency != "USD" {
		t.Errorf("Deposit conversion failed: got %s %s", record.ReceivedAmount, record.ReceivedCurrency)
	}

	// Test withdrawal
	withdrawal := K33Record{
		TypeStatus: "Withdrawal Complete",
		Amount:     "-500",
		Asset:      "USD",
		Timestamp:  "2023/01/16 14:20:30",
		WithdrawalTxhash: "0xdef456",
	}

	records = conv.processK33Record(withdrawal)
	if len(records) != 1 {
		t.Fatalf("Expected 1 record for withdrawal, got %d", len(records))
	}

	record = records[0]
	if record.SentAmount != "500" || record.SentCurrency != "USD" {
		t.Errorf("Withdrawal conversion failed: got %s %s", record.SentAmount, record.SentCurrency)
	}
}

func TestTradePairing(t *testing.T) {
	conv := New()

	// First leg of trade
	buyLeg := K33Record{
		TypeStatus:  "Trade",
		TradeID:     "1000000012345",
		Side:        "Buy",
		Amount:      "1000",
		Asset:       "USD",
		TradeStatus: "Filled",
		Timestamp:   "2023/01/15 10:30:45",
	}

	records := conv.processK33Record(buyLeg)
	if len(records) != 0 {
		t.Errorf("Expected 0 records for first trade leg, got %d", len(records))
	}

	// Second leg of trade
	sellLeg := K33Record{
		TypeStatus:  "Trade",
		TradeID:     "1000000012345",
		Side:        "Sell",
		Amount:      "-0.5",
		Asset:       "BTC",
		TradeStatus: "Filled",
		Timestamp:   "2023/01/15 10:30:45",
	}

	records = conv.processK33Record(sellLeg)
	if len(records) != 1 {
		t.Fatalf("Expected 1 record for complete trade, got %d", len(records))
	}

	record := records[0]
	if record.SentAmount != "0.5" || record.SentCurrency != "BTC" {
		t.Errorf("Trade sell side failed: got %s %s", record.SentAmount, record.SentCurrency)
	}
	if record.ReceivedAmount != "1000" || record.ReceivedCurrency != "USD" {
		t.Errorf("Trade buy side failed: got %s %s", record.ReceivedAmount, record.ReceivedCurrency)
	}
}

func TestFullConversion(t *testing.T) {
	input := `Type/Status,TradeID,Side,Amount,Trade Status,Asset,Credit_old,Credit Balance,Funded_old,Funded Balance,PndWithdrawal_old,PndWithdrawal Balance,Total_old,Total Balance,Timestamp (UTC),UniqueKey,InternalReportID,DepositTxhash,WithdrawalTxhash,SourceAddress,DestinationAddress
Withdrawal Complete,,,-500,,USD,0,0,0,0,500,0,500,0,2023/01/16 14:20:30,test123,1001,,,,TestBank
Trade,1000000012345,Sell,-0.5,Filled,BTC,0,0,1,0.5,0,0,1,0.5,2023/01/15 10:30:45,test456,,,,,
Trade,1000000012345,Buy,1000,Filled,USD,0,0,0,1000,0,0,0,1000,2023/01/15 10:30:45,test456,,,,,`

	output := &strings.Builder{}
	conv := New()

	err := conv.Process(strings.NewReader(input), output)
	if err != nil {
		t.Fatalf("Conversion failed: %v", err)
	}

	result := output.String()
	lines := strings.Split(strings.TrimSpace(result), "\n")
	
	// Should have header + 2 records (1 withdrawal + 1 trade)
	if len(lines) != 3 {
		t.Errorf("Expected 3 lines (header + 2 records), got %d", len(lines))
	}

	// Check that we have the right number of columns
	for i, line := range lines {
		cols := strings.Split(line, ",")
		if len(cols) != 12 {
			t.Errorf("Line %d has %d columns, expected 12", i+1, len(cols))
		}
	}
}