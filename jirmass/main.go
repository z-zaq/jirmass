package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"time"
)

// PageData carries template context for water packaging website
type PageData struct {
	Title        string
	FlashMessage string
	AdminOnline  bool
	CurrentYear  int
}

// DirectOrderRequest represents a customer order for Plain Roll, Printed Film, Printed Packing Bag, or Shrink Wrap
type DirectOrderRequest struct {
	ProductType     string `json:"product_type"`     // "Plain Roll", "Printed Film", "Printed Packing Bag", "Shrink Wrap / SSB & MSB"
	PrintType       string `json:"print_type"`       // "Custom Printed / Branded", "Plain / Standard", etc.
	QuantityRolls   string `json:"quantity_rolls"`   // e.g. "30 Plain Rolls" or "500 kg Printed Film"
	QuantityBags    string `json:"quantity_bags"`    // e.g. "100 Bag Bundles" or "40 Shrink Wrap Rolls"
	CompanyName     string `json:"company_name"`     // Pure water factory / brand name
	CustomerName    string `json:"customer_name"`    // Representative name
	Phone           string `json:"phone"`            // Phone or WhatsApp
	Email           string `json:"email"`            // Contact email
	DeliveryAddress string `json:"delivery_address"` // Factory or depot location
	Notes           string `json:"notes"`            // Artwork, thickness, SSB/MSB specs, delivery notes
}

// OrderResponse confirms receipt of direct order
type OrderResponse struct {
	Success     bool   `json:"success"`
	OrderNumber string `json:"order_number"`
	Message     string `json:"message"`
	AlertSent   bool   `json:"alert_sent"`
}

// SetupRouter sets up web and API routes for the simple water packaging web app
func SetupRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// Static assets handler
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Page endpoints
	mux.HandleFunc("/", HomeHandler)
	mux.HandleFunc("/about", AboutHandler)
	mux.HandleFunc("/contact", ContactHandler)

	// Direct Order placement endpoint
	mux.HandleFunc("/api/order", DirectOrderHandler)

	// Automated customer bot endpoints
	mux.HandleFunc("/api/bot/chat", BotChatHandler)
	mux.HandleFunc("/api/bot/status", BotStatusHandler)

	return mux
}

func main() {
	router := SetupRouter()
	log.Println("Jirmass Water Packaging Web Server running on http://localhost:8081")
	if err := http.ListenAndServe(":8081", router); err != nil {
		log.Fatalf("Server startup failed: %v", err)
	}
}

// HomeHandler serves the simplified single-page water packaging site
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	renderTemplate(w, "templates/home.html", PageData{
		Title:       "Jirmass Plastics - Plain Roll, Printed Film, Packing Bags & Shrink Wrap",
		AdminOnline: false,
		CurrentYear: time.Now().Year(),
	})
}

// AboutHandler serves the about page
func AboutHandler(w http.ResponseWriter, r *http.Request) {
	renderTemplate(w, "templates/about.html", PageData{
		Title:       "About Our Film Manufacturing - Jirmass Plastics",
		CurrentYear: time.Now().Year(),
	})
}

// ContactHandler handles contact page views and direct contact inquiries
func ContactHandler(w http.ResponseWriter, r *http.Request) {
	flashMsg := ""
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err == nil {
			name := r.FormValue("name")
			email := r.FormValue("email")
			phone := r.FormValue("phone")
			message := r.FormValue("message")

			if name != "" && (email != "" || phone != "") && message != "" {
				reply := "Thank you for contacting Jirmass Plastics! Since the admin is currently away, our automated system has logged your inquiry regarding water packaging bags and rolls. Our sales team will call or WhatsApp you promptly."
				logInquiryLocally(name, email, phone, message, reply)
				forwardInquiryToDjango(name, email, phone, message, reply)

				botConfigLock.RLock()
				hook := currentConfig.WebhookURL
				botConfigLock.RUnlock()
				dispatchLiveWebhookAlert("contact_form", fmt.Sprintf("Inquiry from %s (%s)", name, phone), map[string]string{
					"Name":    name,
					"Email":   email,
					"Phone":   phone,
					"Message": message,
				}, hook)

				flashMsg = "Your message has been received! Our factory sales team will reach out to you promptly."
			}
		}
	}

	renderTemplate(w, "templates/contact.html", PageData{
		Title:        "Contact Factory Sales - Jirmass Plastics",
		FlashMessage: flashMsg,
		CurrentYear:  time.Now().Year(),
	})
}

// DirectOrderHandler processes simple direct orders for water packing rolls and bags
func DirectOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req DirectOrderRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		// Fallback to form data parsing
		_ = r.ParseForm()
		req.ProductType = r.FormValue("product_type")
		req.PrintType = r.FormValue("print_type")
		req.QuantityRolls = r.FormValue("quantity_rolls")
		req.QuantityBags = r.FormValue("quantity_bags")
		req.CompanyName = r.FormValue("company_name")
		req.CustomerName = r.FormValue("customer_name")
		req.Phone = r.FormValue("phone")
		req.Email = r.FormValue("email")
		req.DeliveryAddress = r.FormValue("delivery_address")
		req.Notes = r.FormValue("notes")
	}

	// Validate essential direct order fields
	if strings.TrimSpace(req.CustomerName) == "" || (strings.TrimSpace(req.Phone) == "" && strings.TrimSpace(req.Email) == "") {
		http.Error(w, "Contact name and phone number or email are required to place an order", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.QuantityRolls) == "" && strings.TrimSpace(req.QuantityBags) == "" {
		http.Error(w, "Please specify the quantity of Plain Roll, Printed Film, Printed Packing Bag, or Shrink Wrap needed", http.StatusBadRequest)
		return
	}

	// Generate Order ID
	randBytes := make([]byte, 4)
	_, _ = rand.Read(randBytes)
	orderNum := fmt.Sprintf("WTR-%s-%s", time.Now().Format("20060102"), strings.ToUpper(hex.EncodeToString(randBytes)))

	// Asynchronously forward to Django backend and dispatch alerts
	go forwardDirectOrderToDjango(req, orderNum)

	botConfigLock.RLock()
	hook := currentConfig.WebhookURL
	botConfigLock.RUnlock()
	dispatchLiveWebhookAlert("direct_water_order", fmt.Sprintf("Direct Order %s: %s (%s)", orderNum, req.CompanyName, req.CustomerName), map[string]string{
		"Order Number":     orderNum,
		"Product Type":     req.ProductType,
		"Print Option":     req.PrintType,
		"Rolls Quantity":   req.QuantityRolls,
		"Bags Quantity":    req.QuantityBags,
		"Company":          req.CompanyName,
		"Contact Person":   req.CustomerName,
		"Phone / WhatsApp": req.Phone,
		"Email":            req.Email,
		"Delivery Address": req.DeliveryAddress,
		"Notes":            req.Notes,
	}, hook)

	resp := OrderResponse{
		Success:     true,
		OrderNumber: orderNum,
		Message:     "Thank you! Your direct order for water packaging materials (Plain Roll, Printed Film, Printed Packing Bag, Shrink Wrap / SSB & MSB) has been logged. Our sales team has been alerted via email and webhook and will reach out via WhatsApp/phone to confirm dispatch details.",
		AlertSent:   true,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(resp)
}

func forwardDirectOrderToDjango(req DirectOrderRequest, orderNum string) {
	summaryLines := []string{
		fmt.Sprintf("Product: %s (%s)", req.ProductType, req.PrintType),
		fmt.Sprintf("Rolls: %s | Bags: %s", req.QuantityRolls, req.QuantityBags),
		fmt.Sprintf("Company: %s", req.CompanyName),
		fmt.Sprintf("Notes: %s", req.Notes),
	}
	notesSummary := strings.Join(summaryLines, " | ")

	djangoPayload := map[string]interface{}{
		"order_number":     orderNum,
		"customer_name":    fmt.Sprintf("%s (%s)", req.CustomerName, req.CompanyName),
		"customer_email":   req.Email,
		"customer_phone":   req.Phone,
		"shipping_address": req.DeliveryAddress,
		"notes":            notesSummary,
		"total_amount":     "0.00", // Direct wholesale orders are priced per kg/volume on invoice
		"items": []map[string]interface{}{
			{
				"product_name": fmt.Sprintf("%s - %s", req.ProductType, req.PrintType),
				"quantity":     1,
				"unit_price":   "0.00",
			},
		},
	}

	data, err := json.Marshal(djangoPayload)
	if err != nil {
		return
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Post("http://localhost:8000/api/products/orders/", "application/json", strings.NewReader(string(data)))
	if err == nil {
		_ = resp.Body.Close()
	}
}

// BotChatHandler processes automated replies for website visitors
func BotChatHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var chatReq ChatRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&chatReq); err != nil {
		_ = r.ParseForm()
		chatReq.Message = r.FormValue("message")
		chatReq.Name = r.FormValue("name")
		chatReq.Email = r.FormValue("email")
		chatReq.Phone = r.FormValue("phone")
	}

	if stringsTrim := strings.TrimSpace(chatReq.Message); len(stringsTrim) == 0 {
		http.Error(w, "Message content is required", http.StatusBadRequest)
		return
	}

	response := ProcessBotMessage(chatReq)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response)
}

// BotStatusHandler returns the operational status and admin availability
func BotStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	botConfigLock.RLock()
	adminOnline := currentConfig.AdminOnline
	awayMsg := currentConfig.AwayMessage
	hours := currentConfig.OperatingHours
	llmEnabled := currentConfig.LLMEnabled
	llmModel := currentConfig.LLMModel
	botConfigLock.RUnlock()

	status := BotStatusResponse{
		AdminOnline:    adminOnline,
		BotActive:      true,
		AwayMessage:    awayMsg,
		OperatingHours: hours,
		LLMEnabled:     llmEnabled,
		LLMModel:       llmModel,
		ServerTime:     time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(status)
}

func renderTemplate(w http.ResponseWriter, tmplFile string, data PageData) {
	tmpl, err := template.ParseFiles(tmplFile)
	if err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
	}
}
