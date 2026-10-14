package actions

import (
	"net/url"
	"strings"

	"todo/models"

	"github.com/gofrs/uuid"
)

func (as *ActionSuite) Test_TasksIndex_EmptyState() {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "No hay tareas todavía")
}

func (as *ActionSuite) Test_TasksIndex_ListsTasks() {
	as.LoadFixture("tres tareas")

	res := as.HTML("/").Get()
	as.Equal(200, res.Code)

	body := res.Body.String()
	as.Contains(body, "Comprar material de oficina")
	as.Contains(body, "Llamar al gestor")
	as.Contains(body, "Publicar la oferta de empleo")
	as.Contains(body, "Completada")
}

func (as *ActionSuite) Test_TasksCreate_Valid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "  Nueva tarea de prueba  "})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))

	task := &models.Task{}
	as.NoError(as.DB.First(task))
	as.Equal("Nueva tarea de prueba", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksCreate_EmptyTitle_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: ""})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksCreate_OnlySpaces_Invalid() {
	res := as.HTML("/tasks").Post(&models.Task{Title: "     "})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksToggle_CompletesAndReopens() {
	task := &models.Task{Title: "Pendiente de completar"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)

	res = as.HTML("/tasks/%s/toggle", task.ID).Put(url.Values{})
	as.Equal(303, res.Code)

	as.NoError(as.DB.Reload(task))
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksToggle_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s/toggle", id).Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksToggle_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/toggle").Put(url.Values{})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksDestroy() {
	task := &models.Task{Title: "Tarea que se elimina"}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())

	res := as.HTML("/tasks/%s", task.ID).Delete()
	as.Equal(303, res.Code)

	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(0, count)
}

func (as *ActionSuite) Test_TasksDestroy_NotFound() {
	id := uuid.Must(uuid.NewV4())
	res := as.HTML("/tasks/%s", id).Delete()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) taskForEditing(completed bool) *models.Task {
	task := &models.Task{Title: "Título original", Completed: completed}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	as.NoError(as.DB.Reload(task))
	return task
}

func (as *ActionSuite) Test_TasksIndex_EditLink() {
	task := as.taskForEditing(false)
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), "/tasks/"+task.ID.String()+"/edit")
	as.Contains(res.Body.String(), "Editar tarea: Título original")
}

func (as *ActionSuite) Test_TasksEdit_PrefillsTitle() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `value="Título original"`)
	as.Contains(res.Body.String(), `value="PUT"`)
	as.Contains(res.Body.String(), "Guardar cambios")
	as.Contains(res.Body.String(), "Cancelar")
	as.NoError(as.DB.Reload(task))
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksEdit_NotFound() {
	res := as.HTML("/tasks/%s/edit", uuid.Must(uuid.NewV4())).Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid/edit").Get()
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksEdit_EscapesTitle() {
	task := &models.Task{Title: `<img src=x onerror="alert(1)">`}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	res := as.HTML("/tasks/%s/edit", task.ID).Get()
	as.Equal(200, res.Code)
	as.NotContains(res.Body.String(), `<img src=x`)
	as.Contains(res.Body.String(), "&lt;img")
}

func (as *ActionSuite) Test_TasksUpdate_PersistsTitleOnly() {
	task := as.taskForEditing(true)
	id, createdAt, updatedAt := task.ID, task.CreatedAt, task.UpdatedAt
	res := as.HTML("/tasks/%s", id).Put(url.Values{
		"title": {"  Título actualizado  "}, "completed": {"false"},
		"id": {uuid.Must(uuid.NewV4()).String()}, "created_at": {"2000-01-01T00:00:00Z"},
	})
	as.Equal(303, res.Code)
	as.Equal("/", res.Header().Get("Location"))
	as.NoError(as.DB.Reload(task))
	as.Equal("Título actualizado", task.Title)
	as.Equal(id, task.ID)
	as.True(task.Completed)
	as.True(task.CreatedAt.Equal(createdAt))
	as.False(task.UpdatedAt.Before(updatedAt))
	count, err := as.DB.Count(&models.Task{})
	as.NoError(err)
	as.Equal(1, count)
}

func (as *ActionSuite) Test_TasksUpdate_PendingTaskRemainsPending() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"Nuevo título"}, "completed": {"true"}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal("Nuevo título", task.Title)
	as.False(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_EmptyTitle() {
	task := as.taskForEditing(false)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {""}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.Contains(res.Body.String(), `aria-describedby="title-error"`)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_OnlySpaces() {
	task := as.taskForEditing(true)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {"   "}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título es obligatorio.")
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
	as.True(task.Completed)
}

func (as *ActionSuite) Test_TasksUpdate_TitleTooLong() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength+1)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(422, res.Code)
	as.Contains(res.Body.String(), "El título no puede superar los 200 caracteres.")
	as.Contains(res.Body.String(), title)
	as.NoError(as.DB.Reload(task))
	as.Equal("Título original", task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_TitleAtLimit() {
	task := as.taskForEditing(false)
	title := strings.Repeat("á", models.TitleMaxLength)
	res := as.HTML("/tasks/%s", task.ID).Put(url.Values{"title": {title}})
	as.Equal(303, res.Code)
	as.NoError(as.DB.Reload(task))
	as.Equal(title, task.Title)
}

func (as *ActionSuite) Test_TasksUpdate_NotFound() {
	res := as.HTML("/tasks/%s", uuid.Must(uuid.NewV4())).Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksUpdate_InvalidID() {
	res := as.HTML("/tasks/no-es-un-uuid").Put(url.Values{"title": {"Título nuevo"}})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) createTask(title string) *models.Task {
	task := &models.Task{Title: title}
	verrs, err := as.DB.ValidateAndCreate(task)
	as.NoError(err)
	as.False(verrs.HasAny())
	return task
}

// assertOrder comprueba que los títulos aparecen en el cuerpo en el orden indicado.
func (as *ActionSuite) assertOrder(body string, titles ...string) {
	previous := -1
	for _, title := range titles {
		index := strings.Index(body, title)
		as.Greater(index, previous, "%q no aparece en el orden esperado", title)
		previous = index
	}
}

func (as *ActionSuite) indexBody() string {
	res := as.HTML("/").Get()
	as.Equal(200, res.Code)
	return res.Body.String()
}

func (as *ActionSuite) Test_TasksIndex_OrdersByPosition() {
	first := as.createTask("Primera tarea")
	second := as.createTask("Segunda tarea")
	third := as.createTask("Tercera tarea")
	positions := map[*models.Task]int{third: 1, first: 2, second: 3}
	for task, position := range positions {
		as.NoError(as.DB.RawQuery("UPDATE tasks SET position = ? WHERE id = ?", position, task.ID).Exec())
	}

	as.assertOrder(as.indexBody(), "Tercera tarea", "Primera tarea", "Segunda tarea")
}

func (as *ActionSuite) Test_TasksMove_PersistsOrder() {
	as.createTask("Primera tarea")
	as.createTask("Segunda tarea")
	third := as.createTask("Tercera tarea")

	res := as.JSON("/tasks/%s/move", third.ID).Put(map[string]interface{}{"position": 1})
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `"position":1`)

	as.assertOrder(as.indexBody(), "Tercera tarea", "Primera tarea", "Segunda tarea")
}

func (as *ActionSuite) Test_TasksMove_ClampsBeyondEnd() {
	first := as.createTask("Primera tarea")
	as.createTask("Segunda tarea")
	as.createTask("Tercera tarea")

	res := as.JSON("/tasks/%s/move", first.ID).Put(map[string]interface{}{"position": 99})
	as.Equal(200, res.Code)
	as.Contains(res.Body.String(), `"position":3`)

	as.assertOrder(as.indexBody(), "Segunda tarea", "Tercera tarea", "Primera tarea")
}

func (as *ActionSuite) Test_TasksMove_InvalidPosition() {
	as.createTask("Primera tarea")
	as.createTask("Segunda tarea")
	third := as.createTask("Tercera tarea")

	bodies := []map[string]interface{}{
		{"position": 0},
		{},
		{"position": "abc"},
	}
	for _, body := range bodies {
		res := as.JSON("/tasks/%s/move", third.ID).Put(body)
		as.Equal(422, res.Code, "cuerpo %v", body)
		as.Contains(res.Body.String(), "La posición debe ser un número entero mayor que cero.")
	}

	as.assertOrder(as.indexBody(), "Primera tarea", "Segunda tarea", "Tercera tarea")
}

func (as *ActionSuite) Test_TasksMove_NotFound() {
	res := as.JSON("/tasks/%s/move", uuid.Must(uuid.NewV4())).Put(map[string]interface{}{"position": 1})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksMove_InvalidID() {
	res := as.JSON("/tasks/no-es-un-uuid/move").Put(map[string]interface{}{"position": 1})
	as.Equal(404, res.Code)
}

func (as *ActionSuite) Test_TasksCreate_AppendsAtEndAfterReorder() {
	as.createTask("Primera tarea")
	second := as.createTask("Segunda tarea")

	res := as.JSON("/tasks/%s/move", second.ID).Put(map[string]interface{}{"position": 1})
	as.Equal(200, res.Code)

	created := as.HTML("/tasks").Post(&models.Task{Title: "Tercera tarea"})
	as.Equal(303, created.Code)

	as.assertOrder(as.indexBody(), "Segunda tarea", "Primera tarea", "Tercera tarea")
}

func (as *ActionSuite) Test_TasksIndex_RendersReorderHandle() {
	task := as.createTask("Primera tarea")

	body := as.indexBody()
	as.Contains(body, "data-reorder-handle")
	as.Contains(body, "Mover tarea: Primera tarea")
	as.Contains(body, `data-move-url="/tasks/`+task.ID.String()+`/move"`)
	as.Contains(body, "data-reorder-token")
}
