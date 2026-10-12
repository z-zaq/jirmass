from decimal import Decimal
from rest_framework import serializers
from .models import Product, CustomerInquiry, AdminAvailability, Order, OrderItem


class ProductSerializer(serializers.ModelSerializer):
    class Meta:
        model = Product
        fields = [
            "id",
            "name",
            "description",
            "price",
            "stock_quantity",
            "sku",
            "image",
            "is_active",
            "created_at",
            "updated_at",
        ]


class CustomerInquirySerializer(serializers.ModelSerializer):
    class Meta:
        model = CustomerInquiry
        fields = [
            "id",
            "name",
            "email",
            "phone",
            "message",
            "bot_response",
            "status",
            "created_at",
            "updated_at",
        ]
        read_only_fields = ["id", "created_at", "updated_at"]


class AdminAvailabilitySerializer(serializers.ModelSerializer):
    class Meta:
        model = AdminAvailability
        fields = [
            "id",
            "is_online",
            "away_message",
            "admin_email",
            "webhook_url",
            "email_alerts_enabled",
            "webhook_alerts_enabled",
            "updated_at",
        ]
        read_only_fields = ["id", "updated_at"]


class OrderItemSerializer(serializers.ModelSerializer):
    subtotal = serializers.DecimalField(max_digits=12, decimal_places=2, required=False)

    class Meta:
        model = OrderItem
        fields = [
            "id",
            "product",
            "product_name",
            "sku",
            "quantity",
            "unit_price",
            "subtotal",
        ]


class OrderSerializer(serializers.ModelSerializer):
    items = OrderItemSerializer(many=True, required=False)
    order_number = serializers.CharField(required=False)
    total_amount = serializers.DecimalField(max_digits=12, decimal_places=2, required=False)

    class Meta:
        model = Order
        fields = [
            "id",
            "order_number",
            "customer_name",
            "customer_email",
            "customer_phone",
            "shipping_address",
            "notes",
            "total_amount",
            "status",
            "items",
            "created_at",
            "updated_at",
        ]
        read_only_fields = ["id", "created_at", "updated_at"]

    def create(self, validated_data):
        items_data = self.context.get("request").data.get("items", []) if self.context.get("request") else []
        if "items" in validated_data:
            validated_data.pop("items")

        order = Order.objects.create(**validated_data)
        total = Decimal("0.00")

        for item_data in items_data:
            product_id = item_data.get("product_id") or item_data.get("product")
            product = None
            if product_id:
                product = Product.objects.filter(id=product_id).first()

            qty = int(item_data.get("quantity", 1))
            unit_price = Decimal(str(item_data.get("unit_price", product.price if product else "0.00")))
            subtotal = unit_price * qty
            total += subtotal

            OrderItem.objects.create(
                order=order,
                product=product,
                product_name=item_data.get("product_name", product.name if product else "Custom Plastic Item"),
                sku=item_data.get("sku", product.sku if product else ""),
                quantity=qty,
                unit_price=unit_price,
                subtotal=subtotal,
            )

        if total > 0 and (not order.total_amount or order.total_amount == 0):
            order.total_amount = total
            order.save()

        return order
