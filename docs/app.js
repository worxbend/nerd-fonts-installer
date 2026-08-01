// nerd-fonts-installer microsite behaviour: theme toggle, copy button,
// screenshot tabs, and scroll reveals. No dependencies, no build step.

(function () {
  "use strict";

  /* ---------- theme ---------- */

  var root = document.documentElement;
  var toggle = document.getElementById("theme-toggle");

  if (toggle) {
    toggle.addEventListener("click", function () {
      var next = root.dataset.theme === "light" ? "dark" : "light";
      root.dataset.theme = next;
      try {
        localStorage.setItem("nfi-theme", next);
      } catch (e) {
        /* private mode: the choice just does not persist */
      }
    });
  }

  /* ---------- copy buttons ---------- */

  document.querySelectorAll(".copy-btn").forEach(function (btn) {
    btn.addEventListener("click", function () {
      var source = btn.parentElement.querySelector("[data-copy]");
      if (!source) return;

      var text = source.dataset.copy;
      var done = function () {
        btn.classList.add("copied");
        setTimeout(function () {
          btn.classList.remove("copied");
        }, 1600);
      };

      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(done, fallback);
      } else {
        fallback();
      }

      function fallback() {
        var field = document.createElement("textarea");
        field.value = text;
        field.setAttribute("readonly", "");
        field.style.position = "fixed";
        field.style.opacity = "0";
        document.body.appendChild(field);
        field.select();
        try {
          document.execCommand("copy");
          done();
        } catch (e) {
          /* nothing sensible left to try */
        }
        document.body.removeChild(field);
      }
    });
  });

  /* ---------- screenshot tabs ---------- */

  var tabs = Array.prototype.slice.call(document.querySelectorAll(".tab"));
  var caption = document.getElementById("shot-caption");
  var captions = {
    "tab-releases": "Step one: choose which Nerd Fonts release to browse.",
    "tab-families": "Step two: filter and tick families — the plan updates as you go.",
    "tab-install": "Families download, verify, and extract concurrently.",
    "tab-dryrun": "<code>--dry-run</code> shows every URL and destination before anything is written.",
    "tab-names": "<code>--font-names</code> prints YAML for the release you pinned — pipe it, grep it, or save it."
  };

  function select(tab) {
    tabs.forEach(function (other) {
      var panel = document.getElementById(other.getAttribute("aria-controls"));
      var active = other === tab;
      other.setAttribute("aria-selected", active ? "true" : "false");
      if (panel) panel.hidden = !active;
    });
    if (caption && captions[tab.id]) caption.innerHTML = captions[tab.id];
  }

  tabs.forEach(function (tab, index) {
    tab.addEventListener("click", function () {
      select(tab);
    });
    tab.addEventListener("keydown", function (event) {
      var delta = event.key === "ArrowRight" ? 1 : event.key === "ArrowLeft" ? -1 : 0;
      if (!delta) return;
      event.preventDefault();
      var next = tabs[(index + delta + tabs.length) % tabs.length];
      next.focus();
      select(next);
    });
  });

  if (tabs.length) select(tabs[0]);

  /* ---------- scroll reveal ---------- */

  // The head script decides whether reveals are on; if it opted out, the
  // stylesheet has already left every section visible.
  if (!root.classList.contains("js-reveal")) return;

  var targets = document.querySelectorAll(".reveal");
  var observer = new IntersectionObserver(
    function (entries) {
      entries.forEach(function (entry) {
        if (!entry.isIntersecting) return;
        entry.target.classList.add("in");
        observer.unobserve(entry.target);
      });
    },
    { rootMargin: "0px 0px -8% 0px", threshold: 0.06 }
  );

  targets.forEach(function (el, index) {
    // A small stagger so grids animate in as a wave rather than all at once.
    el.style.transitionDelay = (index % 6) * 55 + "ms";
    observer.observe(el);
  });
})();
