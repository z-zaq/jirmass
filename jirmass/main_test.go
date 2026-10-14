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

func TestHealthzHandler(t *testing.T) {
	router := SetupRouter()

	endpoints := []string{"/healthz", "/api/health"}
	for _, ep := range endpoints {
		req := httptest.NewRequest(http.MethodGet, ep, nil)
		rr := httptest.NewRecorder()
		router.ServeHTTP(rr, req)

		if rr.Code != http.StatusOK {
			t.Errorf("Expected 200 OK for %s, got %d", ep, rr.Code)
		}

		var payload map[string]interface{}
		if err := json.Unmarshal(rr.Body.Bytes(), &payload); err != nil {
			t.Fatalf("Failed to parse JSON for %s: %v", ep, err)
		}
		if payload["status"] != "ok" {
			t.Errorf("Expected status 'ok', got %v", payload["status"])
		}
		if payload["service"] != "jirmass-go-frontend" {
			t.Errorf("Expected service 'jirmass-go-frontend', got %v", payload["service"])
		}
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	router := SetupRouter()
	handler := SecurityHeadersMiddleware(router)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if val := rr.Header().Get("X-Content-Type-Options"); val != "nosniff" {
		t.Errorf("Expected X-Content-Type-Options: nosniff, got: %s", val)
	}
	if val := rr.Header().Get("X-Frame-Options"); val != "SAMEORIGIN" {
		t.Errorf("Expected X-Frame-Options: SAMEORIGIN, got: %s", val)
	}
	if val := rr.Header().Get("Referrer-Policy"); val != "strict-origin-when-cross-origin" {
		t.Errorf("Expected Referrer-Policy: strict-origin-when-cross-origin, got: %s", val)
	}
}

func TestRobotsAndSitemap(t *testing.T) {
	router := SetupRouter()

	// Test /robots.txt
	reqRobots := httptest.NewRequest(http.MethodGet, "/robots.txt", nil)
	rrRobots := httptest.NewRecorder()
	router.ServeHTTP(rrRobots, reqRobots)

	if rrRobots.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for /robots.txt, got %d", rrRobots.Code)
	}
	if !strings.Contains(rrRobots.Body.String(), "User-agent: *") {
		t.Errorf("Expected User-agent in /robots.txt")
	}

	// Test /sitemap.xml
	reqSitemap := httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil)
	rrSitemap := httptest.NewRecorder()
	router.ServeHTTP(rrSitemap, reqSitemap)

	if rrSitemap.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for /sitemap.xml, got %d", rrSitemap.Code)
	}
	if !strings.Contains(rrSitemap.Body.String(), "<urlset") {
		t.Errorf("Expected urlset XML in /sitemap.xml")
	}
}

func TestLoginAndStaffPortal(t *testing.T) {
	router := SetupRouter()

	// 1. GET /login returns 200 OK with login form
	reqGet := httptest.NewRequest(http.MethodGet, "/login", nil)
	rrGet := httptest.NewRecorder()
	router.ServeHTTP(rrGet, reqGet)

	if rrGet.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for GET /login, got %d", rrGet.Code)
	}
	if !strings.Contains(rrGet.Body.String(), "Staff Management Portal") {
		t.Errorf("Expected login form on /login")
	}

	// 2. POST /login with empty credentials returns 200 with flash message
	formEmpty := url.Values{"username": {""}, "password": {""}}
	reqEmpty := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formEmpty.Encode()))
	reqEmpty.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrEmpty := httptest.NewRecorder()
	router.ServeHTTP(rrEmpty, reqEmpty)

	if !strings.Contains(rrEmpty.Body.String(), "required") {
		t.Errorf("Expected required fields message on empty login")
	}

	// 3. POST /login with invalid credentials returns 401
	formBad := url.Values{"username": {"admin"}, "password": {"wrongpass"}}
	reqBad := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formBad.Encode()))
	reqBad.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrBad := httptest.NewRecorder()
	router.ServeHTTP(rrBad, reqBad)

	if rrBad.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for bad credentials, got %d", rrBad.Code)
	}

	// 4. POST /login with valid admin credentials returns 303 redirect & sets cookie
	formGood := url.Values{"username": {"admin"}, "password": {"admin123"}}
	reqGood := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(formGood.Encode()))
	reqGood.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rrGood := httptest.NewRecorder()
	router.ServeHTTP(rrGood, reqGood)

	if rrGood.Code != http.StatusSeeOther {
		t.Fatalf("Expected 303 See Other redirect on successful login, got %d", rrGood.Code)
	}

	cookies := rrGood.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == "jirmass_staff_session" {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil || sessionCookie.Value == "" {
		t.Fatalf("Expected jirmass_staff_session cookie to be set")
	}

	// 5. GET /admin-portal without cookie redirects to /login
	reqPortalUnauth := httptest.NewRequest(http.MethodGet, "/admin-portal", nil)
	rrPortalUnauth := httptest.NewRecorder()
	router.ServeHTTP(rrPortalUnauth, reqPortalUnauth)

	if rrPortalUnauth.Code != http.StatusSeeOther || rrPortalUnauth.Header().Get("Location") != "/login" {
		t.Errorf("Expected redirect to /login for unauthenticated /admin-portal access")
	}

	// 6. GET /admin-portal with session cookie succeeds
	reqPortalAuth := httptest.NewRequest(http.MethodGet, "/admin-portal", nil)
	reqPortalAuth.AddCookie(sessionCookie)
	rrPortalAuth := httptest.NewRecorder()
	router.ServeHTTP(rrPortalAuth, reqPortalAuth)

	if rrPortalAuth.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for authenticated /admin-portal, got %d", rrPortalAuth.Code)
	}
	if !strings.Contains(rrPortalAuth.Body.String(), "Staff &amp; Operations Dashboard") &&
		!strings.Contains(rrPortalAuth.Body.String(), "Staff & Operations Dashboard") {
		t.Errorf("Expected Staff Operations Dashboard content")
	}

	// 7. GET /logout clears cookie and redirects to /login
	reqLogout := httptest.NewRequest(http.MethodGet, "/logout", nil)
	reqLogout.AddCookie(sessionCookie)
	rrLogout := httptest.NewRecorder()
	router.ServeHTTP(rrLogout, reqLogout)

	if rrLogout.Code != http.StatusSeeOther || rrLogout.Header().Get("Location") != "/login" {
		t.Errorf("Expected redirect to /login on logout")
	}
}



