package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// BotConfig holds configuration for the automated customer bot.
type BotConfig struct {
	AdminOnline    bool
	AwayMessage    string
	OperatingHours string
	DjangoAPIURL   string
	LLMEnabled     bool
	LLMModel       string
	LLMAPIKey      string
	WebhookURL     string
}

var (
	botConfigLock sync.RWMutex
	currentConfig = BotConfig{
		AdminOnline:    false, // Admin is absent by default, activating auto-reply bot
		AwayMessage:    "The sales admin is currently offline. Our automated bot is active 24/7 to assist with Plain Roll, Printed Film, Printed Packing Bag, and Shrink Wrap / SSB & MSB orders.",
		OperatingHours: "Monday to Friday, 8:00 AM – 5:00 PM (GMT+1). Automated support is 24/7.",
		DjangoAPIURL:   "http://localhost:8000/api/products",
		LLMEnabled:     os.Getenv("GEMINI_API_KEY") != "" || os.Getenv("LLM_API_KEY") != "",
		LLMModel:       getEnvDefault("LLM_MODEL", "gemini-1.5-flash"),
		LLMAPIKey:      getEnvDefault("GEMINI_API_KEY", os.Getenv("LLM_API_KEY")),
		WebhookURL:     os.Getenv("ADMIN_WEBHOOK_URL"),
	}

	inquiriesLock   sync.Mutex
	loggedInquiries []CustomerInquiryRecord
)

func getEnvDefault(key, defVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defVal
}

// CustomerInquiryRecord tracks messages left by customers via the bot
type CustomerInquiryRecord struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Phone       string    `json:"phone"`
	Message     string    `json:"message"`
	BotResponse string    `json:"bot_response"`
	CreatedAt   time.Time `json:"created_at"`
}

// ChatRequest represents an incoming customer message to the bot
type ChatRequest struct {
	Message string `json:"message"`
	Name    string `json:"name,omitempty"`
	Email   string `json:"email,omitempty"`
	Phone   string `json:"phone,omitempty"`
}

// ChatResponse represents the automated reply from the bot
type ChatResponse struct {
	Reply            string   `json:"reply"`
	AdminOnline      bool     `json:"admin_online"`
	IsAwayReply      bool     `json:"is_away_reply"`
	InquiryLogged    bool     `json:"inquiry_logged"`
	IsLLMGenerated   bool     `json:"is_llm_generated"`
	SuggestedPrompts []string `json:"suggested_prompts"`
	Timestamp        string   `json:"timestamp"`
}

// BotStatusResponse reports current admin presence & bot health
type BotStatusResponse struct {
	AdminOnline    bool   `json:"admin_online"`
	BotActive      bool   `json:"bot_active"`
	AwayMessage    string `json:"away_message"`
	OperatingHours string `json:"operating_hours"`
	LLMEnabled     bool   `json:"llm_enabled"`
	LLMModel       string `json:"llm_model"`
	ServerTime     string `json:"server_time"`
}

// ProcessBotMessage analyzes customer input and generates automated responses focused on water packaging
func ProcessBotMessage(req ChatRequest) ChatResponse {
	botConfigLock.RLock()
	adminOnline := currentConfig.AdminOnline
	awayMessage := currentConfig.AwayMessage
	apiKey := currentConfig.LLMAPIKey
	model := currentConfig.LLMModel
	webhookURL := currentConfig.WebhookURL
	botConfigLock.RUnlock()

	userMsg := strings.TrimSpace(req.Message)
	lower := strings.ToLower(userMsg)

	now := time.Now().Format("15:04")
	resp := ChatResponse{
		AdminOnline: adminOnline,
		IsAwayReply: !adminOnline,
		Timestamp:   now,
		SuggestedPrompts: []string{
			"Plain Roll info",
			"Printed Film info",
			"Printed Packing Bag info",
			"Shrink Wrap / SSB & MSB info",
			"Place a direct order",
		},
	}

	// Case 1: Check if this is a lead / direct contact registration
	if req.Email != "" || req.Phone != "" || strings.Contains(lower, "@") {
		email := req.Email
		if email == "" && strings.Contains(lower, "@") {
			for _, part := range strings.Fields(userMsg) {
				if strings.Contains(part, "@") {
					email = strings.Trim(part, ".,;:()")
					break
				}
			}
		}

		name := req.Name
		if name == "" {
			name = "Pure Water Factory Partner"
		}

		replyText := fmt.Sprintf(
			"Thank you %s! Because our sales administration team is currently offline, I have logged your order inquiry and contact details (%s). Our production representative will contact you via WhatsApp/Phone shortly with factory pricing and delivery timelines.",
			name, email,
		)

		logInquiryLocally(name, email, req.Phone, userMsg, replyText)
		forwardInquiryToDjango(name, email, req.Phone, userMsg, replyText)
		dispatchLiveWebhookAlert("customer_inquiry", fmt.Sprintf("Water Packaging Lead: %s", name), map[string]string{
			"Name":    name,
			"Email":   email,
			"Phone":   req.Phone,
			"Message": userMsg,
		}, webhookURL)

		resp.Reply = replyText
		resp.InquiryLogged = true
		return resp
	}

	// Case 2: Inquiries about admin, human agent, or presence
	if containsAny(lower, "admin", "human", "person", "agent", "manager", "staff", "offline", "online", "who are you") {
		if !adminOnline {
			resp.Reply = fmt.Sprintf(
				"Notice: Our human sales managers are currently AWAY (%s). I am Jirmass Bot, your automated assistant. I can help you with Plain Roll, Printed Film, Printed Packing Bag, and Shrink Wrap / SSB & MSB specifications and direct order placement. To request a callback, simply leave your factory name, phone number, and requirements!",
				awayMessage,
			)
		} else {
			resp.Reply = "Our sales admin is currently online. How can we assist your pure water factory today? You can place an order directly or ask about Plain Roll, Printed Film, Bags, or Shrink Wrap specifications."
		}
		return resp
	}

	// Case 3: Try LLM Integration if API Key is configured
	if apiKey != "" {
		llmReply, err := queryLLM(userMsg, apiKey, model)
		if err == nil && len(strings.TrimSpace(llmReply)) > 0 {
			resp.Reply = llmReply
			resp.IsLLMGenerated = true
			resp.SuggestedPrompts = []string{"Place Direct Order", "Pricing per kg", "Delivery details"}
			return resp
		}
	}

	// Case 4: Shrink Wrap / SSB & MSB
	if containsAny(lower, "shrink", "shrink wrap", "ssb", "msb", "ssb&msb", "thermal shrink", "bundling wrap") {
		resp.Reply = "📦 Jirmass Shrink Wrap / SSB & MSB (Industrial Packaging Film):\n\n" +
			"• High-tensile thermal contraction film calibrated for Single Sachet Bag (SSB) and Multiple Sachet Bag (MSB) collation.\n" +
			"• Also optimized for bottled water multipack shrink wrapping.\n" +
			"• Exceptional puncture and tear resilience preventing transit bursts.\n" +
			"• High-clarity dust, moisture, and hygienic barrier maintaining water cleanliness.\n" +
			"• Available in custom widths, micron gauges, and biaxial shrink contraction ratios.\n\n" +
			"👉 You can place a direct order for Shrink Wrap / SSB & MSB using the form on this page!"
		resp.SuggestedPrompts = []string{"Plain Roll info", "Printed Film info", "Printed Packing Bag info", "Place Direct Order"}
		return resp
	}

	// Case 5: Printed Packing Bag (outer bundling sacks)
	if containsAny(lower, "packing bag", "printed packing bag", "printed bag", "outer bag", "bundle", "bale", "sack", "20 sachets") {
		resp.Reply = "🛍️ Jirmass Printed Packing Bag (Outer Bundling Sacks):\n\n" +
			"• Heavy-duty outer bagging sacks engineered to securely bundle and hold 20 sachets of pure water.\n" +
			"• High puncture resistance and reinforced tear-proof bottom seals built for rough transport.\n" +
			"• Custom rotogravure branding with your pure water factory name, depot contact, and logo printing.\n" +
			"• Packaged in convenient bundles (e.g., 500 bags per bundle, 1,000 bags, or 5,000 bag bales).\n\n" +
			"👉 You can place a direct order for Printed Packing Bags using the form on this page!"
		resp.SuggestedPrompts = []string{"Plain Roll info", "Printed Film info", "Shrink Wrap / SSB & MSB info", "Place Direct Order"}
		return resp
	}

	// Case 6: Printed Film (custom rotogravure printed rolls)
	if containsAny(lower, "printed film", "printed roll", "custom print", "rotogravure", "branded roll", "branding", "eye mark", "cylinder") {
		resp.Reply = "🖨️ Jirmass Printed Film (Custom Branded Roll Film):\n\n" +
			"• High-definition rotogravure custom printed sachet water packaging roll film.\n" +
			"• Precision multi-color graphics with factory brand logo, NAFDAC license numbers, and batch zones.\n" +
			"• Standard 320mm roll width with precise photo-eye registration marks for automatic form-fill-seal machines.\n" +
			"• 100% virgin food-grade LDPE with food-safe inks that never bleed or affect water taste.\n" +
			"• Available by roll count or bulk weight (kg).\n\n" +
			"👉 To order Printed Film, use the Direct Order form on this page!"
		resp.SuggestedPrompts = []string{"Plain Roll info", "Printed Packing Bag info", "Place Direct Order"}
		return resp
	}

	// Case 7: Plain Roll (unprinted form-fill-seal film rolls)
	if containsAny(lower, "plain roll", "plain rolls", "plain film", "unprinted", "clear roll", "320mm", "film", "gauge", "micron", "machine", "roll", "rolls", "bag", "bags") {
		resp.Reply = "💧 Jirmass Plain Roll (Form-Fill-Seal Roll Film):\n\n" +
			"• Standard 320mm roll width, calibrated for automated form-fill-seal packaging machines.\n" +
			"• 100% virgin food-grade LDPE with high tensile strength for leak-free sealing.\n" +
			"• Uniform gauge thickness (60–75 microns) and slip additives for jam-free feeding.\n" +
			"• High-clarity transparent unprinted film rolls with zero odor.\n" +
			"• Available by roll count or bulk weight (kg).\n\n" +
			"👉 To order Plain Roll, use the Direct Order form on this page!"
		resp.SuggestedPrompts = []string{"Printed Film info", "Printed Packing Bag info", "Shrink Wrap / SSB & MSB info", "Place Direct Order"}
		return resp
	}

	// Case 8: Direct Order Placement / Pricing inquiries
	if containsAny(lower, "order", "price", "cost", "quote", "rate", "how much", "buy", "wholesale", "direct") {
		resp.Reply = "Factory-Direct Wholesale Pricing:\n\n" +
			"Our water packaging materials (Plain Roll, Printed Film, Printed Packing Bag, Shrink Wrap / SSB & MSB) are priced competitively per roll, per kg, or per bundle based on volume.\n\n" +
			"To place your direct order immediately:\n" +
			"1. Scroll to the 'Direct Order' section on this page.\n" +
			"2. Select your product (Plain Roll, Printed Film, Printed Packing Bag, or Shrink Wrap) and specify quantity.\n" +
			"3. Provide your Phone/WhatsApp and Delivery address.\n\n" +
			"Our sales team receives the alert instantly and will contact you to confirm production and delivery!"
		resp.SuggestedPrompts = []string{"Leave my phone number", "Roll specifications", "Working hours"}
		return resp
	}

	// Case 9: Delivery / Location / Dispatch / Contact
	if containsAny(lower, "delivery", "shipping", "deliver", "ship", "location", "address", "branch", "branches", "where", "factory", "plant", "otukpo", "kaduna", "makurdi", "phone", "tel", "call", "email", "contact", "office") {
		resp.Reply = "📍 Jirmass Plastics Factory & Distribution Locations:\n\n" +
			"1. Otukpo Branch: No.3 New Market Road, Otukpo, Benue State\n" +
			"2. Kaduna Branch: No.3 Kachia Road Along Station Round About, Kaduna\n" +
			"3. Makurdi Branch: No.28 Naka Road, Makurdi\n\n" +
			"📞 Phone / WhatsApp Lines: 08138881463, 09036148838, 07066820641\n" +
			"✉️ Email: jirmassplasticlimited@gmail.com\n\n" +
			"🚚 We dispatch Plain Roll, Printed Film, Printed Packing Bags, and Shrink Wrap / SSB & MSB directly to pure water factories and regional depots nationwide (1–3 business days turnaround)."
		resp.SuggestedPrompts = []string{"Place Direct Order", "Plain Roll info", "Printed Film info", "Shrink Wrap info"}
		return resp
	}

	// Case 10: Working Hours
	if containsAny(lower, "hour", "hours", "time", "open", "schedule", "when") {
		botConfigLock.RLock()
		hours := currentConfig.OperatingHours
		botConfigLock.RUnlock()
		resp.Reply = fmt.Sprintf("Factory operations and sales hours: %s. Our automated bot is available 24/7 to take your orders while staff is away.", hours)
		return resp
	}

	// Case 11: Greetings
	if containsAny(lower, "hello", "hi", "hey", "good morning", "good afternoon", "good evening", "greetings") {
		resp.Reply = "👋 Welcome to Jirmass Plastics! We specialize in Plain Roll, Printed Film, Printed Packing Bag, and Shrink Wrap / SSB & MSB for pure water packaging plants. Our admin is currently away, but you can ask about roll specs, bags, shrink wrap, or place your direct order right here. How can I assist you?"
		return resp
	}

	// Case 12: Closing / Gratitude
	if containsAny(lower, "thank", "thanks", "bye", "goodbye") {
		resp.Reply = "You are most welcome! Thank you for choosing Jirmass Plastics for your water packaging supplies. Reach out anytime if you need more Plain Rolls, Printed Film, Packing Bags, or Shrink Wrap!"
		return resp
	}

	// Default Fallback
	resp.Reply = "Welcome to Jirmass Plastics — manufacturers of Plain Roll, Printed Film, Printed Packing Bag, and Shrink Wrap / SSB & MSB. Since our sales manager is currently offline, you can place a direct order using the form on this page, or reply with your WhatsApp/phone number and requirements to receive a fast callback."
	return resp
}

func queryLLM(prompt, apiKey, model string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	apiURL := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", model, apiKey)

	systemPrompt := "You are Jirmass Bot, the 24/7 automated sales assistant for Jirmass Plastics. " +
		"We specialize EXCLUSIVELY in four industrial water packaging products for pure water factories: " +
		"1) Plain Roll (standard 320mm virgin LDPE roll film, 60–75 microns, jam-free feeding for automatic form-fill-seal machines), " +
		"2) Printed Film (custom rotogravure multi-color printed film rolls with precision photo-eye registration marks and factory branding), " +
		"3) Printed Packing Bag (heavy-duty outer bundling sacks engineered to securely hold and transport 20 sachets of pure water without bursting), and " +
		"4) Shrink Wrap / SSB & MSB (high-tensile thermal contraction film for Single Sachet Bag [SSB] and Multiple Sachet Bag [MSB] collation, plus bottle multipacks). " +
		"Do NOT discuss unrelated plastics like chairs, crates, buckets, or jerrycans. " +
		"The human sales administrator is currently offline/away. Answer questions concisely and professionally, and guide the user to place a direct order or leave their contact details."

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]string{
					{"text": systemPrompt + "\nCustomer inquiry: " + prompt},
				},
			},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("LLM API returned %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) > 0 && len(geminiResp.Candidates[0].Content.Parts) > 0 {
		return strings.TrimSpace(geminiResp.Candidates[0].Content.Parts[0].Text), nil
	}

	return "", fmt.Errorf("no candidates in LLM response")
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}

func logInquiryLocally(name, email, phone, message, botResponse string) {
	inquiriesLock.Lock()
	defer inquiriesLock.Unlock()

	id := len(loggedInquiries) + 1
	loggedInquiries = append(loggedInquiries, CustomerInquiryRecord{
		ID:          id,
		Name:        name,
		Email:       email,
		Phone:       phone,
		Message:     message,
		BotResponse: botResponse,
		CreatedAt:   time.Now(),
	})
}

func forwardInquiryToDjango(name, email, phone, message, botResponse string) {
	go func() {
		payload := map[string]string{
			"name":         name,
			"email":        email,
			"phone":        phone,
			"message":      message,
			"bot_response": botResponse,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}

		client := &http.Client{Timeout: 3 * time.Second}
		backendURL := getBackendURL()
		inquiryURL := strings.TrimRight(backendURL, "/") + "/api/products/inquiries/"
		resp, err := client.Post(inquiryURL, "application/json", bytes.NewBuffer(data))
		if err == nil {
			_ = resp.Body.Close()
		}
	}()
}

func dispatchLiveWebhookAlert(event, title string, details map[string]string, webhookURL string) {
	if webhookURL == "" {
		return
	}
	go func() {
		payload := map[string]interface{}{
			"event":        event,
			"title":        title,
			"details":      details,
			"triggered_at": time.Now().Format(time.RFC3339),
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return
		}
		client := &http.Client{Timeout: 4 * time.Second}
		resp, err := client.Post(webhookURL, "application/json", bytes.NewBuffer(data))
		if err != nil {
			log.Printf("Webhook alert dispatch failed: %v", err)
			return
		}
		_ = resp.Body.Close()
	}()
}
