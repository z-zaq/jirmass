#!/usr/bin/env python3
"""
Jirmass Plastics - Frontend Development Preview Server
Replicates the Go web server (main.go and bot.go) on port :8081
Serving HTML templates, static assets, order API, and chatbot.
"""

import http.server
import json
import mimetypes
import os
import random
import re
import string
import sys
import urllib.parse
import urllib.request
from datetime import datetime

PORT = 8081
BASE_DIR = os.path.dirname(os.path.abspath(__file__))
TEMPLATES_DIR = os.path.join(BASE_DIR, "templates")
STATIC_DIR = os.path.join(BASE_DIR, "static")

def render_template(filename, context):
    filepath = os.path.join(TEMPLATES_DIR, filename)
    with open(filepath, "r", encoding="utf-8") as f:
        content = f.read()
    
    content = content.replace("{{.Title}}", context.get("Title", "Jirmass Plastics"))
    content = content.replace("{{.CurrentYear}}", str(context.get("CurrentYear", datetime.now().year)))
    
    flash = context.get("FlashMessage", "")
    if flash:
        content = re.sub(r"\{\{if \.FlashMessage\}\}(.*?)\{\{end\}\}", r"\1", content, flags=re.DOTALL)
        content = content.replace("{{.FlashMessage}}", flash)
    else:
        content = re.sub(r"\{\{if \.FlashMessage\}\}.*?\{\{end\}\}", "", content, flags=re.DOTALL)
    
    return content.encode("utf-8")

def process_bot_reply(user_msg, name="", email="", phone=""):
    lower = user_msg.lower().strip()
    
    # Lead registration
    if email or phone or "@" in lower:
        client_name = name or "Pure Water Factory Partner"
        client_contact = email or phone
        reply = (
            f"Thank you {client_name}! Because our sales administration team is currently offline, "
            f"I have logged your order inquiry and contact details ({client_contact}). "
            "Our production representative will contact you via WhatsApp/Phone shortly with factory pricing and delivery timelines."
        )
        return {
            "reply": reply,
            "admin_online": False,
            "is_away_reply": True,
            "inquiry_logged": True,
            "timestamp": datetime.now().strftime("%H:%M"),
            "suggested_prompts": ["Place Direct Order", "Pricing per kg", "Delivery details"],
        }
    
    # Admin inquiries
    if any(k in lower for k in ["admin", "human", "person", "agent", "manager", "staff", "offline", "online", "who are you"]):
        reply = (
            "Notice: Our human sales managers are currently AWAY (The sales admin is currently offline. "
            "Our automated bot is active 24/7 to assist with Plain Roll, Printed Film, Printed Packing Bag, "
            "and Shrink Wrap / SSB & MSB orders). I am Jirmass Bot, your automated assistant. I can help you with "
            "specifications and direct order placement. To request a callback, simply leave your factory name and phone number!"
        )
        return {
            "reply": reply,
            "admin_online": False,
            "is_away_reply": True,
            "inquiry_logged": False,
            "timestamp": datetime.now().strftime("%H:%M"),
            "suggested_prompts": ["Plain Roll info", "Printed Film info", "Printed Packing Bag info", "Shrink Wrap / SSB & MSB info"],
        }
    
    # Shrink Wrap / SSB & MSB
    if any(k in lower for k in ["shrink", "shrink wrap", "ssb", "msb", "ssb&msb", "thermal shrink", "bundling wrap"]):
        reply = (
            "📦 Jirmass Shrink Wrap / SSB & MSB (Industrial Packaging Film):\n\n"
            "• High-tensile thermal contraction film calibrated for Single Sachet Bag (SSB) and Multiple Sachet Bag (MSB) collation.\n"
            "• Also optimized for bottled water multipack shrink wrapping.\n"
            "• Exceptional puncture and tear resilience preventing transit bursts.\n"
            "• High-clarity dust, moisture, and hygienic barrier maintaining water cleanliness.\n"
            "• Available in custom widths, micron gauges, and biaxial shrink contraction ratios.\n\n"
            "👉 You can place a direct order for Shrink Wrap / SSB & MSB using the form on this page!"
        )
        return {
            "reply": reply,
            "admin_online": False,
            "is_away_reply": True,
            "inquiry_logged": False,
            "timestamp": datetime.now().strftime("%H:%M"),
            "suggested_prompts": ["Plain Roll info", "Printed Film info", "Printed Packing Bag info", "Place Direct Order"],
        }

    # Printed Packing Bag
    if any(k in lower for k in ["packing bag", "printed packing bag", "printed bag", "outer bag", "bundle", "bale", "sack", "20 sachets"]):
        reply = (
            "🛍️ Jirmass Printed Packing Bag (Outer Bundling Sacks):\n\n"
            "• Heavy-duty outer bagging sacks engineered to securely bundle and hold 20 sachets of pure water.\n"
            "• High puncture resistance and reinforced tear-proof bottom seals built for rough transport.\n"
            "• Custom rotogravure branding with your pure water factory name, depot contact, and logo printing.\n"
            "• Packaged in convenient bundles (e.g., 500 bags per bundle, 1,000 bags, or 5,000 bag bales).\n\n"
            "👉 You can place a direct order for Printed Packing Bags using the form on this page!"
        )
        return {
            "reply": reply,
            "admin_online": False,
            "is_away_reply": True,
            "inquiry_logged": False,
            "timestamp": datetime.now().strftime("%H:%M"),
            "suggested_prompts": ["Plain Roll info", "Printed Film info", "Shrink Wrap / SSB & MSB info", "Place Direct Order"],
        }

    # Printed Film
    if any(k in lower for k in ["printed film", "printed roll", "custom print", "rotogravure", "branded roll", "branding", "eye mark", "cylinder"]):
        reply = (
            "🖨️ Jirmass Printed Film (Custom Branded Roll Film):\n\n"
            "• High-definition rotogravure custom printed sachet water packaging roll film.\n"
            "• Precision multi-color graphics with factory brand logo, NAFDAC license numbers, and batch zones.\n"
            "• Standard 320mm roll width with precise photo-eye registration marks for automatic form-fill-seal machines.\n"
            "• 100% virgin food-grade LDPE with food-safe inks that never bleed or affect water taste.\n"
            "• Available by roll count or bulk weight (kg).\n\n"
            "👉 To order Printed Film, use the Direct Order form on this page!"
        )
        return {
            "reply": reply,
            "admin_online": False,
            "is_away_reply": True,
            "inquiry_logged": False,
            "timestamp": datetime.now().strftime("%H:%M"),
            "suggested_prompts": ["Plain Roll info", "Printed Packing Bag info", "Place Direct Order"],
        }

    # Locations, Branches, Phone, Email & Delivery
    if any(k in lower for k in ["location", "locations", "address", "addresses", "branch", "branches", "office", "offices", "where", "otukpo", "kaduna", "makurdi", "phone", "tel", "call", "whatsapp", "email", "contact", "delivery", "shipping"]):
        reply = (
            "📍 Jirmass Plastics Factory & Distribution Locations:\n\n"
            "1. Otukpo Branch: No.3 New Market Road, Otukpo, Benue State\n"
            "2. Kaduna Branch: No.3 Kachia Road Along Station Round About, Kaduna\n"
            "3. Makurdi Branch: No.28 Naka Road, Makurdi\n\n"
            "📞 Direct Phone/WhatsApp Lines:\n"
            "• 08138881463\n"
            "• 09036148838\n"
            "• 07066820641\n\n"
            "✉️ Official Email: jirmassplasticlimited@gmail.com\n\n"
            "🚚 Nationwide direct factory dispatch available for Plain Roll, Printed Film, Bags, and Shrink Wrap!"
        )
        return {
            "reply": reply,
            "admin_online": False,
            "is_away_reply": True,
            "inquiry_logged": False,
            "timestamp": datetime.now().strftime("%H:%M"),
            "suggested_prompts": ["Place Direct Order", "Plain Roll info", "Printed Film info", "Shrink Wrap info"],
        }

    # Plain Roll
    if any(k in lower for k in ["plain roll", "plain rolls", "plain film", "unprinted", "clear roll", "320mm", "film", "gauge", "micron", "machine", "roll", "rolls", "bag", "bags"]):
        reply = (
            "💧 Jirmass Plain Roll (Form-Fill-Seal Roll Film):\n\n"
            "• Standard 320mm roll width, calibrated for automated form-fill-seal packaging machines.\n"
            "• 100% virgin food-grade LDPE with high tensile strength for leak-free sealing.\n"
            "• Uniform gauge thickness (60–75 microns) and slip additives for jam-free feeding.\n"
            "• High-clarity transparent unprinted film rolls with zero odor.\n"
            "• Available by roll count or bulk weight (kg).\n\n"
            "👉 To order Plain Roll, use the Direct Order form on this page!"
        )
        return {
            "reply": reply,
            "admin_online": False,
            "is_away_reply": True,
            "inquiry_logged": False,
            "timestamp": datetime.now().strftime("%H:%M"),
            "suggested_prompts": ["Printed Film info", "Printed Packing Bag info", "Shrink Wrap / SSB & MSB info", "Place Direct Order"],
        }

    # Greetings
    if any(k in lower for k in ["hello", "hi", "hey", "good morning", "good afternoon", "good evening", "greetings"]):
        reply = (
            "👋 Welcome to Jirmass Plastics! We specialize in Plain Roll, Printed Film, Printed Packing Bag, "
            "and Shrink Wrap / SSB & MSB for pure water packaging plants. Our admin is currently away, but you can ask about "
            "roll specs, bags, shrink wrap, or place your direct order right here. How can I assist you?"
        )
        return {
            "reply": reply,
            "admin_online": False,
            "is_away_reply": True,
            "inquiry_logged": False,
            "timestamp": datetime.now().strftime("%H:%M"),
            "suggested_prompts": ["Plain Roll info", "Printed Film info", "Printed Packing Bag info", "Shrink Wrap / SSB & MSB info"],
        }

    # Default fallback
    reply = (
        "Welcome to Jirmass Plastics — manufacturers of Plain Roll, Printed Film, Printed Packing Bag, "
        "and Shrink Wrap / SSB & MSB. Since our sales manager is currently offline, you can place a direct order "
        "using the form on this page, or reply with your WhatsApp/phone number and requirements to receive a fast callback."
    )
    return {
        "reply": reply,
        "admin_online": False,
        "is_away_reply": True,
        "inquiry_logged": False,
        "timestamp": datetime.now().strftime("%H:%M"),
        "suggested_prompts": ["Plain Roll info", "Printed Film info", "Printed Packing Bag info", "Place Direct Order"],
    }

class JirmassHTTPHandler(http.server.BaseHTTPRequestHandler):
    def log_message(self, format, *args):
        # Concise logging
        sys.stderr.write(f"[{datetime.now().strftime('%H:%M:%S')}] {args[0]} - {args[1]}\n")

    def do_GET(self):
        parsed = urllib.parse.urlparse(self.path)
        path = parsed.path

        # Static assets
        if path.startswith("/static/"):
            rel_path = path[len("/static/"):]
            file_path = os.path.join(STATIC_DIR, rel_path.replace("/", os.sep))
            if os.path.isfile(file_path):
                mime, _ = mimetypes.guess_type(file_path)
                if not mime:
                    if file_path.endswith(".svg"):
                        mime = "image/svg+xml"
                    elif file_path.endswith(".css"):
                        mime = "text/css"
                    elif file_path.endswith(".js"):
                        mime = "application/javascript"
                    else:
                        mime = "application/octet-stream"
                self.send_response(200)
                self.send_header("Content-Type", mime)
                self.send_header("Cache-Control", "no-cache")
                self.end_headers()
                with open(file_path, "rb") as f:
                    self.wfile.write(f.read())
                return
            else:
                self.send_error(404, "Static file not found")
                return

        # Health check
        if path in ("/healthz", "/api/health"):
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps({"status": "ok", "service": "jirmass-dev-server"}).encode("utf-8"))
            return

        # Robots & Sitemap
        if path == "/robots.txt":
            self.send_response(200)
            self.send_header("Content-Type", "text/plain")
            self.end_headers()
            self.wfile.write(b"User-agent: *\nAllow: /\nDisallow: /api/\n")
            return

        if path == "/sitemap.xml":
            xml = (
                '<?xml version="1.0" encoding="UTF-8"?>\n'
                '<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n'
                '  <url><loc>http://localhost:8081/</loc><priority>1.0</priority></url>\n'
                '  <url><loc>http://localhost:8081/about</loc><priority>0.8</priority></url>\n'
                '  <url><loc>http://localhost:8081/contact</loc><priority>0.8</priority></url>\n'
                '</urlset>'
            )
            self.send_response(200)
            self.send_header("Content-Type", "application/xml; charset=utf-8")
            self.end_headers()
            self.wfile.write(xml.encode("utf-8"))
            return

        # Bot status
        if path == "/api/bot/status":
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            data = {
                "admin_online": False,
                "bot_active": True,
                "away_message": "The sales admin is currently offline. Auto-reply is active.",
                "operating_hours": "Monday to Friday, 8:00 AM – 5:00 PM (GMT+1). Automated support is 24/7.",
                "llm_enabled": False,
                "llm_model": "gemini-1.5-flash",
                "server_time": datetime.now().strftime("%H:%M:%S"),
            }
            self.wfile.write(json.dumps(data).encode("utf-8"))
            return

        # HTML Pages
        if path == "/":
            body = render_template("home.html", {
                "Title": "Jirmass Plastics - Plain Roll, Printed Film, Packing Bags & Shrink Wrap",
                "AdminOnline": False,
                "CurrentYear": datetime.now().year,
            })
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.end_headers()
            self.wfile.write(body)
            return

        if path == "/about":
            body = render_template("about.html", {
                "Title": "About Our Manufacturing - Jirmass Plastics",
                "CurrentYear": datetime.now().year,
            })
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.end_headers()
            self.wfile.write(body)
            return

        if path == "/contact":
            body = render_template("contact.html", {
                "Title": "Contact Factory Sales - Jirmass Plastics",
                "FlashMessage": "",
                "CurrentYear": datetime.now().year,
            })
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.end_headers()
            self.wfile.write(body)
            return

        if path == "/login":
            body = render_template("login.html", {
                "Title": "Staff Portal Login - Jirmass Plastics",
                "FlashMessage": "",
                "CurrentYear": datetime.now().year,
            })
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.end_headers()
            self.wfile.write(body)
            return

        if path == "/logout":
            self.send_response(303)
            self.send_header("Location", "/login")
            self.end_headers()
            return

        if path == "/admin-portal":
            body = render_template("portal.html", {
                "Title": "Staff & Operations Portal - Jirmass Plastics",
                "Username": "admin",
                "AdminOnline": False,
                "TotalOrders": 1,
                "TotalInquiries": 0,
                "Orders": [],
                "Inquiries": [],
                "CurrentYear": datetime.now().year,
            })
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.end_headers()
            self.wfile.write(body)
            return

        if path == "/favicon.ico":
            icon_path = os.path.join(STATIC_DIR, "images", "logo-icon.svg")
            if os.path.isfile(icon_path):
                self.send_response(200)
                self.send_header("Content-Type", "image/svg+xml")
                self.send_header("Cache-Control", "no-cache")
                self.end_headers()
                with open(icon_path, "rb") as f:
                    self.wfile.write(f.read())
                return

        self.send_error(404, "Page not found")

    def do_POST(self):
        parsed = urllib.parse.urlparse(self.path)
        path = parsed.path
        length = int(self.headers.get("Content-Length", 0))
        post_data = self.rfile.read(length)

        # Login submission
        if path == "/login":
            params = urllib.parse.parse_qs(post_data.decode("utf-8"))
            username = params.get("username", [""])[0]
            password = params.get("password", [""])[0]
            if username == "admin" and (password == "admin123" or password == "admin"):
                self.send_response(303)
                self.send_header("Location", "/admin-portal")
                self.end_headers()
                return
            else:
                body = render_template("login.html", {
                    "Title": "Staff Portal Login - Jirmass Plastics",
                    "FlashMessage": "Invalid username or password.",
                    "CurrentYear": datetime.now().year,
                })
                self.send_response(401)
                self.send_header("Content-Type", "text/html; charset=utf-8")
                self.end_headers()
                self.wfile.write(body)
                return

        if path == "/admin-portal/toggle-status":
            self.send_response(303)
            self.send_header("Location", "/admin-portal")
            self.end_headers()
            return

        # Contact form submission
        if path == "/contact":
            params = urllib.parse.parse_qs(post_data.decode("utf-8"))
            name = params.get("name", [""])[0]
            flash_msg = f"Thank you {name}, your inquiry has been received! Our factory sales team will reach out promptly."
            body = render_template("contact.html", {
                "Title": "Contact Factory Sales - Jirmass Plastics",
                "FlashMessage": flash_msg,
                "CurrentYear": datetime.now().year,
            })
            self.send_response(200)
            self.send_header("Content-Type", "text/html; charset=utf-8")
            self.end_headers()
            self.wfile.write(body)
            return

        # Direct Order API
        if path == "/api/order":
            try:
                order_data = json.loads(post_data.decode("utf-8"))
            except Exception:
                order_data = urllib.parse.parse_qs(post_data.decode("utf-8"))
                order_data = {k: v[0] for k, v in order_data.items()}

            customer_name = order_data.get("customer_name", "").strip()
            phone = order_data.get("phone", "").strip()
            email = order_data.get("email", "").strip()

            if not customer_name or (not phone and not email):
                self.send_response(400)
                self.send_header("Content-Type", "text/plain")
                self.end_headers()
                self.wfile.write(b"Contact name and phone number or email are required to place an order")
                return

            rand_id = "".join(random.choices(string.ascii_uppercase + string.digits, k=6))
            order_num = f"WTR-{datetime.now().strftime('%Y%m%d')}-{rand_id}"

            # Optionally forward to Django if running
            try:
                django_payload = {
                    "order_number": order_num,
                    "customer_name": f"{customer_name} ({order_data.get('company_name', '')})",
                    "customer_email": email,
                    "customer_phone": phone,
                    "shipping_address": order_data.get("delivery_address", ""),
                    "notes": f"Product: {order_data.get('product_type')} | Rolls: {order_data.get('quantity_rolls')} | Bags: {order_data.get('quantity_bags')}",
                    "total_amount": "0.00",
                    "items": [
                        {
                            "product_name": f"{order_data.get('product_type', 'Water Packaging')} - {order_data.get('print_type', 'Custom')}",
                            "quantity": 1,
                            "unit_price": "0.00",
                        }
                    ],
                }
                req = urllib.request.Request(
                    "http://localhost:8000/api/products/orders/",
                    data=json.dumps(django_payload).encode("utf-8"),
                    headers={"Content-Type": "application/json"},
                )
                with urllib.request.urlopen(req, timeout=1.5):
                    pass
            except Exception:
                pass  # Django backend is optional

            resp_payload = {
                "success": True,
                "order_number": order_num,
                "message": (
                    "Thank you! Your direct order for water packaging materials "
                    "(Plain Roll, Printed Film, Printed Packing Bag, Shrink Wrap / SSB & MSB) has been logged. "
                    "Our sales team will reach out via WhatsApp/phone to confirm dispatch details."
                ),
                "alert_sent": True,
            }
            self.send_response(201)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps(resp_payload).encode("utf-8"))
            return

        # Chatbot API
        if path == "/api/bot/chat":
            try:
                req_json = json.loads(post_data.decode("utf-8"))
            except Exception:
                self.send_error(400, "Invalid JSON")
                return

            user_msg = req_json.get("message", "").strip()
            if not user_msg:
                self.send_error(400, "Message is required")
                return

            res = process_bot_reply(
                user_msg,
                name=req_json.get("name", ""),
                email=req_json.get("email", ""),
                phone=req_json.get("phone", ""),
            )

            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.end_headers()
            self.wfile.write(json.dumps(res).encode("utf-8"))
            return

        self.send_error(404, "Endpoint not found")

def run():
    server_address = ("", PORT)
    httpd = http.server.ThreadingHTTPServer(server_address, JirmassHTTPHandler)
    print(f"Jirmass Plastics Frontend Server running on http://localhost:{PORT}")
    print(f"Serving Plain Roll, Printed Film, Printed Packing Bag, Shrink Wrap / SSB & MSB")
    try:
        httpd.serve_forever()
    except KeyboardInterrupt:
        pass
    finally:
        httpd.server_close()

if __name__ == "__main__":
    run()
