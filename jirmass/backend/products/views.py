import uuid
from decimal import Decimal
from django.db.models import Sum
from rest_framework import generics, status
from rest_framework.response import Response
from rest_framework.views import APIView

from .models import (
    Product,
    CustomerInquiry,
    AdminAvailability,
    Order,
    OrderItem,
    dispatch_admin_notification,
)
from .serializers import (
    ProductSerializer,
    CustomerInquirySerializer,
    AdminAvailabilitySerializer,
    OrderSerializer,
)


class ProductListCreateAPIView(generics.ListCreateAPIView):
    queryset = Product.objects.filter(is_active=True)
    serializer_class = ProductSerializer


class ProductDetailAPIView(generics.RetrieveUpdateDestroyAPIView):
    queryset = Product.objects.all()
    serializer_class = ProductSerializer


class CustomerInquiryListCreateAPIView(generics.ListCreateAPIView):
    queryset = CustomerInquiry.objects.all()
    serializer_class = CustomerInquirySerializer

    def perform_create(self, serializer):
        inquiry = serializer.save()
        # Dispatch live alert to admin (email and webhook)
        dispatch_admin_notification(
            event_type="customer_inquiry",
            title=f"Inquiry from {inquiry.name} ({inquiry.email})",
            details={
                "Customer Name": inquiry.name,
                "Email": inquiry.email,
                "Phone": inquiry.phone or "N/A",
                "Message": inquiry.message,
                "Bot Response": inquiry.bot_response or "Automated guidance provided",
            },
        )


class CustomerInquiryDetailAPIView(generics.RetrieveUpdateDestroyAPIView):
    queryset = CustomerInquiry.objects.all()
    serializer_class = CustomerInquirySerializer


class AdminAvailabilityAPIView(APIView):
    """Returns or updates the current admin presence & alert settings."""

    def get(self, request, *args, **kwargs):
        setting = AdminAvailability.objects.first()
        if not setting:
            setting = AdminAvailability.objects.create(
                is_online=False,
                away_message="Admin is currently offline. Automated assistant is responding to customer inquiries.",
                admin_email="jirmassplasticlimited@gmail.com",
            )
        serializer = AdminAvailabilitySerializer(setting)
        return Response(serializer.data, status=status.HTTP_200_OK)

    def post(self, request, *args, **kwargs):
        setting = AdminAvailability.objects.first()
        if not setting:
            setting = AdminAvailability.objects.create()

        if "is_online" in request.data:
            setting.is_online = bool(request.data.get("is_online"))
        if "away_message" in request.data:
            setting.away_message = request.data.get("away_message")
        if "admin_email" in request.data:
            setting.admin_email = request.data.get("admin_email")
        if "webhook_url" in request.data:
            setting.webhook_url = request.data.get("webhook_url")
        if "email_alerts_enabled" in request.data:
            setting.email_alerts_enabled = bool(request.data.get("email_alerts_enabled"))
        if "webhook_alerts_enabled" in request.data:
            setting.webhook_alerts_enabled = bool(request.data.get("webhook_alerts_enabled"))

        setting.save()
        serializer = AdminAvailabilitySerializer(setting)
        return Response(serializer.data, status=status.HTTP_200_OK)


class OrderListCreateAPIView(generics.ListCreateAPIView):
    queryset = Order.objects.all()
    serializer_class = OrderSerializer

    def perform_create(self, serializer):
        # Auto-generate order number if not supplied
        order_num = self.request.data.get("order_number")
        if not order_num:
            order_num = f"ORD-{uuid.uuid4().hex[:8].upper()}"

        order = serializer.save(order_number=order_num)

        # Dispatch live alert to admin (email and webhook)
        dispatch_admin_notification(
            event_type="new_order",
            title=f"Order {order.order_number} by {order.customer_name} (${order.total_amount})",
            details={
                "Order Number": order.order_number,
                "Customer Name": order.customer_name,
                "Email": order.customer_email,
                "Phone": order.customer_phone,
                "Shipping Address": order.shipping_address,
                "Total Amount": f"${order.total_amount}",
                "Items Count": order.items.count(),
            },
        )


class OrderDetailAPIView(generics.RetrieveUpdateDestroyAPIView):
    queryset = Order.objects.all()
    serializer_class = OrderSerializer


class DashboardStatsAPIView(APIView):
    """Provides high-level dashboard metrics for the administration panel."""

    def get(self, request, *args, **kwargs):
        total_products = Product.objects.filter(is_active=True).count()
        pending_inquiries = CustomerInquiry.objects.filter(status="pending").count()
        total_inquiries = CustomerInquiry.objects.count()
        total_orders = Order.objects.count()
        pending_orders = Order.objects.filter(status="pending").count()
        revenue = Order.objects.aggregate(total=Sum("total_amount"))["total"] or Decimal("0.00")

        setting = AdminAvailability.objects.first()
        is_admin_online = setting.is_online if setting else False

        recent_inquiries = CustomerInquirySerializer(
            CustomerInquiry.objects.all()[:5], many=True
        ).data
        recent_orders = OrderSerializer(
            Order.objects.all()[:5], many=True
        ).data

        return Response(
            {
                "metrics": {
                    "total_products": total_products,
                    "pending_inquiries": pending_inquiries,
                    "total_inquiries": total_inquiries,
                    "total_orders": total_orders,
                    "pending_orders": pending_orders,
                    "total_revenue": float(revenue),
                    "is_admin_online": is_admin_online,
                },
                "recent_inquiries": recent_inquiries,
                "recent_orders": recent_orders,
            },
            status=status.HTTP_200_OK,
        )


class TestAlertAPIView(APIView):
    """Triggers a test email & webhook notification for admin verification."""

    def post(self, request, *args, **kwargs):
        dispatch_admin_notification(
            event_type="test_alert",
            title="System Test Alert Verification",
            details={
                "Message": "This is a verification test of Jirmass automated alerts.",
                "Triggered By": request.data.get("sender", "Admin Console"),
            },
        )
        return Response(
            {"message": "Test alert dispatched to configured email and webhook."},
            status=status.HTTP_200_OK,
        )
