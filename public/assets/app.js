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
          { duration: 200, easing: "cubic-bezier(0.2, 0, 0, 1)" }
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
        flashSaved(task);
        statusBox.textContent = title + ": posición " + position + " de " + total + ".";
      })
      .catch(function () {
        moveTo(task, originalIndex);
        showError("No se ha podido guardar el nuevo orden. Recarga la página e inténtalo de nuevo.");
      });
  }

  // Distancia mínima antes de empezar a arrastrar: así un clic en el asa no mueve nada.
  var DRAG_THRESHOLD = 4;

  // Posición vertical de la tarea sin el desplazamiento que le aplicamos al arrastrarla.
  function naturalTop(task) {
    var transform = task.style.transform;
    task.style.transform = "";
    var top = task.getBoundingClientRect().top;
    task.style.transform = transform;
    return top;
  }

  // Mantiene la tarea pegada al puntero aunque ya haya cambiado de sitio en la lista.
  function follow(clientY) {
    var offset = clientY - drag.grabOffset - naturalTop(drag.task);
    drag.task.style.transform = "translateY(" + offset + "px)";
  }

  function targetIndex(clientY) {
    var others = items().filter(function (el) {
      return el !== drag.task;
    });
    var center = clientY - drag.grabOffset + drag.height / 2;
    for (var i = 0; i < others.length; i++) {
      var rect = others[i].getBoundingClientRect();
      if (center < rect.top + rect.height / 2) {
        return i;
      }
    }
    return others.length;
  }

  function startDrag() {
    drag.active = true;
    drag.task.classList.add("task-dragging");
    list.classList.add("task-list-dragging");
    clearError();
  }

  // Devuelve la tarea a su hueco con una transición corta en vez de un salto.
  function settle(task) {
    task.classList.remove("task-dragging");
    if (reduceMotion.matches || !task.style.transform) {
      task.style.transform = "";
      return;
    }
    task.classList.add("task-settling");
    // Fuerza el estilo actual para que la transición parta de la posición arrastrada.
    task.getBoundingClientRect();
    task.style.transform = "";
    window.setTimeout(function () {
      task.classList.remove("task-settling");
    }, 200);
  }

  function flashSaved(task) {
    if (reduceMotion.matches) {
      return;
    }
    task.classList.remove("task-saved");
    task.getBoundingClientRect();
    task.classList.add("task-saved");
  }

  function endDrag(cancelled) {
    if (!drag) {
      return;
    }
    var current = drag;
    drag = null;
    list.classList.remove("task-list-dragging");
    if (!current.active) {
      return;
    }

    if (cancelled) {
      moveTo(current.task, current.start);
      settle(current.task);
      return;
    }
    settle(current.task);
    if (indexOf(current.task) !== current.start) {
      persist(current.task, current.start);
    }
  }

  list.addEventListener("pointerdown", function (event) {
    var handle = event.target.closest("[data-reorder-handle]");
    if (!handle || event.button !== 0 || items().length < 2) {
      return;
    }
    event.preventDefault();
    handle.setPointerCapture(event.pointerId);
    var task = handle.closest(".task");
    var rect = task.getBoundingClientRect();
    drag = {
      task: task,
      start: indexOf(task),
      startY: event.clientY,
      grabOffset: event.clientY - rect.top,
      height: rect.height,
      active: false,
    };
  });

  list.addEventListener("pointermove", function (event) {
    if (!drag) {
      return;
    }
    if (!drag.active) {
      if (Math.abs(event.clientY - drag.startY) < DRAG_THRESHOLD) {
        return;
      }
      startDrag();
    }

    // Desplaza la página cuando el puntero se acerca a los bordes.
    if (event.clientY < 48) {
      window.scrollBy(0, -12);
    } else if (event.clientY > window.innerHeight - 48) {
      window.scrollBy(0, 12);
    }

    var index = targetIndex(event.clientY);
    if (index !== indexOf(drag.task)) {
      moveTo(drag.task, index);
    }
    follow(event.clientY);
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
