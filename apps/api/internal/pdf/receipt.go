package pdf

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ReceiptInput is the data rendered onto a simple fee receipt PDF.
type ReceiptInput struct {
	InstituteName string
	StudentName   string
	RollNumber    string
	PaymentID     string
	FeeHeadName   string
	Amount        string
	PaidAt        time.Time
	GatewayRef    string
}

// GenerateReceipt builds a minimal valid PDF (no external deps) with text lines.
func GenerateReceipt(in ReceiptInput) ([]byte, error) {
	paid := in.PaidAt
	if paid.IsZero() {
		paid = time.Now().UTC()
	}
	lines := []string{
		"UNICORE FEE RECEIPT",
		"",
		"Institute: " + nullSafe(in.InstituteName),
		"Student: " + nullSafe(in.StudentName),
		"Roll: " + nullSafe(in.RollNumber),
		"Payment ID: " + nullSafe(in.PaymentID),
		"Fee Head: " + nullSafe(in.FeeHeadName),
		"Amount: " + nullSafe(in.Amount),
		"Paid At: " + paid.UTC().Format(time.RFC3339),
	}
	if in.GatewayRef != "" {
		lines = append(lines, "Gateway Ref: "+in.GatewayRef)
	}
	lines = append(lines, "", "This is a computer-generated receipt.")
	return WriteSimplePDF(lines)
}

// WriteSimplePDF writes a single-page PDF with Helvetica text lines.
func WriteSimplePDF(lines []string) ([]byte, error) {
	var content bytes.Buffer
	content.WriteString("BT\n/F1 12 Tf\n50 780 Td\n14 TL\n")
	for i, line := range lines {
		escaped := escapePDFText(line)
		if i == 0 {
			content.WriteString(fmt.Sprintf("(%s) Tj\n", escaped))
		} else {
			content.WriteString(fmt.Sprintf("T*\n(%s) Tj\n", escaped))
		}
	}
	content.WriteString("ET")

	stream := content.Bytes()
	objects := make([][]byte, 0, 6)
	objects = append(objects, []byte("<< /Type /Catalog /Pages 2 0 R >>"))
	objects = append(objects, []byte("<< /Type /Pages /Kids [3 0 R] /Count 1 >>"))
	objects = append(objects, []byte("<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>"))
	objects = append(objects, []byte(fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream)))
	objects = append(objects, []byte("<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>"))

	var buf bytes.Buffer
	buf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, obj := range objects {
		offsets[i+1] = buf.Len()
		buf.WriteString(strconv.Itoa(i+1) + " 0 obj\n")
		buf.Write(obj)
		buf.WriteString("\nendobj\n")
	}
	xrefPos := buf.Len()
	buf.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objects)+1))
	buf.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objects); i++ {
		buf.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	buf.WriteString("trailer\n")
	buf.WriteString(fmt.Sprintf("<< /Size %d /Root 1 0 R >>\n", len(objects)+1))
	buf.WriteString("startxref\n")
	buf.WriteString(strconv.Itoa(xrefPos) + "\n")
	buf.WriteString("%%EOF\n")
	return buf.Bytes(), nil
}

func escapePDFText(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	return s
}

func nullSafe(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
