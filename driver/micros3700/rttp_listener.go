// Package micros3700driver implements the Rooam POS driver for MICROS RES 3700.
// Tickets arrive via the IFS TCP push interface (RTTP) on port 5454.
// The POS terminal sends: <FS>RVC<FS>checkNum<FS>totalCents<FS>phoneNum
// where <FS> = ASCII 0x1C (File Separator / chr(28) in ISL).
// The agent must reply "RECEIVED" to acknowledge each message.
package micros3700driver

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/badpanda83/POSitouch-Integration/entities"
)

const (
	// rttpFieldSep is ASCII 0x1C (File Separator / chr(28) in ISL).
	rttpFieldSep = "\x1c"
	// rttpAck is the acknowledgement string sent back to the POS terminal.
	rttpAck = "RECEIVED"
	// rttpTicketTTL is how long a ticket is kept in memory without a new push.
	rttpTicketTTL = 4 * time.Hour
)

type rttpEntry struct {
	ticket   entities.Ticket
	lastSeen time.Time
}

// RttpListener listens on a TCP port for push messages from MICROS POS terminals
// using the IFS/RTTP protocol. Each incoming message is parsed and stored in an
// in-memory map keyed by check number.
type RttpListener struct {
	port    int
	mu      sync.Mutex
	tickets map[int]rttpEntry
}

// NewRttpListener creates a new RttpListener that will listen on the given port.
func NewRttpListener(port int) *RttpListener {
	return &RttpListener{
		port:    port,
		tickets: make(map[int]rttpEntry),
	}
}

// Start begins accepting TCP connections. It blocks until listening fails.
// Intended to be launched as a goroutine.
func (l *RttpListener) Start() {
	addr := fmt.Sprintf(":%d", l.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("[micros3700][rttp] failed to listen on %s: %v", addr, err)
		return
	}
	log.Printf("[micros3700][rttp] listening on TCP %s", addr)
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("[micros3700][rttp] accept error: %v", err)
			continue
		}
		go l.handleConn(conn)
	}
}

// handleConn processes RTTP messages from a single POS terminal connection.
func (l *RttpListener) handleConn(conn net.Conn) {
	defer conn.Close()
	remote := conn.RemoteAddr().String()
	log.Printf("[micros3700][rttp] connection from %s", remote)

	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if err := l.processMessage(line); err != nil {
			log.Printf("[micros3700][rttp] %s: parse error: %v", remote, err)
			continue
		}
		// Acknowledge the message.
		if _, err := fmt.Fprintf(conn, "%s\r\n", rttpAck); err != nil {
			log.Printf("[micros3700][rttp] %s: write ack error: %v", remote, err)
			return
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("[micros3700][rttp] %s: read error: %v", remote, err)
	}
	log.Printf("[micros3700][rttp] connection closed: %s", remote)
}

// processMessage parses a single RTTP message and stores it in the ticket map.
// Message format (from ISL txmsg chr(28),@RVC,chkNum,itotal,phoneNum):
//
//	\x1C<RVC>\x1C<checkNum>\x1C<totalCents>\x1C<phoneNum>\r\n
func (l *RttpListener) processMessage(line string) error {
	// Split by the field separator (0x1C).  The message starts with the separator
	// so the first element of the split is always an empty string.
	parts := strings.Split(line, rttpFieldSep)

	// After splitting "\x1CRVC\x1CcheckNum\x1CtotalCents\x1CphoneNum" we get:
	// ["", "RVC", "checkNum", "totalCents", "phoneNum"]
	if len(parts) < 5 {
		return fmt.Errorf("expected 5 fields (got %d) in message %q", len(parts), line)
	}

	rvcStr := strings.TrimSpace(parts[1])
	checkStr := strings.TrimSpace(parts[2])
	totalCentsStr := strings.TrimSpace(parts[3])
	phoneNum := strings.TrimSpace(parts[4])

	rvc, err := strconv.Atoi(rvcStr)
	if err != nil {
		return fmt.Errorf("parse RVC %q: %w", rvcStr, err)
	}
	checkNum, err := strconv.Atoi(checkStr)
	if err != nil {
		return fmt.Errorf("parse checkNum %q: %w", checkStr, err)
	}
	totalCents, err := strconv.ParseInt(totalCentsStr, 10, 64)
	if err != nil {
		return fmt.Errorf("parse totalCents %q: %w", totalCentsStr, err)
	}

	total := float64(totalCents) / 100.0
	log.Printf("[micros3700][rttp] received check #%d RVC=%d total=$%.2f phone=%s",
		checkNum, rvc, total, phoneNum)

	t := entities.Ticket{
		Number: checkNum,
		// The RTTP protocol does not carry a table number.  The check number is
		// used as a table identifier so the ticket can be correlated by the
		// cloud layer until a table lookup is possible.
		Table:       checkNum,
		Total:       total,
		CostCenter:  rvc,
		PhoneNumber: phoneNum,
		OpenedAt:    time.Now().UTC().Format(time.RFC3339),
		Open:        true,
		POSType:     "micros3700",
	}

	l.mu.Lock()
	l.tickets[checkNum] = rttpEntry{ticket: t, lastSeen: time.Now()}
	l.mu.Unlock()

	return nil
}

// Tickets returns all non-expired tickets currently held in memory.
// Tickets that have not been updated for more than rttpTicketTTL are evicted.
func (l *RttpListener) Tickets() []entities.Ticket {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	out := make([]entities.Ticket, 0, len(l.tickets))
	for key, entry := range l.tickets {
		if now.Sub(entry.lastSeen) > rttpTicketTTL {
			delete(l.tickets, key)
			continue
		}
		out = append(out, entry.ticket)
	}
	return out
}
