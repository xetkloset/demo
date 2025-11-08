package handler

import (
	"fmt"
	"net/http"
	"strings"
)

// simple in-memory session simulation (for demo only)
var userState = map[string]string{}

func Handler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	r.ParseForm()
	from := r.FormValue("From")
	body := strings.TrimSpace(strings.ToLower(r.FormValue("Body")))
	response := ""

	state := userState[from]

	switch state {
	case "":
		response = "👋 Welcome! Please enter your 4-digit PIN to continue."
		userState[from] = "awaiting_pin"

	case "awaiting_pin":
		if len(body) == 4 {
			response = "Good day 👋\nWhat would you like to do today?\n\n3️⃣ Main Menu\n1️⃣ Check Balance\n2️⃣ Send Money\n3️⃣ Buy Airtime\n4️⃣ Pay Bills\n5️⃣ View Transactions\n6️⃣ Talk to Support"
			userState[from] = "main_menu"
		} else {
			response = "❌ Invalid PIN format. Please enter your 4-digit PIN."
		}

	case "main_menu":
		switch body {
		case "1":
			response = "💰 Your current balance is $480."
			userState[from] = "post_action"
		case "2":
			response = "Who would you like to send money to?"
			userState[from] = "send_money_name"
		case "3":
			response = "Enter amount and mobile number. Example: $2 to 0772123456"
			userState[from] = "airtime_input"
		case "4":
			response = "Pay Bills: Coming soon 💡"
			userState[from] = "post_action"
		case "5":
			response = "Recent Transactions:\n- Sent $20 to Friend ✅\n- Bought Airtime $2 📱\n- Received $50 from Lisa 💵"
			userState[from] = "post_action"
		case "6":
			response = "I can help you with:\n1️⃣ Lost Card\n2️⃣ Transaction Issue\n3️⃣ Talk to Agent"
			userState[from] = "support_menu"
		default:
			response = "Please select a valid option (1–6)."
		}

	case "send_money_name":
		response = fmt.Sprintf("How much would you like to send to %s?", strings.Title(body))
		userState[from] = "send_money_amount"

	case "send_money_amount":
		response = fmt.Sprintf("Send %s to Tawanda M.? ✅ Yes / ❌ No", strings.ToUpper(body))
		userState[from] = "confirm_send"

	case "confirm_send":
		if strings.Contains(body, "yes") || strings.Contains(body, "✅") {
			response = "Please confirm with your PIN."
			userState[from] = "confirm_pin"
		} else {
			response = "Transaction cancelled. Returning to main menu."
			userState[from] = "main_menu"
		}

	case "confirm_pin":
		if len(body) == 4 {
			response = "✅ Transaction successful!\nNew balance: $480.\nReceipt sent.\n\nIs there anything else I can help you with?"
			userState[from] = "end_session"
		} else {
			response = "❌ Invalid PIN. Try again."
		}

	case "airtime_input":
		if strings.Contains(body, "to") {
			response = "✅ Airtime purchase successful!"
			userState[from] = "post_action"
		} else {
			response = "Please enter both amount and number (e.g. $2 to 0772123456)."
		}

	case "support_menu":
		switch body {
		case "1":
			response = "Lost card? Please call our 24/7 line: 0800 123 456 🧾"
		case "2":
			response = "Transaction issue logged. Our support will contact you shortly 🕑"
		case "3":
			response = "Please wait while I connect you to an agent 👩🏾‍💼"
		default:
			response = "Please choose 1️⃣ 2️⃣ or 3️⃣."
			returnXML(w, response)
			return
		}
		response += "\n\nIs there anything else I can help you with?"
		userState[from] = "end_session"

	case "post_action":
		response = "Is there anything else I can help you with?"
		userState[from] = "end_session"

	case "end_session":
		if body == "no" {
			response = "👋 Thank you for using WalletBot. Goodbye!"
			delete(userState, from)
		} else {
			response = "Returning to Main Menu.\n\n1️⃣ Check Balance\n2️⃣ Send Money\n3️⃣ Buy Airtime\n4️⃣ Pay Bills\n5️⃣ View Transactions\n6️⃣ Talk to Support"
			userState[from] = "main_menu"
		}

	default:
		response = "Let's start over. Please enter your 4-digit PIN."
		userState[from] = "awaiting_pin"
	}

	returnXML(w, response)
}

func returnXML(w http.ResponseWriter, msg string) {
	w.Header().Set("Content-Type", "application/xml")
	fmt.Fprintf(w, `<Response><Message>%s</Message></Response>`, msg)
}
