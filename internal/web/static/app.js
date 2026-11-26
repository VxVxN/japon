// Client for the japan web session. Every keystroke (real or button-driven)
// is sent to the server as one key event, exactly like a terminal would
// deliver it; the server replies with the rendered "screen" HTML.
(function () {
  "use strict";

  var display = document.getElementById("display");
  var status = document.getElementById("status");
  var line = document.getElementById("line");
  var ws = null;
  var reconnectDelay = 1000;

  function sendKey(code, rune) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({ type: "key", code: code || "", rune: rune || "" }));
    }
  }

  function connect() {
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(proto + "//" + location.host + "/ws");

    ws.onopen = function () {
      status.textContent = "подключено";
      status.className = "status ok";
      reconnectDelay = 1000;
    };

    ws.onmessage = function (ev) {
      var msg;
      try { msg = JSON.parse(ev.data); } catch (e) { return; }
      if (msg.type === "render") {
        display.innerHTML = msg.html;
        // Keep the kana question in view.
        display.scrollIntoView(false);
      } else if (msg.type === "closed") {
        status.textContent = "сессия завершена";
        status.className = "status bad";
      }
    };

    ws.onclose = function () {
      status.textContent = "переподключение…";
      status.className = "status bad";
      setTimeout(connect, reconnectDelay);
      reconnectDelay = Math.min(reconnectDelay * 2, 15000);
    };
  }

  // Real keyboard: forward everything except plain typing into the visible
  // input box (whose Enter/Backspace are handled below).
  document.addEventListener("keydown", function (ev) {
    if (ev.target === line) {
      if (ev.key === "Enter") {
        ev.preventDefault();
        submitLine();
      } else if (ev.key === "Escape") {
        ev.preventDefault();
        sendKey("Escape", "");
        line.value = "";
      }
      return; // Backspace and typing stay native in the input box
    }
    if (ev.ctrlKey || ev.metaKey || ev.altKey) return;
    switch (ev.key) {
      case "Enter": ev.preventDefault(); sendKey("Enter", ""); break;
      case "Escape": ev.preventDefault(); sendKey("Escape", ""); break;
      case "Backspace": ev.preventDefault(); sendKey("Backspace", ""); break;
      case "Tab": ev.preventDefault(); sendKey("Tab", ""); break;
      default:
        if (ev.key.length === 1) {
          ev.preventDefault();
          sendKey("", ev.key);
        }
    }
  });

  function submitLine() {
    var text = line.value;
    line.value = "";
    for (var i = 0; i < text.length; i++) {
      sendKey("", text[i]);
    }
    sendKey("Enter", "");
  }

  // Menu / action buttons send the corresponding digit.
  Array.prototype.forEach.call(document.querySelectorAll("[data-key]"), function (btn) {
    btn.addEventListener("click", function () { sendKey("", btn.getAttribute("data-key")); });
  });

  document.getElementById("enterBtn").addEventListener("click", function () {
    if (line.value !== "") { submitLine(); } else { sendKey("Enter", ""); }
  });
  document.getElementById("escBtn").addEventListener("click", function () { sendKey("Escape", ""); });
  document.getElementById("bsBtn").addEventListener("click", function () { sendKey("Backspace", ""); });

  connect();
})();
