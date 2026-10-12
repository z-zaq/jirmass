# Static Images & Brand Assets Directory

This folder is the designated storage location for all frontend brand assets, product images, and layout graphics.

---

## 1. Core Water Packaging Product Images

Each of the four core products has a dedicated, pre-styled space on the website calibrated to `.product-image-container` (230px height, `object-fit: cover`) in `static/css/style.css`:

| Product | Current File | Descriptive Visual Specification | Format & Size |
| :--- | :--- | :--- | :--- |
| **Plain Roll** | `static/images/plain-roll.jpg` *(also `.png`)* | **Huge bulk heap of clear film** | Web-optimized 1200×896 (~128 KB) |
| **Printed Film** | `static/images/printed-film.jpg` *(also `.png`)* | **Abstract patterned blue rolls, completely unbranded** | Web-optimized 1200×896 (~163 KB) |
| **Printed Packing Bag** | `static/images/printed-packing-bag.jpg` *(also `.png`)* | **Bags fanned-out with generic Quality/Eco illustrations only** | Web-optimized 1200×896 (~124 KB) |
| **Shrink Wrap / SSB & MSB** | `static/images/shrink-wrap-ssb-msb.jpg` *(also `.png`)* | **Bare shrink rolls alongside multicolor product bundles** | Web-optimized 1200×896 (~140 KB) |

### Frontend Display Features:
- **Responsive 230px Display**: Handled by `.product-image-container` and `.product-img` in `static/css/style.css`.
- **Visual Description Badge**: Rendered beneath the image container via `.product-visual-desc` highlighting the visual overview.
- **Graceful Fallback**: If an image path is invalid or missing, it automatically falls back to `static/images/placeholder-product.svg` without showing broken image icons.

---

## 2. Brand Identity Assets

| Filename | Purpose | Recommended Format & Dimensions |
| :--- | :--- | :--- |
| `logo.svg` *(or `logo.png`)* | Navigation header brand logo | Vector SVG or transparent PNG (approx. 320×64 px) |
| `logo-icon.svg` *(or `favicon.ico`)* | Browser tab favicon and mobile mark | Vector SVG or 32×32 / 64×64 PNG/ICO |
| `hero-bg.svg` *(or `hero-bg.jpg`)* | Homepage banner background graphic | High-res SVG, JPG, or PNG (approx. 1440×600 px) |
| `pattern-bg.svg` *(or `pattern-bg.png`)* | Subtle page background pattern | Repeating tile pattern (optional) |
| `placeholder-product.svg` | Fallback card graphic for missing product images | Vector SVG (1200×896) |

---

## 3. Upload Via Django Admin
Log in to Django Admin at `http://localhost:8000/admin/`:
- **Products Dashboard**: Upload direct product photos under **Products**.
- **Brand Customization**: Upload company logo, hero banner, and favicon under **Brand Graphic & Asset Customizations**.
