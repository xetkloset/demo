package handler

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// TwiML response
type MessageResponse struct {
	XMLName xml.Name `xml:"Response"`
	Message string   `xml:"Message"`
}

// Session store
var sessions = make(map[string]*Session)
var mu sync.Mutex

type Session struct {
	Name        string
	Stage       string
	PIN         string
	Balance     float64
	PendingName string
	PendingAmt  float64
	Transactions []string
}

// Handler function
func Handler(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	from := r.FormValue("From")
	body := strings.TrimSpace(strings.ToLower(r.FormValue("Body")))

	mu.Lock()
	s, ok := sessions[from]
	if !ok {
		s = &Session{Stage: "ask_pin", Balance: 500, Transactions: []string{}}
		sessions[from] = s
	}
	mu.Unlock()

	response := ""

	switch s.Stage {

	case "ask_pin":
		response = "👋 Welcome back! Please enter your 4-digit PIN to continue."
		s.Stage = "verify_pin"

	case "verify_pin":
		if len(body) == 4 && isNumeric(body) {
			s.PIN = body
			s.Stage = "ask_name"
			response = "✅ PIN accepted! Please enter your name to continue."
		} else {
			response = "❌ Invalid PIN. Please enter a 4-digit PIN."
		}

	case "ask_name":
		s.Name = strings.Title(body)
		s.Stage = "main_menu"
		response = mainMenuText(s.Name)

	case "main_menu":
		switch body {
		case "1":
			response = fmt.Sprintf("💰 Your current balance is $%.2f\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit", s.Balance)
			s.Stage = "post_action"
		case "2":
			s.Stage = "send_to"
			response = "Who would you like to send money to?"
		case "3":
			s.Stage = "airtime"
			response = "Enter amount and mobile number (e.g. $2 to 0772123456)"
		case "4":
			response = "⚙️ Bill payment demo not active.\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit"
			s.Stage = "post_action"
		case "5":
			txs := "No transactions yet"
			if len(s.Transactions) > 0 {
				txs = strings.Join(s.Transactions, "\n")
			}
			response = fmt.Sprintf("🧾 Recent Transactions:\n%s\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit", txs)
			s.Stage = "post_action"
		case "6":
			s.Stage = "support"
			response = "I can help you with:\n1️⃣ Lost Card\n2️⃣ Transaction Issue\n3️⃣ Talk to Agent"
		default:
			response = "❓ Please choose a valid option (1–6)."
		}

	case "send_to":
		s.PendingName = strings.Title(body)
		s.Stage = "send_amount"
		response = fmt.Sprintf("How much would you like to send to %s?", s.PendingName)

	case "send_amount":
		amt, err := parseAmount(body)
		if err != nil {
			response = "❌ Invalid amount. Try again (e.g., 20 or $20)."
			break
		}
		s.PendingAmt = amt
		s.Stage = "confirm_send"
		response = fmt.Sprintf("Send $%.2f to %s? ✅ Yes / ❌ No", s.PendingAmt, s.PendingName)

	case "confirm_send":
		if strings.Contains(body, "yes") || body == "✅" {
			if s.Balance >= s.PendingAmt {
				s.Balance -= s.PendingAmt
				tx := fmt.Sprintf("Sent $%.2f to %s ✅", s.PendingAmt, s.PendingName)
				s.Transactions = append([]string{tx}, s.Transactions...) // prepend
				response = fmt.Sprintf("✅ Transaction successful!\nNew balance: $%.2f\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit", s.Balance)
			} else {
				response = "⚠️ Insufficient funds."
			}
			s.Stage = "post_action"
		} else {
			response = "❌ Transaction cancelled.\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit"
			s.Stage = "post_action"
		}

	case "airtime":
		amt, err := parseAmount(body)
		if err != nil {
			response = "❌ Invalid format. Try again (e.g., $2 to 0772123456)."
			break
		}
		if s.Balance >= amt {
			s.Balance -= amt
			tx := fmt.Sprintf("Bought $%.2f airtime 📱", amt)
			s.Transactions = append([]string{tx}, s.Transactions...)
			response = fmt.Sprintf("✅ Airtime purchase successful! New balance: $%.2f\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit", s.Balance)
		} else {
			response = "⚠️ Not enough balance."
		}
		s.Stage = "post_action"

	case "support":
		switch body {
		case "1":
			response = "🧾 Lost Card: Please call 0800 123 456."
		case "2":
			response = "⚙️ Transaction Issue logged."
		case "3":
			response = "👩🏾‍💼 Connecting to an agent..."
		default:
			response = "❓ Please choose 1, 2, or 3."
			return respondXML(w, response)
		}
		response += "\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit"
		s.Stage = "post_action"

	case "post_action":
		if body == "1" {
			s.Stage = "main_menu"
			response = mainMenuText(s.Name)
		} else if body == "0" || strings.Contains(body, "no") {
			delete(sessions, from)
			response = "👋 Thank you for using WalletBot! Goodbye!"
		} else {
			response = "Please choose:\n1️⃣ Main Menu\n0️⃣ Exit"
		}

	default:
		response = "Session expired. Say 'Hi' to start again."
		delete(sessions, from)
	}

	respondXML(w, response)
}

func mainMenuText(name string) string {
	return fmt.Sprintf("Good day, %s 👋\n\nWhat would you like to do today?\n\n1️⃣ Check Balance\n2️⃣ Send Money\n3️⃣ Buy Airtime\n4️⃣ Pay Bills\n5️⃣ View Transactions\n6️⃣ Talk to Support", name)
}

func respondXML(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/xml")
	xml.NewEncoder(w).Encode(MessageResponse{Message: msg})
}

func parseAmount(s string) (float64, error) {
	s = strings.ReplaceAll(s, "$", "")
	fields := strings.Fields(s)
	if len(fields) > 0 {
		s = fields[0]
	}
	return strconv.ParseFloat(s, 64)
}

func isNumeric(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
