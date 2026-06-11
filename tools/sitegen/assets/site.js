// GoWebComponents docs site behaviors: command-K search, gallery filters,
// and copy buttons. Doc pages stay static HTML; this is the only JS they load.
(function () {
  "use strict";

  // ----- search (Cmd/Ctrl-K) -----
  var searchOverlay = document.getElementById("search-overlay");
  var searchInput = document.getElementById("search-input");
  var searchResults = document.getElementById("search-results");
  var searchIndex = null;
  var selectedIndex = 0;

  function rootPrefix() {
    return document.body.getAttribute("data-root") || ".";
  }

  function openSearch() {
    if (!searchOverlay) return;
    searchOverlay.setAttribute("data-open", "true");
    searchInput.value = "";
    renderResults([]);
    searchInput.focus();
    if (searchIndex === null) {
      fetch(rootPrefix() + "/site/search-index.json")
        .then(function (resp) { return resp.json(); })
        .then(function (data) { searchIndex = data; })
        .catch(function () { searchIndex = []; });
    }
  }

  function closeSearch() {
    if (searchOverlay) searchOverlay.removeAttribute("data-open");
  }

  function scoreEntry(entry, terms) {
    var haystack = (entry.title + " " + (entry.tags || "") + " " + (entry.snippet || "")).toLowerCase();
    var titleLower = entry.title.toLowerCase();
    var score = 0;
    for (var i = 0; i < terms.length; i++) {
      var term = terms[i];
      if (haystack.indexOf(term) === -1) return 0;
      score += titleLower.indexOf(term) !== -1 ? 3 : 1;
      if (titleLower.indexOf(term) === 0) score += 2;
    }
    return score;
  }

  function renderResults(entries) {
    if (!searchResults) return;
    searchResults.innerHTML = "";
    selectedIndex = 0;
    if (entries.length === 0) {
      var empty = document.createElement("div");
      empty.className = "search-empty";
      empty.textContent = searchInput.value ? "No matches." : "Type to search docs, examples, and APIs.";
      searchResults.appendChild(empty);
      return;
    }
    entries.forEach(function (entry, i) {
      var link = document.createElement("a");
      link.className = "search-result";
      link.href = rootPrefix() + "/" + entry.href;
      if (i === 0) link.setAttribute("data-selected", "true");
      var kind = document.createElement("span");
      kind.className = "result-kind";
      kind.textContent = entry.kind;
      link.appendChild(kind);
      link.appendChild(document.createTextNode(entry.title));
      if (entry.snippet) {
        var snippet = document.createElement("span");
        snippet.className = "result-snippet";
        snippet.textContent = entry.snippet;
        link.appendChild(snippet);
      }
      searchResults.appendChild(link);
    });
  }

  function currentResults() {
    return searchResults ? Array.prototype.slice.call(searchResults.querySelectorAll(".search-result")) : [];
  }

  function moveSelection(delta) {
    var items = currentResults();
    if (items.length === 0) return;
    items[selectedIndex] && items[selectedIndex].removeAttribute("data-selected");
    selectedIndex = (selectedIndex + delta + items.length) % items.length;
    items[selectedIndex].setAttribute("data-selected", "true");
    items[selectedIndex].scrollIntoView({ block: "nearest" });
  }

  if (searchInput) {
    searchInput.addEventListener("input", function () {
      var terms = searchInput.value.toLowerCase().split(/\s+/).filter(Boolean);
      if (terms.length === 0 || !searchIndex) { renderResults([]); return; }
      var matches = searchIndex
        .map(function (entry) { return { entry: entry, score: scoreEntry(entry, terms) }; })
        .filter(function (item) { return item.score > 0; })
        .sort(function (a, b) { return b.score - a.score; })
        .slice(0, 12)
        .map(function (item) { return item.entry; });
      renderResults(matches);
    });
    searchInput.addEventListener("keydown", function (event) {
      if (event.key === "ArrowDown") { event.preventDefault(); moveSelection(1); }
      if (event.key === "ArrowUp") { event.preventDefault(); moveSelection(-1); }
      if (event.key === "Enter") {
        var items = currentResults();
        if (items[selectedIndex]) window.location.href = items[selectedIndex].href;
      }
    });
  }

  document.addEventListener("keydown", function (event) {
    if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === "k") {
      event.preventDefault();
      openSearch();
    }
    if (event.key === "Escape") closeSearch();
  });

  document.querySelectorAll("[data-search-open]").forEach(function (button) {
    button.addEventListener("click", openSearch);
  });
  if (searchOverlay) {
    searchOverlay.addEventListener("click", function (event) {
      if (event.target === searchOverlay) closeSearch();
    });
  }

  // ----- gallery filters -----
  var filterChips = document.querySelectorAll(".filter-chip[data-filter]");
  if (filterChips.length > 0) {
    filterChips.forEach(function (chip) {
      chip.addEventListener("click", function () {
        filterChips.forEach(function (other) { other.removeAttribute("data-active"); });
        chip.setAttribute("data-active", "true");
        var wanted = chip.getAttribute("data-filter");
        document.querySelectorAll(".catalog-card[data-module]").forEach(function (card) {
          var show = wanted === "all" || card.getAttribute("data-module") === wanted;
          card.style.display = show ? "" : "none";
        });
      });
    });
  }

  // ----- copy buttons on code blocks -----
  document.querySelectorAll(".prose pre, .demo-pane pre").forEach(function (block) {
    var button = document.createElement("button");
    button.className = "copy-button";
    button.type = "button";
    button.textContent = "Copy";
    button.addEventListener("click", function () {
      var code = block.querySelector("code");
      var text = code ? code.textContent : block.textContent;
      navigator.clipboard.writeText(text).then(function () {
        button.textContent = "Copied";
        setTimeout(function () { button.textContent = "Copy"; }, 1200);
      });
    });
    block.appendChild(button);
  });
})();
