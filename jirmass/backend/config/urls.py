from django.contrib import admin
from django.urls import include, path
from products.views import HealthCheckAPIView

urlpatterns = [
    path("admin/", admin.site.urls),
    path("healthz", HealthCheckAPIView.as_view(), name="root-healthz"),
    path("healthz/", HealthCheckAPIView.as_view(), name="root-healthz-slash"),
    path("api/products/", include("products.urls")),
]
