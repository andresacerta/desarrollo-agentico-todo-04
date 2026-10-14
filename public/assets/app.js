// Pide confirmación antes de enviar formularios marcados con data-confirm.
document.addEventListener("submit", function (event) {
  var form = event.target;
  if (form.dataset.confirm && !window.confirm(form.dataset.confirm)) {
    event.preventDefault();
  }

  if (!form.matches("[data-task-toggle]")) {
    return;
  }

  event.preventDefault();

  var task = form.closest(".task");
  var completed = task.classList.toggle("task-completed");
  var button = form.querySelector("button");
  var status = task.querySelector(".task-status");

  var label = completed ? "Reabrir" : "Completar";
  button.querySelector(".toggle-label").textContent = label;
  button.setAttribute("aria-pressed", String(completed));
  button.setAttribute("aria-label", label + " tarea: " + task.querySelector(".task-title").textContent);
  button.setAttribute("title", label + " tarea");

  if (completed && !status) {
    status = document.createElement("span");
    status.className = "task-status";
    status.textContent = "Completada";
    task.insertBefore(status, task.querySelector(".task-actions"));
  } else if (!completed && status) {
    status.remove();
  }

  fetch(form.action);
});

// Reordena las tareas arrastrando el asa (ratón o táctil) o con las flechas
// cuando el asa tiene el foco, y guarda cada movimiento en el servidor.
(function () {
  var list = document.querySelector("[data-task-list]");
  if (!list) {
    return;
  }

  var token = list.dataset.reorderToken;
  var statusBox = document.querySelector("[data-reorder-status]");
  var errorBox = document.querySelector("[data-reorder-error]");
  var reduceMotion = window.matchMedia("(prefers-reduced-motion: reduce)");
  var queue = Promise.resolve();
  var drag = null;

  function items() {
    return Array.prototype.filter.call(list.children, function (el) {
      return el.classList.contains("task");
    });
  }

  function indexOf(task) {
    return items().indexOf(task);
  }

  function titleOf(task) {
    return task.querySelector(".task-title").textContent;
  }

  // Coloca la tarea en la posición index (0-based) entre las demás.
  function placeAt(task, index) {
    var others = items().filter(function (el) {
      return el !== task;
    });
    list.insertBefore(task, others[index] || null);
  }

  function tops() {
    var map = new Map();
    items().forEach(function (el) {
      map.set(el, el.getBoundingClientRect().top);
    });
    return map;
  }

  // Animación FLIP: desliza las tareas desplazadas desde su posición anterior.
  function animateFrom(before, except) {
    if (reduceMotion.matches || !Element.prototype.animate) {
      return;
    }
    items().forEach(function (el) {
      if (el === except || !before.has(el)) {
        return;
      }
      var delta = before.get(el) - el.getBoundingClientRect().top;
      if (delta) {
        el.animate(
          [{ transform: "translateY(" + delta + "px)" }, { transform: "none" }],
          { duration: 150, easing: "ease-out" }
        );
      }
    });
  }

  function moveTo(task, index) {
    var before = tops();
    placeAt(task, index);
    animateFrom(before, task);
  }

  function showError(message) {
    errorBox.textContent = message;
    errorBox.hidden = false;
  }

  function clearError() {
    errorBox.hidden = true;
    errorBox.textContent = "";
  }

  // Las peticiones van en cola para que el servidor aplique los movimientos en orden.
  function persist(task, originalIndex) {
    var position = indexOf(task) + 1;
    var total = items().length;
    var title = titleOf(task);

    queue = queue
      .then(function () {
        return fetch(task.dataset.moveUrl, {
          method: "PUT",
          credentials: "same-origin",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
            "X-CSRF-Token": token,
          },
          body: JSON.stringify({ position: position }),
        });
      })
      .then(function (response) {
        if (!response.ok) {
          throw new Error("HTTP " + response.status);
        }
        clearError();
        statusBox.textContent = title + ": posición " + position + " de " + total + ".";
      })
      .catch(function () {
        moveTo(task, originalIndex);
        showError("No se ha podido guardar el nuevo orden. Recarga la página e inténtalo de nuevo.");
      });
  }

  function endDrag(cancelled) {
    if (!drag) {
      return;
    }
    var current = drag;
    drag = null;
    current.task.classList.remove("task-dragging");
    list.classList.remove("task-list-dragging");

    if (cancelled) {
      moveTo(current.task, current.start);
      return;
    }
    if (indexOf(current.task) !== current.start) {
      persist(current.task, current.start);
    }
  }

  list.addEventListener("pointerdown", function (event) {
    var handle = event.target.closest("[data-reorder-handle]");
    if (!handle || event.button !== 0) {
      return;
    }
    event.preventDefault();
    handle.setPointerCapture(event.pointerId);
    var task = handle.closest(".task");
    drag = { task: task, start: indexOf(task) };
    task.classList.add("task-dragging");
    list.classList.add("task-list-dragging");
  });

  list.addEventListener("pointermove", function (event) {
    if (!drag) {
      return;
    }
    // Desplaza la página cuando el puntero se acerca a los bordes.
    if (event.clientY < 48) {
      window.scrollBy(0, -12);
    } else if (event.clientY > window.innerHeight - 48) {
      window.scrollBy(0, 12);
    }

    var others = items().filter(function (el) {
      return el !== drag.task;
    });
    var index = others.length;
    for (var i = 0; i < others.length; i++) {
      var rect = others[i].getBoundingClientRect();
      if (event.clientY < rect.top + rect.height / 2) {
        index = i;
        break;
      }
    }
    if (index !== indexOf(drag.task)) {
      moveTo(drag.task, index);
    }
  });

  list.addEventListener("pointerup", function () {
    endDrag(false);
  });

  list.addEventListener("pointercancel", function () {
    endDrag(true);
  });

  document.addEventListener("keydown", function (event) {
    if (drag && event.key === "Escape") {
      endDrag(true);
    }
  });

  list.addEventListener("keydown", function (event) {
    var handle = event.target.closest("[data-reorder-handle]");
    if (!handle || drag || (event.key !== "ArrowUp" && event.key !== "ArrowDown")) {
      return;
    }
    event.preventDefault();
    var task = handle.closest(".task");
    var start = indexOf(task);
    var target = start + (event.key === "ArrowUp" ? -1 : 1);
    if (target < 0 || target >= items().length) {
      return;
    }
    moveTo(task, target);
    handle.focus();
    persist(task, start);
  });
})();
