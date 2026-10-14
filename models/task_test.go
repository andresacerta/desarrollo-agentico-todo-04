package models

import "strings"

func (ms *ModelSuite) Test_Task_Create_Valid() {
	task := &Task{Title: "Comprar café"}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	ms.NotZero(task.ID)
	ms.False(task.Completed)

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(1, count)
}

func (ms *ModelSuite) Test_Task_Title_Required() {
	task := &Task{Title: ""}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")

	count, err := ms.DB.Count(&Task{})
	ms.NoError(err)
	ms.Equal(0, count)
}

func (ms *ModelSuite) Test_Task_Title_OnlySpaces_Invalid() {
	task := &Task{Title: "   \t  "}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.Contains(verrs.Get("title"), "El título es obligatorio.")
}

func (ms *ModelSuite) Test_Task_Title_MaxLength() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
}

func (ms *ModelSuite) Test_Task_Title_TooLong_Invalid() {
	task := &Task{Title: strings.Repeat("a", TitleMaxLength+1)}

	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.True(verrs.HasAny())
	ms.NotEmpty(verrs.Get("title"))
}

func (ms *ModelSuite) Test_Task_Toggle_Persists() {
	task := &Task{Title: "Revisar el correo"}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())

	task.Completed = true
	ms.NoError(ms.DB.Update(task))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, task.ID))
	ms.True(reloaded.Completed)
}

func (ms *ModelSuite) createTask(title string) *Task {
	task := &Task{Title: title}
	verrs, err := ms.DB.ValidateAndCreate(task)
	ms.NoError(err)
	ms.False(verrs.HasAny())
	return task
}

func (ms *ModelSuite) orderedTasks() Tasks {
	tasks := Tasks{}
	ms.NoError(ms.DB.Order(TaskOrder).All(&tasks))
	return tasks
}

func (ms *ModelSuite) orderedTitles() []string {
	titles := []string{}
	for _, task := range ms.orderedTasks() {
		titles = append(titles, task.Title)
	}
	return titles
}

func (ms *ModelSuite) orderedPositions() []int {
	positions := []int{}
	for _, task := range ms.orderedTasks() {
		positions = append(positions, task.Position)
	}
	return positions
}

func (ms *ModelSuite) Test_Task_Create_AppendsAtEnd() {
	a := ms.createTask("A")
	b := ms.createTask("B")
	c := ms.createTask("C")

	ms.Equal(1, a.Position)
	ms.Equal(2, b.Position)
	ms.Equal(3, c.Position)
}

func (ms *ModelSuite) Test_Task_MoveTo_Up() {
	ms.createTask("A")
	ms.createTask("B")
	c := ms.createTask("C")

	ms.NoError(c.MoveTo(ms.DB, 1))

	ms.Equal(1, c.Position)
	ms.Equal([]string{"C", "A", "B"}, ms.orderedTitles())
	ms.Equal([]int{1, 2, 3}, ms.orderedPositions())
}

func (ms *ModelSuite) Test_Task_MoveTo_Down() {
	a := ms.createTask("A")
	ms.createTask("B")
	ms.createTask("C")

	ms.NoError(a.MoveTo(ms.DB, 3))

	ms.Equal(3, a.Position)
	ms.Equal([]string{"B", "C", "A"}, ms.orderedTitles())
	ms.Equal([]int{1, 2, 3}, ms.orderedPositions())
}

func (ms *ModelSuite) Test_Task_MoveTo_ClampsOutOfRange() {
	a := ms.createTask("A")
	ms.createTask("B")
	c := ms.createTask("C")

	ms.NoError(a.MoveTo(ms.DB, 99))
	ms.Equal(3, a.Position)
	ms.Equal([]string{"B", "C", "A"}, ms.orderedTitles())

	ms.NoError(c.MoveTo(ms.DB, 0))
	ms.Equal(1, c.Position)
	ms.Equal([]string{"C", "B", "A"}, ms.orderedTitles())
	ms.Equal([]int{1, 2, 3}, ms.orderedPositions())
}

func (ms *ModelSuite) Test_Task_MoveTo_NormalizesGaps() {
	a := ms.createTask("A")
	b := ms.createTask("B")
	c := ms.createTask("C")
	for i, task := range []*Task{a, b, c} {
		ms.NoError(ms.DB.RawQuery("UPDATE tasks SET position = ? WHERE id = ?", (i+1)*10, task.ID).Exec())
	}

	ms.NoError(c.MoveTo(ms.DB, 1))

	ms.Equal([]string{"C", "A", "B"}, ms.orderedTitles())
	ms.Equal([]int{1, 2, 3}, ms.orderedPositions())
}

func (ms *ModelSuite) Test_Task_Create_AfterReorder_AppendsAtEnd() {
	ms.createTask("A")
	b := ms.createTask("B")
	ms.NoError(b.MoveTo(ms.DB, 1))

	ms.createTask("D")

	ms.Equal([]string{"B", "A", "D"}, ms.orderedTitles())
}

func (ms *ModelSuite) Test_Task_MoveTo_DoesNotChangeOtherFields() {
	ms.createTask("A")
	b := ms.createTask("B")
	b.Completed = true
	ms.NoError(ms.DB.Update(b))
	ms.NoError(ms.DB.Reload(b))
	title, completed, updatedAt := b.Title, b.Completed, b.UpdatedAt

	ms.NoError(b.MoveTo(ms.DB, 1))

	reloaded := &Task{}
	ms.NoError(ms.DB.Find(reloaded, b.ID))
	ms.Equal(1, reloaded.Position)
	ms.Equal(title, reloaded.Title)
	ms.Equal(completed, reloaded.Completed)
	ms.True(reloaded.UpdatedAt.Equal(updatedAt))
}
