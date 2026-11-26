// Client for the japan web session. Every keystroke (real or button-driven)
// is sent to the server as one key event, exactly like a terminal would
// deliver it; the server replies with the rendered "screen" text, which this
// script pretty-prints line by line.
(function () {
  "use strict";

  var display = document.getElementById("display");
  var screenEl = document.getElementById("screen");
  var status = document.getElementById("status");
  var line = document.getElementById("line");
  var viewTitle = document.getElementById("view-title");
  var keystrokes = document.getElementById("keystrokes");
  var stageCard = document.querySelector(".stage-card");
  var ws = null;
  var reconnectDelay = 1000;

  // ---------- helpers ----------

  function setChip(cls, text) {
    status.className = "chip " + cls;
    status.querySelector(".chip-text").textContent = text;
  }

  function pushKeystroke(label) {
    if (!label) return;
    var el = document.createElement("span");
    el.className = "ks";
    el.textContent = label;
    keystrokes.appendChild(el);
    while (keystrokes.childNodes.length > 5) {
      keystrokes.removeChild(keystrokes.firstChild);
    }
    setTimeout(function () {
      if (el.parentNode === keystrokes) {
        el.style.transition = "opacity .4s";
        el.style.opacity = "0";
        setTimeout(function () {
          if (el.parentNode === keystrokes) keystrokes.removeChild(el);
        }, 420);
      }
    }, 1600);
  }

  function flashStage() {
    stageCard.classList.remove("flash");
    // restart the animation
    void stageCard.offsetWidth;
    stageCard.classList.add("flash");
  }

  // ---------- sending keys ----------

  function sendKey(code, rune, label) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "key", code: code || "", rune: rune || "" }));
      pushKeystroke(label || rune || code);
      flashStage();
    }
  }

  // ---------- pretty rendering ----------

  var HINTS = [
    /^Type the reading/,
    /^Press /,
    /^You didn't know/,
    /^You already know/,
    /^All phrases learned/,
    /^This will delete/,
    /^Choose a section/
  ];

  function classify(text, prev) {
    if (/^>\s*$/.test(text)) return "ln ln-prompt";
    if (/^Correct!/.test(text)) return "ln ln-correct";
    if (/^Wrong\./.test(text)) return "ln ln-wrong";
    if (/^Statistics \(/.test(text)) return "ln ln-stat";
    if (/^(Known|Not known)[^\n]*:\s*$/.test(text)) return "ln ln-section";
    for (var i = 0; i < HINTS.length; i++) {
      if (HINTS[i].test(text)) return "ln ln-hint";
    }
    if (/^[A-Za-z\d\s"'.,:;!?()-]+$/.test(text) && /[A-Za-z]/.test(text)) {
      return "ln ln-hint";
    }
    if (prev && /^[A-Za-z][A-Za-z\d\s"'.,:;!?()-]*$/.test(prev) && /\s-\s/.test(text)) {
      return "ln ln-correct";
    }
    if (/[぀-ゟ゠-ヿ一-鿿]/.test(text)) {
      var bare = text.trim();
      if (bare.length > 0 && bare.length <= 8 && !/[:()\[\]]/.test(bare)) {
        return "ln ln-kana";
      }
      return "ln ln-option";
    }
    return "ln ln-option";
  }

  function renderScreen(text) {
    var lines = String(text).split("\n");
    display.textContent = "";
    var prev = null;
    for (var i = 0; i < lines.length; i++) {
      var raw = lines[i];
      var span = document.createElement("span");
      if (raw.trim() === "") {
        span.className = "ln";
        span.innerHTML = "&nbsp;";
      } else {
        span.className = classify(raw, prev);
        span.textContent = highlightOption(raw);
      }
      display.appendChild(span);
      prev = raw;
    }
    updateTitle(lines);
    scrollToEnd();
  }

  // Wrap leading "N:" menu numbers in a styled badge (via plain text node
  // replacement using innerHTML on a safe, escaped fragment).
  function highlightOption(raw) {
    return raw; // classification alone keeps things simple and XSS-safe
  }

  function styleOptionNumbers() {
    var opts = display.querySelectorAll(".ln-option");
    Array.prototype.forEach.call(opts, function (el) {
      var m = el.textContent.match(/^(\d):?\s(.*)$/);
      if (!m) return;
      el.textContent = "";
      var b = document.createElement("b");
      b.textContent = m[1] + ":";
      el.appendChild(b);
      el.appendChild(document.createTextNode(" " + m[2]));
    });
  }

  function updateTitle(lines) {
    var title = "Главное меню";
    var joined = lines.join("\n");
    if (/Learning phrases|I can translate/i.test(joined)) title = "Фразы";
    else if (/hiragana/i.test(joined) && /reading|statistics/i.test(joined)) title = "Хирагана";
    else if (/katakana/i.test(joined) && /reading|statistics/i.test(joined)) title = "Катакана";
    else if (/delete your saved/i.test(joined)) title = "Сброс прогресса";
    else if (/Choose a section/.test(joined)) title = "Главное меню";
    viewTitle.textContent = title;
  }

  function scrollToEnd() {
    requestAnimationFrame(function () {
      screenEl.scrollTop = screenEl.scrollHeight;
    });
  }

  // ---------- websocket ----------

  function connect() {
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(proto + "//" + location.host + "/ws");

    ws.onopen = function () {
      setChip("chip-ok", "подключено");
      reconnectDelay = 1000;
    };

    ws.onmessage = function (ev) {
      var msg;
      try { msg = JSON.parse(ev.data); } catch (e) { return; }
      if (msg.type === "render") {
        renderScreen(msg.html);
        styleOptionNumbers();
      } else if (msg.type === "closed") {
        setChip("chip-bad", "сессия завершена");
      }
    };

    ws.onclose = function () {
      setChip("chip-bad", "переподключение…");
      setTimeout(connect, reconnectDelay);
      reconnectDelay = Math.min(reconnectDelay * 2, 15000);
    };
  }

  // ---------- keyboard ----------

  document.addEventListener("keydown", function (ev) {
    if (ev.target === line) {
      if (ev.key === "Enter") {
        ev.preventDefault();
        submitLine();
      } else if (ev.key === "Escape") {
        ev.preventDefault();
        sendKey("Escape", "", "Esc");
        line.value = "";
      }
      return; // Backspace and typing stay native in the input box
    }
    if (ev.ctrlKey || ev.metaKey || ev.altKey) return;
    switch (ev.key) {
      case "Enter": ev.preventDefault(); sendKey("Enter", "", "⏎"); break;
      case "Escape": ev.preventDefault(); sendKey("Escape", "", "Esc"); break;
      case "Backspace": ev.preventDefault(); sendKey("Backspace", "", "⌫"); break;
      case "Tab": ev.preventDefault(); sendKey("Tab", "", "Tab"); break;
      default:
        if (ev.key.length === 1) {
          ev.preventDefault();
          sendKey("", ev.key, ev.key);
        }
    }
  });

  function submitLine() {
    var text = line.value;
    line.value = "";
    for (var i = 0; i < text.length; i++) {
      sendKey("", text[i], null);
    }
    sendKey("Enter", "", "⏎");
  }

  // Menu / action buttons send the corresponding digit.
  Array.prototype.forEach.call(document.querySelectorAll("[data-key]"), function (btn) {
    btn.addEventListener("click", function () {
      var k = btn.getAttribute("data-key");
      sendKey("", k, k);
      line.focus();
    });
  });

  function bind(id, fn) {
    var el = document.getElementById(id);
    if (el) el.addEventListener("click", fn);
  }
  bind("enterBtn", function () {
    if (line.value !== "") { submitLine(); } else { sendKey("Enter", "", "⏎"); }
    line.focus();
  });
  bind("escBtn", function () { sendKey("Escape", "", "Esc"); });
  bind("bsBtn", function () { sendKey("Backspace", "", "⌫"); });

  connect();
})();
