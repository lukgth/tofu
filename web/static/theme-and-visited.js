// theme-and-visited.js — the site's two client-side behaviours:
// the light/dark toggle and the italic styling of already-read post links.
// Loaded with defer; the pre-paint theme boot stays inline in base.html.
(function () {
  "use strict";

  function initThemeToggle() {
    var meta = document.querySelector('meta[name="color-scheme"]');
    var btn = document.getElementById("theme-toggle");
    if (!meta || !btn) return;

    var mq = window.matchMedia("(prefers-color-scheme: dark)");

    function current() {
      if (meta.content === "dark") return "dark";
      if (meta.content === "light") return "light";
      return mq.matches ? "dark" : "light";
    }
    function paint() {
      btn.textContent = current() === "dark" ? "\u2600" : "\u263e";
    }

    btn.addEventListener("click", function () {
      var next = current() === "dark" ? "light" : "dark";
      meta.content = next;
      try { localStorage.setItem("theme", next); } catch (e) {}
      paint();
    });
    mq.addEventListener("change", paint);
    paint();
  }

  // :visited can only restyle color properties and no JS API reads browser
  // history, so track visits ourselves and italicize read post titles.
  function initVisitedPosts() {
    var path = location.pathname;
    if (path.startsWith("/articles/") && path.endsWith(".html") && !path.startsWith("/articles/tag/")) {
      try { localStorage.setItem("v:" + path, 1); } catch (e) {}
    }
    document.querySelectorAll('ul.blog-posts a[href^="/articles/"]').forEach(function (a) {
      try { if (localStorage.getItem("v:" + a.getAttribute("href"))) a.classList.add("visited-post"); } catch (e) {}
    });
  }

  function onReady(fn) {
    if (document.readyState !== "loading") fn();
    else document.addEventListener("DOMContentLoaded", fn);
  }

  onReady(function () {
    initThemeToggle();
    initVisitedPosts();
  });
})();
