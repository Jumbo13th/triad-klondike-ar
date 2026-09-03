(function () {
  const root = document.documentElement;
  const previewPage = new URLSearchParams(window.location.search).get("preview");
  const pages = Array.from(document.querySelectorAll(".page"));

  function assignPageReferences() {
    const errors = [];

    pages.forEach((page, index) => {
      const number = String(index + 1);
      page.dataset.page = number;

      const footer = page.querySelector(":scope > .footer");
      if (footer) {
        const marker = footer.querySelector("[data-page-number]") || footer.lastElementChild;
        if (!marker) {
          errors.push(`page-${number}:missing-footer-number`);
        } else {
          marker.dataset.pageNumber = "";
          marker.textContent = number.padStart(2, "0");
        }
      }
    });

    for (const row of document.querySelectorAll(".toc li")) {
      const link = row.querySelector('a[href^="#"]');
      const marker = row.querySelector(".pg");
      if (!link || !marker) {
        errors.push("toc:malformed-row");
        continue;
      }

      const targetId = decodeURIComponent(link.getAttribute("href").slice(1));
      const target = document.getElementById(targetId);
      const targetPage = target && target.closest(".page");
      if (!targetPage) {
        errors.push(`toc:missing-${targetId || "target"}`);
        continue;
      }
      marker.textContent = String(targetPage.dataset.page).padStart(2, "0");
    }

    return errors;
  }

  const referenceErrors = assignPageReferences();

  if (previewPage) {
    for (const page of pages) {
      if (page.dataset.page !== previewPage) {
        page.style.display = "none";
      }
    }
  }

  function checkLayout() {
    const failures = [...referenceErrors];
    const tolerance = 1;
    const footerGap = 4;

    for (const page of pages) {
      if (getComputedStyle(page).display === "none") {
        continue;
      }

      const number = page.dataset.page || "?";
      const verticalOverflow = page.scrollHeight - page.clientHeight;
      const horizontalOverflow = page.scrollWidth - page.clientWidth;
      if (verticalOverflow > tolerance) {
        failures.push(`${number}:overflow-y-${Math.ceil(verticalOverflow)}`);
      }
      if (horizontalOverflow > tolerance) {
        failures.push(`${number}:overflow-x-${Math.ceil(horizontalOverflow)}`);
      }

      const pageRect = page.getBoundingClientRect();
      const footer = page.querySelector(":scope > .footer");
      const footerRect = footer ? footer.getBoundingClientRect() : null;
      const fileRef = page.querySelector(":scope > .file-ref");
      const fileRefRect = fileRef ? fileRef.getBoundingClientRect() : null;
      const flowChildren = Array.from(page.children).filter((child) => {
        if (child.matches(".edge-band, .file-ref, .footer")) {
          return false;
        }
        const style = getComputedStyle(child);
        return style.display !== "none" && style.position !== "absolute" && style.position !== "fixed";
      });

      for (const child of flowChildren) {
        const rect = child.getBoundingClientRect();
        if (rect.left < pageRect.left - tolerance || rect.right > pageRect.right + tolerance) {
          failures.push(`${number}:content-outside-page`);
          break;
        }
        if (footerRect && rect.bottom > footerRect.top - footerGap) {
          failures.push(`${number}:footer-overlap`);
          break;
        }
      }

      if (fileRefRect && flowChildren.length > 0) {
        const firstRect = flowChildren[0].getBoundingClientRect();
        if (firstRect.top < fileRefRect.bottom + footerGap) {
          failures.push(`${number}:header-overlap`);
        }
      }
      if (footerRect && footerRect.bottom > pageRect.bottom + tolerance) {
        failures.push(`${number}:footer-outside-page`);
      }
    }

    root.dataset.renderedPages = String(pages.length);
    root.dataset.layoutErrors = Array.from(new Set(failures)).join(",");
    root.dataset.layoutReady = "true";
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", checkLayout, { once: true });
  } else {
    checkLayout();
  }

  window.addEventListener("load", checkLayout, { once: true });
  if (document.fonts && document.fonts.ready) {
    document.fonts.ready.then(checkLayout);
  }
  window.setTimeout(checkLayout, 0);
})();
