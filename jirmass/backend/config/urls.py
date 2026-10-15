from django.contrib import admin
from django.http import HttpResponse
from django.urls import include, path
from products.views import HealthCheckAPIView


def health_check(request):
    """Lightweight health check endpoint for Render probes (GET and HEAD)."""
    if request.method == "HEAD":
        response = HttpResponse(status=200, content_type="text/plain")
        response["Content-Length"] = "2"
        return response
    return HttpResponse("OK", status=200, content_type="text/plain")


urlpatterns = [
    path("", health_check, name="root-health"),
    path("health/", health_check, name="health-slash"),
    path("health", health_check, name="health"),
    path("admin/", admin.site.urls),
    path("healthz", HealthCheckAPIView.as_view(), name="root-healthz"),
    path("healthz/", HealthCheckAPIView.as_view(), name="root-healthz-slash"),
    path("api/products/", include("products.urls")),
]

