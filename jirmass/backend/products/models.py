import json
import logging
import threading
import urllib.request
from django.conf import settings
from django.core.mail import send_mail
from django.db import models

logger = logging.getLogger(__name__)


class Product(models.Model):
    name = models.CharField(max_length=200)
    description = models.TextField()
    price = models.DecimalField(max_digits=12, decimal_places=2)
    stock_quantity = models.PositiveIntegerField(default=0)
    sku = models.CharField(max_length=100, unique=True)
    image = models.FileField(
        upload_to="products/",
        blank=True,
        null=True,
        help_text="Upload product photo or graphic (PNG, JPG, SVG, WEBP)",
    )
    is_active = models.BooleanField(default=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    def __str__(self):
        return self.name


class CustomerInquiry(models.Model):
    STATUS_CHOICES = [
        ("pending", "Pending Review"),
        ("responded", "Responded"),
        ("closed", "Closed"),
    ]

    name = models.CharField(max_length=120)
    email = models.EmailField()
    phone = models.CharField(max_length=50, blank=True)
    message = models.TextField()
    bot_response = models.TextField(blank=True)
    status = models.CharField(max_length=20, choices=STATUS_CHOICES, default="pending")
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "Customer Inquiry"
        verbose_name_plural = "Customer Inquiries"
        ordering = ["-created_at"]

    def __str__(self):
        return f"Inquiry from {self.name} ({self.email})"


class AdminAvailability(models.Model):
    is_online = models.BooleanField(default=False)
    away_message = models.TextField(
        default="Admin is currently offline. Automated assistant is responding to customer inquiries."
    )
    admin_email = models.EmailField(default="jirmassplasticlimited@gmail.com")
    webhook_url = models.URLField(blank=True, default="")
    email_alerts_enabled = models.BooleanField(default=True)
    webhook_alerts_enabled = models.BooleanField(default=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "Admin Availability & Alerts Setting"
        verbose_name_plural = "Admin Availability & Alerts Settings"

    def __str__(self):
        status = "Online" if self.is_online else "Offline (Away)"
        return f"Admin Status: {status}"


class Order(models.Model):
    STATUS_CHOICES = [
        ("pending", "Pending Review"),
        ("confirmed", "Confirmed"),
        ("processing", "Processing"),
        ("shipped", "Shipped"),
        ("delivered", "Delivered"),
        ("cancelled", "Cancelled"),
    ]

    order_number = models.CharField(max_length=64, unique=True)
    customer_name = models.CharField(max_length=150)
    customer_email = models.EmailField()
    customer_phone = models.CharField(max_length=50)
    shipping_address = models.TextField()
    notes = models.TextField(blank=True)
    total_amount = models.DecimalField(max_digits=12, decimal_places=2, default=0.0)
    status = models.CharField(max_length=25, choices=STATUS_CHOICES, default="pending")
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "Customer Order"
        verbose_name_plural = "Customer Orders"
        ordering = ["-created_at"]

    def __str__(self):
        return f"Order {self.order_number} - {self.customer_name} (${self.total_amount})"


class OrderItem(models.Model):
    order = models.ForeignKey(Order, related_name="items", on_delete=models.CASCADE)
    product = models.ForeignKey(
        Product, null=True, blank=True, on_delete=models.SET_NULL
    )
    product_name = models.CharField(max_length=200)
    sku = models.CharField(max_length=100, blank=True)
    quantity = models.PositiveIntegerField(default=1)
    unit_price = models.DecimalField(max_digits=12, decimal_places=2)
    subtotal = models.DecimalField(max_digits=12, decimal_places=2)

    def __str__(self):
        return f"{self.quantity}x {self.product_name} (${self.subtotal})"


class BrandAsset(models.Model):
    title = models.CharField(max_length=100, default="Primary Brand Customization")
    logo = models.FileField(
        upload_to="brand/",
        blank=True,
        null=True,
        help_text="Upload your company logo (SVG, PNG, JPG, or WEBP)",
    )
    hero_background = models.FileField(
        upload_to="brand/",
        blank=True,
        null=True,
        help_text="Upload hero banner background graphic (SVG, JPG, PNG, or WEBP)",
    )
    favicon = models.FileField(
        upload_to="brand/",
        blank=True,
        null=True,
        help_text="Upload browser favicon (SVG, ICO, or PNG)",
    )
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        verbose_name = "Brand Graphic & Asset Customization"
        verbose_name_plural = "Brand Graphic & Asset Customizations"

    def __str__(self):
        return self.title


def dispatch_admin_notification(event_type: str, title: str, details: dict):
    """Dispatches live admin notifications (email and webhook) when the admin is away or a high-priority event occurs."""
    setting = AdminAvailability.objects.first()
    if not setting:
        setting = AdminAvailability.objects.create(
            is_online=False,
            admin_email="jirmassplasticlimited@gmail.com",
            email_alerts_enabled=True,
            webhook_alerts_enabled=True,
        )

    # 1. Email notification
    if setting.email_alerts_enabled and setting.admin_email:
        subject = f"[Jirmass Live Alert] {event_type.upper()}: {title}"
        body_lines = [
            f"Event Type: {event_type}",
            f"Summary: {title}",
            f"Admin Status: {'ONLINE' if setting.is_online else 'AWAY / OFFLINE'}",
            "--------------------------------------------------",
        ]
        for key, value in details.items():
            body_lines.append(f"{key}: {value}")
        body_lines.append("--------------------------------------------------")
        body_lines.append(
            "Access Admin Dashboard: http://localhost:8000/admin/ or http://localhost:8081"
        )
        body = "\n".join(body_lines)

        try:
            send_mail(
                subject=subject,
                message=body,
                from_email=getattr(
                    settings, "DEFAULT_FROM_EMAIL", "bot@jirmassplastics.com"
                ),
                recipient_list=[setting.admin_email],
                fail_silently=True,
            )
        except Exception as e:
            logger.warning("Failed to send email alert: %v", e)

    # 2. Webhook notification (dispatched asynchronously)
    if setting.webhook_alerts_enabled and setting.webhook_url:
        payload = {
            "event": event_type,
            "title": title,
            "admin_online": setting.is_online,
            "details": details,
        }

        def post_webhook():
            try:
                data = json.dumps(payload).encode("utf-8")
                req = urllib.request.Request(
                    setting.webhook_url,
                    data=data,
                    headers={"Content-Type": "application/json"},
                )
                with urllib.request.urlopen(req, timeout=4) as _:
                    pass
            except Exception as ex:
                logger.warning("Webhook dispatch failed: %s", ex)

        t = threading.Thread(target=post_webhook, daemon=True)
        t.start()
