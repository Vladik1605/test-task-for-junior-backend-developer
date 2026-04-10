package transporthttp

import (
	"net/http"

	"github.com/gorilla/mux"

	swaggerdocs "example.com/taskservice/internal/transport/http/docs"
	httphandlers "example.com/taskservice/internal/transport/http/handlers"
)

func NewRouter(taskHandler *httphandlers.TaskHandler, templateHandler *httphandlers.TemplateHandler, docsHandler *swaggerdocs.Handler) *mux.Router {
	router := mux.NewRouter().StrictSlash(true)

	router.HandleFunc("/swagger/openapi.json", docsHandler.ServeSpec).Methods(http.MethodGet)
	router.HandleFunc("/swagger/", docsHandler.ServeUI).Methods(http.MethodGet)
	router.HandleFunc("/swagger", docsHandler.RedirectToUI).Methods(http.MethodGet)

	api := router.PathPrefix("/api/v1").Subrouter()

	// Tasks
	api.HandleFunc("/tasks", taskHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/tasks", taskHandler.List).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.Update).Methods(http.MethodPut)
	api.HandleFunc("/tasks/{id:[0-9]+}", taskHandler.Delete).Methods(http.MethodDelete)

	// Task Templates (Recurrence)
	api.HandleFunc("/task-templates", templateHandler.Create).Methods(http.MethodPost)
	api.HandleFunc("/task-templates", templateHandler.List).Methods(http.MethodGet)
	api.HandleFunc("/task-templates/{id:[0-9]+}", templateHandler.GetByID).Methods(http.MethodGet)
	api.HandleFunc("/task-templates/{id:[0-9]+}", templateHandler.Update).Methods(http.MethodPut)
	api.HandleFunc("/task-templates/{id:[0-9]+}", templateHandler.Delete).Methods(http.MethodDelete)

	// Manual generation endpoints for testing
	api.HandleFunc("/task-templates/generate", templateHandler.GenerateForDate).Methods(http.MethodPost)
	api.HandleFunc("/task-templates/run-worker", templateHandler.RunWorkerNow).Methods(http.MethodPost)
	api.HandleFunc("/task-templates/force-advance", templateHandler.ForceAdvanceAll).Methods(http.MethodPost)

	return router
}
