
document.addEventListener("DOMContentLoaded", () => {
    const copyBtn = document.getElementById("copy-btn");
    const shortUrlEl = document.getElementById("short-url");
    const toast = document.getElementById("toast");

    if (copyBtn && shortUrlEl) {
        copyBtn.addEventListener("click", () => {
            const shortUrl = shortUrlEl.textContent.trim();

            navigator.clipboard.writeText(shortUrl)
                .then(() => {
                    toast.classList.remove("translate-y-10", "opacity-0")
                    toast.classList.add("translate-y-0", "opacity-100")

                    setTimeout(() => {
                        toast.classList.remove("translate-y-0", "opacity-100")
                        toast.classList.add("translate-y-10", "opacity-0")
                    }, 3000)
                })

                .catch(err => {
                    console.error("Failed to copy: ", err);
                });
        });
    }
});
