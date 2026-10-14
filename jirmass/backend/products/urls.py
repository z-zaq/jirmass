from django.urls import path
from .views import (
    ProductListCreateAPIView,
    ProductDetailAPIView,
    CustomerInquiryListCreateAPIView,
    CustomerInquiryDetailAPIView,
    AdminAvailabilityAPIView,
    OrderListCreateAPIView,
    OrderDetailAPIView,
    DashboardStatsAPIView,
    TestAlertAPIView,
    HealthCheckAPIView,
    StaffLoginAPIView,
    UserLoginAPIView,
    UserRegisterAPIView,
)

urlpatterns = [
    path("", ProductListCreateAPIView.as_view(), name="product-list-create"),
    path("<int:pk>/", ProductDetailAPIView.as_view(), name="product-detail"),
    path("inquiries/", CustomerInquiryListCreateAPIView.as_view(), name="inquiry-list-create"),
    path("inquiries/<int:pk>/", CustomerInquiryDetailAPIView.as_view(), name="inquiry-detail"),
    path("admin-status/", AdminAvailabilityAPIView.as_view(), name="admin-status"),
    path("orders/", OrderListCreateAPIView.as_view(), name="order-list-create"),
    path("orders/<int:pk>/", OrderDetailAPIView.as_view(), name="order-detail"),
    path("dashboard-stats/", DashboardStatsAPIView.as_view(), name="dashboard-stats"),
    path("test-alert/", TestAlertAPIView.as_view(), name="test-alert"),
    path("health/", HealthCheckAPIView.as_view(), name="backend-health"),
    path("auth/login/", UserLoginAPIView.as_view(), name="user-login"),
    path("auth/staff-login/", StaffLoginAPIView.as_view(), name="staff-login"),
    path("auth/register/", UserRegisterAPIView.as_view(), name="user-register"),
]
