from decimal import Decimal
from django.core import mail
from django.test import TestCase
from django.urls import reverse
from rest_framework import status
from rest_framework.test import APITestCase
from .models import (
    Product,
    CustomerInquiry,
    AdminAvailability,
    Order,
    OrderItem,
    dispatch_admin_notification,
)


class ProductModelTest(TestCase):
    def setUp(self):
        self.product = Product.objects.create(
            name="20L Industrial Jerrycan",
            description="Heavy duty HDPE plastic jerrycan with tamper-evident cap.",
            price=15.50,
            stock_quantity=250,
            sku="JC-20L-HDPE",
            is_active=True,
        )

    def test_product_str(self):
        self.assertEqual(str(self.product), "20L Industrial Jerrycan")

    def test_product_fields(self):
        self.assertEqual(self.product.stock_quantity, 250)
        self.assertTrue(self.product.is_active)
        self.assertEqual(self.product.sku, "JC-20L-HDPE")

    def test_product_image_field_and_water_packaging_products(self):
        p1 = Product.objects.create(
            name="Plain Roll",
            description="320mm LDPE plain roll film",
            price=Decimal("0.00"),
            stock_quantity=500,
            sku="WTR-TEST-PLAIN-ROLL",
            image="products/plain-roll.svg",
            is_active=True,
        )
        p2 = Product.objects.create(
            name="Printed Film",
            description="Custom printed rotogravure roll",
            price=Decimal("0.00"),
            stock_quantity=500,
            sku="WTR-TEST-PRINTED-FILM",
            image="products/printed-film.svg",
            is_active=True,
        )
        p3 = Product.objects.create(
            name="Printed Packing Bag",
            description="20 sachet outer bagging sack",
            price=Decimal("0.00"),
            stock_quantity=1000,
            sku="WTR-TEST-PACKING-BAG",
            image="products/printed-packing-bag.svg",
            is_active=True,
        )
        p4 = Product.objects.create(
            name="Shrink Wrap / SSB & MSB",
            description="SSB and MSB thermal contraction film",
            price=Decimal("0.00"),
            stock_quantity=500,
            sku="WTR-TEST-SHRINK-WRAP",
            image="products/shrink-wrap-ssb-msb.svg",
            is_active=True,
        )
        self.assertEqual(p1.name, "Plain Roll")
        self.assertEqual(p2.name, "Printed Film")
        self.assertEqual(p3.name, "Printed Packing Bag")
        self.assertEqual(p4.name, "Shrink Wrap / SSB & MSB")
        self.assertTrue(p1.image.name.endswith("plain-roll.svg"))
        self.assertTrue(p2.image.name.endswith("printed-film.svg"))
        self.assertTrue(p3.image.name.endswith("printed-packing-bag.svg"))
        self.assertTrue(p4.image.name.endswith("shrink-wrap-ssb-msb.svg"))


class CustomerInquiryModelTest(TestCase):
    def test_inquiry_creation_and_defaults(self):
        inquiry = CustomerInquiry.objects.create(
            name="Amara Okafor",
            email="amara@example.com",
            phone="+2348012345678",
            message="Looking for 500 units of 20L jerrycans.",
            bot_response="Automated quote guidance provided.",
        )
        self.assertEqual(inquiry.status, "pending")
        self.assertIn("Amara Okafor", str(inquiry))
        self.assertIn("amara@example.com", str(inquiry))


class AdminAvailabilityModelTest(TestCase):
    def test_default_availability(self):
        setting = AdminAvailability.objects.create()
        self.assertFalse(setting.is_online)
        self.assertIn("Offline", str(setting))

        setting.is_online = True
        self.assertIn("Online", str(setting))


class OrderModelTest(TestCase):
    def setUp(self):
        self.product = Product.objects.create(
            name="10L Multi-Use Bucket",
            description="Durable bucket",
            price=6.00,
            stock_quantity=200,
            sku="BKT-10L",
            is_active=True,
        )

    def test_order_creation_and_items(self):
        order = Order.objects.create(
            order_number="ORD-TEST-001",
            customer_name="Kofi Mensah",
            customer_email="kofi@example.com",
            customer_phone="+233201122334",
            shipping_address="Tema Industrial Area, Ghana",
            total_amount=Decimal("120.00"),
        )
        item = OrderItem.objects.create(
            order=order,
            product=self.product,
            product_name=self.product.name,
            sku=self.product.sku,
            quantity=20,
            unit_price=Decimal("6.00"),
            subtotal=Decimal("120.00"),
        )
        self.assertEqual(order.items.count(), 1)
        self.assertEqual(str(item), "20x 10L Multi-Use Bucket ($120.00)")
        self.assertIn("ORD-TEST-001", str(order))


class ProductAPITest(APITestCase):
    def setUp(self):
        self.active_product = Product.objects.create(
            name="10L Household Bucket",
            description="Durable plastic bucket with handle.",
            price=6.00,
            stock_quantity=100,
            sku="BKT-10L-01",
            is_active=True,
        )
        self.inactive_product = Product.objects.create(
            name="Discontinued Crate",
            description="Old crate design.",
            price=12.00,
            stock_quantity=0,
            sku="CRT-DISC-01",
            is_active=False,
        )

    def test_list_products_returns_only_active(self):
        url = reverse("product-list-create")
        response = self.client.get(url)
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        skus = [p["sku"] for p in response.data]
        self.assertIn("BKT-10L-01", skus)
        self.assertNotIn("CRT-DISC-01", skus)

    def test_create_product(self):
        url = reverse("product-list-create")
        payload = {
            "name": "5L Water Bottle",
            "description": "Food grade PET container.",
            "price": "3.50",
            "stock_quantity": 400,
            "sku": "PET-5L-01",
            "is_active": True,
        }
        response = self.client.post(url, payload, format="json")
        self.assertEqual(response.status_code, status.HTTP_201_CREATED)
        self.assertEqual(Product.objects.count(), 3)

    def test_retrieve_product_detail(self):
        url = reverse("product-detail", kwargs={"pk": self.active_product.pk})
        response = self.client.get(url)
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        self.assertEqual(response.data["name"], "10L Household Bucket")

    def test_update_product(self):
        url = reverse("product-detail", kwargs={"pk": self.active_product.pk})
        response = self.client.patch(url, {"price": "7.25"}, format="json")
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        self.active_product.refresh_from_db()
        self.assertEqual(float(self.active_product.price), 7.25)


class CustomerInquiryAPITest(APITestCase):
    def test_create_and_list_inquiry(self):
        url = reverse("inquiry-list-create")
        payload = {
            "name": "David Mensah",
            "email": "david@example.com",
            "phone": "+233200112233",
            "message": "Do you deliver to Accra?",
            "bot_response": "Yes, we ship across West Africa.",
        }
        create_res = self.client.post(url, payload, format="json")
        self.assertEqual(create_res.status_code, status.HTTP_201_CREATED)
        self.assertEqual(CustomerInquiry.objects.count(), 1)

        # Check that admin alert email was queued
        self.assertGreaterEqual(len(mail.outbox), 1)
        self.assertIn("David Mensah", mail.outbox[-1].subject)

        list_res = self.client.get(url)
        self.assertEqual(list_res.status_code, status.HTTP_200_OK)
        self.assertEqual(len(list_res.data), 1)


class OrderAPITest(APITestCase):
    def setUp(self):
        self.product = Product.objects.create(
            name="25L Chemical Jerrycan",
            description="UN certified HDPE drum",
            price=18.00,
            stock_quantity=500,
            sku="JC-25L-UN",
            is_active=True,
        )

    def test_create_order_with_items(self):
        url = reverse("order-list-create")
        payload = {
            "customer_name": "Zenith Manufacturing Ltd",
            "customer_email": "orders@zenith.com",
            "customer_phone": "+2348099887766",
            "shipping_address": "Ikeja Industrial Estate, Lagos",
            "items": [
                {
                    "product_id": self.product.id,
                    "product_name": self.product.name,
                    "sku": self.product.sku,
                    "quantity": 50,
                    "unit_price": "18.00",
                }
            ],
        }
        response = self.client.post(url, payload, format="json")
        self.assertEqual(response.status_code, status.HTTP_201_CREATED)
        self.assertEqual(Order.objects.count(), 1)

        order = Order.objects.first()
        self.assertEqual(float(order.total_amount), 900.00)
        self.assertEqual(order.items.count(), 1)

        # Check alert email dispatched
        self.assertGreaterEqual(len(mail.outbox), 1)
        self.assertIn("Zenith Manufacturing Ltd", mail.outbox[-1].subject)


class DashboardStatsAPITest(APITestCase):
    def test_dashboard_stats(self):
        Product.objects.create(
            name="Item A", description="Desc", price=10.0, stock_quantity=10, sku="SKU-A"
        )
        url = reverse("dashboard-stats")
        response = self.client.get(url)
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        self.assertIn("metrics", response.data)
        self.assertIn("total_products", response.data["metrics"])
        self.assertEqual(response.data["metrics"]["total_products"], 1)

    def test_test_alert_endpoint(self):
        url = reverse("test-alert")
        response = self.client.post(url, {"sender": "Test Suite"}, format="json")
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        self.assertGreaterEqual(len(mail.outbox), 1)


class HealthCheckAPITest(APITestCase):
    def test_health_check_endpoint(self):
        url = reverse("backend-health")
        response = self.client.get(url)
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        self.assertEqual(response.data.get("status"), "ok")
        self.assertEqual(response.data.get("database"), "connected")

    def test_root_healthz_endpoint(self):
        url = reverse("root-healthz")
        response = self.client.get(url)
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        self.assertEqual(response.data.get("status"), "ok")
        head_response = self.client.head(url)
        self.assertEqual(head_response.status_code, status.HTTP_200_OK)

    def test_root_health_endpoints_get_and_head(self):
        for path in ["/", "/health/", "/health"]:
            # Test GET probe
            get_resp = self.client.get(path)
            self.assertEqual(get_resp.status_code, status.HTTP_200_OK)
            self.assertEqual(get_resp.content, b"OK")

            # Test HEAD probe (RFC 9110 compliance: status 200 without message body)
            head_resp = self.client.head(path)
            self.assertEqual(head_resp.status_code, status.HTTP_200_OK)
            self.assertEqual(head_resp.content, b"")



class StaffLoginAPITest(APITestCase):
    def setUp(self):
        from django.contrib.auth import get_user_model
        User = get_user_model()
        self.user = User.objects.create_user(
            username="staff_member", password="securepassword123", is_staff=True
        )

    def test_successful_staff_login(self):
        url = reverse("staff-login")
        payload = {"username": "staff_member", "password": "securepassword123"}
        response = self.client.post(url, payload, format="json")
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        self.assertTrue(response.data.get("success"))
        self.assertEqual(response.data.get("username"), "staff_member")
        self.assertTrue(response.data.get("is_staff"))

    def test_invalid_credentials(self):
        url = reverse("staff-login")
        payload = {"username": "staff_member", "password": "wrongpassword"}
        response = self.client.post(url, payload, format="json")
        self.assertEqual(response.status_code, status.HTTP_401_UNAUTHORIZED)
        self.assertFalse(response.data.get("success"))

    def test_empty_credentials(self):
        url = reverse("staff-login")
        response = self.client.post(url, {}, format="json")
        self.assertEqual(response.status_code, status.HTTP_400_BAD_REQUEST)

    def test_customer_email_login(self):
        from django.contrib.auth import get_user_model
        User = get_user_model()
        User.objects.create_user(
            username="aquaflow_client",
            email="manager@aquaflow.com",
            password="clientpass123",
            first_name="AquaFlow Factory",
        )
        url = reverse("user-login")
        # Login using email instead of username
        payload = {"username": "manager@aquaflow.com", "password": "clientpass123"}
        response = self.client.post(url, payload, format="json")
        self.assertEqual(response.status_code, status.HTTP_200_OK)
        self.assertTrue(response.data.get("success"))
        self.assertEqual(response.data.get("username"), "aquaflow_client")
        self.assertFalse(response.data.get("is_staff"))


class UserRegisterAPITest(APITestCase):
    def test_successful_registration(self):
        url = reverse("user-register")
        payload = {
            "name": "Ibrahim Bello",
            "company_name": "Crystal Waters Ltd",
            "email": "ibrahim@crystalwaters.com",
            "password": "strongpassword123",
        }
        response = self.client.post(url, payload, format="json")
        self.assertEqual(response.status_code, status.HTTP_201_CREATED)
        self.assertTrue(response.data.get("success"))
        self.assertEqual(response.data.get("email"), "ibrahim@crystalwaters.com")

    def test_duplicate_email_registration(self):
        from django.contrib.auth import get_user_model
        User = get_user_model()
        User.objects.create_user(
            username="existinguser",
            email="existing@example.com",
            password="password123",
        )
        url = reverse("user-register")
        payload = {
            "name": "New User",
            "email": "existing@example.com",
            "password": "password123",
        }
        response = self.client.post(url, payload, format="json")
        self.assertEqual(response.status_code, status.HTTP_400_BAD_REQUEST)
        self.assertFalse(response.data.get("success"))

    def test_short_password_registration(self):
        url = reverse("user-register")
        payload = {
            "name": "Short Pass User",
            "email": "short@example.com",
            "password": "123",
        }
        response = self.client.post(url, payload, format="json")
        self.assertEqual(response.status_code, status.HTTP_400_BAD_REQUEST)


class OrderAndInquiryFilterAPITest(APITestCase):
    def test_order_email_filter(self):
        Order.objects.create(
            order_number="ORD-FILTER-1",
            customer_name="Client A",
            customer_email="clientA@test.com",
            customer_phone="123",
            shipping_address="Address A",
            total_amount=Decimal("100.00"),
        )
        Order.objects.create(
            order_number="ORD-FILTER-2",
            customer_name="Client B",
            customer_email="clientB@test.com",
            customer_phone="456",
            shipping_address="Address B",
            total_amount=Decimal("200.00"),
        )

        url = reverse("order-list-create")
        res_all = self.client.get(url)
        self.assertEqual(len(res_all.data), 2)

        res_filtered = self.client.get(f"{url}?email=clientA@test.com")
        self.assertEqual(len(res_filtered.data), 1)
        self.assertEqual(res_filtered.data[0]["order_number"], "ORD-FILTER-1")



