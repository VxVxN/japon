// Japan learner — graphical web client.
//
// The server owns the application state and pushes JSON view models over the
// websocket; this script renders them as real UI (menu cards, quiz screens,
// statistics tables) and sends semantic actions back. No terminal emulation.
(function () {
  "use strict";

  var screenEl = document.getElementById("screen");
  var status = document.getElementById("status");
  var viewTitle = document.getElementById("view-title");
  var backBtn = document.getElementById("backBtn");
  var quitBtn = document.getElementById("quitBtn");
  var progressEl = document.getElementById("progress");

  var ws = null;
  var reconnectDelay = 1000;
  var sessionEnded = false;

  // ---------- helpers ----------

  function el(tag, cls, text) {
    var n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text !== undefined && text !== null) n.textContent = text;
    return n;
  }

  function icon(name, cls) {
    var svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    svg.setAttribute("class", "icon" + (cls ? " " + cls : ""));
    var use = document.createElementNS("http://www.w3.org/2000/svg", "use");
    use.setAttribute("href", "#i-" + name);
    svg.appendChild(use);
    return svg;
  }

  function button(label, iconName, cls, onClick) {
    var b = el("button", "btn" + (cls ? " " + cls : ""));
    b.type = "button";
    if (iconName) b.appendChild(icon(iconName));
    b.appendChild(el("span", null, label));
    b.addEventListener("click", onClick);
    return b;
  }

  function setChip(cls, text) {
    status.className = "chip " + cls;
    status.querySelector(".chip-text").textContent = text;
  }

  function sendAction(action, value, raw) {
    if (!ws || ws.readyState !== WebSocket.OPEN || sessionEnded) return;
    var msg = { type: "action", action: action };
    if (value !== undefined && value !== null) msg.value = String(value);
    if (raw !== undefined) msg.raw = raw;
    ws.send(JSON.stringify(msg));
  }

  // ---------- rendering ----------

  function render(state) {
    screenEl.textContent = "";
    var root = el("div", "view-in");
    screenEl.appendChild(root);
    viewTitle.textContent = state.title || "Japan learner";
    backBtn.hidden = !(state.view === "quiz" || state.view === "stats" || state.view === "notice");
    renderProgress(state.progress);

    switch (state.view) {
      case "menu":   renderMenu(root, state); break;
      case "quiz":   renderQuiz(root, state.quiz); break;
      case "stats":  renderStats(root, state.stats); break;
      case "done":   renderDone(root, state); break;
      case "notice": renderNotice(root, state.notice); break;
      default:       root.appendChild(el("p", "loading", "Неизвестный экран."));
    }
  }

  function renderProgress(p) {
    progressEl.textContent = "";
    if (!p) { progressEl.appendChild(el("div", "stats-empty", "нет данных")); return; }
    var rows = [
      ["Хирагана", p.hiraganaKnown, p.hiraganaTotal, ""],
      ["Катакана", p.katakanaKnown, p.katakanaTotal, "p-violet"],
      ["Фразы", p.phrasesKnown, p.phrasesTotal, "p-blue"]
    ];
    rows.forEach(function (r) {
      var row = el("div", "prog-row");
      var top = el("div", "prog-top");
      top.appendChild(el("span", null, r[0]));
      top.appendChild(el("span", "prog-num", r[1] + " / " + r[2]));
      var bar = el("div", "prog-bar");
      var fill = el("div", "prog-fill" + (r[3] ? " " + r[3] : ""));
      fill.style.width = (r[2] > 0 ? Math.round(100 * r[1] / r[2]) : 0) + "%";
      bar.appendChild(fill);
      row.appendChild(top);
      row.appendChild(bar);
      progressEl.appendChild(row);
    });
  }

  function renderMenu(root, state) {
    if (state.subtitle) root.appendChild(el("p", "menu-subtitle", state.subtitle));
    var grid = el("div", "menu-grid");
    (state.menu || []).forEach(function (item) {
      var card = el("button", "menu-card" + (item.style === "danger" ? " danger" : ""));
      card.type = "button";
      var ic = el("div", "mc-icon");
      ic.appendChild(icon(item.icon || "book", "ic-" + item.icon));
      card.appendChild(ic);
      card.appendChild(el("div", "mc-label", item.label));
      card.addEventListener("click", function () { sendAction("select", item.id); });
      grid.appendChild(card);
    });
    root.appendChild(grid);
  }

  function renderQuiz(root, quiz) {
    if (!quiz) return;
    var wrap = el("div", "quiz-wrap");
    root.appendChild(wrap);

    if (quiz.retry) wrap.appendChild(el("div", "badge-retry", "Этот материал был сложным — попробуйте ещё раз"));

    if (quiz.kind === "kana") {
      var hero = el("div", "kana-hero");
      hero.appendChild(el("div", "kana", quiz.prompt));
      wrap.appendChild(hero);
    } else {
      wrap.appendChild(el("div", "phrase-card", quiz.prompt));
    }

    if (quiz.translation) {
      var tr = el("div", "translation");
      tr.appendChild(el("b", null, quiz.kind === "kana" ? "Чтение: " : "Перевод: "));
      tr.appendChild(document.createTextNode(quiz.translation));
      wrap.appendChild(tr);
    }

    if (quiz.input) {
      if (quiz.kind === "kana") {
        var form = el("form", "answer-form");
        var input = el("input");
        input.type = "text";
        input.autocomplete = "off";
        input.spellcheck = false;
        input.placeholder = "Введите чтение ромадзи…";
        var submit = button("", "home", "btn-primary icon-btn");
        submit.type = "submit";
        submit.title = "Отправить ответ";
        submit.replaceChildren(icon("home"));
        form.appendChild(input);
        form.appendChild(submit);
        form.addEventListener("submit", function (ev) {
          ev.preventDefault();
          sendAction("answer", input.value);
          input.value = "";
          input.focus();
        });
        wrap.appendChild(form);
        setTimeout(function () { input.focus(); }, 30);
      } else {
        var acts = el("div", "phrase-actions");
        acts.appendChild(paBtn("Могу перевести", "check", "ok", function () { sendAction("answer", "know"); }));
        acts.appendChild(paBtn("Показать перевод", "eye", "info", function () { sendAction("answer", "show"); }));
        acts.appendChild(paBtn("Не могу перевести", "x", "warn", function () { sendAction("answer", "unknown"); }));
        acts.appendChild(paBtn("В главное меню", "home", "subtle", function () { sendAction("answer", "menu"); }));
        wrap.appendChild(acts);
      }
    }

    if (quiz.hint) {
      var isFb = /^(Верно|Неверно|Пустой)/.test(quiz.hint);
      if (isFb) {
        var fb = el("div", "feedback " + (/^Верно/.test(quiz.hint) ? "ok" : "bad"));
        fb.appendChild(el("div", "fb-title", quiz.hint));
        if (quiz.translation) {
          var line = el("div", "fb-answer");
          line.appendChild(el("b", null, quiz.prompt));
          line.appendChild(document.createTextNode(" — " + quiz.translation));
          fb.appendChild(line);
        }
        // After a correct/incorrect kana answer the session waits for another
        // "answer" action to move on: give the user an explicit continue button.
        var contRow = el("div", "feedback-actions");
        contRow.appendChild(button("Продолжить", "check", "btn-primary", function () {
          sendAction("answer", "continue");
        }));
        fb.appendChild(contRow);
        wrap.appendChild(fb);
        setTimeout(function () {
          var b = contRow.querySelector("button");
          if (b) b.focus();
        }, 30);
      } else {
        wrap.appendChild(el("p", "quiz-hint", quiz.hint));
      }
    }
  }

  function paBtn(label, iconName, cls, onClick) {
    var b = el("button", "btn pa-btn " + cls);
    b.type = "button";
    b.appendChild(icon(iconName));
    b.appendChild(el("span", "pa-label", label));
    b.addEventListener("click", onClick);
    return b;
  }

  function renderStats(root, stats) {
    if (!stats) return;
    var score = el("div", "stats-score");
    score.appendChild(el("span", "big", String(stats.correct)));
    score.appendChild(el("span", "total", "из " + stats.total + " верно"));
    root.appendChild(score);

    root.appendChild(statsSection("Выучено", stats.known, false));
    root.appendChild(statsSection("Осталось", stats.rest, true));

    var acts = el("div", "stats-actions");
    acts.appendChild(button("В главное меню", "home", "btn-primary", function () { sendAction("select", "main"); }));
    root.appendChild(acts);
  }

  function statsSection(title, items, dim) {
    var sec = el("div", "stats-section");
    sec.appendChild(el("h3", null, title));
    if (!items || items.length === 0) {
      sec.appendChild(el("div", "stats-empty", dim ? "Все знаки выучены!" : "Пока ничего не выучено."));
      return sec;
    }
    var grid = el("div", "stats-grid");
    items.forEach(function (it) {
      var cell = el("div", "stat-cell" + (dim ? " dim" : ""));
      cell.appendChild(el("span", "sym", it.symbol));
      cell.appendChild(el("span", "read", it.reading));
      grid.appendChild(cell);
    });
    sec.appendChild(grid);
    return sec;
  }

  function renderDone(root, state) {
    var card = el("div", "center-card");
    var bi = el("div", "big-icon");
    bi.appendChild(icon("trophy"));
    card.appendChild(bi);
    card.appendChild(el("h3", null, state.title || "Готово!"));
    if (state.done) card.appendChild(el("p", null, state.done.message));
    var acts = el("div", "center-actions");
    if (!sessionEnded) {
      acts.appendChild(button("В главное меню", "home", "btn-primary", function () { sendAction("select", "main"); }));
    }
    card.appendChild(acts);
    root.appendChild(card);
  }

  function renderNotice(root, notice) {
    if (!notice) return;
    var card = el("div", "center-card danger");
    var bi = el("div", "big-icon");
    bi.appendChild(icon(notice.confirm ? "reset" : "x"));
    card.appendChild(bi);
    card.appendChild(el("h3", null, "Подтвердите действие"));
    card.appendChild(el("p", null, notice.message));
    var acts = el("div", "center-actions");
    if (notice.confirm) {
      acts.appendChild(button("Да, сбросить", "check", "btn-primary", function () { sendAction("confirm", null, true); }));
      acts.appendChild(button("Отмена", "home", "", function () { sendAction("confirm", null, false); }));
    } else {
      acts.appendChild(button("В главное меню", "home", "btn-primary", function () { sendAction("select", "main"); }));
    }
    card.appendChild(acts);
    root.appendChild(card);
  }

  function renderClosed() {
    sessionEnded = true;
    screenEl.textContent = "";
    var card = el("div", "center-card");
    var bi = el("div", "big-icon");
    bi.appendChild(icon("power"));
    card.appendChild(bi);
    card.appendChild(el("h3", null, "Сессия завершена"));
    card.appendChild(el("p", null, "Обновите страницу, чтобы начать заново."));
    screenEl.appendChild(card);
    backBtn.hidden = true;
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
      if (msg.type === "state") {
        render(msg.state);
      } else if (msg.type === "closed") {
        renderClosed();
      }
    };

    ws.onclose = function () {
      if (sessionEnded) {
        setChip("chip-bad", "сессия завершена");
        return;
      }
      setChip("chip-bad", "переподключение…");
      setTimeout(connect, reconnectDelay);
      reconnectDelay = Math.min(reconnectDelay * 2, 15000);
    };
  }

  backBtn.addEventListener("click", function () { sendAction("select", "main"); });
  quitBtn.addEventListener("click", function () { sendAction("select", "quit"); });

  connect();
})();
