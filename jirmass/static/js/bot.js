// Jirmass Plastics - Automated Customer Support Bot
document.addEventListener("DOMContentLoaded", () => {
    const chatToggle = document.getElementById("chat-toggle-btn");
    const chatWindow = document.getElementById("chat-window");
    const chatClose = document.getElementById("chat-close-btn");
    const chatForm = document.getElementById("chat-form");
    const chatInput = document.getElementById("chat-input");
    const chatMessages = document.getElementById("chat-messages");

    if (!chatToggle || !chatWindow) return;

    let hasGreeted = false;

    // Toggle chat visibility
    function toggleChat(open) {
        if (open === undefined) {
            chatWindow.classList.toggle("hidden");
        } else if (open) {
            chatWindow.classList.remove("hidden");
        } else {
            chatWindow.classList.add("hidden");
        }

        if (!chatWindow.classList.contains("hidden")) {
            if (!hasGreeted) {
                renderWelcomeMessage();
                hasGreeted = true;
            }
            chatInput && chatInput.focus();
        }
    }

    chatToggle.addEventListener("click", () => toggleChat());
    if (chatClose) {
        chatClose.addEventListener("click", () => toggleChat(false));
    }

    // Expose toggleChat globally for "Chat with Assistant" CTA buttons
    window.openJirmassChat = function(prefill) {
        toggleChat(true);
        if (prefill && chatInput) {
            chatInput.value = prefill;
            chatInput.focus();
        }
    };

    // Render Initial Bot Greeting
    function renderWelcomeMessage() {
        appendMessage(
            "bot",
            "👋 Welcome to Jirmass Plastics!\n\nOur administrative team is currently OFFLINE / AWAY. I am Jirmass Bot, your 24/7 automated assistant.\n\nHow can I assist your pure water factory today? You can ask about Plain Roll, Printed Film, Printed Packing Bags, Shrink Wrap / SSB & MSB, place a direct factory order, or request a quick callback."
        );
    }

    // Append Message to UI
    function appendMessage(sender, text) {
        const msgDiv = document.createElement("div");
        msgDiv.className = `msg msg-${sender}`;

        if (sender === "bot") {
            const tag = document.createElement("div");
            tag.className = "bot-tag";
            tag.textContent = "🤖 Jirmass Bot (Auto-Reply)";
            msgDiv.appendChild(tag);
        }

        const body = document.createElement("div");
        body.textContent = text;
        msgDiv.appendChild(body);

        chatMessages.appendChild(msgDiv);
        chatMessages.scrollTop = chatMessages.scrollHeight;
    }

    // Handle user submission
    if (chatForm && chatInput) {
        chatForm.addEventListener("submit", async (e) => {
            e.preventDefault();
            const message = chatInput.value.trim();
            if (!message) return;

            appendMessage("user", message);
            chatInput.value = "";

            // Show typing indicator
            const typingDiv = document.createElement("div");
            typingDiv.className = "msg msg-bot";
            typingDiv.id = "chat-typing";
            typingDiv.innerHTML = "<div class='bot-tag'>🤖 Jirmass Bot</div><em>Typing automated reply...</em>";
            chatMessages.appendChild(typingDiv);
            chatMessages.scrollTop = chatMessages.scrollHeight;

            try {
                const response = await fetch("/api/bot/chat", {
                    method: "POST",
                    headers: {
                        "Content-Type": "application/json",
                    },
                    body: JSON.stringify({ message: message }),
                });

                const typing = document.getElementById("chat-typing");
                if (typing) typing.remove();

                if (response.ok) {
                    const data = await response.json();
                    appendMessage("bot", data.reply || "Thank you. Your inquiry has been received.");
                } else {
                    appendMessage(
                        "bot",
                        "I am currently operating in offline mode. For immediate quote requests, please use our Contact page or leave your email."
                    );
                }
            } catch (err) {
                const typing = document.getElementById("chat-typing");
                if (typing) typing.remove();
                appendMessage(
                    "bot",
                    "We have registered your query. Please leave your email or check back shortly!"
                );
            }
        });
    }

    // Quick suggestion chip helper
    window.sendChatChip = function(query) {
        toggleChat(true);
        if (chatInput) {
            chatInput.value = query;
            chatForm.dispatchEvent(new Event("submit"));
        }
    };
});
