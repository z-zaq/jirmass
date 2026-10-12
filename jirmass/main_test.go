package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestHomeHandler(t *testing.T) {
	router := SetupRouter()

	// 1. Success on root path
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK for '/', got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Plain Roll") {
		t.Errorf("Expected body to contain 'Plain Roll'")
	}
	if !strings.Contains(body, "Printed Film") {
		t.Errorf("Expected body to contain 'Printed Film'")
	}
	if !strings.Contains(body, "Printed Packing Bag") {
		t.Errorf("Expected body to contain 'Printed Packing Bag'")
	}
	if !strings.Contains(body, "Shrink Wrap") {
		t.Errorf("Expected body to contain 'Shrink Wrap'")
	}
	if !strings.Contains(body, "direct-order") {
		t.Errorf("Expected direct-order section on homepage")
	}

	// 2. 404 for unknown path under root handler
	req404 := httptest.NewRequest(http.MethodGet, "/not-found-page", nil)
	rr404 := httptest.NewRecorder()
	router.ServeHTTP(rr404, req404)
	if rr404.Code != http.StatusNotFound {
		t.Errorf("Expected status 404 for unknown route, got %d", rr404.Code)
	}
}

func TestAboutHandler(t *testing.T) {
	router := SetupRouter()

	req := httptest.NewRequest(http.MethodGet, "/about", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK for '/about', got %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "About Jirmass Plastics") {
		t.Errorf("Expected body to contain 'About Jirmass Plastics'")
	}
	if !strings.Contains(body, "Pure Water") {
		t.Errorf("Expected body to focus on pure water packaging")
	}
}

func TestContactHandler(t *testing.T) {
	router := SetupRouter()

	// 1. GET request
	req := httptest.NewRequest(http.MethodGet, "/contact", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK for GET '/contact', got %d", rr.Code)
	}

	// 2. POST form submission
	formData := url.Values{
		"name":    {"Alhassan Danjuma (Crystal Waters)"},
		"email":   {"danjuma@example.com"},
		"phone":   {"+234800112233"},
		"message": {"We need 50 rolls of printed 320mm water film and 200 bundles of outer bags."},
	}
	postReq := httptest.NewRequest(http.MethodPost, "/contact", strings.NewReader(formData.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	postRR := httptest.NewRecorder()
	router.ServeHTTP(postRR, postReq)

	if postRR.Code != http.StatusOK {
		t.Errorf("Expected status 200 OK for POST '/contact', got %d", postRR.Code)
	}
	postBody := postRR.Body.String()
	if !strings.Contains(postBody, "Message Received") {
		t.Errorf("Expected flash message on successful contact submission")
	}
}

func TestDirectOrderHandler(t *testing.T) {
	router := SetupRouter()

	// 1. Method Not Allowed for GET
	getReq := httptest.NewRequest(http.MethodGet, "/api/order", nil)
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405 Method Not Allowed for GET /api/order, got %d", getRR.Code)
	}

	// 2. Bad Request for missing required fields
	invalidPayload, _ := json.Marshal(DirectOrderRequest{
		CustomerName: "",
		Phone:        "",
	})
	invReq := httptest.NewRequest(http.MethodPost, "/api/order", bytes.NewBuffer(invalidPayload))
	invRR := httptest.NewRecorder()
	router.ServeHTTP(invRR, invReq)
	if invRR.Code != http.StatusBadRequest {
		t.Errorf("Expected 400 Bad Request for empty direct order payload, got %d", invRR.Code)
	}

	// 3. Successful Direct Order
	validPayload, _ := json.Marshal(DirectOrderRequest{
		ProductType:     "Both (Rolls & Bags)",
		PrintType:       "Custom Printed / Branded",
		QuantityRolls:   "30 Rolls (320mm)",
		QuantityBags:    "100 Bundles",
		CompanyName:     "Summit Springs Pure Water",
		CustomerName:    "Emeka Nwosu",
		Phone:           "+2348033221100",
		Email:           "emeka@summitsprings.com",
		DeliveryAddress: "Plot 8 Commercial Layout, Enugu",
		Notes:           "Thickness 70 microns, new cylinder artwork ready",
	})

	validReq := httptest.NewRequest(http.MethodPost, "/api/order", bytes.NewBuffer(validPayload))
	validRR := httptest.NewRecorder()
	router.ServeHTTP(validRR, validReq)

	if validRR.Code != http.StatusCreated {
		t.Fatalf("Expected 201 Created for direct order, got %d. Body: %s", validRR.Code, validRR.Body.String())
	}

	var orderResp OrderResponse
	if err := json.Unmarshal(validRR.Body.Bytes(), &orderResp); err != nil {
		t.Fatalf("Failed to parse order JSON: %v", err)
	}

	if !orderResp.Success {
		t.Errorf("Expected order success to be true")
	}
	if !strings.HasPrefix(orderResp.OrderNumber, "WTR-") {
		t.Errorf("Expected order number prefix 'WTR-', got: %s", orderResp.OrderNumber)
	}
}

func TestBotChatHandler(t *testing.T) {
	router := SetupRouter()

	// 1. Method Not Allowed
	getReq := httptest.NewRequest(http.MethodGet, "/api/bot/chat", nil)
	getRR := httptest.NewRecorder()
	router.ServeHTTP(getRR, getReq)
	if getRR.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected status 405 Method Not Allowed for GET /api/bot/chat, got %d", getRR.Code)
	}

	// 2. Empty Message returns 400 Bad Request
	emptyPayload, _ := json.Marshal(ChatRequest{Message: ""})
	emptyReq := httptest.NewRequest(http.MethodPost, "/api/bot/chat", bytes.NewBuffer(emptyPayload))
	emptyRR := httptest.NewRecorder()
	router.ServeHTTP(emptyRR, emptyReq)
	if emptyRR.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 Bad Request for empty message, got %d", emptyRR.Code)
	}

	// 3. Greeting intent focuses on water packaging
	greetPayload, _ := json.Marshal(ChatRequest{Message: "Hello there!"})
	greetReq := httptest.NewRequest(http.MethodPost, "/api/bot/chat", bytes.NewBuffer(greetPayload))
	greetRR := httptest.NewRecorder()
	router.ServeHTTP(greetRR, greetReq)

	if greetRR.Code != http.StatusOK {
		t.Fatalf("Expected 200 OK, got %d", greetRR.Code)
	}
	var greetResp ChatResponse
	if err := json.Unmarshal(greetRR.Body.Bytes(), &greetResp); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if !strings.Contains(strings.ToLower(greetResp.Reply), "water") {
		t.Errorf("Expected greeting reply to mention water packaging, got: %s", greetResp.Reply)
	}

	// 4. Plain Roll query intent
	rollPayload, _ := json.Marshal(ChatRequest{Message: "Tell me about Plain Roll specifications and 320mm film"})
	rollReq := httptest.NewRequest(http.MethodPost, "/api/bot/chat", bytes.NewBuffer(rollPayload))
	rollRR := httptest.NewRecorder()
	router.ServeHTTP(rollRR, rollReq)

	var rollResp ChatResponse
	_ = json.Unmarshal(rollRR.Body.Bytes(), &rollResp)
	if !strings.Contains(rollResp.Reply, "Plain Roll") && !strings.Contains(rollResp.Reply, "320mm") {
		t.Errorf("Expected Plain Roll specifications in reply, got: %s", rollResp.Reply)
	}

	// 5. Printed Film query intent
	filmPayload, _ := json.Marshal(ChatRequest{Message: "Can you do custom Printed Film with our pure water brand logo?"})
	filmReq := httptest.NewRequest(http.MethodPost, "/api/bot/chat", bytes.NewBuffer(filmPayload))
	filmRR := httptest.NewRecorder()
	router.ServeHTTP(filmRR, filmReq)

	var filmResp ChatResponse
	_ = json.Unmarshal(filmRR.Body.Bytes(), &filmResp)
	if !strings.Contains(filmResp.Reply, "Printed Film") {
		t.Errorf("Expected Printed Film details in reply, got: %s", filmResp.Reply)
	}

	// 6. Printed Packing Bag query intent
	bagPayload, _ := json.Marshal(ChatRequest{Message: "What are your Printed Packing Bags for 20 sachets?"})
	bagReq := httptest.NewRequest(http.MethodPost, "/api/bot/chat", bytes.NewBuffer(bagPayload))
	bagRR := httptest.NewRecorder()
	router.ServeHTTP(bagRR, bagReq)

	var bagResp ChatResponse
	_ = json.Unmarshal(bagRR.Body.Bytes(), &bagResp)
	if !strings.Contains(bagResp.Reply, "Printed Packing Bag") && !strings.Contains(bagResp.Reply, "Packing Bag") {
		t.Errorf("Expected Printed Packing Bag specifications in reply, got: %s", bagResp.Reply)
	}

	// 7. Shrink Wrap / SSB & MSB query intent
	shrinkPayload, _ := json.Marshal(ChatRequest{Message: "Tell me about Shrink Wrap / SSB & MSB for bundling water"})
	shrinkReq := httptest.NewRequest(http.MethodPost, "/api/bot/chat", bytes.NewBuffer(shrinkPayload))
	shrinkRR := httptest.NewRecorder()
	router.ServeHTTP(shrinkRR, shrinkReq)

	var shrinkResp ChatResponse
	_ = json.Unmarshal(shrinkRR.Body.Bytes(), &shrinkResp)
	if !strings.Contains(shrinkResp.Reply, "Shrink Wrap") {
		t.Errorf("Expected Shrink Wrap / SSB & MSB specifications in reply, got: %s", shrinkResp.Reply)
	}

	// 8. Lead / Contact Registration intent
	leadPayload, _ := json.Marshal(ChatRequest{
		Name:    "Chinedu (Aqua Pure)",
		Phone:   "+2348055667788",
		Message: "Need quote for 100 printed rolls.",
	})
	leadReq := httptest.NewRequest(http.MethodPost, "/api/bot/chat", bytes.NewBuffer(leadPayload))
	leadRR := httptest.NewRecorder()
	router.ServeHTTP(leadRR, leadReq)

	var leadResp ChatResponse
	_ = json.Unmarshal(leadRR.Body.Bytes(), &leadResp)
	if !leadResp.InquiryLogged {
		t.Errorf("Expected InquiryLogged to be true for lead submission")
	}
}

func TestBotStatusHandler(t *testing.T) {
	router := SetupRouter()

	req := httptest.NewRequest(http.MethodGet, "/api/bot/status", nil)
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for GET /api/bot/status, got %d", rr.Code)
	}

	var statusResp BotStatusResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &statusResp); err != nil {
		t.Fatalf("Failed to parse status JSON: %v", err)
	}
	if !statusResp.BotActive {
		t.Errorf("Expected BotActive to be true")
	}
	if statusResp.AdminOnline != false {
		t.Errorf("Expected AdminOnline to default to false")
	}
}

func TestStaticFiles(t *testing.T) {
	router := SetupRouter()

	// Check CSS file
	reqCSS := httptest.NewRequest(http.MethodGet, "/static/css/style.css", nil)
	rrCSS := httptest.NewRecorder()
	router.ServeHTTP(rrCSS, reqCSS)
	if rrCSS.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for /static/css/style.css, got %d", rrCSS.Code)
	}

	// Check JS files
	reqJS := httptest.NewRequest(http.MethodGet, "/static/js/bot.js", nil)
	rrJS := httptest.NewRecorder()
	router.ServeHTTP(rrJS, reqJS)
	if rrJS.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for /static/js/bot.js, got %d", rrJS.Code)
	}

	reqOrder := httptest.NewRequest(http.MethodGet, "/static/js/order.js", nil)
	rrOrder := httptest.NewRecorder()
	router.ServeHTTP(rrOrder, reqOrder)
	if rrOrder.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for /static/js/order.js, got %d", rrOrder.Code)
	}

	// Check Logos & Brand Assets
	reqLogo := httptest.NewRequest(http.MethodGet, "/static/images/logo.svg", nil)
	rrLogo := httptest.NewRecorder()
	router.ServeHTTP(rrLogo, reqLogo)
	if rrLogo.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for /static/images/logo.svg, got %d", rrLogo.Code)
	}

	// Check Product Images for all four products
	productImages := []string{
		"/static/images/plain-roll.jpg",
		"/static/images/printed-film.jpg",
		"/static/images/printed-packing-bag.jpg",
		"/static/images/shrink-wrap-ssb-msb.jpg",
		"/static/images/placeholder-product.svg",
	}

	for _, imgPath := range productImages {
		reqImg := httptest.NewRequest(http.MethodGet, imgPath, nil)
		rrImg := httptest.NewRecorder()
		router.ServeHTTP(rrImg, reqImg)
		if rrImg.Code != http.StatusOK {
			t.Errorf("Expected 200 OK for product image %s, got %d", imgPath, rrImg.Code)
		}
	}
}
