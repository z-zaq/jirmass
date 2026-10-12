from decimal import Decimal
from django.core.management.base import BaseCommand
from products.models import Product


class Command(BaseCommand):
    help = "Seeds the database with the four specific water packaging products: Plain Roll, Printed Film, Printed Packing Bag, and Shrink Wrap / SSB & MSB."

    def handle(self, *args, **options):
        products_data = [
            {
                "name": "Plain Roll",
                "sku": "WTR-ROLL-PLAIN-320",
                "description": "Standard 320mm virgin LDPE unprinted machine roll film calibrated for form-fill-seal automated sachet packaging machines with uniform 60-75 micron gauge thickness and leak-proof sealing.",
                "price": Decimal("0.00"),
                "stock_quantity": 500,
                "is_active": True,
            },
            {
                "name": "Printed Film",
                "sku": "WTR-FILM-PRINTED-320",
                "description": "High-definition rotogravure multi-color custom branded pure water packaging roll film with factory logo, NAFDAC regulatory license marks, and photo-eye registration marks.",
                "price": Decimal("0.00"),
                "stock_quantity": 500,
                "is_active": True,
            },
            {
                "name": "Printed Packing Bag",
                "sku": "WTR-BAG-PRINTED-20S",
                "description": "Heavy-duty custom printed outer bagging sacks engineered to hold and transport 20 sachets of pure water with high puncture resistance and reinforced tear-proof bottom seals.",
                "price": Decimal("0.00"),
                "stock_quantity": 1000,
                "is_active": True,
            },
            {
                "name": "Shrink Wrap / SSB & MSB",
                "sku": "WTR-SHRINK-SSB-MSB",
                "description": "Industrial thermal contraction shrink film calibrated for Single Sachet Bag (SSB), Multiple Sachet Bag (MSB), and bottled water multipack bundling with high puncture resilience.",
                "price": Decimal("0.00"),
                "stock_quantity": 500,
                "is_active": True,
            },
        ]

        created_count = 0
        updated_count = 0

        for p_data in products_data:
            sku = p_data.pop("sku")
            obj, created = Product.objects.update_or_create(sku=sku, defaults=p_data)
            if created:
                created_count += 1
                self.stdout.write(self.style.SUCCESS(f"Created product: {obj.name} ({sku})"))
            else:
                updated_count += 1
                self.stdout.write(self.style.WARNING(f"Updated product: {obj.name} ({sku})"))

        self.stdout.write(
            self.style.SUCCESS(
                f"Successfully seeded water products: {created_count} created, {updated_count} updated."
            )
        )
