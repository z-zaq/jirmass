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
    serializer_class = CustomerInquirySerializer

    def get_queryset(self):
        qs = CustomerInquiry.objects.all()
        email = self.request.query_params.get("email")
        if email:
            qs = qs.filter(email__iexact=email.strip())
        return qs

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
    serializer_class = OrderSerializer

    def get_queryset(self):
        qs = Order.objects.all()
        email = self.request.query_params.get("email")
        if email:
            qs = qs.filter(customer_email__iexact=email.strip())
        return qs

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


class HealthCheckAPIView(APIView):
    """Health check endpoint for container and uptime monitoring."""

    def get(self, request, *args, **kwargs):
        from django.db import connection
        db_status = "connected"
        try:
            connection.ensure_connection()
        except Exception as e:
            db_status = f"unhealthy: {str(e)}"
            return Response(
                {"status": "degraded", "service": "jirmass-django-backend", "database": db_status},
                status=status.HTTP_503_SERVICE_UNAVAILABLE,
            )

        return Response(
            {"status": "ok", "service": "jirmass-django-backend", "database": db_status},
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


class UserLoginAPIView(APIView):
    """Authenticates customers, staff, and administrative users by username or email."""

    def post(self, request, *args, **kwargs):
        from django.contrib.auth import authenticate
        from django.contrib.auth.models import User

        username = request.data.get("username", "").strip()
        password = request.data.get("password", "")

        if not username or not password:
            return Response(
                {"success": False, "error": "Username or email and password are required."},
                status=status.HTTP_400_BAD_REQUEST,
            )

        user = authenticate(request, username=username, password=password)
        if user is None:
            # Check if input is an email address
            user_obj = User.objects.filter(email__iexact=username).first()
            if user_obj:
                user = authenticate(request, username=user_obj.username, password=password)

        if user is None:
            return Response(
                {"success": False, "error": "Invalid username/email or password."},
                status=status.HTTP_401_UNAUTHORIZED,
            )

        if not user.is_active:
            return Response(
                {"success": False, "error": "Account is disabled."},
                status=status.HTTP_403_FORBIDDEN,
            )

        full_name = user.get_full_name() or user.first_name or user.username
        company_name = user.last_name or ""

        return Response(
            {
                "success": True,
                "user": {
                    "id": user.id,
                    "username": user.username,
                    "email": user.email,
                    "name": full_name,
                    "company_name": company_name,
                    "is_staff": user.is_staff,
                    "is_superuser": user.is_superuser,
                },
                "username": user.username,
                "email": user.email,
                "name": full_name,
                "company_name": company_name,
                "is_staff": user.is_staff,
                "is_superuser": user.is_superuser,
            },
            status=status.HTTP_200_OK,
        )


StaffLoginAPIView = UserLoginAPIView


class UserRegisterAPIView(APIView):
    """Registers a new customer account for water packaging ordering and inquiries."""

    def post(self, request, *args, **kwargs):
        from django.contrib.auth.models import User

        name = request.data.get("name", "").strip()
        company_name = request.data.get("company_name", "").strip()
        email = request.data.get("email", "").strip().lower()
        username = request.data.get("username", "").strip()
        password = request.data.get("password", "")

        if not email or "@" not in email:
            return Response(
                {"success": False, "error": "A valid email address is required."},
                status=status.HTTP_400_BAD_REQUEST,
            )

        if not password or len(password) < 6:
            return Response(
                {"success": False, "error": "Password must be at least 6 characters long."},
                status=status.HTTP_400_BAD_REQUEST,
            )

        if User.objects.filter(email__iexact=email).exists():
            return Response(
                {"success": False, "error": "An account with this email address already exists. Please log in."},
                status=status.HTTP_400_BAD_REQUEST,
            )

        if not username:
            base_username = email.split("@")[0]
            # Strip invalid username chars
            cleaned_base = "".join(c for c in base_username if c.isalnum() or c in "_-")
            if not cleaned_base:
                cleaned_base = "user"
            username = cleaned_base
            counter = 1
            while User.objects.filter(username__iexact=username).exists():
                username = f"{cleaned_base}{counter}"
                counter += 1
        else:
            if User.objects.filter(username__iexact=username).exists():
                return Response(
                    {"success": False, "error": "This username is already taken. Please choose another."},
                    status=status.HTTP_400_BAD_REQUEST,
                )

        user = User.objects.create_user(
            username=username,
            email=email,
            password=password,
            first_name=name,
            last_name=company_name,
        )

        return Response(
            {
                "success": True,
                "message": "Account created successfully.",
                "user": {
                    "id": user.id,
                    "username": user.username,
                    "email": user.email,
                    "name": name or user.username,
                    "company_name": company_name,
                    "is_staff": False,
                    "is_superuser": False,
                },
                "username": user.username,
                "email": user.email,
                "name": name or user.username,
                "company_name": company_name,
                "is_staff": False,
                "is_superuser": False,
            },
            status=status.HTTP_201_CREATED,
        )

