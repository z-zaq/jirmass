package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"
)

// SessionUser represents an authenticated customer or staff member
type SessionUser struct {
	ID          int    `json:"id"`
	Username    string `json:"username"`
	Email       string `json:"email"`
	FullName    string `json:"name"`
	CompanyName string `json:"company_name"`
	Phone       string `json:"phone"`
	IsStaff     bool   `json:"is_staff"`
	IsSuperuser bool   `json:"is_superuser"`
}

// PageData carries template context for water packaging website
type PageData struct {
	Title           string
	FlashMessage    string
	ErrorMessage    string
	AdminOnline     bool
	IsAuthenticated bool
	Username        string
	User            SessionUser
	CurrentYear     int
	Next            string
}

// PortalOrder represents order records fetched for the dashboard
type PortalOrder struct {
	OrderNumber     string `json:"order_number"`
	CustomerName    string `json:"customer_name"`
	CustomerPhone   string `json:"customer_phone"`
	CustomerEmail   string `json:"customer_email"`
	ShippingAddress string `json:"shipping_address"`
	Notes           string `json:"notes"`
	Status          string `json:"status"`
}

// PortalInquiry represents customer inquiries fetched for the dashboard
type PortalInquiry struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Message string `json:"message"`
	Status  string `json:"status"`
}

// PortalData carries template context for the staff operations portal
type PortalData struct {
	Title          string
	Username       string
	AdminOnline    bool
	TotalOrders    int
	TotalInquiries int
	Orders         []PortalOrder
	Inquiries      []PortalInquiry
	FlashMessage   string
	CurrentYear    int
}

// AccountData carries template context for the customer account dashboard
type AccountData struct {
	Title          string
	User           SessionUser
	AdminOnline    bool
	TotalOrders    int
	TotalInquiries int
	Orders         []PortalOrder
	Inquiries      []PortalInquiry
	FlashMessage   string
	CurrentYear    int
}

var (
	sessionLock   sync.RWMutex
	validSessions = make(map[string]SessionUser)
)

func createSession(user SessionUser) string {
	tokenBytes := make([]byte, 24)
	_, _ = rand.Read(tokenBytes)
	token := hex.EncodeToString(tokenBytes)

	sessionLock.Lock()
	validSessions[token] = user
	sessionLock.Unlock()
	return token
}

func getSessionUser(r *http.Request) (SessionUser, bool) {
	for _, cookieName := range []string{"jirmass_session", "jirmass_staff_session"} {
		if cookie, err := r.Cookie(cookieName); err == nil && cookie.Value != "" {
			sessionLock.RLock()
			user, ok := validSessions[cookie.Value]
			sessionLock.RUnlock()
			if ok {
				return user, true
			}
		}
	}
	return SessionUser{}, false
}

func clearSession(w http.ResponseWriter, r *http.Request) {
	for _, cookieName := range []string{"jirmass_session", "jirmass_staff_session"} {
		if cookie, err := r.Cookie(cookieName); err == nil {
			sessionLock.Lock()
			delete(validSessions, cookie.Value)
			sessionLock.Unlock()
		}
		http.SetCookie(w, &http.Cookie{
			Name:     cookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			MaxAge:   -1,
			Expires:  time.Unix(0, 0),
		})
	}
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

// HealthzHandler returns operational status of the Go frontend server
func HealthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ok",
		"service":   "jirmass-go-frontend",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// SecurityHeadersMiddleware adds essential HTTP security headers
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}

// SetupRouter sets up web and API routes for the simple water packaging web app
func SetupRouter() *http.ServeMux {
	mux := http.NewServeMux()

	// Static assets handler with cache control for immediate updates
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle("/static/", http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		fileServer.ServeHTTP(w, r)
	})))

	// Health check endpoints
	mux.HandleFunc("/healthz", HealthzHandler)
	mux.HandleFunc("/api/health", HealthzHandler)

	// SEO & Crawler endpoints
	mux.HandleFunc("/robots.txt", RobotsHandler)
	mux.HandleFunc("/sitemap.xml", SitemapHandler)

	// Page endpoints
	mux.HandleFunc("/", HomeHandler)
	mux.HandleFunc("/about", AboutHandler)
	mux.HandleFunc("/contact", ContactHandler)

	// Direct Order placement endpoint
	mux.HandleFunc("/api/order", DirectOrderHandler)

	// Automated customer bot endpoints
	mux.HandleFunc("/api/bot/chat", BotChatHandler)
	mux.HandleFunc("/api/bot/status", BotStatusHandler)

	// User Authentication & Customer Account endpoints
	mux.HandleFunc("/login", LoginHandler)
	mux.HandleFunc("/register", RegisterHandler)
	mux.HandleFunc("/account", UserAccountHandler)
	mux.HandleFunc("/logout", LogoutHandler)

	// Staff Operations Portal endpoints
	mux.HandleFunc("/admin-portal", AdminPortalHandler)
	mux.HandleFunc("/admin-portal/toggle-status", AdminToggleStatusHandler)

	return mux
}

// RobotsHandler serves standard robots.txt for search engines
func RobotsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintln(w, "User-agent: *\nAllow: /\nDisallow: /api/")
}

// SitemapHandler serves the sitemap XML for search engine indexing
func SitemapHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	xmlContent := `<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>http://localhost:8081/</loc>
    <changefreq>weekly</changefreq>
    <priority>1.0</priority>
  </url>
  <url>
    <loc>http://localhost:8081/about</loc>
    <changefreq>monthly</changefreq>
    <priority>0.8</priority>
  </url>
  <url>
    <loc>http://localhost:8081/contact</loc>
    <changefreq>monthly</changefreq>
    <priority>0.8</priority>
  </url>
</urlset>`
	fmt.Fprint(w, xmlContent)
}

func getBackendURL() string {
	val := os.Getenv("BACKEND_URL")
	if val == "" {
		val = os.Getenv("DJANGO_API_URL")
		if val != "" {
			val = strings.TrimSuffix(val, "/api/products")
		}
	}
	if val == "" {
		return "http://localhost:8000"
	}
	val = strings.TrimSpace(val)
	if !strings.HasPrefix(val, "http://") && !strings.HasPrefix(val, "https://") {
		val = "https://" + val
	}
	return strings.TrimSuffix(val, "/")
}


func main() {
	router := SetupRouter()
	port := getEnvDefault("PORT", "8081")
	handler := SecurityHeadersMiddleware(router)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Jirmass Water Packaging Web Server running on http://localhost:%s\n", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server startup failed: %v", err)
		}
	}()

	<-stop
	log.Println("Shutting down Jirmass Web Server gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown error: %v", err)
	} else {
		log.Println("Jirmass Web Server stopped cleanly.")
	}
}

// LoginHandler processes user and staff authentication
func LoginHandler(w http.ResponseWriter, r *http.Request) {
	next := r.URL.Query().Get("next")
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		if user, ok := getSessionUser(r); ok {
			if next != "" && strings.HasPrefix(next, "/") {
				http.Redirect(w, r, next, http.StatusSeeOther)
				return
			}
			if user.IsStaff {
				http.Redirect(w, r, "/admin-portal", http.StatusSeeOther)
			} else {
				http.Redirect(w, r, "/account", http.StatusSeeOther)
			}
			return
		}
		flash := ""
		if r.URL.Query().Get("logged_out") == "1" {
			flash = "You have been successfully signed out."
		} else if r.URL.Query().Get("registered") == "1" {
			flash = "Account created successfully! Please sign in below."
		}
		botConfigLock.RLock()
		adminOnline := currentConfig.AdminOnline
		botConfigLock.RUnlock()

		renderTemplate(w, "templates/login.html", PageData{
			Title:        "Sign In - Jirmass Plastics",
			FlashMessage: flash,
			AdminOnline:  adminOnline,
			CurrentYear:  time.Now().Year(),
			Next:         next,
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_ = r.ParseForm()
	username := strings.TrimSpace(r.FormValue("username"))
	password := r.FormValue("password")
	next = strings.TrimSpace(r.FormValue("next"))

	botConfigLock.RLock()
	adminOnline := currentConfig.AdminOnline
	botConfigLock.RUnlock()

	if username == "" || password == "" {
		renderTemplate(w, "templates/login.html", PageData{
			Title:        "Sign In - Jirmass Plastics",
			ErrorMessage: "Username/email and password are required.",
			AdminOnline:  adminOnline,
			CurrentYear:  time.Now().Year(),
			Next:         next,
		})
		return
	}

	// Verify against Django authentication API
	backendURL := getBackendURL()
	authURL := strings.TrimRight(backendURL, "/") + "/api/products/auth/login/"
	authPayload, _ := json.Marshal(map[string]string{
		"username": username,
		"password": password,
	})

	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Post(authURL, "application/json", bytes.NewBuffer(authPayload))

	var sessionUser SessionUser
	authenticated := false

	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			var authRes struct {
				Success     bool        `json:"success"`
				User        SessionUser `json:"user"`
				Username    string      `json:"username"`
				Email       string      `json:"email"`
				Name        string      `json:"name"`
				CompanyName string      `json:"company_name"`
				IsStaff     bool        `json:"is_staff"`
				IsSuperuser bool        `json:"is_superuser"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&authRes); err == nil && authRes.Success {
				authenticated = true
				sessionUser = authRes.User
				if sessionUser.Username == "" {
					sessionUser.Username = authRes.Username
				}
				if sessionUser.Email == "" {
					sessionUser.Email = authRes.Email
				}
				if sessionUser.FullName == "" {
					sessionUser.FullName = authRes.Name
				}
				if sessionUser.CompanyName == "" {
					sessionUser.CompanyName = authRes.CompanyName
				}
				sessionUser.IsStaff = authRes.IsStaff
				sessionUser.IsSuperuser = authRes.IsSuperuser
			}
		}
	}

	// Fallback for offline dev
	if !authenticated && username == "admin" && (password == "admin123" || password == "admin") {
		authenticated = true
		sessionUser = SessionUser{
			Username:    "admin",
			Email:       "admin@jirmassplastics.com",
			FullName:    "Administrator",
			CompanyName: "Jirmass Plastics Ltd",
			IsStaff:     true,
			IsSuperuser: true,
		}
	}

	if !authenticated {
		w.WriteHeader(http.StatusUnauthorized)
		renderTemplate(w, "templates/login.html", PageData{
			Title:        "Sign In - Jirmass Plastics",
			ErrorMessage: "Invalid username/email or password. Please verify your credentials.",
			AdminOnline:  adminOnline,
			CurrentYear:  time.Now().Year(),
			Next:         next,
		})
		return
	}

	token := createSession(sessionUser)
	cookie := &http.Cookie{
		Name:     "jirmass_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	}
	http.SetCookie(w, cookie)
	http.SetCookie(w, &http.Cookie{
		Name:     "jirmass_staff_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})

	if next != "" && strings.HasPrefix(next, "/") {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	if sessionUser.IsStaff {
		http.Redirect(w, r, "/admin-portal", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/account", http.StatusSeeOther)
	}
}

// RegisterHandler processes new customer account registration
func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	next := r.URL.Query().Get("next")
	if r.Method == http.MethodGet || r.Method == http.MethodHead {
		if _, ok := getSessionUser(r); ok {
			http.Redirect(w, r, "/account", http.StatusSeeOther)
			return
		}
		botConfigLock.RLock()
		adminOnline := currentConfig.AdminOnline
		botConfigLock.RUnlock()

		renderTemplate(w, "templates/register.html", PageData{
			Title:       "Create Customer Account - Jirmass Plastics",
			AdminOnline: adminOnline,
			CurrentYear: time.Now().Year(),
			Next:        next,
		})
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	companyName := strings.TrimSpace(r.FormValue("company_name"))
	email := strings.TrimSpace(strings.ToLower(r.FormValue("email")))
	phone := strings.TrimSpace(r.FormValue("phone"))
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")
	next = strings.TrimSpace(r.FormValue("next"))

	botConfigLock.RLock()
	adminOnline := currentConfig.AdminOnline
	botConfigLock.RUnlock()

	renderErr := func(msg string) {
		w.WriteHeader(http.StatusBadRequest)
		renderTemplate(w, "templates/register.html", PageData{
			Title:        "Create Customer Account - Jirmass Plastics",
			ErrorMessage: msg,
			AdminOnline:  adminOnline,
			CurrentYear:  time.Now().Year(),
			Next:         next,
		})
	}

	if name == "" || email == "" || password == "" {
		renderErr("Full name, email address, and password are required.")
		return
	}
	if !strings.Contains(email, "@") {
		renderErr("Please enter a valid email address.")
		return
	}
	if len(password) < 6 {
		renderErr("Password must be at least 6 characters long.")
		return
	}
	if password != confirmPassword {
		renderErr("Passwords do not match. Please re-enter your password.")
		return
	}

	// Forward registration to Django backend
	backendURL := getBackendURL()
	regURL := strings.TrimRight(backendURL, "/") + "/api/products/auth/register/"
	regPayload, _ := json.Marshal(map[string]string{
		"name":         name,
		"company_name": companyName,
		"email":        email,
		"password":     password,
	})

	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Post(regURL, "application/json", bytes.NewBuffer(regPayload))
	if err != nil {
		renderErr("Registration service temporarily unavailable. Please try again.")
		return
	}
	defer resp.Body.Close()

	var regRes struct {
		Success bool        `json:"success"`
		Error   string      `json:"error"`
		User    SessionUser `json:"user"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&regRes); err != nil || !regRes.Success {
		errMsg := regRes.Error
		if errMsg == "" {
			errMsg = "Unable to complete registration. Please check your details."
		}
		renderErr(errMsg)
		return
	}

	sessionUser := regRes.User
	if sessionUser.FullName == "" {
		sessionUser.FullName = name
	}
	if sessionUser.CompanyName == "" {
		sessionUser.CompanyName = companyName
	}
	if sessionUser.Email == "" {
		sessionUser.Email = email
	}
	sessionUser.Phone = phone

	token := createSession(sessionUser)
	cookie := &http.Cookie{
		Name:     "jirmass_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	}
	http.SetCookie(w, cookie)
	http.SetCookie(w, &http.Cookie{
		Name:     "jirmass_staff_session",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7,
	})

	if next != "" && strings.HasPrefix(next, "/") {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/account", http.StatusSeeOther)
}

// LogoutHandler clears user session
func LogoutHandler(w http.ResponseWriter, r *http.Request) {
	clearSession(w, r)
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

// UserAccountHandler serves the customer dashboard showing their past orders and inquiries
func UserAccountHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getSessionUser(r)
	if !ok {
		http.Redirect(w, r, "/login?next=/account", http.StatusSeeOther)
		return
	}

	backendURL := getBackendURL()
	client := &http.Client{Timeout: 3 * time.Second}

	var orders []PortalOrder
	orderReqURL := fmt.Sprintf("%s/api/products/orders/?email=%s", strings.TrimRight(backendURL, "/"), url.QueryEscape(user.Email))
	if resp, err := client.Get(orderReqURL); err == nil {
		_ = json.NewDecoder(resp.Body).Decode(&orders)
		_ = resp.Body.Close()
	}

	var inquiries []PortalInquiry
	inquiryReqURL := fmt.Sprintf("%s/api/products/inquiries/?email=%s", strings.TrimRight(backendURL, "/"), url.QueryEscape(user.Email))
	if resp, err := client.Get(inquiryReqURL); err == nil {
		_ = json.NewDecoder(resp.Body).Decode(&inquiries)
		_ = resp.Body.Close()
	}

	botConfigLock.RLock()
	adminOnline := currentConfig.AdminOnline
	botConfigLock.RUnlock()

	renderTemplate(w, "templates/account.html", AccountData{
		Title:          "Customer Account & Orders - Jirmass Plastics",
		User:           user,
		AdminOnline:    adminOnline,
		TotalOrders:    len(orders),
		TotalInquiries: len(inquiries),
		Orders:         orders,
		Inquiries:      inquiries,
		CurrentYear:    time.Now().Year(),
	})
}

// AdminPortalHandler serves the protected staff operations portal
func AdminPortalHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getSessionUser(r)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	if !user.IsStaff {
		http.Redirect(w, r, "/account", http.StatusSeeOther)
		return
	}

	backendURL := getBackendURL()
	client := &http.Client{Timeout: 3 * time.Second}

	var orders []PortalOrder
	if resp, err := client.Get(strings.TrimRight(backendURL, "/") + "/api/products/orders/"); err == nil {
		_ = json.NewDecoder(resp.Body).Decode(&orders)
		_ = resp.Body.Close()
	}

	var inquiries []PortalInquiry
	if resp, err := client.Get(strings.TrimRight(backendURL, "/") + "/api/products/inquiries/"); err == nil {
		_ = json.NewDecoder(resp.Body).Decode(&inquiries)
		_ = resp.Body.Close()
	}

	botConfigLock.RLock()
	adminOnline := currentConfig.AdminOnline
	botConfigLock.RUnlock()

	renderTemplate(w, "templates/portal.html", PortalData{
		Title:          "Staff & Operations Portal - Jirmass Plastics",
		Username:       user.Username,
		AdminOnline:    adminOnline,
		TotalOrders:    len(orders),
		TotalInquiries: len(inquiries),
		Orders:         orders,
		Inquiries:      inquiries,
		CurrentYear:    time.Now().Year(),
	})
}

// AdminToggleStatusHandler switches staff online/offline state
func AdminToggleStatusHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getSessionUser(r)
	if !ok || !user.IsStaff {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	botConfigLock.Lock()
	currentConfig.AdminOnline = !currentConfig.AdminOnline
	newStatus := currentConfig.AdminOnline
	botConfigLock.Unlock()

	go func(online bool) {
		backendURL := getBackendURL()
		data, _ := json.Marshal(map[string]bool{"is_online": online})
		client := &http.Client{Timeout: 3 * time.Second}
		resp, err := client.Post(strings.TrimRight(backendURL, "/") + "/api/products/admin-status/", "application/json", bytes.NewBuffer(data))
		if err == nil {
			_ = resp.Body.Close()
		}
	}(newStatus)

	http.Redirect(w, r, "/admin-portal", http.StatusSeeOther)
}

// HomeHandler serves the simplified single-page water packaging site
func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	user, isAuth := getSessionUser(r)
	botConfigLock.RLock()
	adminOnline := currentConfig.AdminOnline
	botConfigLock.RUnlock()

	renderTemplate(w, "templates/home.html", PageData{
		Title:           "Jirmass Plastics - Plain Roll, Printed Film, Packing Bags & Shrink Wrap",
		AdminOnline:     adminOnline,
		IsAuthenticated: isAuth,
		Username:        user.Username,
		User:            user,
		CurrentYear:     time.Now().Year(),
	})
}

// AboutHandler serves the about page
func AboutHandler(w http.ResponseWriter, r *http.Request) {
	user, isAuth := getSessionUser(r)
	botConfigLock.RLock()
	adminOnline := currentConfig.AdminOnline
	botConfigLock.RUnlock()

	renderTemplate(w, "templates/about.html", PageData{
		Title:           "About Our Film Manufacturing - Jirmass Plastics",
		AdminOnline:     adminOnline,
		IsAuthenticated: isAuth,
		Username:        user.Username,
		User:            user,
		CurrentYear:     time.Now().Year(),
	})
}

// ContactHandler handles contact page views and direct contact inquiries
func ContactHandler(w http.ResponseWriter, r *http.Request) {
	user, isAuth := getSessionUser(r)
	flashMsg := ""
	errMsg := ""

	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err == nil {
			name := strings.TrimSpace(r.FormValue("name"))
			email := strings.TrimSpace(r.FormValue("email"))
			phone := strings.TrimSpace(r.FormValue("phone"))
			message := strings.TrimSpace(r.FormValue("message"))

			if name == "" && user.FullName != "" {
				name = user.FullName
			}
			if email == "" && user.Email != "" {
				email = user.Email
			}
			if phone == "" && user.Phone != "" {
				phone = user.Phone
			}

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
			} else {
				errMsg = "Please fill in your name, contact details, and inquiry message."
			}
		}
	}

	botConfigLock.RLock()
	adminOnline := currentConfig.AdminOnline
	botConfigLock.RUnlock()

	renderTemplate(w, "templates/contact.html", PageData{
		Title:           "Contact Factory Sales - Jirmass Plastics",
		FlashMessage:    flashMsg,
		ErrorMessage:    errMsg,
		AdminOnline:     adminOnline,
		IsAuthenticated: isAuth,
		Username:        user.Username,
		User:            user,
		CurrentYear:     time.Now().Year(),
	})
}

// DirectOrderHandler processes simple direct orders for water packing rolls and bags
func DirectOrderHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	user, _ := getSessionUser(r)

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

	// Auto-populate customer details from user session if left empty
	if strings.TrimSpace(req.CustomerName) == "" && user.FullName != "" {
		req.CustomerName = user.FullName
	}
	if strings.TrimSpace(req.CustomerName) == "" {
		req.CustomerName = user.Username
	}
	if strings.TrimSpace(req.Email) == "" && user.Email != "" {
		req.Email = user.Email
	}
	if strings.TrimSpace(req.CompanyName) == "" && user.CompanyName != "" {
		req.CompanyName = user.CompanyName
	}
	if strings.TrimSpace(req.Phone) == "" && user.Phone != "" {
		req.Phone = user.Phone
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
	backendURL := getBackendURL()
	orderURL := strings.TrimRight(backendURL, "/") + "/api/products/orders/"
	resp, err := client.Post(orderURL, "application/json", strings.NewReader(string(data)))
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

var (
	templateCache     = make(map[string]*template.Template)
	templateCacheLock sync.RWMutex
)

func getTemplate(tmplFile string) (*template.Template, error) {
	templateCacheLock.RLock()
	tmpl, exists := templateCache[tmplFile]
	templateCacheLock.RUnlock()
	if exists {
		return tmpl, nil
	}

	templateCacheLock.Lock()
	defer templateCacheLock.Unlock()

	// Double-check after acquiring write lock
	if tmpl, exists := templateCache[tmplFile]; exists {
		return tmpl, nil
	}

	parsed, err := template.ParseFiles(tmplFile)
	if err != nil {
		return nil, err
	}
	templateCache[tmplFile] = parsed
	return parsed, nil
}

func renderTemplate(w http.ResponseWriter, tmplFile string, data interface{}) {
	tmpl, err := getTemplate(tmplFile)
	if err != nil {
		http.Error(w, "Failed to render template: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, "Failed to execute template: "+err.Error(), http.StatusInternalServerError)
	}
}
