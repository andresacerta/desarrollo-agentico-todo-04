package models

import (
	"fmt"
	"strings"
	"time"

	"github.com/gobuffalo/pop/v6"
	"github.com/gobuffalo/validate/v3"
	"github.com/gofrs/uuid"
)

// TitleMaxLength es el número máximo de caracteres permitido en el título.
const TitleMaxLength = 200

// TaskOrder es el orden en que se muestra la lista de tareas.
const TaskOrder = "position asc, created_at asc, id asc"

// Task representa una tarea de la lista.
type Task struct {
	ID        uuid.UUID `json:"id" db:"id" form:"-"`
	Title     string    `json:"title" db:"title" form:"title"`
	Completed bool      `json:"completed" db:"completed" form:"-"`
	Position  int       `json:"position" db:"position" form:"-"`
	CreatedAt time.Time `json:"created_at" db:"created_at" form:"-"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at" form:"-"`
}

// Tasks es una colección de Task.
type Tasks []Task

// Validate se ejecuta en cada llamada a los métodos "pop.Validate*".
func (t *Task) Validate(tx *pop.Connection) (*validate.Errors, error) {
	verrs := validate.NewErrors()

	title := strings.TrimSpace(t.Title)
	if title == "" {
		verrs.Add("title", "El título es obligatorio.")
	} else if len([]rune(title)) > TitleMaxLength {
		verrs.Add("title", fmt.Sprintf("El título no puede superar los %d caracteres.", TitleMaxLength))
	}

	return verrs, nil
}

// BeforeCreate coloca las tareas nuevas al final de la lista.
func (t *Task) BeforeCreate(tx *pop.Connection) error {
	if t.Position > 0 {
		return nil
	}
	return tx.Store.Get(&t.Position, "SELECT COALESCE(MAX(position), 0) + 1 FROM tasks")
}

// MoveTo coloca la tarea en la posición target (empezando en 1) y renumera
// la lista de forma consecutiva. Las posiciones fuera de rango se acotan.
func (t *Task) MoveTo(tx *pop.Connection, target int) error {
	tasks := Tasks{}
	// FOR UPDATE serializa reordenaciones concurrentes dentro de la transacción de la petición.
	if err := tx.RawQuery("SELECT * FROM tasks ORDER BY " + TaskOrder + " FOR UPDATE").All(&tasks); err != nil {
		return err
	}

	ordered := make(Tasks, 0, len(tasks))
	var moved *Task
	for i := range tasks {
		if tasks[i].ID == t.ID {
			moved = &tasks[i]
			continue
		}
		ordered = append(ordered, tasks[i])
	}
	if moved == nil {
		return fmt.Errorf("la tarea %s no existe", t.ID)
	}

	target = max(1, min(target, len(tasks)))
	ordered = append(ordered[:target-1], append(Tasks{*moved}, ordered[target-1:]...)...)

	for i, task := range ordered {
		position := i + 1
		if task.Position == position {
			continue
		}
		if err := tx.RawQuery("UPDATE tasks SET position = ? WHERE id = ?", position, task.ID).Exec(); err != nil {
			return err
		}
	}

	t.Position = target
	return nil
}
