package ordering

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWritePaymentXML_Basic(t *testing.T) {
	dir := t.TempDir()

	req := PaymentRequest{
		ReferenceNumber: "rooam-pay-001",
		TicketNumber:    62,
		TenderTypeID:    10,
		Amount:          4275, // $42.75
		TipAmount:       500,  // $5.00
		AllowsTips:      true,
		CashierNumber:   "976",
		TerminalNumber:  "1",
		Comment:         "rooam-pay-001",
	}

	if err := WritePaymentXML(req, dir); err != nil {
		t.Fatalf("WritePaymentXML: %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(dir, "ORDER*.XML"))
	if len(files) != 1 {
		t.Fatalf("expected 1 ORDER*.XML file, got %d", len(files))
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var result UpdateOrders
	if err := xml.Unmarshal(data, &result); err != nil {
		t.Fatalf("xml.Unmarshal: %v", err)
	}

	u := result.UpdateOrder
	if u == nil {
		t.Fatal("UpdateOrder is nil")
	}
	if u.CheckNumber != 62 {
		t.Errorf("CheckNumber = %d, want 62", u.CheckNumber)
	}
	if u.Function != 4 {
		t.Errorf("Function = %d, want 4", u.Function)
	}
	if u.ErrorLevel != 2 {
		t.Errorf("ErrorLevel = %d, want 2", u.ErrorLevel)
	}
	if u.ReferenceNumber != "rooam-pay-001" {
		t.Errorf("ReferenceNumber = %q, want %q", u.ReferenceNumber, "rooam-pay-001")
	}
	if u.Check == nil || u.Check.CheckHeader.TerminalNumber != "1" {
		t.Errorf("Check.CheckHeader.TerminalNumber = %q, want %q", u.Check.CheckHeader.TerminalNumber, "1")
	}
	if u.Payment == nil {
		t.Fatal("Payment is nil")
	}
	if u.Payment.PaymentHeader.CashierNumber != "976" {
		t.Errorf("PaymentHeader.CashierNumber = %q, want %q", u.Payment.PaymentHeader.CashierNumber, "976")
	}
	if u.Payment.PaymentHeader.TerminalNumber != "1" {
		t.Errorf("PaymentHeader.TerminalNumber = %q, want %q", u.Payment.PaymentHeader.TerminalNumber, "1")
	}
	if u.Payment.PaymentHeader.PaymentTerminal != "1" {
		t.Errorf("PaymentHeader.PaymentTerminal = %q, want %q", u.Payment.PaymentHeader.PaymentTerminal, "1")
	}
	if len(u.Payment.PaymentDetails) != 1 {
		t.Fatalf("PaymentDetails count = %d, want 1", len(u.Payment.PaymentDetails))
	}
	d := u.Payment.PaymentDetails[0]
	if d.PaymentType != 10 {
		t.Errorf("PaymentType = %d, want 10", d.PaymentType)
	}
	if d.PaymentAmount != 42.75 {
		t.Errorf("PaymentAmount = %v, want 42.75", d.PaymentAmount)
	}
	if d.PaymentTip != 5.00 {
		t.Errorf("PaymentTip = %v, want 5.00", d.PaymentTip)
	}
	if d.PaymentMemo != "*{rooam-pay-001}" {
		t.Errorf("PaymentMemo = %q, want %q", d.PaymentMemo, "*{rooam-pay-001}")
	}
}

func TestWritePaymentXML_NoTip(t *testing.T) {
	dir := t.TempDir()

	req := PaymentRequest{
		ReferenceNumber: "ref-no-tip",
		TicketNumber:    10,
		TenderTypeID:    1,
		Amount:          2000, // $20.00
		CashierNumber:   "5",
	}

	if err := WritePaymentXML(req, dir); err != nil {
		t.Fatalf("WritePaymentXML: %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(dir, "ORDER*.XML"))
	if len(files) != 1 {
		t.Fatalf("expected 1 ORDER*.XML file, got %d", len(files))
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	// PaymentTip should be omitted when AllowsTips is false and TipAmount is 0
	if strings.Contains(string(data), "PaymentTip") {
		t.Error("PaymentTip should not be present when AllowsTips=false and TipAmount=0")
	}
	// PaymentMemo should be omitted when Comment is empty
	if strings.Contains(string(data), "PaymentMemo") {
		t.Error("PaymentMemo should not be present when Comment is empty")
	}
}

func TestWritePaymentXML_AllowsTipsNoAmount(t *testing.T) {
	dir := t.TempDir()

	req := PaymentRequest{
		ReferenceNumber: "ref-allows-tips",
		TicketNumber:    5,
		TenderTypeID:    2,
		Amount:          1500,
		AllowsTips:      true, // tip allowed but $0
		CashierNumber:   "3",
	}

	if err := WritePaymentXML(req, dir); err != nil {
		t.Fatalf("WritePaymentXML: %v", err)
	}

	files, _ := filepath.Glob(filepath.Join(dir, "ORDER*.XML"))
	if len(files) != 1 {
		t.Fatalf("expected 1 ORDER*.XML file, got %d", len(files))
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var result UpdateOrders
	if err := xml.Unmarshal(data, &result); err != nil {
		t.Fatalf("xml.Unmarshal: %v", err)
	}

	d := result.UpdateOrder.Payment.PaymentDetails[0]
	// When AllowsTips=true but TipAmount=0, PaymentTip should be 0.0
	if d.PaymentTip != 0.0 {
		t.Errorf("PaymentTip = %v, want 0.0", d.PaymentTip)
	}
}
