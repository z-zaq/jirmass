from django.contrib import admin
from .models import Product, CustomerInquiry, AdminAvailability, Order, OrderItem, BrandAsset

admin.site.site_header = "Jirmass Plastics Administration"
admin.site.site_title = "Jirmass Admin"
admin.site.index_title = "Operations & Customer Support Dashboard"


@admin.register(Product)
class ProductAdmin(admin.ModelAdmin):
    list_display = ("name", "sku", "price", "stock_quantity", "image", "is_active", "created_at")
    search_fields = ("name", "sku")
    list_filter = ("is_active",)


@admin.register(CustomerInquiry)
class CustomerInquiryAdmin(admin.ModelAdmin):
    list_display = ("name", "email", "phone", "status", "created_at")
    search_fields = ("name", "email", "message")
    list_filter = ("status", "created_at")
    readonly_fields = ("created_at", "updated_at")


@admin.register(AdminAvailability)
class AdminAvailabilityAdmin(admin.ModelAdmin):
    list_display = (
        "__str__",
        "is_online",
        "admin_email",
        "webhook_url",
        "email_alerts_enabled",
        "webhook_alerts_enabled",
        "updated_at",
    )
    list_editable = ("is_online", "email_alerts_enabled", "webhook_alerts_enabled")


class OrderItemInline(admin.TabularInline):
    model = OrderItem
    extra = 0
    readonly_fields = ("product_name", "sku", "quantity", "unit_price", "subtotal")


@admin.register(Order)
class OrderAdmin(admin.ModelAdmin):
    list_display = (
        "order_number",
        "customer_name",
        "customer_email",
        "customer_phone",
        "total_amount",
        "status",
        "created_at",
    )
    list_filter = ("status", "created_at")
    search_fields = ("order_number", "customer_name", "customer_email", "customer_phone")
    inlines = [OrderItemInline]
    readonly_fields = ("order_number", "total_amount", "created_at", "updated_at")


@admin.register(BrandAsset)
class BrandAssetAdmin(admin.ModelAdmin):
    list_display = ("title", "logo", "hero_background", "favicon", "updated_at")
    fieldsets = (
        ("Brand Customization", {
            "description": "Upload your company logo, hero background graphic, and favicon directly here.",
            "fields": ("title", "logo", "hero_background", "favicon"),
        }),
    )
