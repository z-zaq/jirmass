// Jirmass Plastics - Direct Water Packaging Order Controller
document.addEventListener("DOMContentLoaded", () => {
    const orderForm = document.getElementById("direct-order-form");
    if (!orderForm) return;

    orderForm.addEventListener("submit", async (e) => {
        e.preventDefault();

        const submitBtn = orderForm.querySelector("button[type='submit']");
        const originalBtnText = submitBtn ? submitBtn.textContent : "Submit Order";
        if (submitBtn) {
            submitBtn.disabled = true;
            submitBtn.textContent = "Submitting Order...";
        }

        const payload = {
            product_type: document.getElementById("order-product-type").value,
            print_type: document.getElementById("order-print-type").value,
            quantity_rolls: document.getElementById("order-rolls-qty").value.trim(),
            quantity_bags: document.getElementById("order-bags-qty").value.trim(),
            company_name: document.getElementById("order-company-name").value.trim(),
            customer_name: document.getElementById("order-customer-name").value.trim(),
            phone: document.getElementById("order-phone").value.trim(),
            email: document.getElementById("order-email").value.trim(),
            delivery_address: document.getElementById("order-address").value.trim(),
            notes: document.getElementById("order-notes").value.trim(),
        };

        try {
            const res = await fetch("/api/order", {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify(payload),
            });

            if (res.ok) {
                const data = await res.json();
                orderForm.reset();

                const confirmModal = document.getElementById("order-confirmation-modal");
                if (confirmModal) {
                    const numEl = document.getElementById("conf-order-num");
                    if (numEl) numEl.textContent = data.order_number;
                    confirmModal.classList.add("active");
                    confirmModal.focus();
                } else {
                    alert(`Order #${data.order_number} submitted! Our sales team has been alerted via email and webhook.`);
                }
            } else if (res.status === 401) {
                let errData = {};
                try { errData = await res.json(); } catch(e) {}
                const redirectUrl = errData.login_url || "/login?next=/#direct-order";
                if (confirm("Authentication Required: You must sign in or create an account before placing a packaging order.\n\nClick OK to proceed to Sign In.")) {
                    window.location.href = redirectUrl;
                }
            } else {
                const errText = await res.text();
                alert("Order submission error: " + errText);
            }
        } catch (err) {
            alert("Network error submitting order: " + err);
        } finally {
            if (submitBtn) {
                submitBtn.disabled = false;
                submitBtn.textContent = originalBtnText;
            }
        }
    });

    // Close modal on Escape key or outside click
    const confirmModal = document.getElementById("order-confirmation-modal");
    if (confirmModal) {
        confirmModal.addEventListener("click", (e) => {
            if (e.target === confirmModal) {
                confirmModal.classList.remove("active");
            }
        });
        document.addEventListener("keydown", (e) => {
            if (e.key === "Escape" && confirmModal.classList.contains("active")) {
                confirmModal.classList.remove("active");
            }
        });
    }
});
