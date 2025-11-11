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

// Translations map: language -> key -> translated text
var translations = map[string]map[string]string{
	"en": { // English
		"welcome":               "👋 Welcome! Please enter your 4-digit PIN to continue.",
		"pin_accepted":          "✅ PIN accepted! Please enter your name to continue.",
		"pin_invalid":           "❌ Invalid PIN. Please enter a 4-digit PIN.",
		"good_day":              "Good day, %s 👋\n\nWhat would you like to do today?",
		"menu_tip":              "\n\nTip: After entering Loan Menu you can switch roles and regions for demo.",
		"menu_1_balance":        "1️⃣ Check Balance",
		"menu_2_send":           "2️⃣ Send Money",
		"menu_3_airtime":        "3️⃣ Buy Airtime",
		"menu_4_bills":          "4️⃣ Pay Bills",
		"menu_5_transactions":   "5️⃣ View Transactions",
		"menu_6_support":        "6️⃣ Talk to Support",
		"menu_7_loan":           "7️⃣ Microfin Loan 💸",
		"menu_8_language":       "8️⃣ Change Language 🌍",
		"your_balance":          "💰 Your current balance is $%.2f\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit",
		"send_to_who":           "Who would you like to send money to?",
		"send_how_much":         "How much would you like to send to %s?",
		"invalid_amount":        "❌ Invalid amount. Try again (e.g., 20 or $20).",
		"confirm_send":          "Send $%.2f to %s? ✅ Yes / ❌ No",
		"transaction_success":   "✅ Transaction successful!\nNew balance: $%.2f\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit",
		"insufficient_funds":    "⚠️ Insufficient funds.",
		"transaction_cancelled": "❌ Transaction cancelled.\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit",
		"sent_to":               "Sent $%.2f to %s ✅",
		"airtime_prompt":        "Enter amount and mobile number (e.g. $2 to 0772123456)",
		"airtime_invalid":       "❌ Invalid format. Try again (e.g., $2 to 0772123456).",
		"airtime_success":       "✅ Airtime purchase successful! New balance: $%.2f\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit",
		"bought_airtime":        "Bought $%.2f airtime 📱",
		"not_enough_balance":    "⚠️ Not enough balance.",
		"bills_demo":            "⚙️ Bill payment demo not active.\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit",
		"recent_transactions":   "🧾 Recent Transactions:\n%s\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit",
		"no_transactions":       "No transactions yet",
		"support_menu":          "I can help you with:\n1️⃣ Lost Card\n2️⃣ Transaction Issue\n3️⃣ Talk to Agent",
		"support_lost_card":     "🧾 Lost Card: Please call 0800 123 456.",
		"support_issue_logged":  "⚙️ Transaction Issue logged.",
		"support_agent":         "👩🏾‍💼 Connecting to an agent...",
		"choose_valid_support":  "❓ Please choose 1, 2, or 3.",
		"post_action_menu":      "Please choose:\n1️⃣ Main Menu\n0️⃣ Exit",
		"goodbye":               "👋 Thank you for using WalletBot! Goodbye!",
		"choose_valid_option":   "❓ Please choose a valid option (1–8).",
		"loan_menu_title":       "🏦 Microfin Loan Menu — Role: %s | Region: %s\n\n",
		"loan_menu_1":           "1️⃣ Request Loan",
		"loan_menu_2":           "2️⃣ View Loan Status",
		"loan_menu_3":           "3️⃣ Recommend Borrower",
		"loan_menu_4":           "4️⃣ Switch Role",
		"loan_menu_5":           "5️⃣ Borrow Funds",
		"loan_menu_6":           "6️⃣ Approve Loans",
		"loan_menu_0":           "0️⃣ Back to Main Menu",
		"loan_menu_note":        "\n\n(Use numeric choices)",
		"loan_request_name":     "Loan Request — Enter applicant *name*:",
		"loan_request_id":       "Enter applicant ID:",
		"loan_request_region":   "Select applicant region:\n1️⃣ Tabhera\n2️⃣ Nyika",
		"loan_request_amount":   "Enter requested loan amount (e.g., 300):",
		"loan_submitted":        "✅ Loan request submitted with ID: %s\nStatus: pending (awaiting Mufundisi approval)\n\nWould you like to do anything else?\n1️⃣ Main Menu\n0️⃣ Exit",
		"choose_region":         "Please choose 1 for Tabhera or 2 for Nyika.",
		"recommend_title":       "📋 Borrowers awaiting recommendation:\n\n",
		"recommend_none":        "✅ No borrowers awaiting recommendation in your region.",
		"recommend_footer":      "\nReply with a number (1–%d) or 0️⃣ to go back.",
		"recommend_invalid":     "❌ Invalid choice. Please reply with a valid number.",
		"recommend_not_found":   "Loan not found.",
		"recommend_question":    "Would you like to recommend %s?\n1️⃣ Yes\n2️⃣ No",
		"recommend_yes_no":      "Please reply with 1️⃣ Yes or 2️⃣ No.",
		"recommend_success":     "✅ Recommendation recorded for %s.",
		"recommend_already":     "✅ You already recommended this borrower.",
		"recommend_reason":      "Please provide a reason for not recommending:",
		"not_recommended":       "❌ Not recommended (%s).",
		"approver_switch":       "To approve loans switch to role Mufundisi or Elder first. Use Switch Role (option 4).",
		"role_switched":         "🔁 Role switched to %s. Region: %s\n\nGo to Main Menu -> 7 for Microfin Loan.",
		"role_unknown":          "Unknown role. Valid: member, mufundisi, elder, recommender.",
		"switch_role_menu":      "Select your role:\n1️⃣ Member / Requester\n2️⃣ Mufundisi (Approver)\n3️⃣ Elder (Approver)\n4️⃣ Back",
		"language_menu":         "🌍 Choose your language / Sarudza mutauro / Khetha ulimi lwakho:\n\n1️⃣ English\n2️⃣ Shona\n3️⃣ Ndebele\n0️⃣ Back",
		"language_changed":      "✅ Language changed to %s",
	},
	"sn": { // Shona
		"welcome":               "👋 Mauya! Ndapota isa PIN yako ine manhamba mana.",
		"pin_accepted":          "✅ PIN yakagamuchirwa! Ndapota isa zita rako.",
		"pin_invalid":           "❌ PIN isiri yechokwadi. Ndapota isa PIN ine manhamba mana.",
		"good_day":              "Mhoro, %s 👋\n\nUngada kuita chii nhasi?",
		"menu_tip":              "\n\nChiziviso: Mushure mekupinda muMenu yeChikwereti unogona kushandura mabasa nematunhu.",
		"menu_1_balance":        "1️⃣ Tarisa Mari Yangu",
		"menu_2_send":           "2️⃣ Tumira Mari",
		"menu_3_airtime":        "3️⃣ Tenga Airtime",
		"menu_4_bills":          "4️⃣ Bhadhara Mabhiri",
		"menu_5_transactions":   "5️⃣ Ona Zvakaitika",
		"menu_6_support":        "6️⃣ Taura neRubatsiro",
		"menu_7_loan":           "7️⃣ Chikwereti cheMicrofin 💸",
		"menu_8_language":       "8️⃣ Shandura Mutauro 🌍",
		"your_balance":          "💰 Mari yako yakasvika $%.2f\n\nUngade kuita chimwe chinhu here?\n1️⃣ Menu Huru\n0️⃣ Buda",
		"send_to_who":           "Ungade kutumira mari kuna ani?",
		"send_how_much":         "Ungade kutumira mari yakawanda sei kuna %s?",
		"invalid_amount":        "❌ Mari isiri yechokwadi. Edza zvakare (somuenzaniso, 20 kana $20).",
		"confirm_send":          "Tumira $%.2f kuna %s? ✅ Hongu / ❌ Kwete",
		"transaction_success":   "✅ Kutumira kwakafambira mberi!\nMari yatsva: $%.2f\n\nUngade kuita chimwe chinhu here?\n1️⃣ Menu Huru\n0️⃣ Buda",
		"insufficient_funds":    "⚠️ Mari haina kukwana.",
		"transaction_cancelled": "❌ Kutumira kwakamiswa.\n\nUngade kuita chimwe chinhu here?\n1️⃣ Menu Huru\n0️⃣ Buda",
		"sent_to":               "Kutumira $%.2f kuna %s ✅",
		"airtime_prompt":        "Isa mari nenhamba (somuenzaniso $2 ku 0772123456)",
		"airtime_invalid":       "❌ Chisiri chechokwadi. Edza zvakare (somuenzaniso, $2 ku 0772123456).",
		"airtime_success":       "✅ Kutenga airtime kwakafambira mberi! Mari yatsva: $%.2f\n\nUngade kuita chimwe chinhu here?\n1️⃣ Menu Huru\n0️⃣ Buda",
		"bought_airtime":        "Kutenga $%.2f airtime 📱",
		"not_enough_balance":    "⚠️ Mari haina kukwana.",
		"bills_demo":            "⚙️ Kubhadhara mabhiri hakusati kwatanga kushanda.\n\nUngade kuita chimwe chinhu here?\n1️⃣ Menu Huru\n0️⃣ Buda",
		"recent_transactions":   "🧾 Zvakaita Zvekupedzisira:\n%s\n\nUngade kuita chimwe chinhu here?\n1️⃣ Menu Huru\n0️⃣ Buda",
		"no_transactions":       "Hapana zvakaita parizvino",
		"support_menu":          "Ndinogona kukubatsira ne:\n1️⃣ Kadhi Rakarasika\n2️⃣ Dambudziko Rekutumira\n3️⃣ Taura neMumiriri",
		"support_lost_card":     "🧾 Kadhi Rakarasika: Ndapota fona 0800 123 456.",
		"support_issue_logged":  "⚙️ Dambudziko ranyorwa.",
		"support_agent":         "👩🏾‍💼 Tiri kukubatanidza nemumiriri...",
		"choose_valid_support":  "❓ Ndapota sarudza 1, 2, kana 3.",
		"post_action_menu":      "Ndapota sarudza:\n1️⃣ Menu Huru\n0️⃣ Buda",
		"goodbye":               "👋 Tinotenda kushandisa WalletBot! Sara zvakanaka!",
		"choose_valid_option":   "❓ Ndapota sarudza sarudzo chaiyo (1–8).",
		"loan_menu_title":       "🏦 Menu yeChikwereti cheMicrofin — Basa: %s | Dunhu: %s\n\n",
		"loan_menu_1":           "1️⃣ Kumbira Chikwereti",
		"loan_menu_2":           "2️⃣ Ona Chikwereti Changu",
		"loan_menu_3":           "3️⃣ Kurudzira Mukwereti",
		"loan_menu_4":           "4️⃣ Shandura Basa",
		"loan_menu_5":           "5️⃣ Tora Mari Yakabvumidzwa",
		"loan_menu_6":           "6️⃣ Bvumidza Zvikwereti",
		"loan_menu_0":           "0️⃣ Dzokera kuMenu Huru",
		"loan_menu_note":        "\n\n(Shandisa nhamba)",
		"loan_request_name":     "Chikwereti — Isa *zita* remunyoreri:",
		"loan_request_id":       "Isa ID yemunyoreri:",
		"loan_request_region":   "Sarudza dunhu remunyoreri:\n1️⃣ Tabhera\n2️⃣ Nyika",
		"loan_request_amount":   "Isa mari yechikwereti (somuenzaniso, 300):",
		"loan_submitted":        "✅ Chikwereti chaendeswa neID: %s\nChimiro: Chakamirira kubvumidzwa naMufundisi\n\nUngade kuita chimwe chinhu here?\n1️⃣ Menu Huru\n0️⃣ Buda",
		"choose_region":         "Ndapota sarudza 1 yeTabhera kana 2 yeNyika.",
		"recommend_title":       "📋 Vanhu vari kumirira kurudzirwa:\n\n",
		"recommend_none":        "✅ Hapana munhu arikumirira kurudzirwa mudunhu rako.",
		"recommend_footer":      "\nPindura nenhamba (1–%d) kana 0️⃣ kudzokera.",
		"recommend_invalid":     "❌ Sarudzo isiri yechokwadi. Ndapota sarudza nhamba chaiyo.",
		"recommend_not_found":   "Chikwereti hachina kuwanikwa.",
		"recommend_question":    "Ungade kurudzira %s here?\n1️⃣ Hongu\n2️⃣ Kwete",
		"recommend_yes_no":      "Ndapota pindura 1️⃣ Hongu kana 2️⃣ Kwete.",
		"recommend_success":     "✅ Kurudziro kwakanyorwa kuna %s.",
		"recommend_already":     "✅ Watozvikurudzira munhu uyu.",
		"recommend_reason":      "Ndapota ipa chikonzero chekusarudzira:",
		"not_recommended":       "❌ Haina kurudzirwa (%s).",
		"approver_switch":       "Kuti ubvumidze zvikwereti shandura basa kuMufundisi kana Mukuru. Shandisa Shandura Basa (sarudzo 4).",
		"role_switched":         "🔁 Basa rakashandurwa kuita %s. Dunhu: %s\n\nEnda kuMenu Huru -> 7 yeChikwereti.",
		"role_unknown":          "Basa risingazivikanwe. Mabasa: member, mufundisi, elder, recommender.",
		"switch_role_menu":      "Sarudza basa rako:\n1️⃣ Nhengo / Munyoreri\n2️⃣ Mufundisi (Mubvumidzi)\n3️⃣ Mukuru (Mubvumidzi)\n4️⃣ Dzoka",
		"language_menu":         "🌍 Choose your language / Sarudza mutauro / Khetha ulimi lwakho:\n\n1️⃣ English\n2️⃣ Shona\n3️⃣ Ndebele\n0️⃣ Back / Dzoka / Buyela",
		"language_changed":      "✅ Mutauro wakashandurwa kuita %s",
	},
	"nd": { // Ndebele
		"welcome":               "👋 Siyekelele! Sicela ufake i-PIN yakho enezinombolo ezine.",
		"pin_accepted":          "✅ I-PIN yamukelwe! Sicela ufake igama lakho.",
		"pin_invalid":           "❌ I-PIN engalungile. Sicela ufake i-PIN enezinombolo ezine.",
		"good_day":              "Livukile, %s 👋\n\nUfunani ukwenza namhlanje?",
		"menu_tip":              "\n\nIcebo: Ngemva kokungena ku-Menu Yezemalimboleko ungashintsha imihlomba lezifunda.",
		"menu_1_balance":        "1️⃣ Bona Imali Yami",
		"menu_2_send":           "2️⃣ Thumela Imali",
		"menu_3_airtime":        "3️⃣ Thenga I-airtime",
		"menu_4_bills":          "4️⃣ Bhadala Izikweletu",
		"menu_5_transactions":   "5️⃣ Bona Okwenzakeleyo",
		"menu_6_support":        "6️⃣ Khuluma Ngosizo",
		"menu_7_loan":           "7️⃣ Imalimboleko Ye-Microfin 💸",
		"menu_8_language":       "8️⃣ Shintsha Ulimi 🌍",
		"your_balance":          "💰 Imali yakho ifinyelela ku-$%.2f\n\nUfuna ukwenza okunye na?\n1️⃣ I-Menu Enkulu\n0️⃣ Phuma",
		"send_to_who":           "Ufuna ukuthumela imali kubani?",
		"send_how_much":         "Ufuna ukuthumela imali engakanani ku-%s?",
		"invalid_amount":        "❌ Imali engalungile. Zama futhi (isibonelo, 20 kumbe $20).",
		"confirm_send":          "Thumela $%.2f ku-%s? ✅ Yebo / ❌ Hatshi",
		"transaction_success":   "✅ Ukuthumela kuphumelele!\nImali entsha: $%.2f\n\nUfuna ukwenza okunye na?\n1️⃣ I-Menu Enkulu\n0️⃣ Phuma",
		"insufficient_funds":    "⚠️ Imali ayeneli.",
		"transaction_cancelled": "❌ Ukuthumela kuvalwe.\n\nUfuna ukwenza okunye na?\n1️⃣ I-Menu Enkulu\n0️⃣ Phuma",
		"sent_to":               "Ukuthumela $%.2f ku-%s ✅",
		"airtime_prompt":        "Faka imali lenombolo (isibonelo $2 ku-0772123456)",
		"airtime_invalid":       "❌ Akusilo esilungile. Zama futhi (isibonelo, $2 ku-0772123456).",
		"airtime_success":       "✅ Ukuthenga i-airtime kuphumelele! Imali entsha: $%.2f\n\nUfuna ukwenza okunye na?\n1️⃣ I-Menu Enkulu\n0️⃣ Phuma",
		"bought_airtime":        "Ukuthenga $%.2f airtime 📱",
		"not_enough_balance":    "⚠️ Imali ayeneli.",
		"bills_demo":            "⚙️ Ukubhadala izikweletu akusasebenzi okwamanje.\n\nUfuna ukwenza okunye na?\n1️⃣ I-Menu Enkulu\n0️⃣ Phuma",
		"recent_transactions":   "🧾 Okwenzakeleyo Kamuva:\n%s\n\nUfuna ukwenza okunye na?\n1️⃣ I-Menu Enkulu\n0️⃣ Phuma",
		"no_transactions":       "Akulalutho olwenzakeleyo okwamanje",
		"support_menu":          "Ngingakusiza nge:\n1️⃣ Ikhadi Elilahlekileko\n2️⃣ Inkinga Yokuthumela\n3️⃣ Khuluma Lo-agent",
		"support_lost_card":     "🧾 Ikhadi Elilahlekileko: Sicela ubize 0800 123 456.",
		"support_issue_logged":  "⚙️ Inkinga ibhaliwe.",
		"support_agent":         "👩🏾‍💼 Siyakuxhuma lo-agent...",
		"choose_valid_support":  "❓ Sicela ukhethe 1, 2, kumbe 3.",
		"post_action_menu":      "Sicela ukhethe:\n1️⃣ I-Menu Enkulu\n0️⃣ Phuma",
		"goodbye":               "👋 Siyabonga ukusebenzisa i-WalletBot! Sala kuhle!",
		"choose_valid_option":   "❓ Sicela ukhethe okufaneleyo (1–8).",
		"loan_menu_title":       "🏦 I-Menu Yemalimboleko Ye-Microfin — Umhlomba: %s | Isifunda: %s\n\n",
		"loan_menu_1":           "1️⃣ Cela Imalimboleko",
		"loan_menu_2":           "2️⃣ Bona Imalimboleko Yami",
		"loan_menu_3":           "3️⃣ Ncoma Umboleki",
		"loan_menu_4":           "4️⃣ Shintsha Umhlomba",
		"loan_menu_5":           "5️⃣ Thatha Imali Evunyiweyo",
		"loan_menu_6":           "6️⃣ Vumela Amalimboleko",
		"loan_menu_0":           "0️⃣ Buyela ku-Menu Enkulu",
		"loan_menu_note":        "\n\n(Sebenzisa izinombolo)",
		"loan_request_name":     "Imalimboleko — Faka *igama* lomceli:",
		"loan_request_id":       "Faka i-ID yomceli:",
		"loan_request_region":   "Khetha isifunda somceli:\n1️⃣ Tabhera\n2️⃣ Nyika",
		"loan_request_amount":   "Faka imali yemalimboleko (isibonelo, 300):",
		"loan_submitted":        "✅ Imalimboleko ithunyelwe nge-ID: %s\nIsimo: Ilindele ukuvunywa ngu-Mufundisi\n\nUfuna ukwenza okunye na?\n1️⃣ I-Menu Enkulu\n0️⃣ Phuma",
		"choose_region":         "Sicela ukhethe 1 ye-Tabhera kumbe 2 ye-Nyika.",
		"recommend_title":       "📋 Abantu abalindele ukuncomwa:\n\n",
		"recommend_none":        "✅ Akukho muntu olindele ukuncomwa esifundeni sakho.",
		"recommend_footer":      "\nPhendula ngenombolo (1–%d) kumbe 0️⃣ ukubuyela.",
		"recommend_invalid":     "❌ Ukukhetha okungalungile. Sicela ukhethe inombolo efaneleyo.",
		"recommend_not_found":   "Imalimboleko ayitholwa.",
		"recommend_question":    "Ufuna ukuncoma %s na?\n1️⃣ Yebo\n2️⃣ Hatshi",
		"recommend_yes_no":      "Sicela uphendule 1️⃣ Yebo kumbe 2️⃣ Hatshi.",
		"recommend_success":     "✅ Ukuncoma kubhaliwe ku-%s.",
		"recommend_already":     "✅ Usumthembisile umuntu lo.",
		"recommend_reason":      "Sicela unikele isizatho sokungancomi:",
		"not_recommended":       "❌ Akanconywanga (%s).",
		"approver_switch":       "Ukuze uvumele amalimboleko shintsha umhlomba ku-Mufundisi kumbe ku-Elder. Sebenzisa Shintsha Umhlomba (ukukhetha 4).",
		"role_switched":         "🔁 Umhlomba ushintshiwe waba ngu-%s. Isifunda: %s\n\nYiya ku-Menu Enkulu -> 7 Yemalimboleko.",
		"role_unknown":          "Umhlomba ongaziwa. Imihlomba: member, mufundisi, elder, recommender.",
		"switch_role_menu":      "Khetha umhlomba wakho:\n1️⃣ Ilungu / Umceli\n2️⃣ Mufundisi (Umvumeli)\n3️⃣ Elder (Umvumeli)\n4️⃣ Buyela",
		"language_menu":         "🌍 Choose your language / Sarudza mutauro / Khetha ulimi lwakho:\n\n1️⃣ English\n2️⃣ Shona\n3️⃣ Ndebele\n0️⃣ Back / Dzoka / Buyela",
		"language_changed":      "✅ Ulimi lushintshiwe lwaba ngu-%s",
	},
}

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
	TempLoanList     map[string]string `json:"-"` // Maps numbers to loan IDs for recommendation selection
	Language         string            // "en" (English), "sn" (Shona), "nd" (Ndebele)
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
			Language:     "en",
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
			response = getTextf(s.Language, "role_switched", strings.Title(role), s.Region)
			respondXML(w, response)
			return
		default:
			response = getText(s.Language, "role_unknown")
			respondXML(w, response)
			return
		}
	}

	switch s.Stage {

	case "ask_pin":
		response = getText(s.Language, "welcome")
		s.Stage = "verify_pin"

	case "verify_pin":
		if len(body) == 4 && isNumeric(body) {
			s.PIN = body
			s.Stage = "ask_name"
			response = getText(s.Language, "pin_accepted")
		} else {
			response = getText(s.Language, "pin_invalid")
		}

	case "ask_name":
		s.Name = strings.Title(body)
		s.Stage = "main_menu"
		response = mainMenuText(s)

	case "main_menu":
		switch body {
		case "1":
			response = getTextf(s.Language, "your_balance", s.Balance)
			s.Stage = "post_action"
		case "2":
			s.Stage = "send_to"
			response = getText(s.Language, "send_to_who")
		case "3":
			s.Stage = "airtime"
			response = getText(s.Language, "airtime_prompt")
		case "4":
			response = getText(s.Language, "bills_demo")
			s.Stage = "post_action"
		case "5":
			txs := getText(s.Language, "no_transactions")
			if len(s.Transactions) > 0 {
				txs = strings.Join(s.Transactions, "\n")
			}
			response = getTextf(s.Language, "recent_transactions", txs)
			s.Stage = "post_action"
		case "6":
			s.Stage = "support"
			response = getText(s.Language, "support_menu")
		case "7":
			s.Stage = "loan_menu"
			response = loanMenuText(s)
		case "8":
			s.Stage = "language_menu"
			response = getText(s.Language, "language_menu")
		default:
			response = getText(s.Language, "choose_valid_option")
		}

	case "send_to":
		s.PendingName = strings.Title(body)
		s.Stage = "send_amount"
		response = getTextf(s.Language, "send_how_much", s.PendingName)

	case "send_amount":
		amt, err := parseAmount(body)
		if err != nil {
			response = getText(s.Language, "invalid_amount")
			break
		}
		s.PendingAmt = amt
		s.Stage = "confirm_send"
		response = getTextf(s.Language, "confirm_send", s.PendingAmt, s.PendingName)

	case "confirm_send":
		if strings.Contains(body, "yes") || body == "✅" {
			if s.Balance >= s.PendingAmt {
				s.Balance -= s.PendingAmt
				tx := getTextf(s.Language, "sent_to", s.PendingAmt, s.PendingName)
				s.Transactions = append([]string{tx}, s.Transactions...)
				response = getTextf(s.Language, "transaction_success", s.Balance)
			} else {
				response = getText(s.Language, "insufficient_funds")
			}
			s.Stage = "post_action"
		} else {
			response = getText(s.Language, "transaction_cancelled")
			s.Stage = "post_action"
		}

	case "airtime":
		amt, err := parseAmount(body)
		if err != nil {
			response = getText(s.Language, "airtime_invalid")
			break
		}
		if s.Balance >= amt {
			s.Balance -= amt
			tx := getTextf(s.Language, "bought_airtime", amt)
			s.Transactions = append([]string{tx}, s.Transactions...)
			response = getTextf(s.Language, "airtime_success", s.Balance)
		} else {
			response = getText(s.Language, "not_enough_balance")
		}
		s.Stage = "post_action"

	case "support":
		switch body {
		case "1":
			response = getText(s.Language, "support_lost_card")
		case "2":
			response = getText(s.Language, "support_issue_logged")
		case "3":
			response = getText(s.Language, "support_agent")
		default:
			response = getText(s.Language, "choose_valid_support")
			respondXML(w, response)
			return
		}
		response += "\n\n" + getText(s.Language, "post_action_menu")
		s.Stage = "post_action"

	case "post_action":
		if body == "1" {
			s.Stage = "main_menu"
			response = mainMenuText(s)
		} else if body == "0" || strings.Contains(body, "no") {
			delete(sessions, from)
			response = getText(s.Language, "goodbye")
		} else {
			response = getText(s.Language, "post_action_menu")
		}

	// ---------- LANGUAGE MENU ----------
	case "language_menu":
		switch body {
		case "1":
			s.Language = "en"
			response = getTextf(s.Language, "language_changed", "English")
			s.Stage = "main_menu"
			response += "\n\n" + mainMenuText(s)
		case "2":
			s.Language = "sn"
			response = getTextf(s.Language, "language_changed", "Shona")
			s.Stage = "main_menu"
			response += "\n\n" + mainMenuText(s)
		case "3":
			s.Language = "nd"
			response = getTextf(s.Language, "language_changed", "Ndebele")
			s.Stage = "main_menu"
			response += "\n\n" + mainMenuText(s)
		case "0":
			s.Stage = "main_menu"
			response = mainMenuText(s)
		default:
			response = getText(s.Language, "language_menu")
		}

	// ---------- LOAN MENU ----------
	case "loan_menu":
		switch strings.TrimSpace(body) {
		case "1": // Request Loan
			s.Stage = "loan_request_name"
			response = getText(s.Language, "loan_request_name")
		case "2": // View Loan Status
			response = viewLoansForApplicant(s.Name)
		case "3": // Recommend Borrower
			// show list of pending loans in same region
			s.Stage = "recommend_list"
			response = recommendListPrompt(s)
		case "4": // Switch Role
			s.Stage = "switch_role_menu"
			response = switchRoleMenuText(s)
		case "5": // Borrow Funds
			s.Stage = "borrow_list"
			response = borrowListPrompt(s)
		case "6": // Approve Loans (for approvers)
			if s.Role != "mufundisi" && s.Role != "elder" {
				response = getText(s.Language, "approver_switch")
			} else {
				s.Stage = "approver_list"
				response = approverListPrompt(s)
			}
		case "0":
			s.Stage = "main_menu"
			response = mainMenuText(s)
		default:
			response = loanMenuText(s)
		}

	// Loan request sub-steps
	case "loan_request_name":
		s.PendingName = strings.Title(body)
		s.Stage = "loan_request_id"
		response = getText(s.Language, "loan_request_id")

	case "loan_request_id":
		s.PIN = strings.ToUpper(strings.TrimSpace(body)) // temporarily store applicant ID in PIN
		s.Stage = "loan_request_region_choice"
		response = getText(s.Language, "loan_request_region")

	case "loan_request_region_choice":
		if body == "1" {
			s.Region = "Tabhera"
		} else if body == "2" {
			s.Region = "Nyika"
		} else {
			response = getText(s.Language, "choose_region")
			respondXML(w, response)
			return
		}
		s.Stage = "loan_request_amount"
		response = getText(s.Language, "loan_request_amount")

	case "loan_request_amount":
		amt, err := parseAmount(body)
		if err != nil {
			response = getText(s.Language, "invalid_amount")
			break
		}
		loan := createLoan(s.PendingName, s.PIN, s.Region, amt, s.Name)
		response = getTextf(s.Language, "loan_submitted", loan.ID)
		s.Stage = "post_action"

	// Recommend list: user chooses number
	case "recommend_list":
		choice := strings.TrimSpace(body)
		if choice == "0" {
			s.Stage = "loan_menu"
			response = loanMenuText(s)
			respondXML(w, response)
			return
		}

		loanID, ok := s.TempLoanList[choice]
		if !ok {
			response = getText(s.Language, "recommend_invalid")
			respondXML(w, response)
			return
		}

		loanMu.Lock()
		loan, exists := loans[loanID]
		loanMu.Unlock()
		if !exists {
			response = getText(s.Language, "recommend_not_found")
			respondXML(w, response)
			return
		}

		s.Stage = "recommend_action:" + loanID
		response = getTextf(s.Language, "recommend_question", loan.ApplicantName)
		respondXML(w, response)
		return

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
		// recommend action: yes/no
		if strings.HasPrefix(s.Stage, "recommend_action:") {
			loanID := strings.SplitN(s.Stage, ":", 2)[1]

			if body == "1" {
				loanMu.Lock()
				loan, exists := loans[loanID]
				if !exists {
					loanMu.Unlock()
					response = getText(s.Language, "recommend_not_found")
					respondXML(w, response)
					return
				}
				for _, r := range loan.Recommendations {
					if strings.EqualFold(r, s.Name) {
						loanMu.Unlock()
						response = getText(s.Language, "recommend_already")
						respondXML(w, response)
						return
					}
				}
				loan.Recommendations = append(loan.Recommendations, s.Name)
				computeLoanLimits(loan)
				loanMu.Unlock()

				response = getTextf(s.Language, "recommend_success", loan.ApplicantName)
				s.Stage = "loan_menu"
				respondXML(w, response)
				return

			} else if body == "2" {
				s.Stage = "recommend_reason:" + loanID
				response = getText(s.Language, "recommend_reason")
				respondXML(w, response)
				return
			} else {
				response = getText(s.Language, "recommend_yes_no")
				respondXML(w, response)
				return
			}
		}

		// recommend reason
		if strings.HasPrefix(s.Stage, "recommend_reason:") {
			loanID := strings.SplitN(s.Stage, ":", 2)[1]
			reason := strings.TrimSpace(body)
			loanMu.Lock()
			loan, exists := loans[loanID]
			if !exists {
				loanMu.Unlock()
				response = getText(s.Language, "recommend_not_found")
				respondXML(w, response)
				return
			}
			if loan.ApprovalReasons == nil {
				loan.ApprovalReasons = map[string]string{}
			}
			loan.ApprovalReasons[s.Name] = "not recommended: " + reason
			loanMu.Unlock()

			response = getTextf(s.Language, "not_recommended", reason)
			s.Stage = "loan_menu"
			respondXML(w, response)
			return
		}

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

// getText retrieves translated text for the given language and key
func getText(language, key string) string {
	if langMap, ok := translations[language]; ok {
		if text, ok := langMap[key]; ok {
			return text
		}
	}
	// Fallback to English if translation not found
	if langMap, ok := translations["en"]; ok {
		if text, ok := langMap[key]; ok {
			return text
		}
	}
	// Last resort: return the key itself
	return key
}

// getTextf retrieves translated text and formats it with the provided arguments
func getTextf(language, key string, args ...interface{}) string {
	text := getText(language, key)
	return fmt.Sprintf(text, args...)
}

func mainMenuText(s *Session) string {
	menu := getTextf(s.Language, "good_day", s.Name)
	menu += "\n\n"
	menu += getText(s.Language, "menu_1_balance") + "\n"
	menu += getText(s.Language, "menu_2_send") + "\n"
	menu += getText(s.Language, "menu_3_airtime") + "\n"
	menu += getText(s.Language, "menu_4_bills") + "\n"
	menu += getText(s.Language, "menu_5_transactions") + "\n"
	menu += getText(s.Language, "menu_6_support") + "\n"
	menu += getText(s.Language, "menu_7_loan") + "\n"
	menu += getText(s.Language, "menu_8_language")
	menu += getText(s.Language, "menu_tip")
	return menu
}

func loanMenuText(s *Session) string {
	menu := getTextf(s.Language, "loan_menu_title", strings.Title(s.Role), s.Region)
	menu += getText(s.Language, "loan_menu_1") + "\n"
	menu += getText(s.Language, "loan_menu_2") + "\n"
	menu += getText(s.Language, "loan_menu_3") + "\n"
	menu += getText(s.Language, "loan_menu_4") + "\n"
	menu += getText(s.Language, "loan_menu_5") + "\n"
	if s.Role == "mufundisi" || s.Role == "elder" {
		menu += getText(s.Language, "loan_menu_6") + "\n"
	}
	menu += getText(s.Language, "loan_menu_0")
	menu += getText(s.Language, "loan_menu_note")
	return menu
}

func switchRoleMenuText(s *Session) string {
	return getText(s.Language, "switch_role_menu")
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

	// Filter only loans in same region that are pending
	var filtered []*Loan
	for _, l := range loans {
		if l.Region == s.Region && l.Status == "pending" {
			filtered = append(filtered, l)
		}
	}

	if len(filtered) == 0 {
		return getText(s.Language, "recommend_none")
	}

	// Map numbers to loan IDs for this session
	s.TempLoanList = make(map[string]string)
	out := getText(s.Language, "recommend_title")
	for i, l := range filtered {
		index := fmt.Sprintf("%d", i+1)
		s.TempLoanList[index] = l.ID
		out += fmt.Sprintf("%s️⃣ %s | Region: %s | Status: %s | Recs: %d\n",
			index, l.ApplicantName, l.Region, l.Status, len(l.Recommendations))
	}
	out += getTextf(s.Language, "recommend_footer", len(filtered))
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
