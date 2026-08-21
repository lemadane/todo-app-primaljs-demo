import {
   html,
   signal,
   mount,
   onMount
} from "/vendor/primaljs/primal.js";

const todos = signal([]);
const newTitle = signal("");
const message = signal("");

function createTodoModel(todo) {
   return {
      id: todo.id,
      title: signal(todo.title),
      completed: signal(todo.completed)
   };
}

async function loadTodos() {
   const response = await fetch("/api/todos");
   if (!response.ok) {
      throw new Error("Unable to load todos");
   }
   const data = await response.json();
   todos.value = data.map(createTodoModel);
}


async function addTodo(event) {
   event.preventDefault();
   const title = newTitle.value.trim();
   if (!title) {
      return;
   }
   const response = await fetch("/api/todos", {
      method: "POST",
      headers: {
         "Content-Type": "application/json"
      },
      body: JSON.stringify({
         title
      })
   });
   if (!response.ok) {
      message.value = "Unable to create todo";
      return;
   }
   const createdTodo = await response.json();
   todos.value = [
      ...todos.value,
      createTodoModel(createdTodo)
   ];
   newTitle.value = "";
   message.value = "";
}

async function toggleTodo(todo) {
   const completed = !todo.completed.value;
   const response = await fetch(`/api/todos/${todo.id}`,
      {
         method: "PATCH",
         headers: {
            "Content-Type": "application/json"
         },
         body: JSON.stringify({
            completed
         })
      });
   if (!response.ok) {
      message.value = "Unable to update todo";
      return;
   }
   const updatedTodo = await response.json();
   todo.completed.value = updatedTodo.completed;
   message.value = "";
}

async function deleteTodo(todo) {
   const response = await fetch(`/api/todos/${todo.id}`, {
      method: "DELETE"
   });
   if (!response.ok) {
      message.value = "Unable to delete todo";
      return;
   }
   todos.value = todos.value.filter(
      currentTodo => currentTodo.id !== todo.id
   );
   message.value = "";
}

function App() {
   onMount(() => {
      loadTodos().catch(error => {
         console.error(error);
         message.value = "Unable to load todos";
      });
   });
   return html`
         <main class="todo-app">
            <h1>Todo</h1>
            <form onsubmit=${addTodo}>
                <input
                    type="text"
                    placeholder="What needs to be done?"
                    value=${newTitle}
                    oninput=${event => newTitle.value = event.target.value}
                >
                <button type="submit">Add</button>
            </form>
            <p class="message">${message}</p>
            <ul class="todo-list">
                <For each=${todos} key=${todo => todo.id}>
                    ${todo => html`
                        <li class="todo">
                           <input
                                type="checkbox"
                                checked=${todo.completed.value}
                                onchange=${() => toggleTodo(todo)}
                           >
                           <span>${todo.title}</span>
                           <button type="button" onclick=${() => deleteTodo(todo)}>
                              Delete
                           </button>
                        </li>
                    `}
                  <Else>
                     <li>No todos yet.</li>
                  </Else>
               </For>
            </ul>
        </main>
    `;
}

mount(App, "#app");