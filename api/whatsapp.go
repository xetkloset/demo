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

// Session store (shared)
var sessions = make(map[string]*Session)
var mu sync.Mutex

// Global loan store (shared across sessions for demo)
var loans = make(map[string]*Loan)
var loanMu sync.Mutex
var loanCounter int

// Session represents a user session (single shared session per WhatsApp number)
type Session struct {
	Name             string
	Stage            string
	PIN              string
	Balance          float64
	PendingName      string
	PendingAmt       float64
	Transactions     []string
	Role             string // "member", "mufundisi", "elder", "recommender" (we treat recommender as member with flag)
	Region           string // "Tabhera" or "Nyika"
	HasFullRepayment bool   // recommender flag
}

// Loan model
type Loan struct {
	ID               string
	ApplicantName    string
	ApplicantID      string
	Region           string
	RequestedAmount  float64
	Status           string // pending, approved, declined
	MufundisiApproved bool
	ElderApprovals   map[string]bool // keyed by approver name
	ApprovalReasons  map[string]string
	Recommendations  []string // recommender names
	ApprovedLimit    float64
	TermMonths       int
	DeclineReason    string
	Borrowed         float64
}

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
		// default session
		s = &Session{
			Stage:        "ask_pin",
			Balance:      500,
			Transactions: []string{},
			Role:         "member",
			Region:       "Tabhera",
		}
		sessions[from] = s
	}
	mu.Unlock()

	response := ""

	// quick role switch shortcut: "role <name>" still supported, but main UI uses switch role menu
	if strings.HasPrefix(body, "role ") {
		role := strings.TrimSpace(strings.TrimPrefix(body, "role "))
		switch role {
		case "member", "mufundisi", "elder", "recommender":
			s.Role = role
			response = fmt.Sprintf("🔁 Role switched to %s. Region: %s\n\nGo to Main Menu -> 7 for Microfin Loan.", strings.Title(role), s.Region)
			respondXML(w, response)
			return
		default:
			response = "Unknown role. Valid: member, mufundisi, elder, recommender."
			respondXML(w, response)
			return
		}
	}

	switch s.Stage {

	case "ask_pin":
		response = "👋 Welcome! Please enter your 4-digit PIN to continue."
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
		case "7":
			s.Stage = "loan_menu"
			response = loanMenuText(s)
		default:
			response = "❓ Please choose a valid option (1–7)."
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
				s.Transactions = append([]string{tx}, s.Transactions...)
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
			respondXML(w, response)
			return
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

	// ---------- LOAN MENU ----------
	case "loan_menu":
		switch strings.TrimSpace(body) {
		case "1": // Request Loan
			s.Stage = "loan_request_name"
			response = "Loan Request — Enter applicant *name*:"
		case "2": // View Loan Status
			response = viewLoansForApplicant(s.Name)
		case "3": // Recommend Borrower
			// show list of pending loans in same region
			s.Stage = "recommend_list"
			response = recommendListPrompt(s)
		case "4": // Switch Role
			s.Stage = "switch_role_menu"
			response = switchRoleMenuText()
		case "5": // Borrow Funds
			s.Stage = "borrow_list"
			response = borrowListPrompt(s)
		case "6": // Approve Loans (for approvers)
			if s.Role != "mufundisi" && s.Role != "elder" {
				response = "To approve loans switch to role Mufundisi or Elder first. Use Switch Role (option 4)."
			} else {
				s.Stage = "approver_list"
				response = approverListPrompt(s)
			}
		case "7": // Toggle recommender full repayment flag
			s.HasFullRepayment = !s.HasFullRepayment
			response = fmt.Sprintf("🔁 Full repayment status toggled: %v", s.HasFullRepayment)
		case "0":
			s.Stage = "main_menu"
			response = mainMenuText(s.Name)
		default:
			response = loanMenuText(s)
		}

	// Loan request sub-steps
	case "loan_request_name":
		s.PendingName = strings.Title(body)
		s.Stage = "loan_request_id"
		response = "Enter applicant ID:"

	case "loan_request_id":
		s.PIN = strings.ToUpper(strings.TrimSpace(body)) // temporarily store applicant ID in PIN
		s.Stage = "loan_request_region_choice"
		response = "Select applicant region:\n1️⃣ Tabhera\n2️⃣ Nyika"

	case "loan_request_region_choice":
		if body == "1" {
			s.Region = "Tabhera"
		} else if body == "2" {
			s.Region = "Nyika"
		} else {
			response = "Please choose 1 for Tabhera or 2 for Nyika."
			respondXML(w, response)
			return
		}
		s.Stage = "loan_request_amount"
		response = "Enter requested loan amount (e.g., 300):"

	case "loan_request_amount":
		amt, err := parseAmount(body)
		if err != nil {
			response = "❌ Invalid amount. Try again (e.g., 300 or $300)."
			break
		}
		loan := createLoan(s.PendingName, s.PIN, s.Region, amt, s.Name)
		response = fmt.Sprintf("✅ Loan request submitted with ID: %s\nStatus: pending (awaiting Mufundisi approval)\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit", loan.ID)
		s.Stage = "post_action"

	// Recommend list: user types loan ID to recommend or 'back'
	case "recommend_list":
		lid := strings.ToUpper(strings.TrimSpace(body))
		if lid == "BACK" || lid == "0" {
			s.Stage = "loan_menu"
			response = loanMenuText(s)
			respondXML(w, response)
			return
		}
		loanMu.Lock()
		loan, ok := loans[lid]
		if !ok {
			loanMu.Unlock()
			response = "Loan ID not found. Please type a valid Loan ID from the list or 'back'."
			respondXML(w, response)
			return
		}
		// ensure same region
		if !strings.EqualFold(loan.Region, s.Region) {
			loanMu.Unlock()
			response = "You can only recommend borrowers in your region."
			respondXML(w, response)
			return
		}
		// recommender must have full repayment flag
		if !s.HasFullRepayment {
			loanMu.Unlock()
			response = "⚠️ Your recommendation will not count unless you toggle full-repayment status in Loan Menu -> option 7."
			respondXML(w, response)
			return
		}
		// check duplicate
		for _, r := range loan.Recommendations {
			if strings.EqualFold(r, s.Name) {
				loanMu.Unlock()
				response = "✅ You already recommended this borrower."
				respondXML(w, response)
				return
			}
		}
		loan.Recommendations = append(loan.Recommendations, s.Name)
		computeLoanLimits(loan)
		current := loan.ApprovedLimit
		loanMu.Unlock()
		response = fmt.Sprintf("✅ Recommendation recorded for loan %s. Current approved limit: $%.2f", lid, current)
		s.Stage = "loan_menu"

	// Approver list stage: approver chooses loan ID to act on
	case "approver_list":
		if s.Role != "mufundisi" && s.Role != "elder" {
			s.Stage = "loan_menu"
			response = "Switch to approver role first."
			respondXML(w, response)
			return
		}
		lid := strings.ToUpper(strings.TrimSpace(body))
		if lid == "BACK" || lid == "0" {
			s.Stage = "loan_menu"
			response = loanMenuText(s)
			respondXML(w, response)
			return
		}
		loanMu.Lock()
		loan, exists := loans[lid]
		if !exists {
			loanMu.Unlock()
			response = "Loan ID not found. Type the Loan ID shown in the list or 'back'."
			respondXML(w, response)
			return
		}
		if !strings.EqualFold(loan.Region, s.Region) {
			loanMu.Unlock()
			response = "You can only act on loans in your region."
			respondXML(w, response)
			return
		}
		loanMu.Unlock()
		// move to action stage
		s.Stage = "approver_action:" + lid
		response = fmt.Sprintf("You selected loan %s for %s. Type 'approve' to approve or 'decline <reason>' to decline.", lid, loan.ApplicantName)

	// Approver action stage
	default:
		// approver action
		if strings.HasPrefix(s.Stage, "approver_action:") {
			loanID := strings.SplitN(s.Stage, ":", 2)[1]
			cmd := strings.TrimSpace(body)
			loanMu.Lock()
			loan, exists := loans[loanID]
			if !exists {
				loanMu.Unlock()
				response = "Loan not found. Returning to loan menu."
				s.Stage = "loan_menu"
				respondXML(w, response)
				return
			}
			if strings.HasPrefix(cmd, "approve") {
				// record approval
				if s.Role == "mufundisi" {
					loan.MufundisiApproved = true
					if loan.ApprovalReasons == nil {
						loan.ApprovalReasons = map[string]string{}
					}
					loan.ApprovalReasons[s.Name] = "approved"
					computeLoanLimits(loan)
					loan.Status = "approved"
					response = fmt.Sprintf("✅ Mufundisi approved loan %s. Approved limit: $%.2f. Term: %d months.", loan.ID, loan.ApprovedLimit, loan.TermMonths)
				} else if s.Role == "elder" {
					if loan.ElderApprovals == nil {
						loan.ElderApprovals = map[string]bool{}
					}
					loan.ElderApprovals[s.Name] = true
					if loan.ApprovalReasons == nil {
						loan.ApprovalReasons = map[string]string{}
					}
					loan.ApprovalReasons[s.Name] = "approved"
					// If no mufundisi yet, elders shouldn't set status to approved by themselves
					computeLoanLimits(loan)
					if loan.MufundisiApproved {
						loan.Status = "approved"
					}
					response = fmt.Sprintf("✅ Elder approved loan %s. Approved limit: $%.2f. Term: %d months.", loan.ID, loan.ApprovedLimit, loan.TermMonths)
				} else {
					response = "Only Mufundisi or Elder can approve loans."
				}
				loanMu.Unlock()
				s.Stage = "loan_menu"
				respondXML(w, response)
				return
			} else if strings.HasPrefix(cmd, "decline") {
				reason := strings.TrimSpace(strings.TrimPrefix(cmd, "decline"))
				if reason == "" {
					reason = "no reason provided"
				}
				if loan.ApprovalReasons == nil {
					loan.ApprovalReasons = map[string]string{}
				}
				loan.ApprovalReasons[s.Name] = "declined: " + reason
				loan.Status = "declined"
				loan.DeclineReason = reason
				loanMu.Unlock()
				s.Stage = "loan_menu"
				response = fmt.Sprintf("❌ You declined loan %s. Reason: %s", loanID, reason)
				respondXML(w, response)
				return
			} else {
				loanMu.Unlock()
				response = "Unknown command. Type 'approve' or 'decline <reason>'."
				respondXML(w, response)
				return
			}
		}

		// Borrow list stage: user chooses which approved loan to borrow from (if they are the applicant)
		if s.Stage == "borrow_list" {
			lid := strings.ToUpper(strings.TrimSpace(body))
			if lid == "BACK" || lid == "0" {
				s.Stage = "loan_menu"
				response = loanMenuText(s)
				respondXML(w, response)
				return
			}
			loanMu.Lock()
			ln, exists := loans[lid]
			if !exists {
				loanMu.Unlock()
				response = "Loan ID not found. Type the Loan ID or 'back'."
				respondXML(w, response)
				return
			}
			if !strings.EqualFold(ln.ApplicantName, s.Name) {
				loanMu.Unlock()
				response = "You can only borrow from your own approved loans."
				respondXML(w, response)
				return
			}
			if ln.Status != "approved" {
				loanMu.Unlock()
				response = "Loan is not approved yet."
				respondXML(w, response)
				return
			}
			maxAvailable := ln.ApprovedLimit - ln.Borrowed
			if maxAvailable <= 0 {
				loanMu.Unlock()
				response = "No funds available to borrow (limit fully used)."
				respondXML(w, response)
				return
			}
			loanMu.Unlock()
			s.Stage = "borrow_amount:" + lid
			response = fmt.Sprintf("Loan %s approved. Enter amount to borrow (max $%.2f):", lid, maxAvailable)
			respondXML(w, response)
			return
		}

		// borrow amount stage
		if strings.HasPrefix(s.Stage, "borrow_amount:") {
			lid := strings.SplitN(s.Stage, ":", 2)[1]
			amt, err := parseAmount(body)
			if err != nil {
				response = "Invalid amount. Try again."
				respondXML(w, response)
				return
			}
			loanMu.Lock()
			ln, exists := loans[lid]
			if !exists {
				loanMu.Unlock()
				response = "Loan not found."
				loanMu.Unlock()
				s.Stage = "loan_menu"
				respondXML(w, response)
				return
			}
			if !strings.EqualFold(ln.ApplicantName, s.Name) {
				loanMu.Unlock()
				response = "You can only borrow from your own loan."
				respondXML(w, response)
				return
			}
			if ln.Status != "approved" {
				loanMu.Unlock()
				response = "Loan is not approved."
				respondXML(w, response)
				return
			}
			maxAvailable := ln.ApprovedLimit - ln.Borrowed
			if amt <= 0 || amt > maxAvailable {
				loanMu.Unlock()
				response = fmt.Sprintf("Invalid amount. Enter an amount up to $%.2f.", maxAvailable)
				respondXML(w, response)
				return
			}
			// disburse
			ln.Borrowed += amt
			lnStr := fmt.Sprintf("Loan disbursed: $%.2f (Loan ID: %s)", amt, ln.ID)
			s.Balance += amt
			s.Transactions = append([]string{lnStr}, s.Transactions...)
			loanMu.Unlock()
			s.Stage = "loan_menu"
			response = fmt.Sprintf("✅ $%.2f disbursed to your wallet. New balance: $%.2f", amt, s.Balance)
			respondXML(w, response)
			return
		}

		// fallback for unknown states
		response = "Session expired or unknown state. Say 'Hi' to start again."
		delete(sessions, from)
	}

	respondXML(w, response)
}

// ------- Helper UI / logic functions -------

func mainMenuText(name string) string {
	return fmt.Sprintf("Good day, %s 👋\n\nWhat would you like to do today?\n\n1️⃣ Check Balance\n2️⃣ Send Money\n3️⃣ Buy Airtime\n4️⃣ Pay Bills\n5️⃣ View Transactions\n6️⃣ Talk to Support\n7️⃣ Microfin Loan 💸\n\nTip: After entering Loan Menu you can switch roles and regions for demo.", name)
}

func loanMenuText(s *Session) string {
	return fmt.Sprintf("🏦 Microfin Loan Menu — Role: %s | Region: %s\n\n1️⃣ Request Loan\n2️⃣ View Loan Status\n3️⃣ Recommend Borrower\n4️⃣ Switch Role\n5️⃣ Borrow Funds\n6️⃣ Approve Loans (approvers only)\n7️⃣ Toggle full-repayment status (recommenders)\n0️⃣ Back to Main Menu\n\n(Use numeric choices)", strings.Title(s.Role), s.Region)
}

func switchRoleMenuText() string {
	return "Select your role:\n1️⃣ Member / Requester\n2️⃣ Mufundisi (Approver)\n3️⃣ Elder (Approver)\n4️⃣ Back"
}

// respondXML encodes TwiML response
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

// createLoan creates a loan with a unique ID
func createLoan(name, appid, region string, amount float64, submittedBy string) *Loan {
	loanMu.Lock()
	defer loanMu.Unlock()
	loanCounter++
	id := fmt.Sprintf("L%04d", loanCounter)
	ln := &Loan{
		ID:               id,
		ApplicantName:    name,
		ApplicantID:      appid,
		Region:           region,
		RequestedAmount:  amount,
		Status:           "pending",
		ElderApprovals:   map[string]bool{},
		ApprovalReasons:  map[string]string{},
		Recommendations:  []string{},
		ApprovedLimit:    0,
		TermMonths:       0,
		Borrowed:         0,
	}
	// keep initial compute (none approved yet)
	computeLoanLimits(ln)
	loans[id] = ln
	// log creation for debugging in global space (not user facing)
	_ = submittedBy
	return ln
}

// computeLoanLimits calculates ApprovedLimit and TermMonths based on approvals and recommendations
func computeLoanLimits(loan *Loan) {
	base := 0.0
	term := 0
	if loan.MufundisiApproved {
		elderCount := countTrue(loan.ElderApprovals)
		if elderCount >= 2 {
			base = 800
			term = 9
		} else if elderCount == 1 {
			base = 500
			term = 6
		} else {
			base = 300
			term = 6
		}
	} else {
		base = 0
		term = 0
	}
	// count unique recommendations, max 2
	recCount := 0
	seen := map[string]bool{}
	for _, r := range loan.Recommendations {
		n := strings.ToLower(strings.TrimSpace(r))
		if n == "" {
			continue
		}
		if !seen[n] {
			seen[n] = true
			recCount++
			if recCount >= 2 {
				break
			}
		}
	}
	additional := float64(recCount * 100)
	total := base + additional
	if total > 1000 {
		total = 1000
	}
	loan.ApprovedLimit = total
	loan.TermMonths = term
	// if mufundisi approved at least, status can be approved (we still allow elders to increase limit)
	if loan.MufundisiApproved {
		loan.Status = "approved"
	}
}

// viewLoansForApplicant returns readable loans for the caller
func viewLoansForApplicant(name string) string {
	loanMu.Lock()
	defer loanMu.Unlock()
	out := ""
	found := false
	for _, l := range loans {
		if strings.EqualFold(l.ApplicantName, name) {
			found = true
			out += fmt.Sprintf("ID: %s\nApplicant: %s\nRegion: %s\nRequested: $%.2f\nStatus: %s\nApproved Limit: $%.2f\nTerm: %d months\nRecommendations: %d\nApprovals: Mufundisi: %v, Elders: %d\nBorrowed: $%.2f\nDecline reason: %s\n\n",
				l.ID, l.ApplicantName, l.Region, l.RequestedAmount, l.Status, l.ApprovedLimit, l.TermMonths, len(l.Recommendations), l.MufundisiApproved, countTrue(l.ElderApprovals), l.Borrowed, l.DeclineReason)
		}
	}
	if !found {
		return "No loan applications found for you.\n\nTo request a loan: Loan Menu -> 1"
	}
	return out
}

func countTrue(m map[string]bool) int {
	c := 0
	for _, v := range m {
		if v {
			c++
		}
	}
	return c
}

// approverListPrompt lists pending loans in approver's region
func approverListPrompt(s *Session) string {
	loanMu.Lock()
	defer loanMu.Unlock()
	out := "Pending loans in your region:\n\n"
	count := 0
	for _, l := range loans {
		if l.Status == "pending" && strings.EqualFold(l.Region, s.Region) {
			out += fmt.Sprintf("ID: %s | Applicant: %s | Requested: $%.2f\n", l.ID, l.ApplicantName, l.RequestedAmount)
			count++
		}
	}
	if count == 0 {
		out = "No pending loans in your region.\n\nType 0 to go back."
	}
	out += "\n\nType the Loan ID to act on (or 'back')."
	return out
}

// recommendListPrompt lists loans in same region that can be recommended
func recommendListPrompt(s *Session) string {
	loanMu.Lock()
	defer loanMu.Unlock()
	out := "Loans in your region (pending/approved):\n\n"
	count := 0
	for _, l := range loans {
		// allow recommending pending or approved loans in same region
		if strings.EqualFold(l.Region, s.Region) {
			out += fmt.Sprintf("ID: %s | Applicant: %s | Status: %s | Requested: $%.2f | Recs: %d\n", l.ID, l.ApplicantName, l.Status, l.RequestedAmount, len(l.Recommendations))
			count++
		}
	}
	if count == 0 {
		out = "No loans found in your region to recommend."
	}
	out += "\n\nType the Loan ID to recommend or 'back'."
	return out
}

// borrowListPrompt lists approved loans for this session's user
func borrowListPrompt(s *Session) string {
	loanMu.Lock()
	defer loanMu.Unlock()
	out := "Your approved loans:\n\n"
	count := 0
	for _, l := range loans {
		if strings.EqualFold(l.ApplicantName, s.Name) && l.Status == "approved" {
			out += fmt.Sprintf("ID: %s | Limit: $%.2f | Borrowed: $%.2f | Available: $%.2f\n", l.ID, l.ApprovedLimit, l.Borrowed, l.ApprovedLimit-l.Borrowed)
			count++
		}
	}
	if count == 0 {
		out = "You have no approved loans to borrow from.\n\nType 0 to go back."
	}
	out += "\n\nType Loan ID to borrow or 'back'."
	return out
}
